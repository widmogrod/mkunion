package testutils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
)

// Holder has no serde tag; mkunion generated its serde anyway because it holds
// the Tree union, so encoding/json keeps the $type of every variant.
func TestHolder_JSON(t *testing.T) {
	h := Holder{
		One:   &Leaf{Value: 1},
		Many:  []Tree{&Leaf{Value: 2}, shape.Ptr(K("k"))},
		Named: map[string]Tree{"a": &Leaf{Value: 3}},
	}

	data, err := json.Marshal(&h)
	require.NoError(t, err)
	t.Log(string(data))
	assert.Contains(t, string(data), `"$type":"testutils.Leaf"`)
	assert.Contains(t, string(data), `"$type":"testutils.K"`)

	var back Holder
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, h, back)

	viaShared, err := shared.JSONMarshal[Holder](h)
	require.NoError(t, err)
	assert.Equal(t, string(data), string(viaShared))
}

func TestViaAlias_JSON(t *testing.T) {
	v := ViaAlias{List: Trees{&Leaf{Value: 1}, shape.Ptr(K("k"))}}

	data, err := json.Marshal(&v)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"$type":"testutils.Leaf"`)

	var back ViaAlias
	require.NoError(t, json.Unmarshal(data, &back))
	assert.Equal(t, v, back)
}

func TestPlain_NoSerde(t *testing.T) {
	_, ok := any(&Plain{}).(json.Marshaler)
	assert.False(t, ok, "Plain holds no union; encoding/json is enough")
}
