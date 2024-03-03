package stream

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// SpecComparable is a helper function to test if two offsets are comparable
// a must be higher than b in order to pass the test
func SpecComparable(t *testing.T, a, b Offset) {
	t.Run("a < b", func(t *testing.T) {
		cmp, err := OffsetCompare(a, b)
		assert.NoError(t, err)
		assert.Equal(t, int8(-1), cmp)
	})
	t.Run("a = b", func(t *testing.T) {
		cmp, err := OffsetCompare(a, a)
		assert.NoError(t, err)
		assert.Equal(t, int8(0), cmp)
	})
	t.Run("a > b", func(t *testing.T) {
		cmp, err := OffsetCompare(b, a)
		assert.NoError(t, err)
		assert.Equal(t, int8(1), cmp)
	})
}
