package projection

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
	"time"
)

func withTime(hour, minute int) int64 {
	return time.
		Date(2019, 1, 1, hour, minute, 0, 0, time.UTC).
		UnixNano()
}

func printWindow(w *Window) {
	fmt.Printf("Window(%s, %s)\n",
		time.Unix(0, w.Start).Format("15:04"),
		time.Unix(0, w.End).Format("15:04"),
	)
}

func TestWindowing(t *testing.T) {
	list := []Item{
		{
			Key:       "a",
			Data:      nil,
			EventTime: withTime(10, 2),
		},
		{
			Key:       "a",
			Data:      nil,
			EventTime: withTime(10, 28),
		},
	}

	t.Run("assign session windows", func(t *testing.T) {
		result := AssignWindows(list, &SessionWindow{
			GapDuration: 30 * time.Minute,
		})
		expected := []Item{
			{
				Key:       "a",
				Data:      nil,
				EventTime: withTime(10, 2),
				Window: &Window{
					Start: withTime(10, 2),
					End:   withTime(10, 32),
				},
			},
			{
				Key:       "a",
				Data:      nil,
				EventTime: withTime(10, 28),
				Window: &Window{
					Start: withTime(10, 28),
					End:   withTime(10, 58),
				},
			},
		}

		assert.Equal(t, expected, result)
	})
	t.Run("assign sliding windows", func(t *testing.T) {
		result := AssignWindows(list, &SlidingWindow{
			Width:  2 * time.Minute,
			Period: 1 * time.Minute,
		})
		expected := []Item{
			{
				Key:       "a",
				Data:      nil,
				EventTime: withTime(10, 2),
				Window: &Window{
					Start: withTime(10, 1),
					End:   withTime(10, 3),
				},
			},
			{
				Key:       "a",
				Data:      nil,
				EventTime: withTime(10, 2),
				Window: &Window{
					Start: withTime(10, 2),
					End:   withTime(10, 4),
				},
			},
			{
				Key:       "a",
				Data:      nil,
				EventTime: withTime(10, 28),
				Window: &Window{
					Start: withTime(10, 27),
					End:   withTime(10, 29),
				},
			},
			{
				Key:       "a",
				Data:      nil,
				EventTime: withTime(10, 28),
				Window: &Window{
					Start: withTime(10, 28),
					End:   withTime(10, 30),
				},
			},
		}

		assert.Len(t, result, 4)
		if !assert.Equal(t, expected, result) {
			for idx := range result {
				printWindow(result[idx].Window)
				printWindow(expected[idx].Window)
			}
		}
	})
	t.Run("assign fixed windows", func(t *testing.T) {
		result := AssignWindows(list, &FixedWindow{
			Width: 30 * time.Minute,
		})
		expected := []Item{
			{
				Key:       "a",
				Data:      nil,
				EventTime: withTime(10, 2),
				Window: &Window{
					Start: withTime(10, 0),
					End:   withTime(10, 30),
				},
			},
			{
				Key:       "a",
				Data:      nil,
				EventTime: withTime(10, 28),
				Window: &Window{
					Start: withTime(10, 0),
					End:   withTime(10, 30),
				},
			},
		}

		assert.Equal(t, expected, result)
	})
}

