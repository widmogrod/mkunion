package schema

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Compare orders values of different types by type rank:
// None < Bool < Number < String < Binary < List < Map
func TestCompareTypeOrdering(t *testing.T) {
	ordered := []struct {
		name  string
		value Schema
	}{
		{"none", &None{}},
		{"bool", MkBool(true)},
		{"number", MkInt(1)},
		{"string", MkString("some cool string")},
		{"binary", MkBinary([]byte("some cool string"))},
		{"list", &List{}},
		{"map", &Map{}},
	}
	for i, a := range ordered {
		for j, b := range ordered {
			if i == j {
				continue
			}
			want := 1
			if i < j {
				want = -1
			}
			t.Run(fmt.Sprintf("%s and %s = %d", a.name, b.name, want), func(t *testing.T) {
				assert.Equal(t, want, Compare(a.value, b.value))
			})
		}
	}
}

func TestCompare(t *testing.T) {
	useCases := map[string]struct {
		a, b Schema
		cmp  int
	}{
		"nil and nil = 0":    {nil, nil, 0},
		"nil and none = 0":   {nil, &None{}, 0},
		"none and nil = 0":   {&None{}, nil, 0},
		"none and none = 0":  {&None{}, &None{}, 0},
		"true and true = 0":  {MkBool(true), MkBool(true), 0},
		"true and false = 1": {MkBool(true), MkBool(false), 1},
		"string and string = 0": {
			MkString("some cool string"), MkString("some cool string"), 0,
		},
		"list and list = 0": {
			MkList(MkInt(1), MkInt(2), MkInt(3)),
			MkList(MkInt(1), MkInt(2), MkInt(3)),
			0,
		},
		"map and map = 0": {
			MkMap(MkField("a", MkInt(1)), MkField("b", MkInt(2)), MkField("c", MkInt(3))),
			MkMap(MkField("a", MkInt(1)), MkField("b", MkInt(2)), MkField("c", MkInt(3))),
			0,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.cmp, Compare(uc.a, uc.b))
		})
	}
}

// Reproduces: Compare on numbers does int(*x - *y), so any difference
// smaller than 1 collapses to 0 — 1.9 "equals" 1.2 in WHERE and sort.
func TestCompareNumbers(t *testing.T) {
	useCases := map[string]struct {
		a, b Schema
		cmp  int
	}{
		"1 and 1 = 0":       {MkInt(1), MkInt(1), 0},
		"2 and 1 = 1":       {MkInt(2), MkInt(1), 1},
		"1 and 2 = -1":      {MkInt(1), MkInt(2), -1},
		"1.9 and 1.2 = 1":   {MkFloat(1.9), MkFloat(1.2), 1},
		"1.2 and 1.9 = -1":  {MkFloat(1.2), MkFloat(1.9), -1},
		"-1.2 and -1.9 = 1": {MkFloat(-1.2), MkFloat(-1.9), 1},
		"0.1 and 0 = 1":     {MkFloat(0.1), MkFloat(0), 1},
		"huge and tiny = 1 (no int overflow)": {
			MkFloat(math.MaxFloat64), MkFloat(-math.MaxFloat64), 1,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.cmp, Compare(uc.a, uc.b))
		})
	}
}

// schema.Number is float64-backed by design. Integers are exact only up
// to ±2^53; above that adjacent int64 values collapse to the same number.
// This test documents that accepted limitation (see Number in model.go).
func TestNumberInt64PrecisionLimit(t *testing.T) {
	t.Run("integers up to 2^53 stay distinct", func(t *testing.T) {
		limit := int64(1) << 53
		assert.Equal(t, 1, Compare(MkInt(limit), MkInt(limit-1)))
	})
	t.Run("adjacent int64 above 2^53 collapse to equal", func(t *testing.T) {
		big := int64(1) << 60
		assert.Equal(t, 0, Compare(MkInt(big+1), MkInt(big)))
		assert.Equal(t, 0, Compare(MkInt(math.MaxInt64), MkInt(math.MaxInt64-1)))
	})
}

func TestGet(t *testing.T) {
	useCases := map[string]struct {
		data     Schema
		location string
		expected Schema
		found    bool
	}{
		"nested map": {
			data: MkMap(
				MkField("Data", MkMap(
					MkField("Age", MkInt(10)),
				)),
			),
			location: "Data.Age",
			expected: MkInt(10),
			found:    true,
		},
		"nested serialised union # accessor": {
			data: MkMap(
				MkField("Data", MkMap(
					MkField("schema.Map", MkMap(
						MkField("Age", MkMap(
							MkField("schema.Number", MkInt(10)),
						)),
					))))),
			location: "Data[*].Age[*]",
			expected: MkInt(10),
			found:    true,
		},
		"nested serialised union direct accessor": {
			data: MkMap(
				MkField("Data", MkMap(
					MkField("schema.Map", MkMap(
						MkField("Age", MkMap(
							MkField("schema.Number", MkInt(10)),
						)),
					))))),
			location: `Data["schema.Map"].Age["schema.Number"]`,
			expected: MkInt(10),
			found:    true,
		},
		"non existen path": {
			data: MkMap(
				MkField("Data", MkMap(
					MkField("schema.Map", MkMap(
						MkField("Age", MkMap(
							MkField("schema.Number", MkInt(10)),
						)),
					))))),
			location: `Data["schema.Map"].Date`,
			expected: nil,
			found:    false,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			result, found := GetSchema(uc.data, uc.location)
			assert.Equal(t, uc.expected, result)
			assert.Equal(t, uc.found, found)
		})
	}
}
