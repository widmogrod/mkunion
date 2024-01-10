package stream

import "fmt"

func NewInMemoryStream[A any]() *InMemoryStream[A] {
	return &InMemoryStream[A]{}
}

type InMemoryStream[A any] struct {
	values []*Item[A]
}

var _ Stream[int] = (*InMemoryStream[int])(nil)

func (i *InMemoryStream[A]) Push(x *Item[A]) error {
	if x.Offset.IsSet() {
		return ErrOffsetSetOnPush
	}

	i.values = append(i.values, &Item[A]{
		Key:    x.Key,
		Data:   x.Data,
		Offset: MkOffsetFromInt(len(i.values)),
	})
	return nil
}

func (i *InMemoryStream[A]) Pull(fromOffset PullCMD) (*Item[A], error) {
	if len(i.values) == 0 {
		return nil, ErrEndOfStream
	}

	return MatchPullCMDR2(
		fromOffset,
		func(x *FromBeginning) (*Item[A], error) {
			return i.values[0], nil
		},
		func(x *FromOffset) (*Item[A], error) {
			offset, err := ParseOffsetAsInt(x)
			if err != nil {
				return nil, fmt.Errorf("stream.InMemoryStream: Pull: %w", err)
			}

			if offset+1 >= len(i.values) {
				return nil, ErrEndOfStream
			}

			return i.values[offset+1], nil
		},
	)
}
