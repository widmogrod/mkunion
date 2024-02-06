package projection

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/stream"
	"math"
	"math/rand"
	"time"
)

//go:generate go run ../../cmd/mkunion/main.go -v -type-registry

//go:tag mkunion:"Data,noserde"
type (
	Record[A any] struct {
		Key       string
		Data      A
		EventTime EventTime
	}

	Watermark[A any] struct {
		Key       string
		EventTime EventTime
	}
)

type EventTime = int64

func IsWatermarkMarksEndOfStream[A any](x *Watermark[A]) bool {
	if x.EventTime == math.MaxInt64 {
		return true
	}

	return false
}

func EventTimeToStreamEventTime(x EventTime) *stream.EventTime {
	if x == 0 {
		return nil
	}

	return &x
}

func RecordToStreamItem[A any](topic string, x Data[A]) *stream.Item[Data[A]] {
	return MatchDataR1[A, *stream.Item[Data[A]]](x,
		func(x *Record[A]) *stream.Item[Data[A]] {
			return &stream.Item[Data[A]]{
				Topic:     topic,
				Key:       x.Key,
				Data:      x,
				EventTime: EventTimeToStreamEventTime(x.EventTime),
				Offset:    nil,
			}
		},
		func(x *Watermark[A]) *stream.Item[Data[A]] {
			return &stream.Item[Data[A]]{
				Topic:     topic,
				Key:       x.Key,
				Data:      x,
				EventTime: EventTimeToStreamEventTime(x.EventTime),
				Offset:    nil,
			}
		},
	)
}

func StreamItemToRecord[A any](x *stream.Item[Data[A]]) Data[A] {
	return x.Data
}

type (
	PushAndPull[A, B any] interface {
		PullOnly[A]
		PushOnly[B]
		SnapshotContext
	}
	PullOnly[A any] interface {
		PullIn() (Data[A], error)
		SnapshotContext
	}
	PushOnly[A any] interface {
		PushOut(Data[A]) error
		SnapshotContext
	}
	SnapshotContext interface {
		CurrentState() SnapshotState
	}
)

func NewPushAndPullInMemoryContext[A, B any](state *PullPushContextState, in stream.Stream[Data[A]], out stream.Stream[Data[B]]) *PushAndPullInMemoryContext[A, B] {
	return &PushAndPullInMemoryContext[A, B]{
		state:  state,
		input:  in,
		output: out,
	}
}

func NewPushOnlyInMemoryContext[A any](state *PullPushContextState, out stream.Stream[Data[A]]) PushOnly[A] {
	return &PushAndPullInMemoryContext[any, A]{
		state:  state,
		output: out,
	}
}

func NewPullOnlyInMemoryContext[A any](state *PullPushContextState, in stream.Stream[Data[A]]) PullOnly[A] {
	return &PushAndPullInMemoryContext[A, any]{
		state: state,
		input: in,
	}
}

var _ PushAndPull[int, int] = (*PushAndPullInMemoryContext[int, int])(nil)

type PushAndPullInMemoryContext[A, B any] struct {
	state *PullPushContextState

	nextOffset *stream.Offset

	input    stream.Stream[Data[A]]
	output   stream.Stream[Data[B]]
	simulate *SimulateProblem
}

func (c *PushAndPullInMemoryContext[A, B]) PullIn() (Data[A], error) {
	if c.simulate != nil && c.simulate.ErrorOnPullIn != nil {
		if rand.Float64() < c.simulate.ErrorOnPullInProbability {
			return nil, c.simulate.ErrorOnPullIn
		}
	}

	pull := c.pullCommand(c.state, c.nextOffset)
	item, err := c.input.Pull(pull)
	if err != nil {
		return nil, fmt.Errorf("projection.PushAndPullInMemoryContext: PullIn: %w", err)
	}

	if c.nextOffset != nil {
		// save to state only previous offset
		c.state.Offset = c.nextOffset
	}

	c.nextOffset = item.Offset

	result := StreamItemToRecord(item)
	return result, nil
}

func (c *PushAndPullInMemoryContext[A, B]) PushOut(x Data[B]) error {
	if c.simulate != nil && c.simulate.ErrorOnPushOut != nil {
		if rand.Float64() < c.simulate.ErrorOnPushOutProbability {
			return c.simulate.ErrorOnPushOut
		}
	}

	item := RecordToStreamItem(c.state.PushTopic, x)

	err := c.output.Push(item)
	if err != nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: PushOut: %w", err)
	}
	return nil
}

