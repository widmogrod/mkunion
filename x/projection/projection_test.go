package projection

import (
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/stream"
	"math"
	"testing"
	"time"
)

func TestProjection_HappyPath(t *testing.T) {
	dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTime)

	state1 := &PullPushContextState{
		Offset:    nil,
		PullTopic: "",
		PushTopic: "topic-out-1",
	}

	ctx1 := NewPushAndPullInMemoryContext[any, int](state1, dataStream)

	err := DoLoad(ctx1, func(push func(data Data[int]) error) error {
		for i := 0; i < 10; i++ {
			err := push(&Record[int]{
				Key:  fmt.Sprintf("key-%d", i%2),
				Data: i,
			})
			if err != nil {
				return fmt.Errorf("projection.Range: push: %w", err)
			}
		}
		err := push(&Watermark[int]{
			EventTime: math.MaxInt64,
		})
		if err != nil {
			return fmt.Errorf("projection.Range: push: %w", err)
		}
		return nil
	})
	assert.NoError(t, err)

	state2 := &PullPushContextState{
		Offset:    nil,
		PullTopic: "topic-out-1",
		PushTopic: "topic-out-2",
	}
	ctx2 := NewPushAndPullInMemoryContext[int, float64](state2, dataStream)
	err = DoMap[int, float64](ctx2, func(x *Record[int]) *Record[float64] {
		return &Record[float64]{
			Key:       x.Key,
			Data:      float64(x.Data) * 2,
			EventTime: x.EventTime,
		}
	})
	assert.NoError(t, err)

	orderOfEvents := []string{}

	stateC := &JoinContextState{
		Offset1:     nil,
		PullTopic1:  "topic-out-1",
		Offset2:     nil,
		PullTopic2:  "topic-out-2",
		PushTopic:   "topic-out-3",
		LeftOrRight: true,
	}

	err = DoJoin[int, float64, Either[int, float64]](
		dataStream,
		stateC,
		func(x Either[*Record[int], *Record[float64]]) *Record[Either[int, float64]] {
			return MatchEitherR1(
				x,
				func(x *Left[*Record[int], *Record[float64]]) *Record[Either[int, float64]] {
					return &Record[Either[int, float64]]{
						Key: x.Left.Key,
						Data: &Left[int, float64]{
							Left: x.Left.Data,
						},
						EventTime: x.Left.EventTime,
					}
				},
				func(x *Right[*Record[int], *Record[float64]]) *Record[Either[int, float64]] {
					return &Record[Either[int, float64]]{
						Key: x.Right.Key,
						Data: &Right[int, float64]{
							Right: x.Right.Data,
						},
						EventTime: x.Right.EventTime,
					}
				},
			)
		},
	)
	assert.NoError(t, err)

	state3 := &PullPushContextState{
		Offset:    nil,
		PullTopic: "topic-out-3",
		PushTopic: "",
	}
	ctx5 := NewPushAndPullInMemoryContext[Either[int, float64], any](state3, dataStream)
	err = DoSink[Either[int, float64]](ctx5, func(x *Record[Either[int, float64]]) error {
		return MatchEitherR1(
			x.Data,
			func(x *Left[int, float64]) error {
				orderOfEvents = append(orderOfEvents, fmt.Sprintf("left-%d", x.Left))
				return nil
			},
			func(x *Right[int, float64]) error {
				orderOfEvents = append(orderOfEvents, fmt.Sprintf("right-%.2f", x.Right))
				return nil
			},
		)
	})
	assert.NoError(t, err)

	expectedOrder := []string{
		"left-0",
		"right-0.00",
		"left-1",
		"right-2.00",
		"left-2",
		"right-4.00",
		"left-3",
		"right-6.00",
		"left-4",
		"right-8.00",
		"left-5",
		"right-10.00",
		"left-6",
		"right-12.00",
		"left-7",
		"right-14.00",
		"left-8",
		"right-16.00",
		"left-9",
		"right-18.00",
	}

	if diff := cmp.Diff(expectedOrder, orderOfEvents); diff != "" {
		t.Fatalf("NewJoinPushAndPullContext: diff: (-want +got)\n%s", diff)
	}

	windowStore := NewWindowInMemoryStore[string]("window-store")

	state6 := &PullPushContextState{
		Offset:    nil,
		PullTopic: "topic-out-3",
		PushTopic: "topic-out-4",
	}
	ctx6 := NewPushAndPullInMemoryContext[Either[int, float64], string](state6, dataStream)
	wd := &FixedWindow{Width: math.MaxInt64}
	fm := &Discard{}
	td := &AtWatermark{}
	err = DoWindow[Either[int, float64], string](ctx6, windowStore, wd, fm, td, "", func(x Either[int, float64], agg string) (string, error) {
		var concat string
		if agg == "" {
			concat = fmt.Sprintf("%v", x)
		} else {
			concat = fmt.Sprintf("%s,%v", agg, x)
		}

		return concat, nil
	})
	assert.NoError(t, err)

	orderOfEvents = []string{}
	state7 := &PullPushContextState{
		Offset:    nil,
		PullTopic: "topic-out-4",
		PushTopic: "",
	}
	ctx7 := NewPushAndPullInMemoryContext[string, any](state7, dataStream)
	err = DoSink[string](ctx7, func(x *Record[string]) error {
		orderOfEvents = append(orderOfEvents, fmt.Sprintf("record-%s:%s", x.Key, x.Data))
		return nil
	})
	assert.NoError(t, err)

	expectedOrder = []string{
		"record-key-0:&{0},&{0},&{2},&{4},&{4},&{8},&{6},&{12},&{8},&{16}",
		"record-key-1:&{1},&{2},&{3},&{6},&{5},&{10},&{7},&{14},&{9},&{18}",
	}

	if diff := cmp.Diff(expectedOrder, orderOfEvents); diff != "" {
		t.Fatalf("NewJoinPushAndPullContext: diff: (-want +got)\n%s", diff)
	}
}

