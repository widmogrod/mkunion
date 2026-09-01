package schemaless

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
)

// fakeDynamoDB records inputs and replays scripted outputs.
type fakeDynamoDB struct {
	getItemOut *dynamodb.GetItemOutput
	getItemErr error
	getItemIn  *dynamodb.GetItemInput

	transactErr error
	transactIn  *dynamodb.TransactWriteItemsInput

	scanOut *dynamodb.ScanOutput
	scanErr error
	scanIn  *dynamodb.ScanInput
}

func (f *fakeDynamoDB) GetItem(_ context.Context, params *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	f.getItemIn = params
	return f.getItemOut, f.getItemErr
}

func (f *fakeDynamoDB) TransactWriteItems(_ context.Context, params *dynamodb.TransactWriteItemsInput, _ ...func(*dynamodb.Options)) (*dynamodb.TransactWriteItemsOutput, error) {
	f.transactIn = params
	return &dynamodb.TransactWriteItemsOutput{}, f.transactErr
}

func (f *fakeDynamoDB) Scan(_ context.Context, params *dynamodb.ScanInput, _ ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error) {
	f.scanIn = params
	return f.scanOut, f.scanErr
}

func dynamoRecord(t *testing.T, id string, version uint16) map[string]types.AttributeValue {
	t.Helper()
	rec := Record[schema.Schema]{
		ID:      id,
		Type:    "test",
		Data:    schema.MkString("payload-" + id),
		Version: version,
	}
	item := schema.ToDynamoDB(schema.FromGo[Record[schema.Schema]](rec))
	m, ok := item.(*types.AttributeValueMemberM)
	require.True(t, ok)
	return m.Value
}

