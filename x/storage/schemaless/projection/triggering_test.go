package projection

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"math"
	"testing"
	"time"
)

func TestTriggers(t *testing.T) {
	useCases := map[string]struct {
		td       TriggerDescription
		wd       WindowDescription
		fm       WindowFlushMode
		expected []Item
	}{
		"should trigger window emitting once at period 100ms, and 10 items arrives as 1 item": {
			td: &AllOf{
				Triggers: []TriggerDescription{
					&AtPeriod{
						Duration: 100 * time.Millisecond,
					},
					&AtWatermark{},
				},
			},
			wd: &FixedWindow{
				Width: 100 * time.Millisecond,
			},
			fm: &Discard{},
			expected: []Item{
				{
					Key: "key",
					Data: schema.MkList(
						schema.MkInt(0), schema.MkInt(1), schema.MkInt(2), schema.MkInt(3), schema.MkInt(4),
						schema.MkInt(5), schema.MkInt(6), schema.MkInt(7), schema.MkInt(8), schema.MkInt(9),
					),
					EventTime: withTime(10, 0) + (100 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0),
						End:   withTime(10, 0) + (100 * int64(time.Millisecond)),
					},
				},
				{
					Key: "key",
					Data: schema.MkList(
						schema.MkInt(10), schema.MkInt(11), schema.MkInt(12), schema.MkInt(13), schema.MkInt(14),
						schema.MkInt(15), schema.MkInt(16), schema.MkInt(17), schema.MkInt(18),
						// it should fit in 100ms window, but due timeouts being part of process time, not event time,
						// it's not guaranteed that when system will receive event at 10.1s, it will be processed at 10.2s
						schema.MkInt(19),
					),
					EventTime: withTime(10, 0) + (200 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0) + (100 * int64(time.Millisecond)),
						End:   withTime(10, 0) + (200 * int64(time.Millisecond)),
					},
				},
			},
		},
		"should trigger window emitting when window size reach 2 item": {
			td: &AtWindowItemSize{
				Number: 2,
			},
			wd: &FixedWindow{
				Width: 100 * time.Millisecond,
			},
			fm: &Discard{},
			expected: []Item{
				{
					Key: "key",
					Data: schema.MkList(
						schema.MkInt(0), schema.MkInt(1),
					),
					EventTime: withTime(10, 0) + (100 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0),
						End:   withTime(10, 0) + (100 * int64(time.Millisecond)),
					},
				},
				{
					Key: "key",
					Data: schema.MkList(
						schema.MkInt(2), schema.MkInt(3),
					),
					EventTime: withTime(10, 0) + (100 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0),
						End:   withTime(10, 0) + (100 * int64(time.Millisecond)),
					},
				},
				{
					Key: "key",
					Data: schema.MkList(
						schema.MkInt(4), schema.MkInt(5),
					),
					EventTime: withTime(10, 0) + (100 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0),
						End:   withTime(10, 0) + (100 * int64(time.Millisecond)),
					},
				},
			},
		},
		"should trigger window flush at watermark": {
			td: &AtWatermark{},
			wd: &FixedWindow{
				Width: 100 * time.Millisecond,
			},
			fm: &Discard{},
			expected: []Item{
				{
					Key: "key",
					Data: schema.MkList(
						schema.MkInt(0), schema.MkInt(1), schema.MkInt(2), schema.MkInt(3), schema.MkInt(4),
						schema.MkInt(5), schema.MkInt(6), schema.MkInt(7), schema.MkInt(8), schema.MkInt(9),
					),
					EventTime: withTime(10, 0) + (100 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0),
						End:   withTime(10, 0) + (100 * int64(time.Millisecond)),
					},
				},
			},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			returning := &ListAssert{t: t}

			trigger := NewTriggerManager(uc.td)

			timeTickers := NewTimeTicker()
			timeTickers.Register(uc.td, trigger)
			defer timeTickers.Unregister(uc.td)

			wb := NewWindowBuffer(uc.wd, trigger)

			trigger.WhenTrigger(func(kw *KeyedWindow) {
				wb.EachKeyedWindow(kw, func(group *ItemGroupedByWindow) {
					returning.Returning(ToElement(group))
					wb.RemoveItemGropedByWindow(group)
				})
			})

			for item := range GenerateItemsEvery(withTime(10, 0), 20, 10*time.Millisecond) {
				wb.Append(item)
			}

			// trigger watermark that there won't be any more events
			trigger.SignalWatermark(math.MaxInt64)

			time.Sleep(100 * time.Millisecond)
			for i, expected := range uc.expected {
				returning.AssertAt(i, expected)
			}
		})
	}
}

