package testutil

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"math"
	"runtime"
	"testing"
)

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
		//t.Skip("not implemented")
		if runtime.GOARCH != "arm64" {
			t.Skip("skipping test that are for ARM64")
		}

		s := schema.FromGo(max)
		assert.Equal(t, &schema.Map{
			"Int":     schema.MkInt(math.MaxInt),
			"Int8":    schema.MkInt(math.MaxInt8),
			"Int16":   schema.MkInt(math.MaxInt16),
			"Int32":   schema.MkInt(math.MaxInt32),
			"Int64":   schema.MkInt(math.MaxInt64),
			"Float32": schema.MkFloat(math.MaxFloat32),
			"Float64": schema.MkFloat(math.MaxFloat64),
			"Uint":    schema.MkUint(math.MaxUint),
			"Uint8":   schema.MkUint(math.MaxInt8),
			"Uint16":  schema.MkUint(math.MaxUint16),
			"Uint32":  schema.MkUint(math.MaxUint32),
			"Uint64":  schema.MkUint(math.MaxUint64),
		}, s)
	})
}
