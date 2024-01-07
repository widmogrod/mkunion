package projection

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"time"
)

//go:generate go run ../../../../cmd/mkunion/main.go
//go:generate go run ../../../../cmd/mkunion/main.go serde

//go:tag mkunion:"WindowDescription"
type (
	SessionWindow struct {
		GapDuration time.Duration
	}
	SlidingWindow struct {
		Width  time.Duration
		Period time.Duration
	}
	FixedWindow struct {
		Width time.Duration
	}
)

func AssignWindows(x []Item, wd WindowDescription) []Item {
	return MatchWindowDescriptionR1(
		wd,
		func(wd *SessionWindow) []Item {
			return assignSessionWindows(x, wd)
		},
		func(wd *SlidingWindow) []Item {
			return assignSlidingWindows(x, wd)
		},
		func(wd *FixedWindow) []Item {
			return assignFixedWindows(x, wd)
		},
	)
}

func assignFixedWindows(x []Item, wd *FixedWindow) []Item {
	result := make([]Item, 0, len(x))
	for _, item := range x {
		start := item.EventTime - item.EventTime%wd.Width.Nanoseconds()
		end := start + wd.Width.Nanoseconds()
		result = append(result, Item{
			Key:       item.Key,
			Data:      item.Data,
			EventTime: item.EventTime,
			Window: &Window{
				Start: start,
				End:   end,
			},
		})
	}
	return result
}

func assignSlidingWindows(x []Item, wd *SlidingWindow) []Item {
	result := make([]Item, 0, len(x))
	for _, item := range x {
		eventTime := time.Unix(0, item.EventTime)
		// slicing window is [start, end)
		// left side inclusive, and right side exclusive,
		// so we need to add 1 period to the end
		for start := eventTime.Add(-wd.Width).Add(wd.Period); start.UnixNano() <= item.EventTime; start = start.Add(wd.Period) {
			result = append(result, Item{
				Key:       item.Key,
				Data:      item.Data,
				EventTime: item.EventTime,
				Window: &Window{
					Start: start.UnixNano(),
					End:   start.Add(wd.Width).UnixNano(),
				},
			})
		}
	}
	return result
}

func assignSessionWindows(x []Item, wd *SessionWindow) []Item {
	result := make([]Item, 0, len(x))
	for _, item := range x {
		result = append(result, Item{
			Key:       item.Key,
			Data:      item.Data,
			EventTime: item.EventTime,
			Window: &Window{
				Start: item.EventTime,
				End:   time.Unix(0, item.EventTime).Add(wd.GapDuration).UnixNano(),
			},
		})
	}
	return result
}

func MergeWindows(x []ItemGroupedByKey, wd WindowDescription) []ItemGroupedByKey {
	return MatchWindowDescriptionR1(
		wd,
		func(wd *SessionWindow) []ItemGroupedByKey {
			return mergeSessionWindows(x, wd)
		},
		func(wd *SlidingWindow) []ItemGroupedByKey {
			// assumption here is that before calling MergeWindows,
			// items got assigned window using the same WindowDefinition,
			// so we can assume that all items in the group have the same value for sliding & fixed windows
			// that don't need to be adjusted, like in session windows
			return x
		},
		func(wd *FixedWindow) []ItemGroupedByKey {
			// assumption here is that before calling MergeWindows,
			// items got assigned window using the same WindowDefinition,
			// so we can assume that all items in the group have the same value for sliding & fixed windows
			// that don't need to be adjusted, like in session windows
			return x
		},
	)
}

func winNo(w *Window, min int64, wd *SessionWindow) int64 {
	return int64((time.Unix(0, w.Start).Sub(time.Unix(0, min)) + wd.GapDuration) / wd.GapDuration)
}

