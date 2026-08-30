package schemaless

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/localstackutil"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
)

// Integration repro for DynamoDB query bugs:
//   - empty filter broke "list all" (ValidationException)
//   - Not{Or{a,b}} rendered as `NOT a OR b` (no parentheses)
//   - pagination cursors dropped the record type
//   - pagination on a deleted cursor record must terminate
func TestDynamoDBQueryBugs(t *testing.T) {
	address := os.Getenv("AWS_ENDPOINT_URL")
	if address == "" {
		t.Skip(`Skipping test because:
- AWS_ENDPOINT_URL that points to localstack is not set.
- Assuming localstack is not running.

To run this test, please set AWS_ENDPOINT_URL to the address of your localstack, like:
	export AWS_ENDPOINT_URL=http://localhost:4566
`)
	}

	tableName := "test-query-bugs"

	awscfg, err := localstackutil.LoadLocalStackAwsConfig(context.Background())
	require.NoError(t, err, "while loading localstack config")

	d := dynamodb.NewFromConfig(awscfg)
	require.NoError(t, setupDynamoDB(d, tableName), "while setting up dynamodb")

	repo := NewDynamoDBRepository[ExampleRecord](d, tableName)

	// 5 records of type ExampleRecord + 2 records of another type
	otherRecords := Save(
		Record[ExampleRecord]{ID: "o1", Type: "OtherRecord", Data: ExampleRecord{Name: "Olga", Age: 50}},
		Record[ExampleRecord]{ID: "o2", Type: "OtherRecord", Data: ExampleRecord{Name: "Omar", Age: 51}},
	)

	_, err = repo.UpdateRecords(exampleUpdateRecords)
	require.NoError(t, err, "while saving example records")
	_, err = repo.UpdateRecords(otherRecords)
	require.NoError(t, err, "while saving other records")

	// collectAllPages follows Next cursors with a loop guard, so a
	// pagination bug fails the test instead of hanging it.
	collectAllPages := func(t *testing.T, query FindingRecords[Record[ExampleRecord]], mutateAfterFirstPage func(page PageResult[Record[ExampleRecord]])) []Record[ExampleRecord] {
		t.Helper()
		var found []Record[ExampleRecord]
		page, err := repo.FindingRecords(query)
		require.NoError(t, err, "while finding records (first page)")
		if mutateAfterFirstPage != nil {
			mutateAfterFirstPage(page)
		}

		const maxPages = 20
		pages := 0
		for {
			pages++
			require.LessOrEqual(t, pages, maxPages, "pagination should terminate, possible infinite loop")

			found = append(found, page.Items...)
			if !page.HasNext() {
				return found
			}

			page, err = repo.FindingRecords(*page.Next)
			require.NoError(t, err, "while finding records (page %d)", pages+1)
		}
	}

	t.Run("empty query lists all records", func(t *testing.T) {
		// Used to fail with ValidationException, because FilterExpression was set to "".
		found := collectAllPages(t, FindingRecords[Record[ExampleRecord]]{}, nil)
		assert.Len(t, found, 7, "empty query should return all records")
	})

	t.Run("pagination keeps the record type", func(t *testing.T) {
		// Next used to drop RecordType. With no Where clause the second page
		// then had an empty filter (ValidationException); with a Where clause
		// it leaked records of other types.
		found := collectAllPages(t, FindingRecords[Record[ExampleRecord]]{
			RecordType: "ExampleRecord",
			Limit:      2,
		}, nil)

		assert.Len(t, found, 5, "should find only records of requested type")
		for _, record := range found {
			assert.Equal(t, "ExampleRecord", record.Type)
		}
	})

	t.Run("NOT over OR is grouped", func(t *testing.T) {
		// NOT (ID = "123" OR ID = "124") used to render without parentheses,
		// which DynamoDB reads as (NOT ID = "123") OR ID = "124" —
		// and that wrongly includes ID = "124".
		found := collectAllPages(t, FindingRecords[Record[ExampleRecord]]{
			RecordType: "ExampleRecord",
			Where: &predicate.WherePredicates{
				Predicate: &predicate.Not{
					P: &predicate.Or{L: []predicate.Predicate{
						&predicate.Compare{Location: "ID", Operation: "=", BindValue: &predicate.BindValue{BindName: ":a"}},
						&predicate.Compare{Location: "ID", Operation: "=", BindValue: &predicate.BindValue{BindName: ":b"}},
					}},
				},
				Params: predicate.ParamBinds{
					":a": schema.MkString("123"),
					":b": schema.MkString("124"),
				},
			},
		}, nil)

		assert.Len(t, found, 3, "should exclude both records named in NOT (a OR b)")
		for _, record := range found {
			assert.NotEqual(t, "123", record.ID)
			assert.NotEqual(t, "124", record.ID)
		}
	})

	t.Run("pagination terminates when cursor record is deleted", func(t *testing.T) {
		var deletedID string
		found := collectAllPages(t, FindingRecords[Record[ExampleRecord]]{
			RecordType: "ExampleRecord",
			Limit:      2,
		}, func(page PageResult[Record[ExampleRecord]]) {
			// the scan cursor points at the last scanned item of the page;
			// delete it before asking for the next page
			require.NotEmpty(t, page.Items)
			last := page.Items[len(page.Items)-1]
			deletedID = last.ID
			_, err := repo.UpdateRecords(Delete(last))
			require.NoError(t, err, "while deleting cursor record")
		})

		assert.Len(t, found, 5, "first page still contains the deleted record, later pages the rest")
		assert.NotEmpty(t, deletedID)
	})
}