func (c *PushAndPullInMemoryContext[A, B]) CurrentState() SnapshotState {
	return c.state
}

func (c *PushAndPullInMemoryContext[A, B]) pullCommand(x *PullPushContextState, next *stream.Offset) stream.PullCMD {
	if x.Offset == nil && next == nil {
		return &stream.FromBeginning{
			Topic: x.PullTopic,
		}
	}

	if next != nil {
		return &stream.FromOffset{
			Topic:  x.PullTopic,
			Offset: next,
		}
	}

	return &stream.FromOffset{
		Topic:  x.PullTopic,
		Offset: x.Offset,
	}
}

type SimulateProblem struct {
	ErrorOnPullInProbability float64
	ErrorOnPullIn            error

	ErrorOnPushOutProbability float64
	ErrorOnPushOut            error
}

func (c *PushAndPullInMemoryContext[A, B]) SimulateRuntimeProblem(x *SimulateProblem) {
	c.simulate = x
}

func DoLoad[A any](ctx PushOnly[A], f func(push func(record Data[A]) error) error) error {
	err := f(func(record Data[A]) error {
		return ctx.PushOut(record)
	})
	if err != nil {
		return fmt.Errorf("projection.DoLoad: load: %w", err)
	}
	return nil
}

func DoMap[A, B any](ctx PushAndPull[A, B], f func(*Record[A]) *Record[B]) error {
	timer := time.NewTimer(1 * time.Second)
	defer timer.Stop()

	keyToKeysJoin := make(map[string]map[string]struct{})
	joinedWatermark := make(map[string]int64)

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("projection.DoMap: timeout; %w", stream.ErrNoMoreNewDataInStream)
		default:
			val, err := ctx.PullIn()
			if err != nil {
				if errors.Is(err, stream.ErrNoMoreNewDataInStream) {
					return nil
				}
				return fmt.Errorf("projection.DoMap: pull: %w", err)
			}

			err = MatchDataR1(
				val,
				func(x *Record[A]) error {
					y := f(x)

					keys, ok := keyToKeysJoin[y.Key]
					if !ok {
						keys = make(map[string]struct{})
						keyToKeysJoin[y.Key] = keys
					}

					err = ctx.PushOut(y)
					if err != nil {
						return fmt.Errorf("projection.DoMap: push: %w", err)
					}

					keys[y.Key] = struct{}{}

					return nil
				},
				func(x *Watermark[A]) error {
					for key := range keyToKeysJoin[x.Key] {
						if _, ok := joinedWatermark[key]; !ok {
							joinedWatermark[key] = 0
						}

						if joinedWatermark[key] >= x.EventTime {
							return nil
						}

						joinedWatermark[key] = x.EventTime

						err := ctx.PushOut(&Watermark[B]{
							Key:       key,
							EventTime: x.EventTime,
						})
						if err != nil {
							return fmt.Errorf("projection.DoMap: push wattermark for key %s: %w", key, err)
						}
					}

					return nil
				},
			)

			if err != nil {
				return fmt.Errorf("projection.DoMap: map: %w", err)
			}
		}
	}
}

