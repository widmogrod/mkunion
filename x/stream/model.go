package stream

import "fmt"

//go:generate go run ../../cmd/mkunion

var (
	ErrEndOfStream     = fmt.Errorf("end of stream")
	ErrOffsetSetOnPush = fmt.Errorf("offset set on push")
	ErrEmptyCommand    = fmt.Errorf("empty command")
	ErrNoTopicWithName = fmt.Errorf("no topic with name")
	ErrEmptyTopic      = fmt.Errorf("no topic specified")
	ErrEmptyKey        = fmt.Errorf("empty key")
	ErrSimulatedError  = fmt.Errorf("simulated error")
)

//go:tag mkunion:"PullCMD"
type (
	FromBeginning struct {
		Topic Topic
	}
	FromOffset struct {
		Topic  Topic
		Offset *Offset
	}
)

type Topic = string

//go:tag serde:"json"
type Offset string

func (o *Offset) IsSet() bool {
	return o != nil && *o != ""
}

type Item[A any] struct {
	Topic     string
	Key       string
	Data      A
	EventTime *EventTime
	Offset    *Offset
}

type EventTime = int64

type Stream[A any] interface {
	Push(x *Item[A]) error
	Pull(offset PullCMD) (*Item[A], error)
}

var (
	ErrParsingOffsetEmptyOffset = fmt.Errorf("offset parsing empty value of offset")
	ErrParsingOffsetParser      = fmt.Errorf("offset parser error")
)

func MkOffsetFromInt(x int) *Offset {
	result := Offset(fmt.Sprintf("%d", x))
	return &result
}

func ParseOffsetAsInt(x *Offset) (int, error) {
	if x == nil {
		return 0, ErrParsingOffsetEmptyOffset
	}

	var result int

	_, err := fmt.Sscanf(string(*x), "%d", &result)
	if err != nil {
		return 0, fmt.Errorf("stream.ParseOffsetAsInt: %w; %w", err, ErrParsingOffsetParser)
	}

	return result, nil
}

func MkEventTimeFromInt(x int64) *EventTime {
	return &x
}
