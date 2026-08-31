package schemaless

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/localstackutil"
)

// DynamoDB-specific regression: pagination must terminate when the record the
// scan cursor points at is deleted between pages. This is not part of the
// cross-backend spec suite (x/storage/schemaless/spec), because the in-memory
// repository's ID-position cursor cannot honour it.
func TestDynamoDBPaginationSurvivesDeletedCursor(t *testing.T) {
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
	_, err = repo.UpdateRecords(exampleUpdateRecords)
	require.NoError(t, err, "while saving example records")

	page, err := repo.FindingRecords(FindingRecords[Record[ExampleRecord]]{
		RecordType: "ExampleRecord",
		Limit:      2,
	})
	require.NoError(t, err, "while finding records (first page)")

	// the scan cursor points at the last scanned item of the page;
	// delete it before asking for the next page
	require.NotEmpty(t, page.Items)
	last := page.Items[len(page.Items)-1]
	_, err = repo.UpdateRecords(Delete(last))
	require.NoError(t, err, "while deleting cursor record")

	found := append([]Record[ExampleRecord]{}, page.Items...)
	const maxPages = 20
	pages := 1
	for page.HasNext() {
		pages++
		require.LessOrEqual(t, pages, maxPages, "pagination should terminate, possible infinite loop")

		page, err = repo.FindingRecords(*page.Next)
		require.NoError(t, err, "while finding records (page %d)", pages)
		found = append(found, page.Items...)
	}

	assert.Len(t, found, 5, "first page still contains the deleted record, later pages the rest")
}

func setupDynamoDB(d *dynamodb.Client, tableName string) error {
	// clean database, if exists
	_, _ = d.DeleteTable(context.TODO(), &dynamodb.DeleteTableInput{
		TableName: &tableName,
	})

	_, err := d.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("ID"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("Type"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("ID"),
				KeyType:       types.KeyTypeHash,
			},
			{
				AttributeName: aws.String("Type"),
				KeyType:       types.KeyTypeRange,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
		TableName:   &tableName,
	})

	return err
}
