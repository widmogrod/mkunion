package shared

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func typeOf(x any) reflect.Type {
	return reflect.TypeOf(x)
}

//go:tag shape:"-"
type plainStruct struct {
	Name string
	Age  int
}

// registeredUnion mimics what mkunion generates: an interface type whose
// serde is registered under its full type name.
//go:tag shape:"-"
type registeredUnion interface{ isRegistered() }

//go:tag shape:"-"
type variantA struct{ Value string }

func (variantA) isRegistered() {}

const registeredUnionName = "github.com/widmogrod/mkunion/x/shared.registeredUnion"

var serdeErr = errors.New("scripted serde failure")

func init() {
	JSONMarshallerRegister[registeredUnion](
		registeredUnionName,
		func(data []byte) (registeredUnion, error) {
			if string(data) == `"fail"` {
				return nil, serdeErr
			}
			var v variantA
			if err := json.Unmarshal(data, &v); err != nil {
				return nil, err
			}
			return v, nil
		},
		func(u registeredUnion) ([]byte, error) {
			if v, ok := u.(variantA); ok {
				if v.Value == "fail" {
					return nil, serdeErr
				}
				return json.Marshal(v)
			}
			return nil, fmt.Errorf("unknown variant %T", u)
		},
	)
}

func TestJSONRoundTripNative(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		data, err := JSONMarshal[int](42)
		require.NoError(t, err)
		assert.Equal(t, "42", string(data))

		back, err := JSONUnmarshal[int](data)
		require.NoError(t, err)
		assert.Equal(t, 42, back)
	})

	t.Run("string", func(t *testing.T) {
		data, err := JSONMarshal[string]("hi")
		require.NoError(t, err)

		back, err := JSONUnmarshal[string](data)
		require.NoError(t, err)
		assert.Equal(t, "hi", back)
	})

	t.Run("plain struct uses the native fallback", func(t *testing.T) {
		data, err := JSONMarshal[plainStruct](plainStruct{Name: "Ala", Age: 7})
		require.NoError(t, err)

		back, err := JSONUnmarshal[plainStruct](data)
		require.NoError(t, err)
		assert.Equal(t, plainStruct{Name: "Ala", Age: 7}, back)
	})

	t.Run("map of raw messages", func(t *testing.T) {
		in := map[string]json.RawMessage{"a": json.RawMessage(`1`)}
		data, err := JSONMarshal[map[string]json.RawMessage](in)
		require.NoError(t, err)

		back, err := JSONUnmarshal[map[string]json.RawMessage](data)
		require.NoError(t, err)
		assert.Equal(t, in, back)
	})
}

func TestJSONMarshalEdgeCases(t *testing.T) {
	t.Run("nil any marshals to nil without error", func(t *testing.T) {
		data, err := JSONMarshal[any](nil)
		require.NoError(t, err)
		assert.Nil(t, data)
	})

	t.Run("unmarshalable type errors", func(t *testing.T) {
		_, err := JSONMarshal[chan int](make(chan int))
		assert.ErrorContains(t, err, "unsupported type")
	})

	t.Run("type with MarshalJSON short-circuits", func(t *testing.T) {
		ts := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
		data, err := JSONMarshal[time.Time](ts)
		require.NoError(t, err)

		expected, err := json.Marshal(ts)
		require.NoError(t, err)
		assert.Equal(t, expected, data)
	})
}

func TestJSONUnmarshalEdgeCases(t *testing.T) {
	t.Run("malformed native input errors", func(t *testing.T) {
		_, err := JSONUnmarshal[int]([]byte("not-json"))
		assert.Error(t, err)
	})

	t.Run("malformed struct input errors", func(t *testing.T) {
		_, err := JSONUnmarshal[plainStruct]([]byte(`{"Name":`))
		assert.Error(t, err)
	})

	t.Run("type with UnmarshalJSON short-circuits", func(t *testing.T) {
		ts := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
		data, err := json.Marshal(ts)
		require.NoError(t, err)

		back, err := JSONUnmarshal[time.Time](data)
		require.NoError(t, err)
		assert.True(t, ts.Equal(back))
	})

	t.Run("UnmarshalJSON error propagates", func(t *testing.T) {
		_, err := JSONUnmarshal[time.Time]([]byte(`"not-a-time"`))
		assert.ErrorContains(t, err, "destination ptr unmarshal")
	})
}

func TestJSONRegisteredSerde(t *testing.T) {
	t.Run("marshal routes through the registered serde", func(t *testing.T) {
		data, err := JSONMarshal[registeredUnion](variantA{Value: "x"})
		require.NoError(t, err)
		assert.JSONEq(t, `{"Value":"x"}`, string(data))
	})

	t.Run("unmarshal routes through the registered serde", func(t *testing.T) {
		back, err := JSONUnmarshal[registeredUnion]([]byte(`{"Value":"x"}`))
		require.NoError(t, err)
		assert.Equal(t, variantA{Value: "x"}, back)
	})

	t.Run("serde marshal error propagates", func(t *testing.T) {
		_, err := JSONMarshal[registeredUnion](variantA{Value: "fail"})
		assert.ErrorIs(t, err, serdeErr)
	})

	t.Run("serde unmarshal error propagates", func(t *testing.T) {
		_, err := JSONUnmarshal[registeredUnion]([]byte(`"fail"`))
		assert.ErrorIs(t, err, serdeErr)
	})

	t.Run("null input yields the zero value without calling the serde", func(t *testing.T) {
		back, err := JSONUnmarshal[registeredUnion]([]byte("null"))
		require.NoError(t, err)
		assert.Nil(t, back)
	})

	t.Run("nil input yields the zero value", func(t *testing.T) {
		back, err := JSONUnmarshal[registeredUnion](nil)
		require.NoError(t, err)
		assert.Nil(t, back)
	})

	t.Run("registration also fills the type registry", func(t *testing.T) {
		typ, found := TypeRegistryLoad(registeredUnionName)
		assert.True(t, found)
		assert.Nil(t, typ, "the stored zero value of an interface type is nil")
	})
}

func TestFullTypeName(t *testing.T) {
	assert.Equal(t, "int", FullTypeName(typeOf(0)))
	assert.Equal(t, "int", FullTypeName(typeOf((*int)(nil))), "pointers are dereferenced")
	assert.Equal(t, "time.Time", FullTypeName(typeOf(time.Time{})))
	assert.Equal(t, "github.com/widmogrod/mkunion/x/shared.plainStruct", FullTypeName(typeOf(plainStruct{})))
}

func TestJSONIsNativePath(t *testing.T) {
	assert.True(t, JSONIsNativePath("x"))
	assert.True(t, JSONIsNativePath(42))
	assert.True(t, JSONIsNativePath([]byte("x")))
	assert.True(t, JSONIsNativePath(plainStruct{}), "any non-nil value matches the any case")
	assert.False(t, JSONIsNativePath(nil), "nil interfaces take the registry path")
}
