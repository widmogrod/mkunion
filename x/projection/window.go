package projection

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/stream"
	"math"
	"time"
)

//go:generate go run ../../cmd/mkunion/main.go

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

type Window struct {
	Start int64
	End   int64
}

type WindowID = string
type WindowKey = string

type WindowRecord[A any] struct {
	Key    WindowKey
	Window *Window
	Record A
	//Count  uint64 // How many items were added to this window, this is useful for AtCount TriggerDescription
}

func MkWindowID(key string, window *Window) WindowID {
	return fmt.Sprintf("%s_%s", key, WindowToKey(window))
}

func WindowToKey(w *Window) WindowKey {
	return fmt.Sprintf("%d-%d", w.Start, w.End)
}

func MkWindow(eventTime EventTime, wd WindowDescription) []*Window {
	return MatchWindowDescriptionR1(
		wd,
		func(x *SessionWindow) []*Window {
			return []*Window{
				{
					Start: eventTime,
					End:   time.Unix(0, eventTime).Add(x.GapDuration).UnixNano(),
				},
			}
		},
		func(x *SlidingWindow) []*Window {
			var result []*Window
			timeObject := time.Unix(0, eventTime)
			// slicing window is [start, end)
			// left side inclusive, and right side exclusive,
			// so we need to add 1 period to the end
			for start := timeObject.Add(-x.Width).Add(x.Period); start.UnixNano() <= eventTime; start = start.Add(x.Period) {
				result = append(result, &Window{
					Start: start.UnixNano(),
					End:   start.Add(x.Width).UnixNano(),
				})
			}
			return result
		},
		func(x *FixedWindow) []*Window {
			start := eventTime - eventTime%x.Width.Nanoseconds()
			end := start + x.Width.Nanoseconds()
			return []*Window{
				{
					Start: start,
					End:   end,
				},
			}
		},
	)
}

func WindowToRecord[A any](key string, window WindowRecord[A]) *Record[A] {
	return &Record[A]{
		Key:       key,
		Data:      window.Record,
		EventTime: window.Window.End,
	}
}

type WindowSnapshotState struct {
	snapshotState SnapshotState
	wd            WindowDescription
	fm            WindowFlushMode
	td            TriggerDescription
}

func DoWindow[A, B any](
	ctx PushAndPull[A, B],
	wd WindowDescription,
	fm WindowFlushMode,
	td TriggerDescription,
	merge func(x A, agg B) (B, error),
) error {
	store := NewWindowInMemoryStore[B]("window")

	// recovery from failure:
	// to avoid any double processing of data process should work only on data from last snapshot
	// load all windows after last snapshot and before last watermark to memory
	// for each window apply function
	// save window

	flush := MatchWindowFlushModeR1(
		fm,
		func(x *Discard) func() error {
			return func() error {
				where, err := TriggerDescriptionToWhere(td)
				if err != nil {
					return fmt.Errorf("projection.DoWindow: flush trigger description to whare: %w", err)
				}

				where.Params[":watermark"] = schema.MkInt(math.MaxInt64)

				find := &schemaless.FindingRecords[schemaless.Record[*WindowRecord[B]]]{
					RecordType: "window",
					Where:      where,
				}

				for {
					records, err := store.store.FindingRecords(*find)
					if err != nil {
						return fmt.Errorf("projection.DoWindow: flush find: %w", err)
					}

					for _, record := range records.Items {
						err := ctx.PushOut(WindowToRecord(record.Data.Key, *record.Data))
						if err != nil {
							return fmt.Errorf("projection.DoWindow: flush push: %w", err)
						}
					}

					if len(records.Items) > 0 {
						err = store.store.UpdateRecords(schemaless.Delete(records.Items...))
						if err != nil {
							return fmt.Errorf("projection.DoWindow: flush delete: %w", err)
						}
					}

					if records.HasNext() {
						find = records.Next
						continue
					}

					return nil
				}
			}
		},
	)

	for {
		item, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, stream.ErrEndOfStream) {
				err = flush()
				if err != nil {
					return fmt.Errorf("projection.DoWindow: flush on end of stream: %w", err)
				}
				return nil
			}
			return fmt.Errorf("projection.DoWindow: pull: %w", err)
		}

		err = MatchDataR1(
			item,
			func(x *Record[A]) error {
				dataKey := x.Key
				for _, w := range MkWindow(x.EventTime, wd) {
					windowID := MkWindowID(x.Key, w)
					windowRecord, err := store.Load(windowID)
					if err != nil {
						if errors.Is(err, ErrWindowNotFound) {
							var zero B
							result, err := merge(x.Data, zero)
							if err != nil {
								return fmt.Errorf("projection.DoWindow: merge first key=%s window=%s: %w", dataKey, windowID, err)
							}

							err = store.Save(&WindowRecord[B]{
								Key:    x.Key,
								Window: w,
								Record: result,
							})

							if err != nil {
								return fmt.Errorf("projection.DoWindow: save key=%s window=%s: %w", dataKey, windowID, err)
							}

							continue
						}

						return fmt.Errorf("projection.DoWindow: load key=%s window=%s: %w", dataKey, windowID, err)
					} else {
						result, err := merge(x.Data, windowRecord.Record)
						if err != nil {
							return fmt.Errorf("projection.DoWindow: merge key=%s window=%s: %w", dataKey, windowID, err)
						}

						err = store.Save(&WindowRecord[B]{
							Key:    x.Key,
							Window: w,
							Record: result,
						})
						if err != nil {
							return fmt.Errorf("projection.DoWindow: save key=%s window=%s: %w", dataKey, windowID, err)
						}
					}
				}

				return nil
			},
			func(x *Watermark[A]) error {
				err := flush()
				if err != nil {
					return fmt.Errorf("projection.DoWindow: flush on watermark: %w", err)
				}
				return nil
			},
		)

		if err != nil {
			return fmt.Errorf("projection.DoWindow: merge data %T: %w", item, err)
		}
	}
}

func NewWindowInMemoryStore[A any](recordType string) *WindowInMemoryStore[A] {
	return &WindowInMemoryStore[A]{
		store:      schemaless.NewInMemoryRepository[*WindowRecord[A]](),
		recordType: recordType,
	}
}

var (
	ErrWindowNotFound = fmt.Errorf("window not found")
)

type WindowInMemoryStore[A any] struct {
	store      schemaless.Repository[*WindowRecord[A]]
	recordType string
}

func (w *WindowInMemoryStore[A]) Load(id WindowID) (*WindowRecord[A], error) {
	result, err := w.store.Get(id, w.recordType)
	if err != nil {
		if errors.Is(err, schemaless.ErrNotFound) {
			return nil, ErrWindowNotFound
		}

		return nil, fmt.Errorf("projection.WindowInMemoryStore: Load: %w", err)
	}

	return result.Data, nil
}

func (w *WindowInMemoryStore[A]) Save(x *WindowRecord[A]) error {
	update := schemaless.Save(schemaless.Record[*WindowRecord[A]]{
		ID:   MkWindowID(x.Key, x.Window),
		Type: w.recordType,
		Data: x,
	})
	update.UpdatingPolicy = schemaless.PolicyOverwriteServerChanges
	err := w.store.UpdateRecords(update)

	if err != nil {
		return fmt.Errorf("projection.WindowInMemoryStore: Save: %w", err)
	}

	return nil
}
