package stream

import (
	"fmt"
	"math/rand"
	"time"
)

func WithSystemTime() EventTime {
	return time.Now().UnixNano()
}

func WithSystemTimeFixed(x EventTime) func() EventTime {
	return func() EventTime {
		return x
	}
}

func NewInMemoryStream[A any](systemTime func() EventTime) *InMemoryStream[A] {
	return &InMemoryStream[A]{
		systemTime: systemTime,
		values:     make(map[Topic][]*Item[A]),
	}
}

type InMemoryStream[A any] struct {
	values     map[Topic][]*Item[A]
	systemTime func() EventTime
	simulate   *SimulateProblem
}

var _ Stream[int] = (*InMemoryStream[int])(nil)

func (i *InMemoryStream[A]) Push(x *Item[A]) error {
	if i.simulate != nil && i.simulate.ErrorOnPush != nil {
		if rand.Float64() < i.simulate.ErrorOnPushProbability {
			return i.simulate.ErrorOnPush
		}
	}

	if x.Topic == "" {
		return ErrEmptyTopic
	}
	if x.Key == "" {
		return ErrEmptyKey
	}
	if x.Offset.IsSet() {
		return ErrOffsetSetOnPush
	}

	if i.values[x.Topic] == nil {
		i.values[x.Topic] = make([]*Item[A], 0)
	}

	i.values[x.Topic] = append(i.values[x.Topic], &Item[A]{
		Topic:     x.Topic,
		Key:       x.Key,
		Data:      x.Data,
		EventTime: i.ensureEventTime(x.EventTime),
		Offset:    MkOffsetFromInt(len(i.values[x.Topic])),
	})
	return nil
}

func (i *InMemoryStream[A]) Pull(fromOffset PullCMD) (*Item[A], error) {
	if i.simulate != nil && i.simulate.ErrorOnPull != nil {
		if rand.Float64() < i.simulate.ErrorOnPullProbability {
			return nil, i.simulate.ErrorOnPull
		}
	}

	if fromOffset == nil {
		return nil, ErrEmptyCommand
	}

	return MatchPullCMDR2(
		fromOffset,
		func(x *FromBeginning) (*Item[A], error) {
			if x.Topic == "" {
				return nil, ErrEmptyTopic
			}

			if i.values[x.Topic] == nil {
				return nil, ErrEndOfStream
			}

			if len(i.values[x.Topic]) == 0 {
				return nil, ErrEndOfStream
			}

			return i.values[x.Topic][0], nil
		},
		func(x *FromOffset) (*Item[A], error) {
			if x.Topic == "" {
				return nil, ErrEmptyTopic
			}

			offset, err := ParseOffsetAsInt(x.Offset)
			if err != nil {
				return nil, fmt.Errorf("stream.InMemoryStream: Pull %+#v: %w", x, err)
			}

			if i.values[x.Topic] == nil {
				return nil, ErrEndOfStream
			}

			if len(i.values[x.Topic]) == 0 {
				return nil, ErrEndOfStream
			}

			if offset+1 >= len(i.values[x.Topic]) {
				return nil, ErrEndOfStream
			}

			return i.values[x.Topic][offset+1], nil
		},
	)
}

func (i *InMemoryStream[A]) ensureEventTime(eventTime *EventTime) *EventTime {
	if eventTime != nil {
		return eventTime
	}

	result := i.systemTime()
	return &result
}

type SimulateProblem struct {
	ErrorOnPullProbability float64
	ErrorOnPull            error

	ErrorOnPushProbability float64
	ErrorOnPush            error
}

func (i *InMemoryStream[A]) SimulateRuntimeProblem(x *SimulateProblem) {
	i.simulate = x
}