func mergeSessionWindows(x []ItemGroupedByKey, wd *SessionWindow) []ItemGroupedByKey {
	result := make([]ItemGroupedByKey, 0, len(x))
	for _, group := range x {
		var min int64
		for _, item := range group.Data {
			if min > item.Window.Start {
				min = item.Window.Start
			}
		}

		window := map[int64]*Window{}
		for _, item := range group.Data {
			// detect where in which session window item belongs
			// if in window session there are no items, then leave element as is
			// when there are items, then merge them and set window to the min start and max end of elements in this window

			windowNo := winNo(item.Window, min, wd)
			if _, ok := window[windowNo]; !ok {
				window[windowNo] = &Window{
					Start: item.Window.Start,
					End:   item.Window.End,
				}
			} else {
				w := window[windowNo]
				if w.Start > item.Window.Start {
					w.Start = item.Window.Start
				}
				if w.End < item.Window.End {
					w.End = item.Window.End
				}
			}
		}

		newGroup := ItemGroupedByKey{
			Key:  group.Key,
			Data: make([]Item, 0, len(group.Data)),
		}
		for _, item := range group.Data {
			windowNo := winNo(item.Window, min, wd)
			newGroup.Data = append(newGroup.Data, Item{
				Key:       item.Key,
				Data:      item.Data,
				EventTime: item.EventTime,
				Window: &Window{
					Start: window[windowNo].Start,
					End:   window[windowNo].End,
				},
			})
		}

		result = append(result, newGroup)
	}

	return result
}

func DropTimestamps(x []Item) []Item {
	result := make([]Item, 0, len(x))
	for _, item := range x {
		result = append(result, Item{
			Key:       item.Key,
			Data:      item.Data,
			EventTime: 0,
			Window:    item.Window,
		})
	}
	return result
}

func GroupByKey(x []Item) []ItemGroupedByKey {
	result := make([]*ItemGroupedByKey, 0, 0)
	groups := map[string]*ItemGroupedByKey{}
	for _, item := range x {
		group, ok := groups[item.Key]
		if !ok {
			group = &ItemGroupedByKey{Key: item.Key}
			groups[item.Key] = group
			result = append(result, group)
		}
		group.Data = append(group.Data, item)
	}

	// yet another workaround for unordered maps in golang
	final := make([]ItemGroupedByKey, 0, len(result))
	for _, group := range result {
		final = append(final, *group)
	}

	return final
}

func GroupAlsoByWindow(x []ItemGroupedByKey) []ItemGroupedByWindow {
	result := make([]ItemGroupedByWindow, 0, len(x))
	windowGroups := map[int64]map[int64]*ItemGroupedByWindow{}

	for _, group := range x {
		for _, item := range group.Data {
			if _, ok := windowGroups[item.Window.Start]; !ok {
				windowGroups[item.Window.Start] = map[int64]*ItemGroupedByWindow{}
			}
			if _, ok := windowGroups[item.Window.Start][item.Window.End]; !ok {
				windowGroups[item.Window.Start][item.Window.End] = &ItemGroupedByWindow{
					Key:    group.Key,
					Data:   &schema.List{},
					Window: item.Window,
				}
			}

			windowGroups[item.Window.Start][item.Window.End].Data =
				schema.AppendList(windowGroups[item.Window.Start][item.Window.End].Data, item.Data)
		}

		// because golang maps are not ordered,
		// to create ordered result we need to iterate over data again in order to get ordered result
		for _, item := range group.Data {
			if _, ok := windowGroups[item.Window.Start]; !ok {
				continue
			}
			if _, ok := windowGroups[item.Window.Start][item.Window.End]; !ok {
				continue
			}

			result = append(result, *windowGroups[item.Window.Start][item.Window.End])
			delete(windowGroups[item.Window.Start], item.Window.End)
		}
	}

	return result
}

func ExpandToElements(x []ItemGroupedByWindow) []Item {
	result := make([]Item, 0, len(x))
	for _, group := range x {
		result = append(result, ToElement(&group))
	}
	return result
}

func ToElement(group *ItemGroupedByWindow) Item {
	return Item{
		Key:       group.Key,
		Data:      group.Data,
		EventTime: group.Window.End,
		Window:    group.Window,
	}
}

func WindowKey(window *Window) string {
	return fmt.Sprintf("%d.%d", window.Start, window.End)
}

func KeyedWindowKey(x *KeyedWindow) string {
	return fmt.Sprintf("%s:%s", x.Key, WindowKey(x.Window))
}

type KeyedWindow struct {
	Key    string
	Window *Window
}

func ToKeyedWindowFromItem(x *Item) *KeyedWindow {
	return &KeyedWindow{
		Key:    x.Key,
		Window: x.Window,
	}
}

