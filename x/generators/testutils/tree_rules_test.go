package testutils

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
)

// A nil variant pointer is "no value". It encodes as null, like encoding/json
// does for any nil pointer, and null decodes back to a nil union.
func TestTree_JSON_nil_variant_encodes_as_null(t *testing.T) {
	var typedNil Tree = (*Leaf)(nil)

	out, err := TreeToJSON(typedNil)
	require.NoError(t, err)
	assert.Equal(t, "null", string(out))

	out, err = shared.JSONMarshal[Tree](typedNil)
	require.NoError(t, err)
	assert.Equal(t, "null", string(out))

	out, err = (*Leaf)(nil).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "null", string(out), "the variant alone follows the same rule")

	out, err = json.Marshal(typedNil)
	require.NoError(t, err)
	assert.Equal(t, "null", string(out), "encoding/json agrees")

	back, err := TreeFromJSON([]byte("null"))
	require.NoError(t, err)
	assert.Nil(t, back)
}

// $type picks the variant. When the payload key is missing or null, the
// variant is its zero value. Both spellings decode the same way.
func TestTree_JSON_type_without_payload_is_zero_value(t *testing.T) {
	cases := map[string]struct {
		in   string
		want Tree
	}{
		"struct, no payload":    {`{"$type":"testutils.Leaf"}`, &Leaf{}},
		"struct, null payload":  {`{"$type":"testutils.Leaf","testutils.Leaf":null}`, &Leaf{}},
		"string alias, no key":  {`{"$type":"testutils.K"}`, shape.Ptr(K(""))},
		"map alias, no key":     {`{"$type":"testutils.Ma"}`, new(Ma)},
		"slice alias, null":     {`{"$type":"testutils.La","testutils.La":null}`, new(La)},
		"payload wins when set": {`{"$type":"testutils.Leaf","testutils.Leaf":{"Value":7}}`, &Leaf{Value: 7}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := TreeFromJSON([]byte(c.in))
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

// Decoding errors name the problem and can be matched with errors.Is.
func TestTree_JSON_decode_errors(t *testing.T) {
	t.Run("no $type and no variant key", func(t *testing.T) {
		for _, in := range []string{`{}`, `{   }`, `{"other":1}`} {
			got, err := TreeFromJSON([]byte(in))
			assert.Nil(t, got, in)
			assert.ErrorIs(t, err, shared.ErrNoType, in)
		}
	})

	t.Run("unknown $type", func(t *testing.T) {
		got, err := TreeFromJSON([]byte(`{"$type":"testutils.Nope"}`))
		assert.Nil(t, got)
		assert.ErrorIs(t, err, shared.ErrUnknownType)
		assert.EqualError(t, err, "testutils.TreeFromJSON: unknown type: testutils.Nope")
	})

	t.Run("inputs shorter than four bytes are errors, not panics", func(t *testing.T) {
		for _, in := range []string{`1`, `""`, `[]`, `nul`} {
			_, err := TreeFromJSON([]byte(in))
			assert.Error(t, err, in)
		}
	})

	t.Run("short value for a pointer field is an error, not a panic", func(t *testing.T) {
		_, err := TreeFromJSON([]byte(`{"$type":"testutils.Branch","testutils.Branch":{"L":1}}`))
		assert.Error(t, err)
	})

	t.Run("JSONUnmarshal wraps the same sentinels", func(t *testing.T) {
		_, err := shared.JSONUnmarshal[Tree]([]byte(`{"$type":"testutils.Nope"}`))
		assert.True(t, errors.Is(err, shared.ErrUnknownType), err)
	})
}
