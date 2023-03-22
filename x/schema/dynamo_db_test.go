package schema

import (
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnwrapDynamoDB(t *testing.T) {
	exampleDDBType := types.AttributeValueMemberM{
		Value: map[string]types.AttributeValue{
			"string": &types.AttributeValueMemberS{
				Value: "bar",
			},
			"string set": &types.AttributeValueMemberSS{
				Value: []string{"bar", "baz"},
			},
			"number": &types.AttributeValueMemberN{
				Value: "1",
			},
			"number set": &types.AttributeValueMemberNS{
				Value: []string{"1", "2"},
			},
			"binary": &types.AttributeValueMemberB{
				Value: []byte("bar"),
			},
			"binary set": &types.AttributeValueMemberBS{
				Value: [][]byte{[]byte("bar"), []byte("baz")},
			},
			"bool": &types.AttributeValueMemberBOOL{
				Value: true,
			},
			"null": &types.AttributeValueMemberNULL{
				Value: true,
			},
			"list": &types.AttributeValueMemberL{
				Value: []types.AttributeValue{
					&types.AttributeValueMemberS{
						Value: "bar",
					},
					&types.AttributeValueMemberS{
						Value: "baz",
					},
				},
			},
			"map": &types.AttributeValueMemberM{
				Value: map[string]types.AttributeValue{
					"foo": &types.AttributeValueMemberS{
						Value: "bar",
					},
				},
			},
		},
	}

	var result any = nil
	err := attributevalue.Unmarshal(&exampleDDBType, &result)
	assert.NoError(t, err)

	grandTruthJSONRepresentation, err := json.Marshal(result)
	assert.NoError(t, err)

	schemed, err := FromDynamoDB(&exampleDDBType)
	assert.NoError(t, err)

	t.Run("FromDynamoDB should produce the same result as the original", func(t *testing.T) {
		jsoned, err := ToJSON(schemed)
		assert.NoError(t, err)

		// schema and dynamodb unmarshalling should produce the same result
		assert.JSONEq(t, string(grandTruthJSONRepresentation), string(jsoned))
	})

	t.Run("ToDynamoDB should product the same result as the original", func(t *testing.T) {
		dynamed := ToDynamoDB(schemed)
		var result2 any = nil
		err = attributevalue.Unmarshal(dynamed, &result2)
		assert.NoError(t, err)

		jsonRepresentation2, err := json.Marshal(result2)
		assert.NoError(t, err)

		assert.JSONEq(t, string(grandTruthJSONRepresentation), string(jsonRepresentation2))
	})

	t.Run("UnwrapDynamoDB should produce the same result as the original", func(t *testing.T) {
		dynamoJSON, err := FromJSON([]byte(`{
"M": {
	"string": {
		"S": "bar"
	},
	"string set": {	
		"SS": ["bar", "baz"]
	},
	"number": {
		"N": "1"
	},
	"number set": {
		"NS": ["1", "2"]
	},
	"binary": {
		"B": "YmFy"
	},
	"binary set": {
		"BS": ["YmFy", "YmF6"]
	},
	"bool": {
		"BOOL": true
	},
	"null": {
		"NULL": true
	},
	"list": {
		"L": [
			{
				"S": "bar"
			},
			{
				"S": "baz"
			}
		]
	},
	"map": {
		"M": {
			"foo": {
				"S": "bar"
			}
		}
	}
}}`))
		assert.NoError(t, err)

		unwrapped, err := UnwrapDynamoDB(dynamoJSON)
		assert.NoError(t, err)

		jsonRepresentation3, err := ToJSON(unwrapped)
		assert.NoError(t, err)

		assert.JSONEq(t, string(grandTruthJSONRepresentation), string(jsonRepresentation3))
	})
}
