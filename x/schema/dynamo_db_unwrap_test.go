package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wrapped builds the schema form of a DynamoDB-JSON attribute, e.g.
// {"S": "bar"} or {"N": "1"} — the input UnwrapDynamoDB decodes.
func wrapped(tag string, value Schema) Schema {
	return MkMap(MkField(tag, value))
}

func TestUnwrapDynamoDBScalars(t *testing.T) {
	useCases := map[string]struct {
		in   Schema
		want Schema
	}{
		"S unwraps to string":     {wrapped("S", MkString("bar")), MkString("bar")},
		"N unwraps to number":     {wrapped("N", MkString("1.5")), MkFloat(1.5)},
		"BOOL unwraps to bool":    {wrapped("BOOL", MkBool(true)), MkBool(true)},
		"NULL unwraps to none":    {wrapped("NULL", MkBool(true)), &None{}},
		"B passes through as-is":  {wrapped("B", MkString("YmFy")), MkString("YmFy")},
		"SS unwraps to string list": {
			wrapped("SS", MkList(MkString("a"), MkString("b"))),
			MkList(MkString("a"), MkString("b")),
		},
		"NS unwraps to number list": {
			wrapped("NS", MkList(MkFloat(1), MkFloat(2))),
			MkList(MkFloat(1), MkFloat(2)),
		},
		"BS passes elements through": {
			wrapped("BS", MkList(MkString("YmFy"))),
			MkList(MkString("YmFy")),
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got, err := UnwrapDynamoDB(uc.in)
			require.NoError(t, err)
			assert.Equal(t, uc.want, got)
		})
	}
}

func TestUnwrapDynamoDBComposites(t *testing.T) {
	t.Run("M unwraps recursively", func(t *testing.T) {
		in := wrapped("M", MkMap(
			MkField("name", wrapped("S", MkString("Ala"))),
			MkField("age", wrapped("N", MkString("42"))),
		))
		got, err := UnwrapDynamoDB(in)
		require.NoError(t, err)
		assert.Equal(t, MkMap(
			MkField("name", MkString("Ala")),
			MkField("age", MkFloat(42)),
		), got)
	})

	t.Run("L unwraps each element", func(t *testing.T) {
		in := wrapped("L", MkList(
			wrapped("S", MkString("a")),
			wrapped("N", MkString("1")),
		))
		got, err := UnwrapDynamoDB(in)
		require.NoError(t, err)
		assert.Equal(t, MkList(MkString("a"), MkFloat(1)), got)
	})

	t.Run("multi-key map is treated as a plain map of wrapped values", func(t *testing.T) {
		in := MkMap(
			MkField("a", wrapped("S", MkString("x"))),
			MkField("b", wrapped("BOOL", MkBool(false))),
		)
		got, err := UnwrapDynamoDB(in)
		require.NoError(t, err)
		assert.Equal(t, MkMap(
			MkField("a", MkString("x")),
			MkField("b", MkBool(false)),
		), got)
	})

	t.Run("empty map unwraps to an empty map", func(t *testing.T) {
		got, err := UnwrapDynamoDB(MkMap())
		require.NoError(t, err)
		assert.Equal(t, MkMap(), got)
	})

	t.Run("deep nesting unwraps all levels", func(t *testing.T) {
		in := wrapped("M", MkMap(
			MkField("outer", wrapped("L", MkList(
				wrapped("M", MkMap(MkField("inner", wrapped("S", MkString("deep"))))),
			))),
		))
		got, err := UnwrapDynamoDB(in)
		require.NoError(t, err)
		assert.Equal(t, MkMap(
			MkField("outer", MkList(
				MkMap(MkField("inner", MkString("deep"))),
			)),
		), got)
	})
}

func TestUnwrapDynamoDBErrors(t *testing.T) {
	useCases := map[string]Schema{
		"nil data":               nil,
		"non-map data":           MkInt(1),
		"list data":              MkList(MkInt(1)),
		"N with unparsable text": wrapped("N", MkString("not-a-number")),
		"SS with non-list":       wrapped("SS", MkString("a")),
		"NS with non-list":       wrapped("NS", MkString("1")),
		"BS with non-list":       wrapped("BS", MkString("x")),
		"M with non-map":         wrapped("M", MkString("x")),
		"L with non-list":        wrapped("L", MkString("x")),
		// a single unknown key cannot be told apart from a wrapper tag,
		// so it is rejected rather than guessed at
		"single-key plain map":       MkMap(MkField("a", MkInt(1))),
		"error inside L propagates":  wrapped("L", MkList(wrapped("N", MkString("nope")))),
		"error inside M propagates":  wrapped("M", MkMap(MkField("x", wrapped("N", MkString("nope"))))),
		"error in multi-key map":     MkMap(MkField("a", MkInt(1)), MkField("b", MkInt(2))),
	}
	for name, in := range useCases {
		t.Run(name, func(t *testing.T) {
			_, err := UnwrapDynamoDB(in)
			assert.Error(t, err)
		})
	}
}