func DoJoin[A, B, C any](
	ctxA PushAndPull[A, C],
	ctxB PushAndPull[B, C],
	f func(x Either[*Record[A], *Record[B]]) *Record[C],
) error {
	delay := 1 * time.Second
	timer := time.NewTimer(delay)
	defer timer.Stop()

	leftKeyToKeysJoin := make(map[string]map[string]struct{})
	rightKeyToKeysJoin := make(map[string]map[string]struct{})
	leftKeyWatermark := make(map[string]int64)
	rightKeyWatermark := make(map[string]int64)
	leftOrRight := true

	flushWatermarks := func(
		eventKey string,
		eventTime EventTime,
		keysToJoin map[string]map[string]struct{},
		leftKeyWatermark map[string]int64,
		rightKeyWatermark map[string]int64,
	) error {
		var watermarksToPush map[string]*Watermark[C]
		var clean []func()
		for key := range keysToJoin[eventKey] {
			// no watermark for keys recorded, then init
			if _, ok := leftKeyWatermark[key]; !ok {
				leftKeyWatermark[key] = 0
			}

			// has bigger watermark, then skip
			if leftKeyWatermark[key] < eventTime {
				leftKeyWatermark[key] = eventTime
			}

			// other side has no watermark for key, then skip
			if _, ok := rightKeyWatermark[key]; !ok {
				continue
			}

			// all conditions for publishing watermark are met
			if watermarksToPush == nil {
				watermarksToPush = make(map[string]*Watermark[C])
			}

			if _, ok := watermarksToPush[key]; ok {
				continue
			}

			watermarksToPush[key] = &Watermark[C]{
				Key:       key,
				EventTime: min(leftKeyWatermark[key], rightKeyWatermark[key]),
			}

			// publish smallest watermark
			if leftKeyWatermark[key] <= rightKeyWatermark[key] {
				// remove smallest watermark
				clean = append(clean, func() {
					delete(leftKeyWatermark, key)
				})
			} else if leftKeyWatermark[key] > rightKeyWatermark[key] {
				// remove smallest watermark
				clean = append(clean, func() {
					delete(rightKeyWatermark, key)
				})
			}
		}

		// publish watermarks
		for _, watermark := range watermarksToPush {
			// output stream in ctxA and ctxB is the same
			// so we push only once
			err := ctxA.PushOut(watermark)
			if err != nil {
				return fmt.Errorf("push wattermark for key %s: %w", watermark.Key, err)
			}
		}

		// clean up lowest watermarks
		for _, c := range clean {
			c()
		}

		return nil
	}

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("projection.DoJoin: timeout; %w", stream.ErrNoMoreNewDataInStream)
		default:
			if leftOrRight {
				val, err := ctxA.PullIn()
				if err != nil {
					if errors.Is(err, stream.ErrNoMoreNewDataInStream) {
						return nil
					}
					return fmt.Errorf("projection.DoJoin: pull left: %w", err)
				}

				err = MatchDataR1(
					val,
					func(x *Record[A]) error {
						y := f(&Left[*Record[A], *Record[B]]{
							Left: x,
						})

						keys, ok := leftKeyToKeysJoin[y.Key]
						if !ok {
							keys = make(map[string]struct{})
							leftKeyToKeysJoin[y.Key] = keys
						}

						err = ctxA.PushOut(y)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push left: %w", err)
						}

						keys[y.Key] = struct{}{}

						return nil
					},
					func(x *Watermark[A]) error {
						err := flushWatermarks(x.Key, x.EventTime, leftKeyToKeysJoin,
							leftKeyWatermark, rightKeyWatermark)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: flush watermarks left side: %w", err)
						}

						return nil
					},
				)

				if err != nil {
					return fmt.Errorf("projection.DoJoin: left; %w", err)
				}
			} else {
				val, err := ctxB.PullIn()
				if err != nil {
					if errors.Is(err, stream.ErrNoMoreNewDataInStream) {
						return nil
					}
					return fmt.Errorf("projection.DoJoin: pull right: %w", err)
				}

				err = MatchDataR1(
					val,
					func(x *Record[B]) error {
						y := f(&Right[*Record[A], *Record[B]]{
							Right: x,
						})

						keys, ok := rightKeyToKeysJoin[y.Key]
						if !ok {
							keys = make(map[string]struct{})
							rightKeyToKeysJoin[y.Key] = keys
						}

						err = ctxA.PushOut(y)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push right: %w", err)
						}

						keys[y.Key] = struct{}{}

						return nil
					},
					func(x *Watermark[B]) error {
						err := flushWatermarks(x.Key, x.EventTime, rightKeyToKeysJoin,
							rightKeyWatermark, leftKeyWatermark)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: flush watermarks right side: %w", err)
						}

						return nil
					},
				)

				if err != nil {
					return fmt.Errorf("projection.DoJoin: right; %w", err)
				}
			}
		}

		leftOrRight = !leftOrRight
		timer.Reset(delay)
	}
}

func DoSink[A any](ctx PullOnly[A], f func(*Record[A]) error) error {
	for {
		val, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, stream.ErrNoMoreNewDataInStream) {
				return nil
			}
			return fmt.Errorf("projection.DoSink: pull: %w", err)
		}

		err = MatchDataR1(
			val,
			func(x *Record[A]) error {
				return f(x)
			},
			func(x *Watermark[A]) error {
				// TODO do snapshot
				return nil
			},
		)
		if err != nil {
			return fmt.Errorf("projection.DoSink: sink: %w", err)
		}
	}
}

//go:tag mkunion:"Either,noserde"
type (
	Left[A, B any] struct {
		Left A
	}
	Right[A, B any] struct {
		Right B
	}
)
