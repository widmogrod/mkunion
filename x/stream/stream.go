package stream

import "fmt"

var (
	ErrEndOfStream     = fmt.Errorf("end of stream")
	ErrOffsetSetOnPush = fmt.Errorf("offset set on push")
)

type Item[A any] struct {
	Key    string
	Data   A
	Offset int
}

type Stream[A any] interface {
	Push(x *Item[A]) error
	Pull(offset int) (*Item[A], error)
}

func NewInMemoryStream[A any]() *InMemoryStream[A] {
	return &InMemoryStream[A]{}
}

type InMemoryStream[A any] struct {
	values []*Item[A]
}

var _ Stream[int] = (*InMemoryStream[int])(nil)

func (i *InMemoryStream[A]) Push(x *Item[A]) error {
	if x.Offset != 0 {
		return ErrOffsetSetOnPush
	}

	i.values = append(i.values, &Item[A]{
		Key:    x.Key,
		Data:   x.Data,
		Offset: len(i.values),
	})
	return nil
}

func (i *InMemoryStream[A]) Pull(fromOffset int) (*Item[A], error) {
	if fromOffset+1 >= len(i.values) {
		return nil, ErrEndOfStream
	}

	// Pull from the offset excluding the offset itself
	return i.values[fromOffset+1], nil
}
