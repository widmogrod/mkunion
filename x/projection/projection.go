package projection

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/stream"
)

//go:generate go run ../../cmd/mkunion/main.go

//go:tag mkunion:"Data,noserde"
type (
	Record[A any] stream.Item[A]

	Watermark[A any] struct {
		EventTime EventTime
	}
)

type EventTime = int64

func RecordToStreamItem[A any](x Data[A]) *stream.Item[A] {
	return MatchDataR1[A, *stream.Item[A]](x,
		func(x *Record[A]) *stream.Item[A] {
			return &stream.Item[A]{
				Key:  x.Key,
				Data: x.Data,
			}
		},
		func(x *Watermark[A]) *stream.Item[A] {
			var zero A
			return &stream.Item[A]{
				Key:  "$watermark",
				Data: zero,
			}
		},
	)
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

func NewPushAndPullInMemoryContext[A, B any](in stream.Stream[A], out stream.Stream[B]) *PushAndPullInMemoryContext[A, B] {
	return &PushAndPullInMemoryContext[A, B]{
		state:  SnapshotState{},
		input:  in,
		output: out,
	}
}

func NewPushOnlyInMemoryContext[A any](out stream.Stream[A]) PushOnly[A] {
	return &PushAndPullInMemoryContext[any, A]{
		state:  SnapshotState{},
		output: out,
	}
}

func NewPullOnlyInMemoryContext[A any](in stream.Stream[A]) PullOnly[A] {
	return &PushAndPullInMemoryContext[A, any]{
		state: SnapshotState{},
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
		return nil, c.simulate.ErrorOnPullIn
	}

	pull := c.pullCommand(c.state)
	item, err := c.input.Pull(pull)
	if err != nil {
		return nil, fmt.Errorf("projection.PushAndPullInMemoryContext: PullIn: %w", err)
	}

	c.state.Offset = item.Offset

	result := &Record[A]{
		Key:       item.Key,
		Data:      item.Data,
		EventTime: item.EventTime,
		Offset:    item.Offset,
	}

	return result, nil
}

func (c *PushAndPullInMemoryContext[A, B]) PushOut(x Data[B]) error {
	if c.simulate != nil && c.simulate.ErrorOnPushOut != nil {
		return c.simulate.ErrorOnPushOut
	}

	item := RecordToStreamItem(x)

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
		return &stream.FromBeginning{}
	}

	return x.Offset
}

type SimulateProblem struct {
	ErrorOnPullIn  error
	ErrorOnPushOut error
}

func (c *PushAndPullInMemoryContext[A, B]) SimulateRuntimeProblem(x *SimulateProblem) {
	c.simulate = x
}

type SnapshotState struct {
	ID        string
	Offset    *stream.Offset
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
		Completed: x.Completed,
	}
}

func (c *SnapshotStore) SaveSnapshot(x SnapshotState) error {
	if x.ID == "" {
		return fmt.Errorf("projection.SnapshotStore: SaveSnapshot: empty id")
	}

	c.snapshots[x.ID] = copySnapshot(&x)
	return nil
}

func (c *SnapshotStore) LoadLastSnapshot(id string) (*SnapshotState, error) {
	if id == "" {
		return nil, fmt.Errorf("projection.SnapshotStore: LoadLastSnapshot: empty id")
	}

	snapshot, ok := c.snapshots[id]
	if !ok {
		return nil, fmt.Errorf("projection.SnapshotStore: LoadLastSnapshot: snapshot not found")
	}

	return copySnapshot(snapshot), nil
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

func DoMap[A, B any](ctx PushAndPull[A, B], f func(Data[A]) Data[B]) error {
	for {
		val, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, stream.ErrEndOfStream) {
				return nil
			}
			return fmt.Errorf("projection.DoMap: pull: %w", err)
		}

		err = ctx.PushOut(f(val))
		if err != nil {
			return fmt.Errorf("projection.DoMap: push: %w", err)
		}
	}
}

func DoSink[A any](ctx PullOnly[A], f func(Data[A]) error) error {
	for {
		val, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, stream.ErrEndOfStream) {
				return nil
			}
			return fmt.Errorf("projection.DoSink: pull: %w", err)
		}

		err = f(val)
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
	b stream.Stream[B],
	out stream.Stream[C],
) PushAndPull[Either[A, B], C] {
	return &InMemoryJoinContext[A, B, C]{
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
			pull = &stream.FromBeginning{}
		} else {
			pull = i.offsetA
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

		return &Record[Either[A, B]]{
			Key: val.Key,
			Data: &Left[A, B]{
				Left: val.Data,
			},
			EventTime: val.EventTime,
			Offset:    val.Offset,
		}, nil
	} else if !i.endB && i.mod == false {
		i.mod = !i.mod

		var pull stream.PullCMD
		if i.offsetB == nil {
			pull = &stream.FromBeginning{}
		} else {
			pull = i.offsetB
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

		return &Record[Either[A, B]]{
			Key: val.Key,
			Data: &Right[A, B]{
				Right: val.Data,
			},
			EventTime: val.EventTime,
			Offset:    val.Offset,
		}, nil
	}

	i.mod = !i.mod
	return i.PullIn()
}

func (i *InMemoryJoinContext[A, B, C]) PushOut(x Data[C]) error {
	item := RecordToStreamItem(x)

	err := i.output.Push(item)
	if err != nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: PushOut: %w", err)
	}
	return nil
}