func TestProjection_Recovery(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	probabilityOfFailure := 0.20
	recoveryAttempts := uint8(50)

	dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTimeFixed(0))

	dataStream.SimulateRuntimeProblem(&stream.SimulateProblem{
		ErrorOnPullProbability: probabilityOfFailure,
		ErrorOnPush:            fmt.Errorf("simulated push error"),

		ErrorOnPushProbability: probabilityOfFailure,
		ErrorOnPull:            fmt.Errorf("simulated pull error"),
	})

	var store schemaless.Repository[SnapshotState] = schemaless.NewInMemoryRepository[SnapshotState]()

	recovery :=
		NewRecoveryOptions(
			"recovery-load",
			func() SnapshotState {
				return &PullPushContextState{
					Offset:    nil,
					PullTopic: "",
					PushTopic: "topic-out-1",
				}
			},
			store,
		).
			WithMaxRecoveryAttempts(recoveryAttempts).
			WithAutoSnapshot(false)

	err := Recovery[*PullPushContextState](
		recovery,
		func(state *PullPushContextState) (*PushAndPullInMemoryContext[any, int], error) {
			ctx := NewPushAndPullInMemoryContext[any, int](state, dataStream)
			ctx.SimulateRuntimeProblem(&SimulateProblem{
				ErrorOnPushOutProbability: probabilityOfFailure,
				ErrorOnPushOut:            fmt.Errorf("simulated push error"),
				ErrorOnPullInProbability:  probabilityOfFailure,
				ErrorOnPullIn:             fmt.Errorf("simulated pull error"),
			})
			return ctx, nil
		},
		func(ctx1 *PushAndPullInMemoryContext[any, int]) error {
			state := ctx1.CurrentState()
			startValue := 0
			off := state.(*PullPushContextState).Offset
			if off.IsSet() {
				_, err := fmt.Sscanf(string(*off), "%d", &startValue)
				if err != nil {
					return fmt.Errorf("projection.Load: parse offset: %w", err)
				}
			}

			for i := startValue; i < 10; i++ {
				time.Sleep(50 * time.Millisecond)
				et := 50 * time.Millisecond
				et *= time.Duration(i)
				err := ctx1.PushOut(&Record[int]{
					Key:       fmt.Sprintf("key-%d", i%2),
					Data:      i,
					EventTime: int64(et),
				})
				if err != nil {
					return fmt.Errorf("projection.Range: push: %w", err)
				}

				off := stream.Offset(fmt.Sprintf("%d", i+1))
				err = recovery.Snapshot(&PullPushContextState{
					Offset:    &off,
					PullTopic: "",
					PushTopic: "topic-out-1",
				})
				if err != nil {
					return fmt.Errorf("projection.Range: snapshot: %w", err)
				}
			}

			err := ctx1.PushOut(&Watermark[int]{
				EventTime: math.MaxInt64,
			})
			if err != nil {
				return fmt.Errorf("projection.Range: push: %w", err)
			}

			return nil
		},
	)
	assert.NoError(t, err)

	t.Log("WINDOW")

	recovery =
		NewRecoveryOptions(
			"recovery-window",
			func() SnapshotState {
				state := &PullPushContextState{
					Offset:    nil,
					PullTopic: "topic-out-1",
					PushTopic: "topic-out-2",
				}
				return state
			},
			store,
		).
			WithMaxRecoveryAttempts(recoveryAttempts).
			WithAutoSnapshot(true)

	//TODO:
	// - [√] Data should have offset field, otherwise we can't recover from the last offset
	// - [√] ctx.AckOffset(item) for consistent asynchronous snapshots
	// - [√] mkunion generate type registry for parametrised interface types like Record[int] or Record[float64]
	// - [√] DoWindow state should catch also last watermark with last offset, so that windowStore keys that have windows closed, can be deleted

	windowStore := NewWindowInMemoryStore[float64]("window-store")

	err = Recovery(
		recovery,
		func(state *PullPushContextState) (*PushAndPullInMemoryContext[int, float64], error) {
			ctx := NewPushAndPullInMemoryContext[int, float64](state, dataStream)
			ctx.SimulateRuntimeProblem(&SimulateProblem{
				ErrorOnPushOutProbability: probabilityOfFailure,
				ErrorOnPushOut:            fmt.Errorf("simulated push error"),
				ErrorOnPullInProbability:  probabilityOfFailure,
				ErrorOnPullIn:             fmt.Errorf("simulated pull error"),
			})
			return ctx, nil
		},
		func(ctx *PushAndPullInMemoryContext[int, float64]) error {
			wd := &FixedWindow{Width: 200 * time.Millisecond}
			fm := &Discard{}
			td := &AtWatermark{}
			return DoWindow[int, float64](ctx, windowStore, wd, fm, td, 0, func(x int, agg float64) (float64, error) {
				time.Sleep(50 * time.Millisecond)
				return float64(x) + agg, nil
			})
		},
	)
	assert.NoError(t, err)

	t.Log("SINK")

	var orderOfEvents []string
	var orderOfUniquer = make(map[string]struct{})

	recovery =
		NewRecoveryOptions(
			"recovery-sink",
			func() SnapshotState {
				return &PullPushContextState{
					Offset:    nil,
					PullTopic: "topic-out-2",
					PushTopic: "topic-out-3",
				}
			},
			store,
		).WithMaxRecoveryAttempts(recoveryAttempts)

	err = Recovery(
		recovery,
		func(state *PullPushContextState) (*PushAndPullInMemoryContext[float64, float64], error) {
			ctx := NewPushAndPullInMemoryContext[float64, float64](state, dataStream)
			ctx.SimulateRuntimeProblem(&SimulateProblem{
				ErrorOnPushOutProbability: probabilityOfFailure,
				ErrorOnPushOut:            fmt.Errorf("simulated push error"),
				ErrorOnPullInProbability:  probabilityOfFailure,
				ErrorOnPullIn:             fmt.Errorf("simulated pull error"),
			})
			return ctx, nil
		},
		func(ctx *PushAndPullInMemoryContext[float64, float64]) error {
			timer := time.NewTimer(100 * time.Millisecond)
			defer timer.Stop()
			for {
				select {
				case <-timer.C:
				default:
					val, err := ctx.PullIn()
					if err != nil {
						if errors.Is(err, stream.ErrNoMoreNewDataInStream) {
							return nil
						}
						return fmt.Errorf("projection.DoSink: pull: %w", err)
					}
					timer.Reset(100 * time.Millisecond)

					log.Debugf("projection.DoSink: pull value: %#v", val)

					err = MatchDataR1(
						val.Data,
						func(x *Record[float64]) error {
							time.Sleep(50 * time.Millisecond)

							entry := fmt.Sprintf("record-%s:%2f", x.Key, x.Data)
							if _, ok := orderOfUniquer[entry]; ok {
								return nil
							}
							orderOfUniquer[entry] = struct{}{}
							orderOfEvents = append(orderOfEvents, entry)

							return nil
						},
						func(x *Watermark[float64]) error {
							err := ctx.AckWatermark(&x.EventTime)
							if err != nil {
								return fmt.Errorf("projection.DoSink: ack watermark: %w", err)
							}

							return nil
						},
					)
					if err != nil {
						return fmt.Errorf("projection.DoSink: sink: %w", err)
					}
					err = ctx.AckOffset(val.Offset)
					if err != nil {
						return fmt.Errorf("projection.DoSink: ack offset: %w", err)
					}
				}
			}
		},
	)
	assert.NoError(t, err)

	expectedOrder := []string{
		"record-key-0:2.000000",
		"record-key-1:4.000000",
		"record-key-0:10.000000",
		"record-key-1:12.000000",
		"record-key-0:8.000000",
		"record-key-1:9.000000",
	}

	if diff := cmp.Diff(expectedOrder, orderOfEvents); diff != "" {
		t.Fatalf("diff: (-want +got)\n%s", diff)
	}

	results, err := store.FindingRecords(schemaless.FindingRecords[schemaless.Record[SnapshotState]]{
		RecordType: RecoveryRecordType,
	})
	assert.NoError(t, err)
	assert.Len(t, results.Items, 3)

	for _, result := range results.Items {
		t.Logf("recovery state: %v", result)
	}
}
