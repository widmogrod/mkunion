package schemaless

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/localstackutil"
)

// Reproduces the missing SavingPolicy check in DynamoDBRepository.UpdateRecords:
// PolicyOverwriteServerChanges must let a writer holding a stale version win,
// the way InMemoryRepository does, instead of failing with ErrVersionConflict.
func TestDynamoDBRepository_SavingPolicy(t *testing.T) {
	address := os.Getenv("AWS_ENDPOINT_URL")
	if address == "" {
		t.Skip(`Skipping test because:
- AWS_ENDPOINT_URL that points to localstack is not set.
- Assuming localstack is not running.

To run this test, please set AWS_ENDPOINT_URL to the address of your localstack, like:
	export AWS_ENDPOINT_URL=http://localhost:4566
`)
	}

	tableName := "test-repo-saving-policy"

	awscfg, err := localstackutil.LoadLocalStackAwsConfig(context.Background())
	require.NoError(t, err, "while loading localstack config")

	d := dynamodb.NewFromConfig(awscfg)
	require.NoError(t, setupDynamoDB(d, tableName), "while setting up dynamodb")

	repo := NewDynamoDBRepository[ExampleRecord](d, tableName)

	// first write creates the record with version 1
	saved, err := repo.UpdateRecords(Save(Record[ExampleRecord]{
		ID:   "policy-1",
		Type: "ExampleRecord",
		Data: ExampleRecord{Name: "initial", Age: 10},
	}))
	require.NoError(t, err, "while saving initial record")
	stale := saved.Saved["policy-1"]

	// a concurrent writer bumps the server version to 2,
	// which makes the first writer's copy stale
	concurrent := stale
	concurrent.Data.Name = "concurrent"
	saved, err = repo.UpdateRecords(Save(concurrent))
	require.NoError(t, err, "while saving concurrent update")
	serverVersion := saved.Saved["policy-1"].Version

	t.Run("PolicyIfServerNotChanged rejects a stale write", func(t *testing.T) {
		rejected := stale
		rejected.Data.Name = "rejected"
		_, err := repo.UpdateRecords(UpdateRecords[Record[ExampleRecord]]{
			UpdatingPolicy: PolicyIfServerNotChanged,
			Saving: map[string]Record[ExampleRecord]{
				rejected.ID: rejected,
			},
		})
		assert.ErrorIs(t, err, ErrVersionConflict)
	})

	t.Run("PolicyOverwriteServerChanges lets a stale write win", func(t *testing.T) {
		overwrite := stale
		overwrite.Data.Name = "overwrite"
		_, err := repo.UpdateRecords(UpdateRecords[Record[ExampleRecord]]{
			UpdatingPolicy: PolicyOverwriteServerChanges,
			Saving: map[string]Record[ExampleRecord]{
				overwrite.ID: overwrite,
			},
		})
		require.NoError(t, err, "overwrite policy must not fail on version conflict")

		stored, err := repo.Get(overwrite.ID, overwrite.Type)
		require.NoError(t, err, "while reading record back")
		assert.Equal(t, "overwrite", stored.Data.Name, "stale write should overwrite server state")
		// the server increments Version itself, so the number keeps growing
		// no matter how stale the writer's copy was
		assert.Equal(t, serverVersion+1, stored.Version, "version should keep increasing monotonically")

		// a second stale overwrite must bump the version again
		overwrite.Data.Name = "overwrite-again"
		_, err = repo.UpdateRecords(UpdateRecords[Record[ExampleRecord]]{
			UpdatingPolicy: PolicyOverwriteServerChanges,
			Saving: map[string]Record[ExampleRecord]{
				overwrite.ID: overwrite,
			},
		})
		require.NoError(t, err, "second overwrite must not fail")

		stored, err = repo.Get(overwrite.ID, overwrite.Type)
		require.NoError(t, err, "while reading record back")
		assert.Equal(t, "overwrite-again", stored.Data.Name)
		assert.Equal(t, serverVersion+2, stored.Version, "each overwrite bumps the server-side version by exactly one")
	})
}
