package projection

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/stream"
	"math"
	"testing"
)

func TestProjection_HappyPath(t *testing.T) {
	out1 := stream.NewInMemoryStream[int](stream.WithSystemTime)
	state1 := NewSnapshotStateForInMemoryContext("load-1", "topic-in-1", "topic-out-1")
	ctx1 := NewPushOnlyInMemoryContext[int](state1, out1)

	err := DoLoad(ctx1, func(push func(*Record[int]) error) error {
		for i := 0; i < 10; i++ {
			err := push(&Record[int]{
				Key:  fmt.Sprintf("key-%d", i%2),
				Data: i,
			})
			if err != nil {
				return fmt.Errorf("projection.Range: push: %w", err)
			}
		}
		return nil
	})
	assert.NoError(t, err)

	out2 := stream.NewInMemoryStream[float64](stream.WithSystemTime)
	state2 := NewSnapshotStateForInMemoryContext("map-1", "topic-out-1", "topic-out-2")
	ctx2 := NewPushAndPullInMemoryContext[int, float64](state2, out1, out2)
	err = DoMap[int, float64](ctx2, func(x *Record[int]) *Record[float64] {
		return &Record[float64]{
			Key:       x.Key,
			Data:      float64(x.Data) * 2,
			EventTime: x.EventTime,
		}
	})
	assert.NoError(t, err)

	orderOfEvents := []string{}
	out3 := stream.NewInMemoryStream[float64](stream.WithSystemTime)
	ctx4 := NewJoinInMemoryContext[int, float64, float64](
		out1, "topic-out-1",
		out2, "topic-out-2",
		out3, "topic-out-3")

	err = DoSink[Either[int, float64]](ctx4, func(x *Record[Either[int, float64]]) error {
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

	out4 := stream.NewInMemoryStream[string](stream.WithSystemTime)
	ctx5 := NewJoinInMemoryContext[int, float64, string](
		out1, "topic-out-1",
		out2, "topic-out-2",
		out4, "topic-out-4")
	err = DoWindow(ctx5, func(x Either[int, float64], agg string) (string, error) {
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
	state3 := NewSnapshotStateForInMemoryContext("sink-2", "topic-out-4", "topic-out-3")
	ctx6 := NewPullOnlyInMemoryContext[string](state3, out4)
	err = DoSink[string](ctx6, func(x *Record[string]) error {
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
	probabilityOfFailure := 0.05
	recoveryAttempts := uint8(math.MaxUint8)

	out1 := stream.NewInMemoryStream[int](stream.WithSystemTime)
	out1.SimulateRuntimeProblem(&stream.SimulateProblem{
		ErrorOnPullProbability: probabilityOfFailure,
		ErrorOnPush:            fmt.Errorf("simulated push error"),

		ErrorOnPushProbability: probabilityOfFailure,
		ErrorOnPull:            fmt.Errorf("simulated pull error"),
	})

	store := NewSnapshotStore()

	recovery := NewRecoveryOptions("recovery-load", "in-1", "out-1", store).WithMaxRecoveryAttempts(recoveryAttempts)
	err := Recovery(recovery, func(state SnapshotState) error {
		ctx1 := NewPushOnlyInMemoryContext[int](state, out1)
		InjectRuntimeProblem(ctx1, &SimulateProblem{
			ErrorOnPushOutProbability: probabilityOfFailure,
			ErrorOnPushOut:            fmt.Errorf("simulated push error"),
			ErrorOnPullInProbability:  probabilityOfFailure,
			ErrorOnPullIn:             fmt.Errorf("simulated pull error"),
		})

		return DoLoad(ctx1, func(push func(*Record[int]) error) error {
			for i := 0; i < 10; i++ {
				err := push(&Record[int]{
					Key:       fmt.Sprintf("key-%d", i%2),
					Data:      i,
					EventTime: stream.WithSystemTime(),
				})
				if err != nil {
					return fmt.Errorf("projection.Range: push: %w", err)
				}
			}
			return nil
		})
	})
	assert.NoError(t, err)

	var orderOfEvents []string
	var orderOfUniquer = make(map[string]struct{})

	out2 := stream.NewInMemoryStream[float64](stream.WithSystemTime)
	out2.SimulateRuntimeProblem(&stream.SimulateProblem{
		ErrorOnPullProbability: probabilityOfFailure,
		ErrorOnPush:            fmt.Errorf("simulated push error"),

		ErrorOnPushProbability: probabilityOfFailure,
		ErrorOnPull:            fmt.Errorf("simulated pull error"),
	})

	recovery = NewRecoveryOptions("recovery-load", "out-1", "out-2", store).WithMaxRecoveryAttempts(recoveryAttempts)
	err = Recovery(recovery, func(state SnapshotState) error {
		ctx2 := NewPushAndPullInMemoryContext[int, float64](state, out1, out2)
		InjectRuntimeProblem(ctx2, &SimulateProblem{
			ErrorOnPushOutProbability: probabilityOfFailure,
			ErrorOnPushOut:            fmt.Errorf("simulated push error"),
			ErrorOnPullInProbability:  probabilityOfFailure,
			ErrorOnPullIn:             fmt.Errorf("simulated pull error"),
		})
		return DoSink[int](ctx2, func(x *Record[int]) error {
			entry := fmt.Sprintf("record-%s:%d", x.Key, x.Data)
			if _, ok := orderOfUniquer[entry]; ok {
				return nil
			}
			orderOfUniquer[entry] = struct{}{}
			orderOfEvents = append(orderOfEvents, entry)
			return nil
		})
	})
	assert.NoError(t, err)

	expectedOrder := []string{
		"record-key-0:0",
		"record-key-1:1",
		"record-key-0:2",
		"record-key-1:3",
		"record-key-0:4",
		"record-key-1:5",
		"record-key-0:6",
		"record-key-1:7",
		"record-key-0:8",
		"record-key-1:9",
	}

	if diff := cmp.Diff(expectedOrder, orderOfEvents); diff != "" {
		t.Fatalf("diff: (-want +got)\n%s", diff)
	}
}
