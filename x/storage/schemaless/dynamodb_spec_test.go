package schemaless_test

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/localstackutil"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/spec"
)

func dynamoDBClient(t *testing.T) *dynamodb.Client {
	t.Helper()
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

	return dynamodb.NewFromConfig(awscfg)
}

// dynamoDBCapabilities: DynamoDB downgrades two capabilities.
//   - SortByDataField: DynamoDB cannot sort on attributes outside the key.
//   - BackwardPagination: scans only produce a forward cursor.
func dynamoDBCapabilities() spec.Capabilities {
	return spec.FullCapabilities().
		WithoutSortByDataField().
		WithoutBackwardPagination()
}

func TestDynamoDBRepositorySpec(t *testing.T) {
	client := dynamoDBClient(t)
	const tableName = "spec-test-repo-record"
	require.NoError(t, recreateDynamoDBTable(client, tableName), "while setting up dynamodb table")

	spec.RunRepositorySpec(t,
		func(t *testing.T) spec.Repo {
			// the suite namespaces record types, so one shared table is fine
			return schemaless.NewDynamoDBRepository[spec.Data](client, tableName)
		},
		dynamoDBCapabilities(),
	)
}

func TestDynamoDBComplexQuerySpec(t *testing.T) {
	client := dynamoDBClient(t)
	// a dedicated table: every payload type gets its own namespace, so an
	// unfiltered query never decodes foreign payloads
	const tableName = "spec-test-vehicle-record"
	require.NoError(t, recreateDynamoDBTable(client, tableName), "while setting up dynamodb table")

	spec.RunComplexQuerySpec(t,
		func(t *testing.T) spec.ComplexRepo {
			return schemaless.NewDynamoDBRepository[spec.Vehicle](client, tableName)
		},
		dynamoDBCapabilities(),
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
