package stream

import (
	"fmt"
	"math/rand"
)

func init() {
	RegisterOffsetCompare("i", InMemoryOffsetCompare)
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

var _ Stream[any] = (*InMemoryStream[any])(nil)

func (i *InMemoryStream[A]) Push(x *Item[A]) error {
	if i.simulate != nil && i.simulate.ErrorOnPush != nil {
		if rand.Float64() < i.simulate.ErrorOnPushProbability {
			return fmt.Errorf("stream.InMemoryStream.Push: %w; %w", i.simulate.ErrorOnPush, ErrSimulatedError)
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
		Offset:    mkInMemoryOffsetFromInt(len(i.values[x.Topic])),
	})
	return nil
}

func (i *InMemoryStream[A]) Pull(fromOffset PullCMD) (*Item[A], error) {
	if i.simulate != nil && i.simulate.ErrorOnPull != nil {
		if rand.Float64() < i.simulate.ErrorOnPullProbability {
			return nil, fmt.Errorf("stream.InMemoryStream.Pull: %w; %w", i.simulate.ErrorOnPull, ErrSimulatedError)
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

			if _, ok := i.values[x.Topic]; !ok {
				return nil, ErrNoTopicWithName
			}

			if len(i.values[x.Topic]) == 0 {
				return nil, ErrNoMoreNewDataInStream
			}

			return i.values[x.Topic][0], nil
		},
		func(x *FromOffset) (*Item[A], error) {
			if x.Topic == "" {
				return nil, ErrEmptyTopic
			}

			offset, err := parseInMemoryOffsetAsInt(x.Offset)
			if err != nil {
				return nil, fmt.Errorf("stream.InMemoryStream: Pull %+#v: %w", x, err)
			}

			if _, ok := i.values[x.Topic]; !ok {
				return nil, ErrNoTopicWithName
			}

			if len(i.values[x.Topic]) == 0 {
				return nil, ErrNoMoreNewDataInStream
			}

			if offset+1 >= len(i.values[x.Topic]) {
				return nil, ErrNoMoreNewDataInStream
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

func InMemoryOffsetCompare(a Offset, b Offset) (int8, error) {
	var resultA, resultB int
	_, err := fmt.Sscanf(string(a), "i:%d", &resultA)
	if err != nil {
		return 0, fmt.Errorf("stream.InMemoryOffsetCompare: left side; %w; %w", err, ErrParsingOffsetParser)
	}

	_, err = fmt.Sscanf(string(b), "i:%d", &resultB)
	if err != nil {
		return 0, fmt.Errorf("stream.InMemoryOffsetCompare: right side; %w; %w", err, ErrParsingOffsetParser)
	}

	return int8(resultA - resultB), nil
}

func mkInMemoryOffsetFromInt(x int) *Offset {
	result := Offset(fmt.Sprintf("i:%d", x))
	return &result
}

func parseInMemoryOffsetAsInt(x *Offset) (int, error) {
	var result int
	_, err := fmt.Sscanf(string(*x), "i:%d", &result)
	if err != nil {
		return 0, fmt.Errorf("stream.parseInMemoryOffsetAsInt: %w; %w", err, ErrParsingOffsetParser)
	}

	return result, nil
}
