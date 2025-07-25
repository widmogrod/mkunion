package projection

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/stream"
	"math"
	"math/rand"
	"time"
)

var (
	ErrStateAckNilOffset    = errors.New("cannot acknowledge nil offset")
	ErrStateAckNilWatermark = errors.New("cannot acknowledge nil watermark")
)

const (
	KeySystemWatermark = "watermark"
)

//go:tag mkunion:"Data[A]"
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
		PullIn() (*stream.Item[Data[A]], error)
		AckOffset(offset *stream.Offset) error

		PushOut(Data[B]) error
		AckWatermark(watermark *stream.EventTime) error
		LastWatermark() EventTime
	}
	SnapshotContext interface {
		CurrentState() SnapshotState
	}
)

func NewPushAndPullInMemoryContext[A, B any](state *PullPushContextState, stream stream.Stream[schema.Schema]) *PushAndPullInMemoryContext[A, B] {
	return &PushAndPullInMemoryContext[A, B]{
		state:  state,
		stream: stream,
	}
}

var _ PushAndPull[int, int] = (*PushAndPullInMemoryContext[int, int])(nil)

type PushAndPullInMemoryContext[A, B any] struct {
	state    *PullPushContextState
	stream   stream.Stream[schema.Schema]
	simulate *SimulateProblem
}

func (c *PushAndPullInMemoryContext[A, B]) PullIn() (*stream.Item[Data[A]], error) {
	if c.simulate != nil && c.simulate.ErrorOnPullIn != nil {
		if rand.Float64() < c.simulate.ErrorOnPullInProbability {
			return nil, c.simulate.ErrorOnPullIn
		}
	}

	result, err := pullFrom(c.stream, c.state)
	if err != nil {
		return nil, fmt.Errorf("projection.PushAndPullInMemoryContext: PullIn: %w", err)
	}

	item, err := itemToTyped[A](result)
	if err != nil {
		return nil, fmt.Errorf("projection.PushAndPullInMemoryContext: PullIn: type conversion; %w", err)
	}

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

	err := pushOut[B](c.stream, c.state, x)
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

func pullFrom(s stream.Stream[schema.Schema], state SnapshotState) (*stream.Item[schema.Schema], error) {
	return MatchSnapshotStateR2(
		state,
		func(x *PullPushContextState) (*stream.Item[schema.Schema], error) {
			var cmd stream.PullCMD
			if x.Offset == nil {
				cmd = &stream.FromBeginning{
					Topic: x.PullTopic,
				}
			} else {
				cmd = &stream.FromOffset{
					Topic:  x.PullTopic,
					Offset: x.Offset,
				}
			}

			result, err := s.Pull(cmd)
			if err != nil {
				return nil, fmt.Errorf("projection.PullFrom: %w", err)
			}

			return result, nil
		},
		func(x *JoinContextState) (*stream.Item[schema.Schema], error) {
			var cmd stream.PullCMD

			var offset *stream.Offset
			var pullTopic string

			if x.LeftOrRight {
				offset = x.Offset1
				pullTopic = x.PullTopic1
			} else {
				offset = x.Offset2
				pullTopic = x.PullTopic2
			}

			if offset != nil {
				cmd = &stream.FromOffset{
					Topic:  pullTopic,
					Offset: offset,
				}
			} else {
				cmd = &stream.FromBeginning{
					Topic: pullTopic,
				}
			}

			result, err := s.Pull(cmd)
			if err != nil {
				return nil, fmt.Errorf("projection.PullFrom: %w", err)
			}

			return result, nil
		},
	)
}

func pushOut[A any](s stream.Stream[schema.Schema], state SnapshotState, data Data[A]) error {
	return MatchSnapshotStateR1(
		state,
		func(x *PullPushContextState) error {
			result, err := RecordToStreamItem(x.PushTopic, data)
			if err != nil {
				return fmt.Errorf("projection.pushOut: %w", err)
			}

			item := &stream.Item[schema.Schema]{
				Topic:     result.Topic,
				Key:       result.Key,
				Data:      schema.FromGo(result.Data),
				EventTime: result.EventTime,
				Offset:    result.Offset,
			}

			return s.Push(item)
		},
		func(x *JoinContextState) error {
			result, err := RecordToStreamItem(x.PushTopic, data)
			if err != nil {
				return fmt.Errorf("projection.pushOut: %w", err)
			}

			item := &stream.Item[schema.Schema]{
				Topic:     result.Topic,
				Key:       result.Key,
				Data:      schema.FromGo(result.Data),
				EventTime: result.EventTime,
				Offset:    result.Offset,
			}

			return s.Push(item)
		},
	)
}

func itemToTyped[A any](item *stream.Item[schema.Schema]) (*stream.Item[Data[A]], error) {
	data := schema.ToGo[Data[A]](item.Data)

	result := &stream.Item[Data[A]]{
		Topic:     item.Topic,
		Key:       item.Key,
		Data:      data,
		EventTime: item.EventTime,
		Offset:    item.Offset,
	}

	if result.Key == KeySystemWatermark {
		result.Data = &Watermark[A]{
			EventTime: *item.EventTime,
		}
	}

	return result, nil
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

func DoLoad[A any](ctx PushAndPull[any, A], f func(push func(record Data[A]) error) error) error {
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
			if IsWatermarkMarksEndOfStream(ctx.LastWatermark()) {
				return nil
			}

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

			err = ctx.AckOffset(val.Offset)
			if err != nil {
				return fmt.Errorf("projection.DoMap: ack offset: %w", err)
			}
		}
	}
}

