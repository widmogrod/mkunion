package stream

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
)

var (
	ErrTypedTopicMismatch = errors.New("typed topic don't match")
)

func NewTypedStreamTopic[A any](
	stream *InMemoryStream[schema.Schema],
	topic Topic,

) *TypedStreamTopic[A] {
	return &TypedStreamTopic[A]{
		stream: stream,
		topic:  topic,
	}
}

var _ Stream[any] = (*TypedStreamTopic[any])(nil)

type TypedStreamTopic[A any] struct {
	stream *InMemoryStream[schema.Schema]
	topic  Topic
}

func (t *TypedStreamTopic[A]) Push(x *Item[A]) error {
	if x.Topic != t.topic {
		return fmt.Errorf(
			"stream.TypedStreamTopic: Push: %w; invalid topic %s, expects %s",
			ErrTypedTopicMismatch, x.Topic, t.topic)
	}

	err := t.stream.Push(&Item[schema.Schema]{
		Topic:     x.Topic,
		Key:       x.Key,
		Data:      schema.FromGo(x.Data),
		EventTime: x.EventTime,
	})

	if err != nil {
		return fmt.Errorf("stream.TypedStreamTopic: Push: %w", err)
	}

	return nil
}

func (t *TypedStreamTopic[A]) Pull(offset PullCMD) (*Item[A], error) {
	err := MatchPullCMDR1(
		offset,
		func(x *FromBeginning) error {
			if x.Topic != t.topic {
				return fmt.Errorf(
					"stream.TypedStreamTopic: Pull(FromBeginning): %w; invalid topic %s, expects %s",
					ErrTypedTopicMismatch, x.Topic, t.topic)
			}

			return nil
		},
		func(x *FromOffset) error {
			if x.Topic != t.topic {
				return fmt.Errorf(
					"stream.TypedStreamTopic: Pull(FromOffset): %w; invalid topic %s, expects %s",
					ErrTypedTopicMismatch, x.Topic, t.topic)
			}

			return nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("stream.TypedStreamTopic: Pull: %w", err)
	}

	item, err := t.stream.Pull(offset)
	if err != nil {
		return nil, fmt.Errorf("stream.TypedStreamTopic: Pull: %w", err)
	}

	return &Item[A]{
		Topic:     item.Topic,
		Key:       item.Key,
		Data:      schema.ToGo[A](item.Data),
		EventTime: item.EventTime,
		Offset:    item.Offset,
	}, nil
}
