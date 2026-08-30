package schema

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Reproduces: ToDynamoDB writes numbers with fmt.Sprintf("%f", ...),
// which keeps only 6 decimal places and expands large values.
// Numbers must survive a ToDynamoDB -> FromDynamoDB round trip unchanged.
func TestDynamoDBNumberRoundTrip(t *testing.T) {
	useCases := map[string]Schema{
		"int":                  MkInt(42),
		"negative int":         MkInt(-42),
		"float":                MkFloat(1.5),
		"more than 6 decimals": MkFloat(1.123456789),
		"small float":          MkFloat(0.0000001),
		"large float":          MkFloat(1e21),
		"max safe float int":   MkInt(1 << 53),
		"int64 above 2^53":     MkInt((1 << 60) + 1),
		"max int64":            MkInt(math.MaxInt64),
	}
	for name, in := range useCases {
		t.Run(name, func(t *testing.T) {
			out, err := FromDynamoDB(ToDynamoDB(in))
			require.NoError(t, err)
			assert.Equal(t, in, out)
		})
	}
}

// DynamoDB numbers should be written in a canonical short form,
// not as "%f" (e.g. MkInt(7) must not become "7.000000").
func TestDynamoDBNumberFormat(t *testing.T) {
	useCases := map[string]struct {
		in   Schema
		want string
	}{
		"int is written without decimals": {MkInt(7), "7"},
		"float keeps all decimals":        {MkFloat(1.123456789), "1.123456789"},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			av, ok := ToDynamoDB(uc.in).(*types.AttributeValueMemberN)
			require.True(t, ok)
			assert.Equal(t, uc.want, av.Value)
		})
	}
}

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

	t.Run("ToDynamoDB should product the same result as the original", func(t *testing.T) {
		dynamed := ToDynamoDB(schemed)
		var result2 any = nil
		err = attributevalue.Unmarshal(dynamed, &result2)
		assert.NoError(t, err)

		jsonRepresentation2, err := json.Marshal(result2)
		assert.NoError(t, err)

		assert.JSONEq(t, string(grandTruthJSONRepresentation), string(jsonRepresentation2))
	})
}
