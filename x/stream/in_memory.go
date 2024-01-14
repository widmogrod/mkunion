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
	}
}

type InMemoryStream[A any] struct {
	values     []*Item[A]
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

	if x.Offset.IsSet() {
		return ErrOffsetSetOnPush
	}

	i.values = append(i.values, &Item[A]{
		Key:       x.Key,
		Data:      x.Data,
		EventTime: i.ensureEventTime(x.EventTime),
		Offset:    MkOffsetFromInt(len(i.values)),
	})
	return nil
}

func (i *InMemoryStream[A]) Pull(fromOffset PullCMD) (*Item[A], error) {
	if i.simulate != nil && i.simulate.ErrorOnPull != nil {
		if rand.Float64() < i.simulate.ErrorOnPullProbability {
			return nil, i.simulate.ErrorOnPull
		}
	}

	if len(i.values) == 0 {
		return nil, ErrEndOfStream
	}

	if fromOffset == nil {
		return nil, ErrEmptyCommand
	}

	return MatchPullCMDR2(
		fromOffset,
		func(x *FromBeginning) (*Item[A], error) {
			return i.values[0], nil
		},
		func(x *FromOffset) (*Item[A], error) {
			offset, err := ParseOffsetAsInt(x)
			if err != nil {
				return nil, fmt.Errorf("stream.InMemoryStream: Pull %+#v: %w", x, err)
			}

			if offset+1 >= len(i.values) {
				return nil, ErrEndOfStream
			}

			return i.values[offset+1], nil
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