func TestMergeWindows(t *testing.T) {
	list := []Item{
		{
			Key:       "k1",
			Data:      schema.MkString("v1"),
			EventTime: withTime(13, 2),
		},
		{
			Key:       "k2",
			Data:      schema.MkString("v2"),
			EventTime: withTime(13, 14),
		},
		{
			Key:       "k1",
			Data:      schema.MkString("v3"),
			EventTime: withTime(13, 57),
		},
		{
			Key:       "k1",
			Data:      schema.MkString("v4"),
			EventTime: withTime(13, 20),
		},
	}

	list2 := AssignWindows(list, &SessionWindow{
		GapDuration: 30 * time.Minute,
	})
	assert.Equal(t, []Item{
		{
			Key:       "k1",
			Data:      schema.MkString("v1"),
			EventTime: withTime(13, 2),
			Window: &Window{
				Start: withTime(13, 2),
				End:   withTime(13, 32),
			},
		},
		{
			Key:       "k2",
			Data:      schema.MkString("v2"),
			EventTime: withTime(13, 14),
			Window: &Window{
				Start: withTime(13, 14),
				End:   withTime(13, 44),
			},
		},
		{
			Key:       "k1",
			Data:      schema.MkString("v3"),
			EventTime: withTime(13, 57),
			Window: &Window{
				Start: withTime(13, 57),
				End:   withTime(14, 27),
			},
		},
		{
			Key:       "k1",
			Data:      schema.MkString("v4"),
			EventTime: withTime(13, 20),
			Window: &Window{
				Start: withTime(13, 20),
				End:   withTime(13, 50),
			},
		},
	}, list2, "AssignWindows")

	list3 := DropTimestamps(list2)
	assert.Equal(t, []Item{
		{
			Key:       "k1",
			Data:      schema.MkString("v1"),
			EventTime: 0,
			Window: &Window{
				Start: withTime(13, 2),
				End:   withTime(13, 32),
			},
		},
		{
			Key:       "k2",
			Data:      schema.MkString("v2"),
			EventTime: 0,
			Window: &Window{
				Start: withTime(13, 14),
				End:   withTime(13, 44),
			},
		},
		{
			Key:       "k1",
			Data:      schema.MkString("v3"),
			EventTime: 0,
			Window: &Window{
				Start: withTime(13, 57),
				End:   withTime(14, 27),
			},
		},
		{
			Key:       "k1",
			Data:      schema.MkString("v4"),
			EventTime: 0,
			Window: &Window{
				Start: withTime(13, 20),
				End:   withTime(13, 50),
			},
		},
	}, list3, "DropTimestamps")

	list4 := GroupByKey(list3)
	assert.Equal(t, []ItemGroupedByKey{
		{
			Key: "k1",
			Data: []Item{
				{
					Key:       "k1",
					Data:      schema.MkString("v1"),
					EventTime: 0,
					Window: &Window{
						Start: withTime(13, 2),
						End:   withTime(13, 32),
					},
				},
				{
					Key:       "k1",
					Data:      schema.MkString("v3"),
					EventTime: 0,
					Window: &Window{
						Start: withTime(13, 57),
						End:   withTime(14, 27),
					},
				},
				{
					Key:       "k1",
					Data:      schema.MkString("v4"),
					EventTime: 0,
					Window: &Window{
						Start: withTime(13, 20),
						End:   withTime(13, 50),
					},
				},
			},
		},
		{
			Key: "k2",
			Data: []Item{
				{
					Key:       "k2",
					Data:      schema.MkString("v2"),
					EventTime: 0,
					Window: &Window{
						Start: withTime(13, 14),
						End:   withTime(13, 44),
					},
				},
			},
		},
	}, list4, "GroupByKey")

	list5 := MergeWindows(list4, &SessionWindow{
		GapDuration: 30 * time.Minute,
	})
	assert.Equal(t, []ItemGroupedByKey{
		{
			Key: "k1",
			Data: []Item{
				{
					Key:       "k1",
					Data:      schema.MkString("v1"),
					EventTime: 0,
					Window: &Window{
						Start: withTime(13, 2),
						End:   withTime(13, 50),
					},
				},
				{
					Key:       "k1",
					Data:      schema.MkString("v3"),
					EventTime: 0,
					Window: &Window{
						Start: withTime(13, 57),
						End:   withTime(14, 27),
					},
				},
				{
					Key:       "k1",
					Data:      schema.MkString("v4"),
					EventTime: 0,
					Window: &Window{
						Start: withTime(13, 2),
						End:   withTime(13, 50),
					},
				},
			},
		},
		{
			Key: "k2",
			Data: []Item{
				{
					Key:       "k2",
					Data:      schema.MkString("v2"),
					EventTime: 0,
					Window: &Window{
						Start: withTime(13, 14),
						End:   withTime(13, 44),
					},
				},
			},
		},
	}, list5, "MergeWindows")

	list6 := GroupAlsoByWindow(list5)
	assert.Equal(t, []ItemGroupedByWindow{
		{
			Key:  "k1",
			Data: schema.MkList(schema.MkString("v1"), schema.MkString("v4")),
			Window: &Window{
				Start: withTime(13, 2),
				End:   withTime(13, 50),
			},
		},
		{
			Key:  "k1",
			Data: schema.MkList(schema.MkString("v3")),
			Window: &Window{
				Start: withTime(13, 57),
				End:   withTime(14, 27),
			},
		},
		{
			Key:  "k2",
			Data: schema.MkList(schema.MkString("v2")),
			Window: &Window{
				Start: withTime(13, 14),
				End:   withTime(13, 44),
			},
		},
	}, list6, "GroupAlsoByWindow")

	list7 := ExpandToElements(list6)
	assert.Equal(t, []Item{
		{
			Key:       "k1",
			Data:      schema.MkList(schema.MkString("v1"), schema.MkString("v4")),
			EventTime: withTime(13, 50),
			Window: &Window{
				Start: withTime(13, 2),
				End:   withTime(13, 50),
			},
		},
		{
			Key:       "k1",
			Data:      schema.MkList(schema.MkString("v3")),
			EventTime: withTime(14, 27),
			Window: &Window{
				Start: withTime(13, 57),
				End:   withTime(14, 27),
			},
		},
		{
			Key:       "k2",
			Data:      schema.MkList(schema.MkString("v2")),
			EventTime: withTime(13, 44),
			Window: &Window{
				Start: withTime(13, 14),
				End:   withTime(13, 44),
			},
		},
	}, list7)
}

