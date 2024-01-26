package projection

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/stream"
	"math/rand"
	"time"
)

//go:generate go run ../../cmd/mkunion/main.go

//go:tag mkunion:"Data,noserde"
type (
	Record[A any] struct {
		Key       string
		Data      A
		EventTime EventTime
	}

	Watermark[A any] struct {
		Key       string
		EventTime EventTime
	}
)

type EventTime = int64

func MkEventTimeFromTime(x time.Time) EventTime {
	return x.UnixNano()
}

func RecordToStreamItem[A any](topic string, x Data[A]) *stream.Item[A] {
	return MatchDataR1[A, *stream.Item[A]](x,
		func(x *Record[A]) *stream.Item[A] {
			return &stream.Item[A]{
				Topic:     topic,
				Key:       x.Key,
				Data:      x.Data,
				EventTime: &x.EventTime,
				Offset:    nil,
			}
		},
		func(x *Watermark[A]) *stream.Item[A] {
			var zero A
			return &stream.Item[A]{
				Topic:     topic,
				Key:       x.Key,
				Data:      zero,
				EventTime: &x.EventTime,
				Offset:    nil,
			}
		},
	)
}

func StreamItemToRecord[A any](x *stream.Item[A]) Data[A] {
	return &Record[A]{
		Key:       x.Key,
		Data:      x.Data,
		EventTime: *x.EventTime,
	}
}
func StreamItemToRecordSetData[A, B any](x *stream.Item[A], data B) *Record[B] {
	return &Record[B]{
		Key:       x.Key,
		Data:      data,
		EventTime: *x.EventTime,
	}
}

type (
	PushAndPull[A, B any] interface {
		PullOnly[A]
		PushOnly[B]
		SnapshotContext
	}
	PullOnly[A any] interface {
		PullIn() (Data[A], error)
		SnapshotContext
	}
	PushOnly[A any] interface {
		PushOut(Data[A]) error
		SnapshotContext
	}
	SnapshotContext interface {
		CurrentState() SnapshotState
	}
)

func NewPushAndPullInMemoryContext[A, B any](state *PullPushContextState, in stream.Stream[A], out stream.Stream[B]) *PushAndPullInMemoryContext[A, B] {
	return &PushAndPullInMemoryContext[A, B]{
		state:  state,
		input:  in,
		output: out,
	}
}

func NewPushOnlyInMemoryContext[A any](state *PullPushContextState, out stream.Stream[A]) PushOnly[A] {
	return &PushAndPullInMemoryContext[any, A]{
		state:  state,
		output: out,
	}
}

func NewPullOnlyInMemoryContext[A any](state *PullPushContextState, in stream.Stream[A]) PullOnly[A] {
	return &PushAndPullInMemoryContext[A, any]{
		state: state,
		input: in,
	}
}

var _ PushAndPull[int, int] = (*PushAndPullInMemoryContext[int, int])(nil)

type PushAndPullInMemoryContext[A, B any] struct {
	state *PullPushContextState

	input    stream.Stream[A]
	output   stream.Stream[B]
	simulate *SimulateProblem
}

func (c *PushAndPullInMemoryContext[A, B]) PullIn() (Data[A], error) {
	if c.simulate != nil && c.simulate.ErrorOnPullIn != nil {
		if rand.Float64() < c.simulate.ErrorOnPullInProbability {
			return nil, c.simulate.ErrorOnPullIn
		}
	}

	pull := c.pullCommand(c.state)
	item, err := c.input.Pull(pull)
	if err != nil {
		return nil, fmt.Errorf("projection.PushAndPullInMemoryContext: PullIn: %w", err)
	}

	c.state.Offset = item.Offset

	result := StreamItemToRecord(item)
	return result, nil
}

func (c *PushAndPullInMemoryContext[A, B]) PushOut(x Data[B]) error {
	if c.simulate != nil && c.simulate.ErrorOnPushOut != nil {
		if rand.Float64() < c.simulate.ErrorOnPushOutProbability {
			return c.simulate.ErrorOnPushOut
		}
	}

	item := RecordToStreamItem(c.state.PushTopic, x)

	err := c.output.Push(item)
	if err != nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: PushOut: %w", err)
	}
	return nil
}

func (c *PushAndPullInMemoryContext[A, B]) CurrentState() SnapshotState {
	return c.state
}

func (c *PushAndPullInMemoryContext[A, B]) pullCommand(x *PullPushContextState) stream.PullCMD {
	if x.Offset == nil {
		return &stream.FromBeginning{
			Topic: x.PullTopic,
		}
	}

	return &stream.FromOffset{
		Topic:  x.PullTopic,
		Offset: x.Offset,
	}
}

type SimulateProblem struct {
	ErrorOnPullInProbability float64
	ErrorOnPullIn            error

	ErrorOnPushOutProbability float64
	ErrorOnPushOut            error
}

func (c *PushAndPullInMemoryContext[A, B]) SimulateRuntimeProblem(x *SimulateProblem) {
	c.simulate = x
}

type simulationProblemAware interface {
	SimulateRuntimeProblem(x *SimulateProblem)
}

