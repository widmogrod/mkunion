package schemaless

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
)

func cmp(location, operation, bindName string) *predicate.Compare {
	return &predicate.Compare{
		Location:  location,
		Operation: operation,
		BindValue: &predicate.BindValue{BindName: bindName},
	}
}

// Repro: Not{Or{a,b}} used to render as `NOT a OR b`, which DynamoDB
// parses as `(NOT a) OR b` — the negation applied only to the first operand.
func TestDynamoDBExpression_NotOverOr_IsParenthesised(t *testing.T) {
	repo := &DynamoDBRepository[ExampleRecord]{}
	expr, _, _, err := repo.buildFilterExpression(FindingRecords[Record[ExampleRecord]]{
		Where: &predicate.WherePredicates{
			Predicate: &predicate.Not{
				P: &predicate.Or{L: []predicate.Predicate{
					cmp("ID", "=", ":a"),
					cmp("ID", "=", ":b"),
				}},
			},
			Params: predicate.ParamBinds{
				":a": schema.MkString("1"),
				":b": schema.MkString("2"),
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "NOT (#ID = :a OR #ID = :b)", expr)
}

// Repro: `t AND (x OR y)` used to render as `t AND x OR y`, which DynamoDB
// parses as `(t AND x) OR y` because AND binds tighter than OR.
func TestDynamoDBExpression_AndOverOr_IsParenthesised(t *testing.T) {
	repo := &DynamoDBRepository[ExampleRecord]{}
	expr, _, _, err := repo.buildFilterExpression(FindingRecords[Record[ExampleRecord]]{
		RecordType: "ExampleRecord",
		Where: &predicate.WherePredicates{
			Predicate: &predicate.Or{L: []predicate.Predicate{
				cmp("ID", "=", ":a"),
				cmp("ID", "=", ":b"),
			}},
			Params: predicate.ParamBinds{
				":a": schema.MkString("1"),
				":b": schema.MkString("2"),
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "#Type = :Type AND (#ID = :a OR #ID = :b)", expr)
}

// Repro: an empty query (no RecordType, no Where) used to produce
// FilterExpression="" in the ScanInput, which DynamoDB rejects with
// ValidationException. "List all" must send no filter at all.
func TestDynamoDBScanInput_EmptyQuery_HasNoFilter(t *testing.T) {
	repo := &DynamoDBRepository[ExampleRecord]{tableName: "test"}
	input, err := repo.buildScanInput(FindingRecords[Record[ExampleRecord]]{})
	assert.NoError(t, err)
	assert.Nil(t, input.FilterExpression, "empty query must not set FilterExpression")
	assert.Nil(t, input.ExpressionAttributeNames)
	assert.Nil(t, input.ExpressionAttributeValues)
}

// Repro: a Literal bind value used to panic("implement me").
func TestDynamoDBExpression_Literal_DoesNotPanic(t *testing.T) {
	repo := &DynamoDBRepository[ExampleRecord]{}
	assert.NotPanics(t, func() {
		expr, values, _, err := repo.buildFilterExpression(FindingRecords[Record[ExampleRecord]]{
			Where: &predicate.WherePredicates{
				Predicate: &predicate.Compare{
					Location:  "Data.Age",
					Operation: ">",
					BindValue: &predicate.Literal{Value: schema.MkInt(10)},
				},
				Params: predicate.ParamBinds{},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, "#Data.#Age > :lit0", expr)
		assert.Contains(t, values, ":lit0")
	})
}

// Repro: a location with a list index used to panic("implement me").
func TestDynamoDBExpression_LocationIndex_DoesNotPanic(t *testing.T) {
	repo := &DynamoDBRepository[ExampleRecord]{}
	assert.NotPanics(t, func() {
		expr, _, _, err := repo.buildFilterExpression(FindingRecords[Record[ExampleRecord]]{
			Where: &predicate.WherePredicates{
				Predicate: cmp("Data[0]", "=", ":a"),
				Params:    predicate.ParamBinds{":a": schema.MkString("x")},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, "#Data[0] = :a", expr)
	})
}

// Repro: a Locatable bind value (compare two fields) used to panic("implement me").
func TestDynamoDBExpression_Locatable_DoesNotPanic(t *testing.T) {
	repo := &DynamoDBRepository[ExampleRecord]{}
	assert.NotPanics(t, func() {
		expr, _, _, err := repo.buildFilterExpression(FindingRecords[Record[ExampleRecord]]{
			Where: &predicate.WherePredicates{
				Predicate: &predicate.Compare{
					Location:  "ID",
					Operation: "=",
					BindValue: &predicate.Locatable{Location: "Type"},
				},
				Params: predicate.ParamBinds{},
			},
		})
		assert.NoError(t, err)
		assert.Equal(t, "#ID = #Type", expr)
	})
}

// Repro: the Next page query used to drop RecordType, so the second page
// was no longer restricted to the requested record type (and with no Where
// it produced an empty FilterExpression → ValidationException).
func TestDynamoDBNextPage_KeepsRecordType(t *testing.T) {
	repo := &DynamoDBRepository[ExampleRecord]{tableName: "test"}
	query := FindingRecords[Record[ExampleRecord]]{
		RecordType: "ExampleRecord",
		Limit:      2,
	}
	lastKey := map[string]types.AttributeValue{
		"ID":   &types.AttributeValueMemberS{Value: "123"},
		"Type": &types.AttributeValueMemberS{Value: "ExampleRecord"},
	}
	next, err := repo.nextPageQuery(query, lastKey)
	assert.NoError(t, err)
	assert.Equal(t, "ExampleRecord", next.RecordType, "Next must keep the RecordType")
	assert.NotNil(t, next.After)
}