func DoJoin[A, B, C any](
	dataStream stream.Stream[schema.Schema],
	state *JoinContextState,
	f func(x Either[*Record[A], *Record[B]]) *Record[C],
) error {
	delay := 1 * time.Second
	timer := time.NewTimer(delay)
	defer timer.Stop()

	watermarkA := state.Watermark
	watermarkB := state.Watermark

	for {
		select {
		case <-timer.C:
			return fmt.Errorf("projection.DoJoin: timeout; %w", stream.ErrNoMoreNewDataInStream)
		default:
			if IsWatermarkMarksEndOfStream(state.Watermark) {
				return nil
			}

			item, err := pullFrom(dataStream, state)
			if err != nil {
				if errors.Is(err, stream.ErrNoMoreNewDataInStream) {
					return nil
				}
				return fmt.Errorf("projection.DoJoin: pull; %w", err)
			}

			if state.LeftOrRight {
				val, err := itemToTyped[A](item)
				if err != nil {
					return fmt.Errorf("projection.DoJoin: covnert left; %w", err)
				}

				err = MatchDataR1(
					val.Data,
					func(x *Record[A]) error {
						y := f(&Left[*Record[A], *Record[B]]{
							Left: x,
						})

						err := pushOut[C](dataStream, state, y)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push left: %w", err)
						}

						return nil
					},
					func(x *Watermark[A]) error {
						watermarkA = x.EventTime
						minWatermark := min(watermarkA, watermarkB)
						if state.Watermark >= minWatermark {
							// we already processed this watermarks
							return nil
						}
						err := pushOut[C](dataStream, state, &Watermark[C]{
							EventTime: minWatermark,
						})
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push left wattermark  %w", err)
						}

						state.Watermark = minWatermark

						return nil
					},
				)

				if err != nil {
					return fmt.Errorf("projection.DoJoin: left; %w", err)
				}

				state.Offset1 = val.Offset
			} else {
				val, err := itemToTyped[B](item)
				if err != nil {
					return fmt.Errorf("projection.DoJoin: covnert right; %w", err)
				}

				err = MatchDataR1(
					val.Data,
					func(x *Record[B]) error {
						y := f(&Right[*Record[A], *Record[B]]{
							Right: x,
						})

						err := pushOut[C](dataStream, state, y)
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push right: %w", err)
						}

						return nil
					},
					func(x *Watermark[B]) error {
						watermarkB = x.EventTime

						minWatermark := min(watermarkA, watermarkB)
						if state.Watermark >= minWatermark {
							// we already processed this watermarks
							return nil
						}
						err := pushOut[C](dataStream, state, &Watermark[C]{
							EventTime: minWatermark,
						})
						if err != nil {
							return fmt.Errorf("projection.DoJoin: push right wattermark  %w", err)
						}

						state.Watermark = minWatermark

						return nil
					},
				)

				if err != nil {
					return fmt.Errorf("projection.DoJoin: left; %w", err)
				}

				state.Offset2 = val.Offset
			}

			state.LeftOrRight = !state.LeftOrRight
		}

		timer.Reset(delay)
	}
}

func DoSink[A any](ctx PushAndPull[A, any], f func(*Record[A]) error) error {
	for {
		if IsWatermarkMarksEndOfStream(ctx.LastWatermark()) {
			return nil
		}

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
				return ctx.AckWatermark(&x.EventTime)
			},
		)
		if err != nil {
			return fmt.Errorf("projection.DoSink: sink: %w", err)
		}
		err = ctx.AckOffset(val.Offset)
		if err != nil {
			return fmt.Errorf("projection.DoSink: ack offset: %w", err)
		}
	}
}

//go:tag mkunion:"Either[A, B]"
type (
	Left[A, B any] struct {
		Left A
	}
	Right[A, B any] struct {
		Right B
	}
)
