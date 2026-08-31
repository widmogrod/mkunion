package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromPrimitiveGoValues(t *testing.T) {
	useCases := map[string]struct {
		in   any
		want Schema
	}{
		"bool":    {true, MkBool(true)},
		"int":     {42, MkInt(42)},
		"int8":    {int8(42), MkInt(42)},
		"int16":   {int16(42), MkInt(42)},
		"int32":   {int32(42), MkInt(42)},
		"int64":   {int64(42), MkInt(42)},
		"uint":    {uint(42), MkUint(42)},
		"uint8":   {uint8(42), MkUint(42)},
		"uint16":  {uint16(42), MkUint(42)},
		"uint32":  {uint32(42), MkUint(42)},
		"uint64":  {uint64(42), MkUint(42)},
		"float32": {float32(1.5), MkFloat(1.5)},
		"float64": {1.5, MkFloat(1.5)},
		"string":  {"x", MkString("x")},
		"bytes":   {[]byte{1, 2}, MkBinary([]byte{1, 2})},
		"nil":     {nil, MkNone()},
		"slice of any": {
			[]any{1, "x"},
			MkList(MkInt(1), MkString("x")),
		},
		"map of any": {
			map[any]any{"a": 1},
			MkMap(MkField("a", MkInt(1))),
		},
		// json.Unmarshal into any produces this shape; it used to panic
		"map of string to any": {
			map[string]any{"a": 1},
			MkMap(MkField("a", MkInt(1))),
		},
		"schema value passes through": {
			MkString("already-schema"), MkString("already-schema"),
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.want, FromPrimitiveGo(uc.in))
		})
	}

	t.Run("unknown type panics", func(t *testing.T) {
		assert.Panics(t, func() {
			FromPrimitiveGo(struct{ A int }{})
		})
	})
}

// IsPrimitive reports pointers as primitive, so FromGo routes them here;
// they must convert (nil to None, non-nil to the pointed value), not panic.
func TestFromPrimitiveGoPointers(t *testing.T) {
	i := 42
	f := 1.5
	s := "x"
	b := true
	raw := []byte{1}

	useCases := map[string]struct {
		in   any
		want Schema
	}{
		"pointer to int":    {&i, MkInt(42)},
		"pointer to float":  {&f, MkFloat(1.5)},
		"pointer to string": {&s, MkString("x")},
		"pointer to bool":   {&b, MkBool(true)},
		"pointer to bytes":  {&raw, MkBinary([]byte{1})},
		"nil pointer":       {(*int)(nil), MkNone()},
		"nil string ptr":    {(*string)(nil), MkNone()},
		"nil bytes ptr":     {(*[]byte)(nil), MkNone()},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.want, FromPrimitiveGo(uc.in))
		})
	}

	t.Run("FromGo with a pointer no longer panics", func(t *testing.T) {
		got := FromGo[*int](&i)
		assert.Equal(t, MkInt(42), got)

		got = FromGo[*int](nil)
		assert.Equal(t, MkNone(), got)
	})
}

func TestCompareUnequalValues(t *testing.T) {
	useCases := map[string]struct {
		a, b Schema
		want int
	}{
		"false < true":        {MkBool(false), MkBool(true), -1},
		"strings by order":    {MkString("a"), MkString("b"), -1},
		"strings reversed":    {MkString("b"), MkString("a"), 1},
		"binary by bytes":     {MkBinary([]byte{1}), MkBinary([]byte{2}), -1},
		"binary reversed":     {MkBinary([]byte{2}), MkBinary([]byte{1}), 1},
		"shorter list first":  {MkList(MkInt(1)), MkList(MkInt(1), MkInt(2)), -1},
		"longer list last":    {MkList(MkInt(1), MkInt(2)), MkList(MkInt(1)), 1},
		"list by element":     {MkList(MkInt(1)), MkList(MkInt(2)), -1},
		"equal nested lists":  {MkList(MkList(MkInt(1))), MkList(MkList(MkInt(1))), 0},
		"smaller map first":   {MkMap(), MkMap(MkField("a", MkInt(1))), -1},
		"bigger map last":     {MkMap(MkField("a", MkInt(1))), MkMap(), 1},
		"map by shared value": {MkMap(MkField("a", MkInt(1))), MkMap(MkField("a", MkInt(2))), -1},
		"map with a different key": {
			MkMap(MkField("a", MkInt(1))),
			MkMap(MkField("b", MkInt(1))),
			-1,
		},
		"nil sorts before values": {nil, MkInt(1), -1},
		"values sort after nil":   {MkInt(1), nil, 1},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.want, Compare(uc.a, uc.b))
		})
	}

	t.Run("map comparison is symmetric for disjoint keys", func(t *testing.T) {
		a := MkMap(MkField("a", MkInt(1)))
		b := MkMap(MkField("b", MkInt(1)))
		require.Equal(t, -1, Compare(a, b))
		require.Equal(t, -1, Compare(b, a),
			"documents current behavior: disjoint same-size maps both report -1, ordering is not antisymmetric")
	})
}
