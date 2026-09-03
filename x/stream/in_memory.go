package stream

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shared"
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

	data, err := detach[A](x.Data)
	if err != nil {
		return fmt.Errorf("stream.InMemoryStream.Push: %w", err)
	}

	if i.values[x.Topic] == nil {
		i.values[x.Topic] = make([]*Item[A], 0)
	}

	i.values[x.Topic] = append(i.values[x.Topic], &Item[A]{
		Topic:     x.Topic,
		Key:       x.Key,
		Data:      data,
		EventTime: i.ensureEventTime(x.EventTime),
		Offset:    mkInMemoryOffsetFromInt(len(i.values[x.Topic])),
	})
	return nil
}

// detach deep-copies a value through serialization, the way a broker
// captures a message: the stream must never share memory with callers.
func detach[A any](x A) (A, error) {
	bytes, err := shared.JSONMarshal[A](x)
	if err != nil {
		return x, fmt.Errorf("stream.detach: marshal; %w", err)
	}
	result, err := shared.JSONUnmarshal[A](bytes)
	if err != nil {
		return x, fmt.Errorf("stream.detach: unmarshal; %w", err)
	}
	return result, nil
}

// detachItem returns a copy of a stored item with detached data, so a
// consumer mutating what it pulled cannot rewrite the log.
func detachItem[A any](x *Item[A]) (*Item[A], error) {
	data, err := detach[A](x.Data)
	if err != nil {
		return nil, err
	}
	copied := *x
	copied.Data = data
	return &copied, nil
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

			return detachItem(i.values[x.Topic][0])
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

			return detachItem(i.values[x.Topic][offset+1])
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
