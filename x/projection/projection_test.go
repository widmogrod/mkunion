package projection

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProjection(t *testing.T) {
	out1 := NewInMemoryStream[int]()
	ctx1 := NewPushOnlyInMemoryContext[int](out1)

	err := Range(ctx1, 10)
	assert.NoError(t, err)

	out2 := NewInMemoryStream[float64]()
	ctx2 := NewPushAndPullInMemoryContext[int, float64](out1, out2)
	err = Do[int, float64](ctx2, func(x int) float64 {
		return float64(x) * 2
	})
	assert.NoError(t, err)

	ctx3 := NewPullOnlyInMemoryContext[float64](out2)
	err = Drain[float64](ctx3, t.Logf)
	assert.NoError(t, err)
}