func ToKeyedWindowFromGrouped(x *ItemGroupedByWindow) *KeyedWindow {
	return &KeyedWindow{
		Key:    x.Key,
		Window: x.Window,
	}
}

func KeyWithNamespace(key string, namespace string) string {
	return fmt.Sprintf("%s:%s", namespace, key)
}

func NewWindowBuffer(wd WindowDescription, sig WindowBufferSignaler) *WindowBuffer {
	return &WindowBuffer{
		wd:           wd,
		sig:          sig,
		windowGroups: map[string]*ItemGroupedByWindow{},
	}
}

type WindowBuffer struct {
	wd           WindowDescription
	sig          WindowBufferSignaler
	windowGroups map[string]*ItemGroupedByWindow
}

func (wb *WindowBuffer) Append(x Item) {
	list1 := AssignWindows([]Item{x}, wb.wd)
	list2 := DropTimestamps(list1)
	list3 := GroupByKey(list2)
	list4 := MergeWindows(list3, wb.wd)
	wb.GroupAlsoByWindow(list4)
}

// FlushItemGroupedByWindow makes sure that windows that needs to be flushed are delivered to the function f.
//
// Some operations that require aggregate or aggregateAndRetract are not expressed by window buffer,
// but by the function f that is responsible for grouping windows and knowing whenever value of window was calculated,
// and aggregation can add previous value, or retract previous value.
//
// Snapshotting process works as follows:
// - store information about last message that was successfully processed
// - store outbox of windows that were successfully processed and need to be flushed
//
// When process is restarted, it will:
// - restore information about last message that was successfully processed, and ask runtime to continue sending messages from that point
// - start emptying outbox
//
// Flush process works as follows:
// - for each window in outbox, call flush function, that function needs to return OK or error
// - if flush function returns OK, remove window from outbox
// - if flush function returns error, stop flushing, and retry on next flush
//
// Because each of the processes is independent by key, we can retry flushing only for windows that failed to flush.
// Because each of outbox pattern, we have order of windows guaranteed.
// _
// Because we make failure first class citizen, client can define failure stream and decide that after N retries,
// message should be sent to dead letter queue, or other error handling mechanism.
//
// Because we can model backfilling as a failure, we can use same mechanism to backfill windows that failed to flush,
// in the same way as we would backfill normal messages from time window
//
// Backfill is the same as using already existing DAG, but only with different input.

func (wb *WindowBuffer) EachItemGroupedByWindow(f func(group *ItemGroupedByWindow)) {
	for _, group := range wb.windowGroups {
		f(group)
	}
}

func (wb *WindowBuffer) EachKeyedWindow(kw *KeyedWindow, f func(group *ItemGroupedByWindow)) {
	key := KeyedWindowKey(kw)
	if group, ok := wb.windowGroups[key]; ok {
		f(group)
	}
}

func (wb *WindowBuffer) RemoveItemGropedByWindow(item *ItemGroupedByWindow) {
	kw := ToKeyedWindowFromGrouped(item)
	key := KeyedWindowKey(kw)
	delete(wb.windowGroups, key)
	wb.sig.SignalWindowDeleted(kw)
}

func (wb *WindowBuffer) GroupAlsoByWindow(x []ItemGroupedByKey) {
	for _, group := range x {
		for _, item := range group.Data {
			kw := ToKeyedWindowFromItem(&item)
			key := KeyedWindowKey(kw)
			if _, ok := wb.windowGroups[key]; !ok {
				wb.windowGroups[key] = &ItemGroupedByWindow{
					Key:    group.Key,
					Data:   &schema.List{},
					Window: item.Window,
				}

				wb.sig.SignalWindowCreated(kw)
			}

			wb.windowGroups[key].Data = schema.AppendList(
				wb.windowGroups[key].Data,
				item.Data,
			)

			wb.sig.SignalWindowSizeReached(kw, len(*wb.windowGroups[key].Data))
		}
	}
}

// Problem with tests that use period based tirggers
// result in windows that due to internal latency, may not have all events at the time of trigger
// but watermark could say, hey wait with this trigger, because I see event that will land in this window
// so trigger at the arrival of watermark

func NewWindowTrigger(w *Window, td TriggerDescription) *WindowTrigger {
	wt := &WindowTrigger{
		w:  w,
		td: td,
	}
	wt.init()

	return wt
}

