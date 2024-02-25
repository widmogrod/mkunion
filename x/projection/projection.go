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

var (
	ErrStateAckNilOffset    = errors.New("cannot acknowledge nil offset")
	ErrStateAckNilWatermark = errors.New("cannot acknowledge nil watermark")
)

const (
	KeySystemWatermark = "watermark"
)

//go:tag mkunion:"Data,noserde"
type (
	Record[A any] struct {
		Key       string
		Data      A
		EventTime EventTime
	}

	Watermark[A any] struct {
		EventTime EventTime
	}
)

type EventTime = int64

func IsWatermarkMarksEndOfStream(x EventTime) bool {
	if x == math.MaxInt64 {
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

func RecordToStreamItem[A any](topic string, x Data[A]) (*stream.Item[Data[A]], error) {
	return MatchDataR2[A, *stream.Item[Data[A]]](
		x,
		func(x *Record[A]) (*stream.Item[Data[A]], error) {
			if x.Key == KeySystemWatermark {
				return nil, fmt.Errorf("projection.RecordToStreamItem: key %s is reserved", KeySystemWatermark)
			}

			return &stream.Item[Data[A]]{
				Topic:     topic,
				Key:       x.Key,
				Data:      x,
				EventTime: EventTimeToStreamEventTime(x.EventTime),
				Offset:    nil,
			}, nil
		},
		func(x *Watermark[A]) (*stream.Item[Data[A]], error) {
			return &stream.Item[Data[A]]{
				Topic:     topic,
				Key:       KeySystemWatermark,
				Data:      x,
				EventTime: EventTimeToStreamEventTime(x.EventTime),
				Offset:    nil,
			}, nil
		},
	)
}

type (
	PushAndPull[A, B any] interface {
		PullOnly[A]
		PushOnly[B]
		SnapshotContext
	}
	PullOnly[A any] interface {
		PullIn() (*stream.Item[Data[A]], error)
		AckOffset(offset *stream.Offset) error
		SnapshotContext
	}
	PushOnly[A any] interface {
		PushOut(Data[A]) error
		AckWatermark(watermark *stream.EventTime) error
		SnapshotContext
	}
	SnapshotContext interface {
		CurrentState() SnapshotState
		LastWatermark() EventTime
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

func (c *PushAndPullInMemoryContext[A, B]) PullIn() (*stream.Item[Data[A]], error) {
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

	if item.Key == KeySystemWatermark {
		item.Data = &Watermark[A]{
			EventTime: *item.EventTime,
		}
	}

	if c.nextOffset != nil {
		// save to state only previous offset
		c.state.Offset = c.nextOffset
	}

	c.nextOffset = item.Offset

	return item, nil
}

func (c *PushAndPullInMemoryContext[A, B]) AckOffset(offset *stream.Offset) error {
	if offset == nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: AckOffset:  %w", ErrStateAckNilOffset)
	}

	c.state.Offset = offset
	return nil
}

func (c *PushAndPullInMemoryContext[A, B]) PushOut(x Data[B]) error {
	if c.simulate != nil && c.simulate.ErrorOnPushOut != nil {
		if rand.Float64() < c.simulate.ErrorOnPushOutProbability {
			return c.simulate.ErrorOnPushOut
		}
	}

	item, err := RecordToStreamItem(c.state.PushTopic, x)
	if err != nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: PushOut: %w", err)
	}

	err = c.output.Push(item)
	if err != nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: PushOut: %w", err)
	}
	return nil
}

func (c *PushAndPullInMemoryContext[A, B]) AckWatermark(watermark *stream.EventTime) error {
	if watermark == nil {
		return fmt.Errorf("projection.PushAndPullInMemoryContext: AckWatermark: %w", ErrStateAckNilWatermark)
	}
	c.state.Watermark = watermark
	return nil
}

func (c *PushAndPullInMemoryContext[A, B]) CurrentState() SnapshotState {
	return c.state
}
func (c *PushAndPullInMemoryContext[A, B]) LastWatermark() EventTime {
	if c.state.Watermark == nil {
		return 0
	}

	return *c.state.Watermark
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
				val.Data,
				func(x *Record[A]) error {
					y := f(x)

					err = ctx.PushOut(y)
					if err != nil {
						return fmt.Errorf("projection.DoMap: push: %w", err)
					}

					err = ctx.AckOffset(val.Offset)
					if err != nil {
						return fmt.Errorf("projection.DoMap: ack offset: %w", err)
					}

					return nil
				},
				func(x *Watermark[A]) error {
					if ctx.LastWatermark() >= x.EventTime {
						// we already processed this watermark
						return nil
					}

					err := ctx.PushOut(&Watermark[B]{
						EventTime: x.EventTime,
					})
					if err != nil {
						return fmt.Errorf("projection.DoMap: push wattermark  %w", err)
					}

					err = ctx.AckWatermark(&x.EventTime)
					if err != nil {
						return fmt.Errorf("projection.DoMap: ack watermark: %w", err)
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

	leftOrRight := true

	watermarkA := ctxA.LastWatermark()
	watermarkB := ctxB.LastWatermark()

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
					val.Data,
					func(x *Record[A]) error {
						y := f(&Left[*Record[A], *Record[B]]{
							Left: x,
						})

						err = ctxA.PushOut(y)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push left: %w", err)
						}

						err = ctxA.AckOffset(val.Offset)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: ack left offset: %w", err)
						}

						return nil
					},
					func(x *Watermark[A]) error {
						watermarkA = x.EventTime
						minWatermark := min(watermarkA, watermarkB)
						if ctxA.LastWatermark() >= minWatermark {
							// we already processed this watermarks
							return nil
						}

						err := ctxA.PushOut(&Watermark[C]{
							EventTime: minWatermark,
						})
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push left wattermark  %w", err)
						}
						err = ctxA.AckWatermark(&minWatermark)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: ack left watermarks: %w", err)
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
					val.Data,
					func(x *Record[B]) error {
						y := f(&Right[*Record[A], *Record[B]]{
							Right: x,
						})

						err = ctxB.PushOut(y)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push right: %w", err)
						}

						err = ctxB.AckOffset(val.Offset)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: ack right offset: %w", err)
						}

						return nil
					},
					func(x *Watermark[B]) error {
						watermarkB = x.EventTime
						minWatermark := min(watermarkA, watermarkB)
						if ctxB.LastWatermark() >= minWatermark {
							// we already processed this watermarks
							return nil
						}

						err := ctxB.PushOut(&Watermark[C]{
							EventTime: minWatermark,
						})
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push right wattermark  %w", err)
						}
						err = ctxB.AckWatermark(&minWatermark)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: ack right watermarks: %w", err)
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
			val.Data,
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
