package spec

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/localstackutil"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

// TestInMemoryRepositorySpec: the in-memory repository defines the full
// contract, so it runs with FullCapabilities and skips nothing.
func TestInMemoryRepositorySpec(t *testing.T) {
	RunRepositorySpec(t,
		func(t *testing.T) Repo {
			return schemaless.NewInMemoryRepository[Data]()
		},
		FullCapabilities(),
	)
}

// TestOpenSearchRepositorySpec: OpenSearch downgrades two capabilities.
//   - BackwardPagination: search_after cursors only move forward.
//   - AtomicBatch: each record is written in its own request, so a failing
//     batch can leave earlier records durably written.
func TestOpenSearchRepositorySpec(t *testing.T) {
	address := os.Getenv("OPENSEARCH_ADDRESS")
	if address == "" {
		t.Skip(`Skipping test because:
- OPENSEARCH_ADDRESS is not set.
- Assuming OpenSearch is not running.

To run this test, please set OPENSEARCH_ADDRESS to the address of your OpenSearch instance, like:
	export OPENSEARCH_ADDRESS=http://localhost:9200
`)
	}

	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{address},
	})
	require.NoError(t, err)

	const indexName = "spec-test-records-index"

	// searching an index that was never written to is an error, not an empty
	// result; one overwrite-policy write makes sure the index exists
	warmup := schemaless.NewOpenSearchRepository[Data](client, indexName)
	command := schemaless.Save(schemaless.Record[Data]{
		ID: "warmup", Type: "spec-warmup", Data: Data{Name: "Warmup", Age: 1},
	})
	command.UpdatingPolicy = schemaless.PolicyOverwriteServerChanges
	_, err = warmup.UpdateRecords(command)
	require.NoError(t, err, "while creating the shared test index")

	RunRepositorySpec(t,
		func(t *testing.T) Repo {
			// the suite namespaces record types, so one shared index is fine
			return schemaless.NewOpenSearchRepository[Data](client, indexName)
		},
		FullCapabilities().
			WithoutBackwardPagination().
			WithoutAtomicBatch(),
	)
}

// TestDynamoDBRepositorySpec: DynamoDB downgrades two capabilities.
//   - SortByDataField: DynamoDB cannot sort on attributes outside the key.
//   - BackwardPagination: scans only produce a forward cursor.
func TestDynamoDBRepositorySpec(t *testing.T) {
	address := os.Getenv("AWS_ENDPOINT_URL")
	if address == "" {
		t.Skip(`Skipping test because:
- AWS_ENDPOINT_URL that points to localstack is not set.
- Assuming localstack is not running.

To run this test, please set AWS_ENDPOINT_URL to the address of your localstack, like:
	export AWS_ENDPOINT_URL=http://localhost:4566
`)
	}

	awscfg, err := localstackutil.LoadLocalStackAwsConfig(context.Background())
	require.NoError(t, err, "while loading localstack config")

	client := dynamodb.NewFromConfig(awscfg)
	const tableName = "spec-test-repo-record"
	require.NoError(t, recreateDynamoDBTable(client, tableName), "while setting up dynamodb table")

	RunRepositorySpec(t,
		func(t *testing.T) Repo {
			// the suite namespaces record types, so one shared table is fine
			return schemaless.NewDynamoDBRepository[Data](client, tableName)
		},
		FullCapabilities().
			WithoutSortByDataField().
			WithoutBackwardPagination(),
	)
}

func recreateDynamoDBTable(d *dynamodb.Client, tableName string) error {
	_, _ = d.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})

	_, err := d.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("ID"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("Type"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("ID"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("Type"), KeyType: types.KeyTypeRange},
		},
		BillingMode: types.BillingModePayPerRequest,
		TableName:   aws.String(tableName),
	})
	return err
}
