package schema

import (
	"github.com/stretchr/testify/assert"
	"math"
	"runtime"
	"testing"
)

type Max struct {
	Int   int
	Int8  int8
	Int16 int16
	Int32 int32
	Int64 int64

	Float32 float32
	Float64 float64

	Uint   uint
	Uint8  uint8
	Uint16 uint16
	Uint32 uint32
	Uint64 uint64
}

func TestMaxScalars(t *testing.T) {
	max := Max{
		Int:     math.MaxInt,
		Int8:    math.MaxInt8,
		Int16:   math.MaxInt16,
		Int32:   math.MaxInt32,
		Int64:   math.MaxInt64,
		Float32: math.MaxFloat32,
		Float64: math.MaxFloat64,
		Uint:    math.MaxUint,
		Uint8:   math.MaxInt8,
		Uint16:  math.MaxUint16,
		Uint32:  math.MaxUint32,
		Uint64:  math.MaxUint64,
	}

	t.Run("max scalars for respective values contain correct value", func(t *testing.T) {
		t.Skip("not implemented")
		if runtime.GOARCH != "arm64" {
			t.Skip("skipping test that are for ARM64")
		}

		s := FromPrimitiveGo(max)
		assert.Equal(t, &Map{
			"Int":     MkInt(math.MaxInt),
			"Int8":    MkInt(math.MaxInt8),
			"Int16":   MkInt(math.MaxInt16),
			"Int32":   MkInt(math.MaxInt32),
			"Int64":   MkInt(math.MaxInt64),
			"Float32": MkFloat(math.MaxFloat32),
			"Float64": MkFloat(math.MaxFloat64),
			//"Uint":    MkInt(math.MaxUint),
			"Uint8":  MkInt(math.MaxInt8),
			"Uint16": MkInt(math.MaxUint16),
			"Uint32": MkInt(math.MaxUint32),
			//"Uint64":  MkInt(math.MaxUint64),
		}, s)
	})
}
