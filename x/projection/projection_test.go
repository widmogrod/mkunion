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

	err := DoLoad(ctx1, func(push func(Record[int]) error) error {
		for i := 0; i < 10; i++ {
			err := push(&Data[int]{
				Key:  fmt.Sprintf("key-%d", i),
				Data: i,
			})
			if err != nil {
				return fmt.Errorf("projection.Range: push: %w", err)
			}
		}
		return nil
	})
	assert.NoError(t, err)

	out2 := stream.NewInMemoryStream[float64]()
	ctx2 := NewPushAndPullInMemoryContext[int, float64](out1, out2)
	err = DoMap[int, float64](ctx2, func(x Record[int]) Record[float64] {
		return MatchRecordR1(
			x,
			func(x *Data[int]) Record[float64] {
				return &Data[float64]{
					Key:  x.Key,
					Data: float64(x.Data) * 2,
				}
			},
			func(x *Watermark[int]) Record[float64] {
				return &Watermark[float64]{
					EventTime: x.EventTime,
				}
			},
		)
	})
	assert.NoError(t, err)

	ctx4 := DoJoin[int, float64](out1, out2)
	err = DoSink(ctx4, func(x Record[Either[int, float64]]) error {
		return MatchRecordR1(
			x,
			func(x *Data[Either[int, float64]]) error {
				return MatchEitherR1(
					x.Data,
					func(x *Left[int, float64]) error {
						t.Log("left", x.Left)
						return nil
					},
					func(x *Right[int, float64]) error {
						t.Log("right", x.Right)
						return nil
					},
				)
			},
			func(x *Watermark[Either[int, float64]]) error {
				t.Log("watermark")
				return nil
			},
		)
	})
	assert.NoError(t, err)
}