type WindowTrigger struct {
	w             *Window
	td            TriggerDescription
	ts            *TriggerState
	shouldTrigger bool
}

func (wt *WindowTrigger) init() {
	if wt.ts != nil {
		return
	}

	wt.ts = wt.initState(wt.td)
}

func Bool(b bool) *bool {
	return &b
}

func (wt *WindowTrigger) initState(td TriggerDescription) *TriggerState {
	return MatchTriggerDescriptionR1(
		td,
		func(x *AtPeriod) *TriggerState {
			return &TriggerState{
				desc: td,
			}
		},
		func(x *AtWindowItemSize) *TriggerState {
			return &TriggerState{
				desc: td,
			}
		},
		func(x *AtWatermark) *TriggerState {
			return &TriggerState{
				desc: &AtWatermark{
					Timestamp: wt.w.End,
				},
			}
		},
		func(x *AnyOf) *TriggerState {
			result := &TriggerState{
				desc: td,
			}

			for _, desc := range x.Triggers {
				result.nexts = append(result.nexts, wt.initState(desc))
			}

			return result
		},
		func(x *AllOf) *TriggerState {
			result := &TriggerState{
				desc: td,
			}

			for _, desc := range x.Triggers {
				result.nexts = append(result.nexts, wt.initState(desc))
			}

			return result
		},
	)
}

func (wt *WindowTrigger) ReceiveEvent(triggerType TriggerType) {
	// continue evaluation until state is true
	// but when it's true, we don't need to evaluate
	// conditions for window flush
	if !wt.ts.isTrue() {
		result := wt.ts.evaluate(triggerType, 0)
		wt.ts.result = &result
	}

	wt.shouldTrigger = wt.ts.isTrue()
}

func (wt *WindowTrigger) ShouldTrigger() bool {
	return wt.shouldTrigger
}

func (wt *WindowTrigger) Reset() {
	wt.shouldTrigger = false
	wt.ts = wt.initState(wt.td)
}

//go:generate mkunion match -name=EvaluateTrigger
type EvaluateTrigger[T0 TriggerDescription, T1 TriggerType] interface {
	MatchPeriod(*AtPeriod, *AtPeriod)
	MatchCount(*AtWindowItemSize, *AtWindowItemSize)
	MatchWatermark(*AtWatermark, *AtWatermark)
	MatchAnyOfAny(*AnyOf, TriggerType)
	MatchAllOfAny(*AllOf, TriggerType)
	MatchDefault(T0, T1)
}

type TriggerState struct {
	desc   TriggerDescription
	nexts  []*TriggerState
	result *bool
}

func (ts *TriggerState) isTrue() bool {
	return ts.result != nil && *ts.result
}

func (ts *TriggerState) evaluate(triggerType TriggerType, depth int) bool {
	return EvaluateTriggerR1(
		ts.desc, triggerType,
		func(x0 *AtPeriod, x1 *AtPeriod) bool {
			return x0.Duration == x1.Duration
		},
		func(x0 *AtWindowItemSize, x1 *AtWindowItemSize) bool {
			return x0.Number == x1.Number
		},
		func(x0 *AtWatermark, x1 *AtWatermark) bool {
			return x0.Timestamp <= x1.Timestamp
		},
		func(x0 *AnyOf, x1 TriggerType) bool {
			found := false
			for _, state := range ts.nexts {
				if !state.isTrue() {
					matched := state.evaluate(triggerType, depth+1)
					if matched {
						state.result = Bool(true)
					}
				}

				// be exhaustive, and allow other triggers to be evaluated
				// that way, with different triggers an complete state can be build
				found = found || state.isTrue()
			}

			return found
		},
		func(x0 *AllOf, x1 TriggerType) bool {
			found := true
			for _, state := range ts.nexts {
				if !state.isTrue() {
					result := state.evaluate(triggerType, depth+1)
					if result {
						state.result = Bool(true)
					}
				}

				// be exhaustive, and allow other triggers to be evaluated
				// that way, with different triggers an complete state can be build
				found = found && state.isTrue()
			}

			return found
		},
		func(x0 TriggerDescription, x1 TriggerType) bool {
			return false
		},
	)
}
