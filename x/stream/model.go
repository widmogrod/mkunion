package stream

import (
	"fmt"
	"time"
)

var (
	ErrNoMoreNewDataInStream = fmt.Errorf("no more new data in stream")
	ErrOffsetSetOnPush       = fmt.Errorf("offset set on push")
	ErrEmptyCommand          = fmt.Errorf("empty command")
	ErrNoTopicWithName       = fmt.Errorf("no topic with name")
	ErrEmptyTopic            = fmt.Errorf("no topic specified")
	ErrEmptyKey              = fmt.Errorf("empty key")
	ErrSimulatedError        = fmt.Errorf("simulated error")
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
	ErrOffsetNotComparable      = fmt.Errorf("offset not comparable")
)

func MkEventTimeFromInt(x int64) *EventTime {
	return &x
}

func WithSystemTime() EventTime {
	return time.Now().UnixNano()
}

func WithSystemTimeFixed(x EventTime) func() EventTime {
	return func() EventTime {
		return x
	}
}