func TestDynamoDBGet(t *testing.T) {
	t.Run("found record converts back", func(t *testing.T) {
		fake := &fakeDynamoDB{getItemOut: &dynamodb.GetItemOutput{Item: dynamoRecord(t, "1", 3)}}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		got, err := repo.Get("1", "test")
		require.NoError(t, err)
		assert.Equal(t, "1", got.ID)
		assert.Equal(t, uint16(3), got.Version)
		assert.Equal(t, schema.MkString("payload-1"), got.Data)

		require.NotNil(t, fake.getItemIn)
		assert.Equal(t, "table", *fake.getItemIn.TableName)
		assert.True(t, *fake.getItemIn.ConsistentRead, "reads must be consistent")
	})

	t.Run("empty item is ErrNotFound", func(t *testing.T) {
		fake := &fakeDynamoDB{getItemOut: &dynamodb.GetItemOutput{}}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		_, err := repo.Get("missing", "test")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("client error is ErrInternalError", func(t *testing.T) {
		fake := &fakeDynamoDB{getItemErr: errors.New("aws down")}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		_, err := repo.Get("1", "test")
		assert.ErrorIs(t, err, ErrInternalError)
	})
}

func TestDynamoDBUpdateRecords(t *testing.T) {
	record := func(id string, version uint16) Record[schema.Schema] {
		return Record[schema.Schema]{
			ID: id, Type: "test",
			Data:    schema.MkString("x"),
			Version: version,
		}
	}

	t.Run("empty command is rejected", func(t *testing.T) {
		repo := NewDynamoDBRepository[schema.Schema](&fakeDynamoDB{}, "table")
		_, err := repo.UpdateRecords(UpdateRecords[Record[schema.Schema]]{})
		assert.ErrorIs(t, err, ErrEmptyCommand)
	})

	t.Run("save issues a conditional put and bumps the version", func(t *testing.T) {
		fake := &fakeDynamoDB{}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		result, err := repo.UpdateRecords(Save(record("1", 4)))
		require.NoError(t, err)

		saved := result.Saved["1:test"]
		assert.Equal(t, uint16(5), saved.Version, "client-side version guess is incremented")

		require.NotNil(t, fake.transactIn)
		require.Len(t, fake.transactIn.TransactItems, 1)
		put := fake.transactIn.TransactItems[0].Put
		require.NotNil(t, put)
		assert.Contains(t, *put.ConditionExpression, "Version = :version")
	})

	t.Run("overwrite policy delegates the version bump to the server", func(t *testing.T) {
		fake := &fakeDynamoDB{}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		cmd := Save(record("1", 4))
		cmd.UpdatingPolicy = PolicyOverwriteServerChanges
		_, err := repo.UpdateRecords(cmd)
		require.NoError(t, err)

		require.Len(t, fake.transactIn.TransactItems, 1)
		update := fake.transactIn.TransactItems[0].Update
		require.NotNil(t, update)
		assert.Contains(t, *update.UpdateExpression, "if_not_exists(#version, :zero) + :one")
	})

	t.Run("delete issues delete items and reports them", func(t *testing.T) {
		fake := &fakeDynamoDB{}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		result, err := repo.UpdateRecords(Delete(record("1", 4)))
		require.NoError(t, err)
		assert.Contains(t, result.Deleted, "1:test")

		require.Len(t, fake.transactIn.TransactItems, 1)
		assert.NotNil(t, fake.transactIn.TransactItems[0].Delete)
	})

	t.Run("conditional check failure is a version conflict", func(t *testing.T) {
		fake := &fakeDynamoDB{
			transactErr: &types.TransactionCanceledException{
				CancellationReasons: []types.CancellationReason{
					{Code: aws.String("ConditionalCheckFailed")},
				},
			},
		}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		_, err := repo.UpdateRecords(Save(record("1", 4)))
		assert.ErrorIs(t, err, ErrVersionConflict)
	})

	t.Run("other transaction errors pass through", func(t *testing.T) {
		fake := &fakeDynamoDB{transactErr: errors.New("throttled")}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		_, err := repo.UpdateRecords(Save(record("1", 4)))
		assert.ErrorContains(t, err, "throttled")
	})
}

func TestDynamoDBFindingRecords(t *testing.T) {
	t.Run("scan results convert and filters carry the record type", func(t *testing.T) {
		fake := &fakeDynamoDB{scanOut: &dynamodb.ScanOutput{
			Items: []map[string]types.AttributeValue{
				dynamoRecord(t, "1", 1),
				dynamoRecord(t, "2", 2),
			},
		}}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		page, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{
			RecordType: "test",
			Limit:      10,
		})
		require.NoError(t, err)
		require.Len(t, page.Items, 2)
		assert.Nil(t, page.Next, "no LastEvaluatedKey means no next page")

		require.NotNil(t, fake.scanIn)
		assert.Contains(t, *fake.scanIn.FilterExpression, "#Type")
		assert.Equal(t, int32(10), *fake.scanIn.Limit)
	})

	t.Run("list-all sends no filter expression at all", func(t *testing.T) {
		fake := &fakeDynamoDB{scanOut: &dynamodb.ScanOutput{}}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		_, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{})
		require.NoError(t, err)
		assert.Nil(t, fake.scanIn.FilterExpression,
			"DynamoDB rejects empty filter expressions, so none must be sent")
	})

	t.Run("last evaluated key produces a next-page cursor that resumes", func(t *testing.T) {
		fake := &fakeDynamoDB{scanOut: &dynamodb.ScanOutput{
			Items: []map[string]types.AttributeValue{dynamoRecord(t, "1", 1)},
			LastEvaluatedKey: map[string]types.AttributeValue{
				"ID":   &types.AttributeValueMemberS{Value: "1"},
				"Type": &types.AttributeValueMemberS{Value: "test"},
			},
		}}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		page, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{RecordType: "test"})
		require.NoError(t, err)
		require.NotNil(t, page.Next)
		require.NotNil(t, page.Next.After)

		// feeding the cursor back resumes from the reported key
		fake.scanOut = &dynamodb.ScanOutput{}
		_, err = repo.FindingRecords(*page.Next)
		require.NoError(t, err)
		require.NotNil(t, fake.scanIn.ExclusiveStartKey)
		id, ok := fake.scanIn.ExclusiveStartKey["ID"].(*types.AttributeValueMemberS)
		require.True(t, ok)
		assert.Equal(t, "1", id.Value)
	})

	t.Run("malformed after cursor errors", func(t *testing.T) {
		fake := &fakeDynamoDB{}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		after := "{not-json"
		_, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{After: &after})
		assert.ErrorContains(t, err, "after cursor")
	})

	t.Run("scan error passes through", func(t *testing.T) {
		fake := &fakeDynamoDB{scanErr: errors.New("throttled")}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		_, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{})
		assert.ErrorContains(t, err, "throttled")
	})

	t.Run("unconvertible item errors", func(t *testing.T) {
		fake := &fakeDynamoDB{scanOut: &dynamodb.ScanOutput{
			Items: []map[string]types.AttributeValue{
				// Version must be a number; a string breaks the conversion
				{"Version": &types.AttributeValueMemberS{Value: "not-a-number"}},
			},
		}}
		repo := NewDynamoDBRepository[schema.Schema](fake, "table")

		_, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{})
		assert.Error(t, err)
	})
}
