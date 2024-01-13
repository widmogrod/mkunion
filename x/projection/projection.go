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

type Window struct {
	Start int64
	End   int64
}

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
		input:  in,
		output: out,
	}
}

func NewPushOnlyInMemoryContext[A any](out stream.Stream[A]) PushOnly[A] {
	return &PushAndPullInMemoryContext[any, A]{
		output: out,
	}
}

func NewPullOnlyInMemoryContext[A any](in stream.Stream[A]) PullOnly[A] {
	return &PushAndPullInMemoryContext[A, any]{
		input: in,
	}
}

var _ PushAndPull[int, int] = (*PushAndPullInMemoryContext[int, int])(nil)

type PushAndPullInMemoryContext[A, B any] struct {
	offset *stream.Offset
	input  stream.Stream[A]
	output stream.Stream[B]
}

func (c *PushAndPullInMemoryContext[A, B]) PullIn() (Data[A], error) {
	var pull stream.PullCMD
	if c.offset == nil {
		pull = &stream.FromBeginning{}
	} else {
		pull = c.offset
	}

	item, err := c.input.Pull(pull)
	if err != nil {
		return nil, fmt.Errorf("projection.PushAndPullInMemoryContext: PullIn: %w", err)
	}

	c.offset = item.Offset

	result := &Record[A]{
		Key:  item.Key,
		Data: item.Data,
		//EventTime: 0,
		//Window:    nil,
	}

	return result, nil
}

func (c *PushAndPullInMemoryContext[A, B]) PushOut(x Data[B]) error {
	item := RecordToStreamItem(x)

	err := c.output.Push(item)
	if err != nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: PushOut: %w", err)
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

func DoLoad[A any](ctx PushOnly[A], f func(push func(Data[A]) error) error) error {
	err := f(ctx.PushOut)
	if err != nil {
		return fmt.Errorf("projection.DoLoad: load: %w", err)
	}
	return nil
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

func NewJoinInMemoryContext[A, B any](a stream.Stream[A], b stream.Stream[B]) *InMemoryJoinContext[A, B] {
	return &InMemoryJoinContext[A, B]{
		a: a,
		b: b,

		mod: true,
	}
}

type InMemoryJoinContext[A, B any] struct {
	a stream.Stream[A]
	b stream.Stream[B]

	mod  bool
	endA bool
	endB bool

	offsetA *stream.Offset
	offsetB *stream.Offset
}

var _ PullOnly[Either[any, any]] = (*InMemoryJoinContext[any, any])(nil)

func (i *InMemoryJoinContext[A, B]) PullIn() (Data[Either[A, B]], error) {
	if i.endA && i.endB {
		return nil, stream.ErrEndOfStream
	}

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
			//EventTime: 0,
			//Window:    nil,
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
			//EventTime: 0,
			//Window:    nil,
		}, nil
	}

	i.mod = !i.mod
	return i.PullIn()
}

func DoJoin[A, B any](a stream.Stream[A], b stream.Stream[B]) PullOnly[Either[A, B]] {
	return NewJoinInMemoryContext(a, b)
}
