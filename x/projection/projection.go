package projection

import (
	"errors"
	"fmt"
)

var (
	ErrEndOfStream   = fmt.Errorf("end of stream")
	ErrInvalidOffset = fmt.Errorf("invalid offset")
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

func (i *InMemoryStream[A]) Pull(offset int) (Item[A], error) {
	if offset == len(i.values) {
		var res Item[A]
		return res, ErrEndOfStream
	} else if offset > len(i.values) {
		var res Item[A]
		return res, ErrInvalidOffset
	}

	return i.values[offset], nil
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
	}
}

func NewPushOnlyInMemoryContext[A any](out Stream[A]) PushOnly[A] {
	return &PushAndPullInMemoryContext[any, A]{
		output: out,
	}
}

func NewPullOnlyInMemoryContext[A any](in Stream[A]) PullOnly[A] {
	return &PushAndPullInMemoryContext[A, any]{
		input: in,
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

	c.offset++
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
	for i := 0; i < numbers; i++ {
		err := ctx.PushOut(i)
		if err != nil {
			return fmt.Errorf("projection.Range: push: %w", err)
		}
	}

	return nil
}

func Drain[A any](ctx PullOnly[A], logf func(string, ...any)) error {
	for {
		val, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, ErrEndOfStream) {
				return nil
			}
			return fmt.Errorf("projection.Drain: pull: %w", err)
		}
		logf("drain: %v, %v", val, err)
	}
}

func Do[A, B any](ctx PushAndPull[A, B], f func(A) B) error {
	for {
		val, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, ErrEndOfStream) {
				return nil
			}
			return fmt.Errorf("projection.Do: pull: %w", err)
		}

		err = ctx.PushOut(f(val))
		if err != nil {
			return fmt.Errorf("projection.Do: push: %w", err)
		}
	}
}