func InjectRuntimeProblem(ctx any, x *SimulateProblem) {
	if ctx, ok := ctx.(simulationProblemAware); ok {
		ctx.SimulateRuntimeProblem(x)
	}
}

func DoLoad[A any](ctx PushOnly[A], f func(push func(record *Record[A]) error) error) error {
	err := f(func(record *Record[A]) error {
		return ctx.PushOut(record)
	})
	if err != nil {
		return fmt.Errorf("projection.DoLoad: load: %w", err)
	}
	return nil
}

func DoMap[A, B any](ctx PushAndPull[A, B], f func(*Record[A]) *Record[B]) error {
	for {
		val, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, stream.ErrEndOfStream) {
				return nil
			}
			return fmt.Errorf("projection.DoMap: pull: %w", err)
		}

		err = MatchDataR1(
			val,
			func(x *Record[A]) error {
				y := f(x)
				err = ctx.PushOut(y)
				if err != nil {
					return fmt.Errorf("projection.DoMap: push: %w", err)
				}

				return nil
			},
			func(x *Watermark[A]) error {
				// TODO do snapshot
				return nil
			},
		)

		if err != nil {
			return fmt.Errorf("projection.DoMap: map: %w", err)
		}
	}
}

func DoSink[A any](ctx PullOnly[A], f func(*Record[A]) error) error {
	for {
		val, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, stream.ErrEndOfStream) {
				return nil
			}
			return fmt.Errorf("projection.DoSink: pull: %w", err)
		}

		err = MatchDataR1(
			val,
			func(x *Record[A]) error {
				return f(x)
			},
			func(x *Watermark[A]) error {
				// TODO do snapshot
				return nil
			},
		)
		if err != nil {
			return fmt.Errorf("projection.DoSink: sink: %w", err)
		}
	}
}

//go:tag mkunion:"Either,noserde"
type (
	Left[A, B any] struct {
		Left A
	}
	Right[A, B any] struct {
		Right B
	}
)

func NewJoinInMemoryContext[A, B, C any](
	state *JoinContextState,
	a stream.Stream[A],
	b stream.Stream[B],
	out stream.Stream[C],
) PushAndPull[Either[A, B], C] {
	return &InMemoryJoinContext[A, B, C]{
		state:  state,
		a:      a,
		b:      b,
		output: out,
		mod:    true,
	}
}

type InMemoryJoinContext[A, B, C any] struct {
	a stream.Stream[A]
	b stream.Stream[B]

	output stream.Stream[C]

	state *JoinContextState

	mod  bool
	endA bool
	endB bool
}

var _ PushAndPull[Either[any, any], any] = (*InMemoryJoinContext[any, any, any])(nil)

func (i *InMemoryJoinContext[A, B, C]) PullIn() (Data[Either[A, B]], error) {
	if i.endA && i.endB {
		return nil, stream.ErrEndOfStream
	}

	// TODO add watermark support

	if !i.endA && i.mod == true {
		i.mod = !i.mod

		var pull stream.PullCMD
		if i.state.Offset1 == nil {
			pull = &stream.FromBeginning{
				Topic: i.state.PullTopic1,
			}
		} else {
			pull = &stream.FromOffset{
				Topic:  i.state.PullTopic1,
				Offset: i.state.Offset1,
			}
		}

		val, err := i.a.Pull(pull)
		if err != nil {
			if errors.Is(err, stream.ErrEndOfStream) {
				i.endA = true
				return i.PullIn()
			}
			return nil, fmt.Errorf("projection.InMemoryJoinContext: PullIn left: %w", err)
		}

		i.state.Offset1 = val.Offset

		return StreamItemToRecordSetData[A, Either[A, B]](val, &Left[A, B]{Left: val.Data}), nil
	} else if !i.endB && i.mod == false {
		i.mod = !i.mod

		var pull stream.PullCMD
		if i.state.Offset2 == nil {
			pull = &stream.FromBeginning{
				Topic: i.state.PullTopic2,
			}
		} else {
			pull = &stream.FromOffset{
				Topic:  i.state.PullTopic2,
				Offset: i.state.Offset2,
			}
		}

		val, err := i.b.Pull(pull)
		if err != nil {
			if errors.Is(err, stream.ErrEndOfStream) {
				i.endB = true
				return i.PullIn()
			}
			return nil, fmt.Errorf("projection.InMemoryJoinContext: PullIn right: %w", err)
		}

		i.state.Offset2 = val.Offset

		return StreamItemToRecordSetData[B, Either[A, B]](val, &Right[A, B]{Right: val.Data}), nil
	}

	i.mod = !i.mod
	return i.PullIn()
}

func (i *InMemoryJoinContext[A, B, C]) PushOut(x Data[C]) error {
	item := RecordToStreamItem(i.state.PushTopic, x)

	err := i.output.Push(item)
	if err != nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: PushOut: %w", err)
	}
	return nil
}

func (i *InMemoryJoinContext[A, B, C]) CurrentState() SnapshotState {
	return i.state
}
