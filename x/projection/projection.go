package projection

import (
	"errors"
	"fmt"
)

var (
	ErrEndOfStream = fmt.Errorf("end of stream")
)

type Item[A any] struct {
	Value  A
	Offset int
}

type Stream[A any] interface {
	Push(x A) error
	Pull(offset int) (Item[A], error)
}

func NewInMemoryStream[A any]() *InMemoryStream[A] {
	return &InMemoryStream[A]{}
}

type InMemoryStream[A any] struct {
	values []Item[A]
}

var _ Stream[int] = (*InMemoryStream[int])(nil)

func (i *InMemoryStream[A]) Push(x A) error {
	i.values = append(i.values, Item[A]{
		Value:  x,
		Offset: len(i.values),
	})
	return nil
}

func (i *InMemoryStream[A]) Pull(fromOffset int) (Item[A], error) {
	if fromOffset+1 >= len(i.values) {
		var res Item[A]
		return res, ErrEndOfStream
	}

	// Pull from the offset excluding the offset itself
	return i.values[fromOffset+1], nil
}

type (
	PushAndPull[A, B any] interface {
		PullOnly[A]
		PushOnly[B]
	}

	PullOnly[A any] interface {
		PullIn() (A, error)
	}
	PushOnly[A any] interface {
		PushOut(A) error
	}
)

func NewPushAndPullInMemoryContext[A, B any](in Stream[A], out Stream[B]) *PushAndPullInMemoryContext[A, B] {
	return &PushAndPullInMemoryContext[A, B]{
		input:  in,
		output: out,
		offset: -1,
	}
}

func NewPushOnlyInMemoryContext[A any](out Stream[A]) PushOnly[A] {
	return &PushAndPullInMemoryContext[any, A]{
		output: out,
		offset: -1,
	}
}

func NewPullOnlyInMemoryContext[A any](in Stream[A]) PullOnly[A] {
	return &PushAndPullInMemoryContext[A, any]{
		input:  in,
		offset: -1,
	}
}

var _ PushAndPull[int, int] = (*PushAndPullInMemoryContext[int, int])(nil)

type PushAndPullInMemoryContext[A, B any] struct {
	offset int
	input  Stream[A]
	output Stream[B]
}

func (c *PushAndPullInMemoryContext[A, B]) PullIn() (A, error) {
	item, err := c.input.Pull(c.offset)
	if err != nil {
		return item.Value, fmt.Errorf("projection.PushAndPullInMemoryContext: PullIn: %w", err)
	}

	c.offset = item.Offset
	return item.Value, nil
}

func (c *PushAndPullInMemoryContext[A, B]) PushOut(x B) error {
	err := c.output.Push(x)
	if err != nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: PushOut: %w", err)
	}
	return nil
}

func Range(ctx PushOnly[int], numbers int) error {
	return DoLoad(ctx, func(push func(int) error) error {
		for i := 0; i < numbers; i++ {
			err := push(i)
			if err != nil {
				return fmt.Errorf("projection.Range: push: %w", err)
			}
		}
		return nil
	})
}

func DoMap[A, B any](ctx PushAndPull[A, B], f func(A) B) error {
	for {
		val, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, ErrEndOfStream) {
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

func DoLoad[A any](ctx PushOnly[A], f func(push func(A) error) error) error {
	err := f(ctx.PushOut)
	if err != nil {
		return fmt.Errorf("projection.DoLoad: load: %w", err)
	}
	return nil
}

func DoSink[A any](ctx PullOnly[A], f func(A) error) error {
	for {
		val, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, ErrEndOfStream) {
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

type Either[A, B any] struct {
	Left  *A
	Right *B
}

func NewJoinInMemoryContext[A, B any](a Stream[A], b Stream[B]) PullOnly[*Either[A, B]] {
	return &InMemoryJoinContext[A, B]{
		a: a,
		b: b,

		mod: true,

		offsetA: -1,
		offsetB: -1,
	}
}

type InMemoryJoinContext[A, B any] struct {
	a Stream[A]
	b Stream[B]

	mod  bool
	endA bool
	endB bool

	offsetA int
	offsetB int
}

var _ PullOnly[*Either[any, any]] = (*InMemoryJoinContext[any, any])(nil)

func (i *InMemoryJoinContext[A, B]) PullIn() (*Either[A, B], error) {
	if i.endA && i.endB {
		return nil, ErrEndOfStream
	}

	if !i.endA && i.mod == true {
		i.mod = !i.mod
		val, err := i.a.Pull(i.offsetA)
		if err != nil {
			if errors.Is(err, ErrEndOfStream) {
				i.endA = true
				return i.PullIn()
			}
			return nil, fmt.Errorf("projection.InMemoryJoinContext: PullIn left: %w", err)
		}

		i.offsetA = val.Offset

		return &Either[A, B]{
			Left: &val.Value,
		}, nil
	} else if !i.endB && i.mod == false {
		i.mod = !i.mod
		val, err := i.b.Pull(i.offsetB)
		if err != nil {
			if errors.Is(err, ErrEndOfStream) {
				i.endB = true
				return i.PullIn()
			}
			return nil, fmt.Errorf("projection.InMemoryJoinContext: PullIn right: %w", err)
		}

		i.offsetB = val.Offset

		return &Either[A, B]{
			Right: &val.Value,
		}, nil
	}

	i.mod = !i.mod
	return i.PullIn()
}

func DoJoin[A, B any](a Stream[A], b Stream[B]) PullOnly[*Either[A, B]] {
	return NewJoinInMemoryContext(a, b)
}
