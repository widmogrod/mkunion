package projection

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/stream"
	"testing"
)

func TestProjection(t *testing.T) {
	out1 := stream.NewInMemoryStream[int]()
	ctx1 := NewPushOnlyInMemoryContext[int](out1)

	err := DoLoad(ctx1, func(push func(int) error) error {
		for i := 0; i < 10; i++ {
			err := push(i)
			if err != nil {
				return fmt.Errorf("projection.Range: push: %w", err)
			}
		}
		return nil
	})
	assert.NoError(t, err)

	out2 := stream.NewInMemoryStream[float64]()
	ctx2 := NewPushAndPullInMemoryContext[int, float64](out1, out2)
	err = DoMap[int, float64](ctx2, func(x int) float64 {
		return float64(x) * 2
	})
	assert.NoError(t, err)

	ctx4 := DoJoin[int, float64](out1, out2)
	err = DoSink(ctx4, func(x *Either[int, float64]) error {
		if x.Left != nil {
			t.Logf("left  = %v", *x.Left)
		} else {
			t.Logf("right = %v", *x.Right)
		}
		return nil
	})
}