func TestWindowMerginOnly(t *testing.T) {
	list := []ItemGroupedByKey{
		{
			Key: "k1",
			Data: []Item{
				{
					Key:       "k1",
					Data:      schema.MkString("v1"),
					EventTime: withTime(13, 2),
				},
				{
					Key:       "k1",
					Data:      schema.MkString("v3"),
					EventTime: withTime(13, 57),
				},
				{
					Key:       "k1",
					Data:      schema.MkString("v4"),
					EventTime: withTime(13, 20),
				},
			},
		},
		{
			Key: "k2",
			Data: []Item{
				{
					Key:       "k2",
					Data:      schema.MkString("v2"),
					EventTime: withTime(13, 14),
				},
			},
		},
	}

	t.Run("merge session windows", func(t *testing.T) {
		window := &SessionWindow{
			GapDuration: 30 * time.Minute,
		}
		var list2 []ItemGroupedByKey
		for _, item := range list {
			list2 = append(list2, ItemGroupedByKey{
				Key:  item.Key,
				Data: AssignWindows(item.Data, window),
			})
		}
		result := MergeWindows(list2, window)
		assert.Equal(t, []ItemGroupedByKey{
			{
				Key: "k1",
				Data: []Item{
					{
						Key:       "k1",
						Data:      schema.MkString("v1"),
						EventTime: withTime(13, 2),
						Window: &Window{
							Start: withTime(13, 2),
							End:   withTime(13, 50),
						},
					},
					{
						Key:       "k1",
						Data:      schema.MkString("v3"),
						EventTime: withTime(13, 57),
						Window: &Window{
							Start: withTime(13, 57),
							End:   withTime(14, 27),
						},
					},
					{
						Key:       "k1",
						Data:      schema.MkString("v4"),
						EventTime: withTime(13, 20),
						Window: &Window{
							Start: withTime(13, 2),
							End:   withTime(13, 50),
						},
					},
				},
			},
			{
				Key: "k2",
				Data: []Item{
					{
						Key:       "k2",
						Data:      schema.MkString("v2"),
						EventTime: withTime(13, 14),
						Window: &Window{
							Start: withTime(13, 14),
							End:   withTime(13, 44),
						},
					},
				},
			},
		}, result, "MergeWindows")
	})
	t.Run("merge sliding windows", func(t *testing.T) {
		window := &SlidingWindow{
			Width:  2 * time.Minute,
			Period: 1 * time.Minute,
		}
		var list2 []ItemGroupedByKey
		for _, item := range list {
			list2 = append(list2, ItemGroupedByKey{
				Key:  item.Key,
				Data: AssignWindows(item.Data, window),
			})
		}

		result := MergeWindows(list2, window)
		assert.Equal(t, []ItemGroupedByKey{
			{
				Key: "k1",
				Data: []Item{
					{
						Key:       "k1",
						Data:      schema.MkString("v1"),
						EventTime: withTime(13, 2),
						Window: &Window{
							Start: withTime(13, 1),
							End:   withTime(13, 3),
						},
					},
					{
						Key:       "k1",
						Data:      schema.MkString("v1"),
						EventTime: withTime(13, 2),
						Window: &Window{
							Start: withTime(13, 2),
							End:   withTime(13, 4),
						},
					},
					{
						Key:       "k1",
						Data:      schema.MkString("v3"),
						EventTime: withTime(13, 57),
						Window: &Window{
							Start: withTime(13, 56),
							End:   withTime(13, 58),
						},
					},
					{
						Key:       "k1",
						Data:      schema.MkString("v3"),
						EventTime: withTime(13, 57),
						Window: &Window{
							Start: withTime(13, 57),
							End:   withTime(13, 59),
						},
					},
					{
						Key:       "k1",
						Data:      schema.MkString("v4"),
						EventTime: withTime(13, 20),
						Window: &Window{
							Start: withTime(13, 19),
							End:   withTime(13, 21),
						},
					},
					{
						Key:       "k1",
						Data:      schema.MkString("v4"),
						EventTime: withTime(13, 20),
						Window: &Window{
							Start: withTime(13, 20),
							End:   withTime(13, 22),
						},
					},
				},
			},
			{
				Key: "k2",
				Data: []Item{
					{
						Key:       "k2",
						Data:      schema.MkString("v2"),
						EventTime: withTime(13, 14),
						Window: &Window{
							Start: withTime(13, 13),
							End:   withTime(13, 15),
						},
					},
					{
						Key:       "k2",
						Data:      schema.MkString("v2"),
						EventTime: withTime(13, 14),
						Window: &Window{
							Start: withTime(13, 14),
							End:   withTime(13, 16),
						},
					},
				},
			},
		}, result, "MergeWindows")
	})
	t.Run("merge fixed windows", func(t *testing.T) {
		window := &FixedWindow{
			Width: 30 * time.Minute,
		}
		var list2 []ItemGroupedByKey
		for _, item := range list {
			list2 = append(list2, ItemGroupedByKey{
				Key:  item.Key,
				Data: AssignWindows(item.Data, window),
			})
		}
		result := MergeWindows(list2, window)
		assert.Equal(t, []ItemGroupedByKey{
			{
				Key: "k1",
				Data: []Item{
					{
						Key:       "k1",
						Data:      schema.MkString("v1"),
						EventTime: withTime(13, 2),
						Window: &Window{
							Start: withTime(13, 0),
							End:   withTime(13, 30),
						},
					},
					{
						Key:       "k1",
						Data:      schema.MkString("v3"),
						EventTime: withTime(13, 57),
						Window: &Window{
							Start: withTime(13, 30),
							End:   withTime(14, 0),
						},
					},
					{
						Key:       "k1",
						Data:      schema.MkString("v4"),
						EventTime: withTime(13, 20),
						Window: &Window{
							Start: withTime(13, 0),
							End:   withTime(13, 30),
						},
					},
				},
			},
			{
				Key: "k2",
				Data: []Item{
					{
						Key:       "k2",
						Data:      schema.MkString("v2"),
						EventTime: withTime(13, 14),
						Window: &Window{
							Start: withTime(13, 0),
							End:   withTime(13, 30),
						},
					},
				},
			},
		}, result, "MergeWindows")

		byWindow := GroupAlsoByWindow(result)

		assert.Equal(t, []ItemGroupedByWindow{
			{
				Key:  "k1",
				Data: schema.MkList(schema.MkString("v1"), schema.MkString("v4")),
				Window: &Window{
					Start: withTime(13, 0),
					End:   withTime(13, 30),
				},
			},
			{
				Key:  "k1",
				Data: schema.MkList(schema.MkString("v3")),
				Window: &Window{
					Start: withTime(13, 30),
					End:   withTime(14, 0),
				},
			},
			{
				Key:  "k2",
				Data: schema.MkList(schema.MkString("v2")),
				Window: &Window{
					Start: withTime(13, 0),
					End:   withTime(13, 30),
				},
			},
		}, byWindow, "MergeWindows")
	})
}

