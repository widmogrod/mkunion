package projection

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/stream"
	"math/rand"
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
	}
	PullOnly[A any] interface {
		PullIn() (Data[A], error)
	}
	PushOnly[A any] interface {
		PushOut(Data[A]) error
	}
)

func NewPushAndPullInMemoryContext[A, B any](state SnapshotState, in stream.Stream[A], out stream.Stream[B]) *PushAndPullInMemoryContext[A, B] {
	return &PushAndPullInMemoryContext[A, B]{
		state:  state,
		input:  in,
		output: out,
	}
}

func NewPushOnlyInMemoryContext[A any](state SnapshotState, out stream.Stream[A]) PushOnly[A] {
	return &PushAndPullInMemoryContext[any, A]{
		state:  state,
		output: out,
	}
}

func NewPullOnlyInMemoryContext[A any](state SnapshotState, in stream.Stream[A]) PullOnly[A] {
	return &PushAndPullInMemoryContext[A, any]{
		state: state,
		input: in,
	}
}

var _ PushAndPull[int, int] = (*PushAndPullInMemoryContext[int, int])(nil)

type PushAndPullInMemoryContext[A, B any] struct {
	state         SnapshotState
	snapshotStore *SnapshotStore

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

func (c *PushAndPullInMemoryContext[A, B]) pullCommand(x SnapshotState) stream.PullCMD {
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

func NewSnapshotStateForInMemoryContext(id, pullTopic, pushTopic string) SnapshotState {
	return SnapshotState{
		ID:        id,
		PullTopic: pullTopic,
		PushTopic: pushTopic,
		Offset:    nil,
	}
}

var (
	ErrSnapshotIDEmpty  = fmt.Errorf("empty snapshot id")
	ErrSnapshotNotFound = fmt.Errorf("snapshot not found")
)

type SnapshotState struct {
	ID        string
	Offset    *stream.Offset
	PullTopic stream.Topic
	PushTopic stream.Topic
	Completed bool
}

func NewSnapshotStore() *SnapshotStore {
	return &SnapshotStore{
		snapshots: make(map[string]*SnapshotState),
	}
}

type SnapshotStore struct {
	snapshots map[string]*SnapshotState
}

func copySnapshot(x *SnapshotState) *SnapshotState {
	return &SnapshotState{
		ID:        x.ID,
		Offset:    x.Offset,
		PullTopic: x.PullTopic,
		PushTopic: x.PushTopic,
		Completed: x.Completed,
	}
}

func (c *SnapshotStore) SaveSnapshot(x SnapshotState) error {
	if x.ID == "" {
		return fmt.Errorf("projection.SnapshotStore: SaveSnapshot: %w", ErrSnapshotIDEmpty)
	}

	c.snapshots[x.ID] = copySnapshot(&x)
	return nil
}

func (c *SnapshotStore) LoadLastSnapshot(id string) (*SnapshotState, error) {
	if id == "" {
		return nil, fmt.Errorf("projection.SnapshotStore: LoadLastSnapshot: %w", ErrSnapshotIDEmpty)
	}

	snapshot, ok := c.snapshots[id]
	if !ok {
		return nil, fmt.Errorf("projection.SnapshotStore: LoadLastSnapshot: %w", ErrSnapshotNotFound)
	}

	return copySnapshot(snapshot), nil
}

func (c *SnapshotStore) InitSnapshot(id, pullTopic, pushTopic string) *SnapshotState {
	return &SnapshotState{
		ID:        id,
		Offset:    nil,
		PullTopic: pullTopic,
		PushTopic: pushTopic,
		Completed: false,
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
	a stream.Stream[A],
	aTopic stream.Topic,
	b stream.Stream[B],
	bTopic stream.Topic,
	out stream.Stream[C],
	outTopic stream.Topic,
) PushAndPull[Either[A, B], C] {
	return &InMemoryJoinContext[A, B, C]{
		a:      a,
		b:      b,
		output: out,
		mod:    true,

		aTopic:   aTopic,
		bTopic:   bTopic,
		outTopic: outTopic,
	}
}

type InMemoryJoinContext[A, B, C any] struct {
	a stream.Stream[A]
	b stream.Stream[B]

	output stream.Stream[C]

	aTopic   stream.Topic
	bTopic   stream.Topic
	outTopic stream.Topic

	mod  bool
	endA bool
	endB bool

	offsetA *stream.Offset
	offsetB *stream.Offset
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
		if i.offsetA == nil {
			pull = &stream.FromBeginning{
				Topic: i.aTopic,
			}
		} else {
			pull = &stream.FromOffset{
				Topic:  i.aTopic,
				Offset: i.offsetA,
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

		i.offsetA = val.Offset

		return StreamItemToRecordSetData[A, Either[A, B]](val, &Left[A, B]{Left: val.Data}), nil
	} else if !i.endB && i.mod == false {
		i.mod = !i.mod

		var pull stream.PullCMD
		if i.offsetB == nil {
			pull = &stream.FromBeginning{
				Topic: i.bTopic,
			}
		} else {
			pull = &stream.FromOffset{
				Topic:  i.bTopic,
				Offset: i.offsetB,
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

		i.offsetB = val.Offset

		return StreamItemToRecordSetData[B, Either[A, B]](val, &Right[A, B]{Right: val.Data}), nil
	}

	i.mod = !i.mod
	return i.PullIn()
}

func (i *InMemoryJoinContext[A, B, C]) PushOut(x Data[C]) error {
	item := RecordToStreamItem(i.outTopic, x)

	err := i.output.Push(item)
	if err != nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: PushOut: %w", err)
	}
	return nil
}
