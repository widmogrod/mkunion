package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAsFromNumber(t *testing.T) {
	t.Run("every numeric target type", func(t *testing.T) {
		n := MkInt(42)

		i, ok := As[int](n)
		assert.True(t, ok)
		assert.Equal(t, int(42), i)

		i8, ok := As[int8](n)
		assert.True(t, ok)
		assert.Equal(t, int8(42), i8)

		i16, ok := As[int16](n)
		assert.True(t, ok)
		assert.Equal(t, int16(42), i16)

		i32, ok := As[int32](n)
		assert.True(t, ok)
		assert.Equal(t, int32(42), i32)

		i64, ok := As[int64](n)
		assert.True(t, ok)
		assert.Equal(t, int64(42), i64)

		u, ok := As[uint](n)
		assert.True(t, ok)
		assert.Equal(t, uint(42), u)

		u8, ok := As[uint8](n)
		assert.True(t, ok)
		assert.Equal(t, uint8(42), u8)

		u16, ok := As[uint16](n)
		assert.True(t, ok)
		assert.Equal(t, uint16(42), u16)

		u32, ok := As[uint32](n)
		assert.True(t, ok)
		assert.Equal(t, uint32(42), u32)

		u64, ok := As[uint64](n)
		assert.True(t, ok)
		assert.Equal(t, uint64(42), u64)

		f32, ok := As[float32](n)
		assert.True(t, ok)
		assert.Equal(t, float32(42), f32)

		f64, ok := As[float64](n)
		assert.True(t, ok)
		assert.Equal(t, float64(42), f64)
	})

	t.Run("fraction truncates toward zero for int targets", func(t *testing.T) {
		i, ok := As[int](MkFloat(1.9))
		assert.True(t, ok)
		assert.Equal(t, 1, i)

		i, ok = As[int](MkFloat(-1.9))
		assert.True(t, ok)
		assert.Equal(t, -1, i)
	})

	t.Run("number to non-numeric targets fails", func(t *testing.T) {
		_, ok := As[string](MkInt(1))
		assert.False(t, ok)

		_, ok = As[bool](MkInt(1))
		assert.False(t, ok)

		_, ok = As[[]byte](MkInt(1))
		assert.False(t, ok)
	})
}

func TestAsFromString(t *testing.T) {
	t.Run("string and bytes", func(t *testing.T) {
		s, ok := As[string](MkString("hello"))
		assert.True(t, ok)
		assert.Equal(t, "hello", s)

		b, ok := As[[]byte](MkString("hello"))
		assert.True(t, ok)
		assert.Equal(t, []byte("hello"), b)
	})

	t.Run("numeric strings parse", func(t *testing.T) {
		f64, ok := As[float64](MkString("1.5"))
		assert.True(t, ok)
		assert.Equal(t, 1.5, f64)

		f32, ok := As[float32](MkString("1.5"))
		assert.True(t, ok)
		assert.Equal(t, float32(1.5), f32)

		i, ok := As[int](MkString("-42"))
		assert.True(t, ok)
		assert.Equal(t, -42, i)

		i8, ok := As[int8](MkString("127"))
		assert.True(t, ok)
		assert.Equal(t, int8(127), i8)

		i16, ok := As[int16](MkString("1000"))
		assert.True(t, ok)
		assert.Equal(t, int16(1000), i16)

		i32, ok := As[int32](MkString("100000"))
		assert.True(t, ok)
		assert.Equal(t, int32(100000), i32)

		i64, ok := As[int64](MkString("9007199254740993"))
		assert.True(t, ok)
		assert.Equal(t, int64(9007199254740993), i64)
	})

	t.Run("malformed numeric strings fail", func(t *testing.T) {
		_, ok := As[int](MkString("not-a-number"))
		assert.False(t, ok)

		_, ok = As[float64](MkString(""))
		assert.False(t, ok)

		_, ok = As[int64](MkString("1.5"))
		assert.False(t, ok)

		_, ok = As[float32](MkString("not-a-float"))
		assert.False(t, ok)
	})

	t.Run("out of range numeric strings fail", func(t *testing.T) {
		_, ok := As[int8](MkString("128"))
		assert.False(t, ok)

		_, ok = As[int16](MkString("40000"))
		assert.False(t, ok)

		_, ok = As[int32](MkString("3000000000"))
		assert.False(t, ok)
	})

	t.Run("string to unsupported targets fails", func(t *testing.T) {
		// unsigned targets have no parse path from strings
		_, ok := As[uint](MkString("42"))
		assert.False(t, ok)

		_, ok = As[bool](MkString("true"))
		assert.False(t, ok)
	})
}