type triggerCase struct {
	trigger       TriggerType
	shouldTrigger bool
}

func TestWindowTrigger(t *testing.T) {
	useCases := map[string]struct {
		w      *Window
		td     TriggerDescription
		events []triggerCase
	}{
		"should trigger aways at correct AtPeriod": {
			w: &Window{
				Start: withTime(13, 0),
				End:   withTime(13, 30),
			},
			td: &AtPeriod{
				Duration: 10 * time.Minute,
			},
			events: []triggerCase{
				{
					trigger: &AtPeriod{
						Duration: 1 * time.Minute,
					},
					shouldTrigger: false,
				},
				{
					trigger: &AtPeriod{
						Duration: 10 * time.Minute,
					},
					shouldTrigger: true,
				},
				// at this point, we know that we see all triggers
				// that window need to be flushed
				{
					trigger: &AtPeriod{
						Duration: 10 * time.Minute,
					},
					shouldTrigger: true,
				},
				{
					trigger: &AtPeriod{
						Duration: 1 * time.Minute,
					},
					shouldTrigger: true,
				},
				{
					trigger: &AtPeriod{
						Duration: 10 * time.Minute,
					},
					shouldTrigger: true,
				},
			},
		},
		"should trigger when all conditions are true": {
			w: &Window{
				Start: withTime(13, 0),
				End:   withTime(13, 30),
			},
			td: &AllOf{
				Triggers: []TriggerDescription{
					&AtPeriod{
						Duration: 10 * time.Minute,
					},
					&AtPeriod{
						Duration: 1 * time.Minute,
					},
				},
			},
			events: []triggerCase{
				{
					trigger: &AtPeriod{
						Duration: 1 * time.Minute,
					},
					shouldTrigger: false,
				},
				{
					trigger: &AtPeriod{
						Duration: 3 * time.Minute,
					},
					shouldTrigger: false,
				},
				{
					trigger: &AtPeriod{
						Duration: 10 * time.Minute,
					},
					shouldTrigger: true,
				},
			},
		},
		"should trigger when any conditions are true": {
			w: &Window{
				Start: withTime(13, 0),
				End:   withTime(13, 30),
			},
			td: &AnyOf{
				Triggers: []TriggerDescription{
					&AtPeriod{
						Duration: 10 * time.Minute,
					},
					&AtWindowItemSize{
						Number: 2,
					},
				},
			},
			events: []triggerCase{
				{
					trigger: &AtWindowItemSize{
						Number: 1,
					},
					shouldTrigger: false,
				},
				{
					trigger: &AtPeriod{
						Duration: 10 * time.Minute,
					},
					shouldTrigger: true,
				},
			},
		},
		"should trigger window flush after timeout, but if watermark says that there are events that will fall under this window, then let's wait": {
			w: &Window{
				Start: withTime(13, 0),
				End:   withTime(13, 30),
			},
			td: &AnyOf{
				Triggers: []TriggerDescription{
					// when watermark says that there most likely won't be any older events
					// so that we can flush this window
					&AtWatermark{},
					// or we are at flush period, but watermark haven't reached this window yet
					// so wait, and flush imminently, when watermark will reach this window
					&AllOf{
						Triggers: []TriggerDescription{
							&AtWatermark{},
							&AtPeriod{
								Duration: 10 * time.Minute,
							},
						},
					},
					// for very low latency granularity, we can flush window on every event
					&AtWindowItemSize{
						Number: 1,
					},
				},
			},
			events: []triggerCase{
				{
					trigger: &AtPeriod{
						Duration: 10 * time.Minute,
					},
					shouldTrigger: false,
				},
				{
					trigger: &AtWatermark{
						// watermark is at 13:00, so we should wait, since window ends at 13:30
						Timestamp: withTime(13, 0),
					},
					shouldTrigger: false,
				},
				{
					trigger: &AtWatermark{
						Timestamp: withTime(13, 30),
					},
					shouldTrigger: true,
				},
			},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			wt := NewWindowTrigger(uc.w, uc.td)
			for idx, event := range uc.events {
				wt.ReceiveEvent(event.trigger)

				if event.shouldTrigger {
					assert.Truef(t, wt.ShouldTrigger(), "should flush on trigger but didn't; idx=%d", idx)
				} else {
					assert.Falsef(t, wt.ShouldTrigger(), "should NOT flush on trigger but did; idx=%d", idx)
				}
			}
		})
	}
}