func TestAggregate(t *testing.T) {
	// arithmetic sum of series 0..9, 10..19, 0 .. 19
	// 45, 145, 190
	useCases := map[string]struct {
		td       TriggerDescription
		wd       WindowDescription
		fm       WindowFlushMode
		expected []Item
	}{
		"should trigger window emitting evey period 100ms, and 10 items arrives as 1 item, late arrivals are new aggregations": {
			td: &AtPeriod{
				Duration: 100 * time.Millisecond,
			},
			wd: &FixedWindow{
				Width: 100 * time.Millisecond,
			},
			fm: &Discard{},
			expected: []Item{
				{
					Key:       "key",
					Data:      schema.MkInt(45), // arithmetic sum fo series 0..9
					EventTime: withTime(10, 0) + (100 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0),
						End:   withTime(10, 0) + (100 * int64(time.Millisecond)),
					},
				},
				{
					Key:       "key",
					Data:      schema.MkInt(126),
					EventTime: withTime(10, 0) + (200 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0) + (100 * int64(time.Millisecond)),
						End:   withTime(10, 0) + (200 * int64(time.Millisecond)),
					},
				},
			},
		},
		"should trigger window emitting evey period 100ms, and 10 items arrives as 1 item, late arrivals use past aggregation as base": {
			td: &AtPeriod{
				Duration: 100 * time.Millisecond,
			},
			wd: &FixedWindow{
				Width: 100 * time.Millisecond,
			},
			fm: &Accumulate{},
			expected: []Item{
				{
					Key:       "key",
					Data:      schema.MkInt(45), // arithmetic sum fo series 0..9
					EventTime: withTime(10, 0) + (100 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0),
						End:   withTime(10, 0) + (100 * int64(time.Millisecond)),
					},
				},
				// this window is incomplete, and will be remitted
				{
					Key:       "key",
					Data:      schema.MkInt(126),
					EventTime: withTime(10, 0) + (200 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0) + (100 * int64(time.Millisecond)),
						End:   withTime(10, 0) + (200 * int64(time.Millisecond)),
					},
				},
				// here is complete aggregation in effect.
				{
					Key:       "key",
					Data:      schema.MkInt(145), // arithmetic sum of series 10..19
					EventTime: withTime(10, 0) + (200 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0) + (100 * int64(time.Millisecond)),
						End:   withTime(10, 0) + (200 * int64(time.Millisecond)),
					},
				},
			},
		},
		"should trigger window emitting every period 100ms, and 10 items arrives as 1 item, late arrivals use past aggregation as base, and retract last change": {
			td: &AtPeriod{
				Duration: 100 * time.Millisecond,
			},
			wd: &FixedWindow{
				Width: 100 * time.Millisecond,
			},
			fm: &AccumulatingAndRetracting{},
			expected: []Item{
				{
					Key:       "key",
					Data:      schema.MkInt(45), // arithmetic sum fo series 0..9
					EventTime: withTime(10, 0) + (100 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0),
						End:   withTime(10, 0) + (100 * int64(time.Millisecond)),
					},
					Type: ItemAggregation,
				},
				// this window is incomplete, and will be remitted
				{
					Key:       "key",
					Data:      schema.MkInt(126),
					EventTime: withTime(10, 0) + (200 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0) + (100 * int64(time.Millisecond)),
						End:   withTime(10, 0) + (200 * int64(time.Millisecond)),
					},
					Type: ItemAggregation,
				},
				// here is retracting and aggregate in effect.
				{
					Key: "key",
					Data: PackRetractAndAggregate(
						schema.MkInt(126), // retract previous
						schema.MkInt(145), // aggregate new
					),
					EventTime: withTime(10, 0) + (200 * int64(time.Millisecond)),
					Window: &Window{
						Start: withTime(10, 0) + (100 * int64(time.Millisecond)),
						End:   withTime(10, 0) + (200 * int64(time.Millisecond)),
					},
					Type: ItemRetractAndAggregate,
				},
			},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			returning := &ListAssert{t: t}

			trigger := NewTriggerManager(uc.td)

			timeTickers := NewTimeTicker()
			timeTickers.Register(uc.td, trigger)
			defer timeTickers.Unregister(uc.td)

			wb := NewWindowBuffer(uc.wd, trigger)

			handler2 := &AccumulateDiscardRetractHandler{
				fm: uc.fm,
				mapf: &SimpleProcessHandler{
					P: func(item Item, returning func(Item)) error {
						returning(Item{
							Key: item.Key,
							Data: schema.MkInt(schema.Reduce[int64](
								item.Data,
								0,
								func(s schema.Schema, i int64) int64 {
									x, err := schema.ToGoG[float64](s)
									if err != nil {
										panic(err)
									}
									return int64(x) + i
								},
							)),
							EventTime: item.EventTime,
							Window:    item.Window,
						})
						return nil
					}},
				mergef: &MergeHandler[int]{
					Combine: func(a, b int) (int, error) {
						return a + b, nil
					},
				},
				bag: NewInMemoryBagOf[Item](),
			}

			trigger.WhenTrigger(func(kw *KeyedWindow) {
				wb.EachKeyedWindow(kw, func(group *ItemGroupedByWindow) {
					err := handler2.Process(ToElement(group), returning.Returning)
					assert.NoError(t, err)
					wb.RemoveItemGropedByWindow(group)
				})
			})

			for item := range GenerateItemsEvery(withTime(10, 0), 20, 10*time.Millisecond) {
				wb.Append(item)
			}

			// trigger watermark that there won't be any more events
			trigger.SignalWatermark(math.MaxInt64)

			time.Sleep(100 * time.Millisecond)
			for i, expected := range uc.expected {
				returning.AssertAt(i, expected)
			}
		})
	}
}