func TestAsFromBoolBinaryListMap(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		b, ok := As[bool](MkBool(true))
		assert.True(t, ok)
		assert.True(t, b)

		_, ok = As[string](MkBool(true))
		assert.False(t, ok)

		_, ok = As[int](MkBool(true))
		assert.False(t, ok)
	})

	t.Run("binary", func(t *testing.T) {
		b, ok := As[[]byte](MkBinary([]byte{1, 2}))
		assert.True(t, ok)
		assert.Equal(t, []byte{1, 2}, b)

		s, ok := As[string](MkBinary([]byte("raw")))
		assert.True(t, ok)
		assert.Equal(t, "raw", s)

		_, ok = As[int](MkBinary([]byte{1}))
		assert.False(t, ok)
	})

	t.Run("collections never convert to primitives", func(t *testing.T) {
		_, ok := As[int](MkList(MkInt(1)))
		assert.False(t, ok)

		_, ok = As[string](MkMap(MkField("a", MkInt(1))))
		assert.False(t, ok)
	})
}

func TestAsNilAndNone(t *testing.T) {
	// A is always a value type, so the zero value never equals nil and
	// nil/None report ok=false with the zero value.
	t.Run("nil schema", func(t *testing.T) {
		i, ok := As[int](nil)
		assert.False(t, ok)
		assert.Equal(t, 0, i)

		s, ok := As[string](nil)
		assert.False(t, ok)
		assert.Equal(t, "", s)
	})

	t.Run("none value", func(t *testing.T) {
		i, ok := As[int](&None{})
		assert.False(t, ok)
		assert.Equal(t, 0, i)
	})
}

func TestAsDefault(t *testing.T) {
	assert.Equal(t, 42, AsDefault[int](MkInt(42), 7))
	assert.Equal(t, 7, AsDefault[int](MkString("nope"), 7))
	assert.Equal(t, 7, AsDefault[int](nil, 7))
	assert.Equal(t, "x", AsDefault[string](&None{}, "x"))
}

func TestGetSchemaDefault(t *testing.T) {
	data := MkMap(
		MkField("Age", MkInt(10)),
		MkField("Name", MkString("Ala")),
	)

	t.Run("found and convertible", func(t *testing.T) {
		assert.Equal(t, 10, GetSchemaDefault[int](data, "Age", 7))
	})
	t.Run("path not found", func(t *testing.T) {
		assert.Equal(t, 7, GetSchemaDefault[int](data, "Nope", 7))
	})
	t.Run("found but not convertible", func(t *testing.T) {
		assert.Equal(t, 7, GetSchemaDefault[int](data, "Name", 7))
	})
	t.Run("unparsable location", func(t *testing.T) {
		assert.Equal(t, 7, GetSchemaDefault[int](data, `Age[`, 7))
	})
}

// Documents accepted current behavior: numeric conversions use plain Go
// conversions, so values outside the target range wrap or saturate the
// way Go conversions do — no range error is reported.
func TestAsNumberConversionQuirks(t *testing.T) {
	t.Run("overflowing int8 wraps", func(t *testing.T) {
		i8, ok := As[int8](MkInt(200))
		assert.True(t, ok)
		assert.Equal(t, int8(-56), i8)
	})
	t.Run("negative to uint64 reports ok with a platform-defined value", func(t *testing.T) {
		// float64 -> uint64 of a negative value is implementation-defined
		// in Go (0 on arm64, wraparound on amd64), so only ok is asserted
		u, ok := As[uint64](MkInt(-1))
		assert.True(t, ok)
		_ = u
	})
}
