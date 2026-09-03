package projection

import (
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/stream"
)

// TestDoWindow_FaultInjection_ConvergesToFaultFreeResult drives DoWindow
// through Recovery with simulated faults on every pull and push, over more
// windows than one flush page holds. Whatever the fault schedule, the
// output must converge to exactly the fault-free aggregates: duplicates
// are allowed (at-least-once), but every emitted value must be the final
// aggregate of its window - never a partial built from re-merged records.
func TestDoWindow_FaultInjection_ConvergesToFaultFreeResult(t *testing.T) {
	// keys*windowsPerKey = 39 windows > DefaultWindowFlushPageSize, so the
	// flush spans multiple pages; 78 records is deliberately not a multiple
	// of DefaultSnapshotEveryNRecords, so a crash near the watermark always
	// has an unpersisted tail of records to replay
	const (
		keys             = 13
		windowsPerKey    = 3
		recordsPerWindow = 2
	)
	windowWidth := int64(100) // nanoseconds

	dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTimeFixed(0))

	// fault-free load; expected[key][windowEnd] = sum of records in that window
	expected := map[string]map[EventTime]int{}
	loadCtx := NewPushAndPullInMemoryContext[any, int](&PullPushContextState{
		PushTopic: "in",
	}, dataStream)
	// each window gets one record early in the stream and one late, in
	// reverse window order: the tail of the stream then belongs to the
	// windows that a flush deletes first, so a replay of the tail after a
	// crashed flush hits exactly the windows whose dedup offsets are gone
	err := DoLoad(loadCtx, func(push func(record Data[int]) error) error {
		for r := 0; r < recordsPerWindow; r++ {
			for w := 0; w < windowsPerKey; w++ {
				windowIdx := w
				if r%2 == 1 {
					windowIdx = windowsPerKey - 1 - w
				}
				for k := 0; k < keys; k++ {
					keyIdx := k
					if r%2 == 1 {
						keyIdx = keys - 1 - k
					}
					key := fmt.Sprintf("key-%d", keyIdx)
					if expected[key] == nil {
						expected[key] = map[EventTime]int{}
					}
					windowEnd := EventTime(int64(windowIdx)*windowWidth + windowWidth)
					value := windowIdx*100 + keyIdx*10 + r
					expected[key][windowEnd] += value
					if err := push(&Record[int]{
						Key:       key,
						Data:      value,
						EventTime: int64(windowIdx) * windowWidth,
					}); err != nil {
						return err
					}
				}
			}
		}
		return push(&Watermark[int]{EventTime: math.MaxInt64})
	})
	require.NoError(t, err)

	// from now on every stream pull and push can fail
	dataStream.SimulateRuntimeProblem(&stream.SimulateProblem{
		ErrorOnPullProbability: 0.1,
		ErrorOnPull:            fmt.Errorf("simulated pull error"),
		ErrorOnPushProbability: 0.1,
		ErrorOnPush:            fmt.Errorf("simulated push error"),
	})

	store := schemaless.NewInMemoryRepository[SnapshotState]()
	recovery := NewRecoveryOptions(
		"recovery-window-convergence",
		func() SnapshotState {
			return &PullPushContextState{
				PullTopic: "in",
				PushTopic: "out",
			}
		},
		store,
	).WithMaxRecoveryAttempts(math.MaxUint8)

	windowStore := NewWindowInMemoryStore[int]("window")
	err = Recovery(
		recovery,
		func(state *PullPushContextState) (*PushAndPullInMemoryContext[int, int], error) {
			ctx := NewPushAndPullInMemoryContext[int, int](state, dataStream)
			ctx.SimulateRuntimeProblem(&SimulateProblem{
				ErrorOnPullInProbability:  0.1,
				ErrorOnPullIn:             fmt.Errorf("simulated pull error"),
				ErrorOnPushOutProbability: 0.1,
				ErrorOnPushOut:            fmt.Errorf("simulated push error"),
			})
			return ctx, nil
		},
		func(ctx *PushAndPullInMemoryContext[int, int]) error {
			return DoWindow[int, int](ctx, windowStore, &FixedWindow{Width: time.Duration(windowWidth)}, &Discard{}, &AtWatermark{}, 0,
				func(x int, agg int) (int, error) {
					return x + agg, nil
				}, recovery)
		},
	)
	require.NoError(t, err)

	// fault-free read-back of everything the window stage emitted
	dataStream.SimulateRuntimeProblem(nil)
	got := map[string]map[EventTime][]int{}
	sawEndOfStream := false
	readCtx := NewPushAndPullInMemoryContext[int, int](&PullPushContextState{
		PullTopic: "out",
	}, dataStream)
	for {
		item, err := readCtx.PullIn()
		if errors.Is(err, stream.ErrNoMoreNewDataInStream) {
			break
		}
		require.NoError(t, err)
		require.NoError(t, readCtx.AckOffset(item.Offset))

		MatchDataR0(
			item.Data,
			func(x *Record[int]) {
				if got[x.Key] == nil {
					got[x.Key] = map[EventTime][]int{}
				}
				got[x.Key][x.EventTime] = append(got[x.Key][x.EventTime], x.Data)
			},
			func(x *Watermark[int]) {
				if IsWatermarkMarksEndOfStream(x.EventTime) {
					sawEndOfStream = true
				}
			},
		)
	}

	assert.True(t, sawEndOfStream, "downstream never received the end-of-stream watermark")

	// every window must be emitted, and every emission (duplicates included)
	// must carry the fault-free aggregate
	for key, windows := range expected {
		for windowEnd, want := range windows {
			values := got[key][windowEnd]
			if assert.NotEmpty(t, values, "window key=%s end=%d was never flushed", key, windowEnd) {
				for _, v := range values {
					assert.Equal(t, want, v, "window key=%s end=%d emitted %d, fault-free value is %d", key, windowEnd, v, want)
				}
			}
		}
	}

	// nothing extra was invented
	for key, windows := range got {
		for windowEnd := range windows {
			_, ok := expected[key][windowEnd]
			assert.True(t, ok, "unexpected window key=%s end=%d", key, windowEnd)
		}
	}
}
