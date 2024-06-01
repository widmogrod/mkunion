package projection

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/stream"
	"time"
)

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
	Offset stream.Offset
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

//type WindowSnapshotState struct {
//	snapshotState PullPushContextState
//	wd            WindowDescription
//	fm            WindowFlushMode
//	td            TriggerDescription
//}

func DoWindow[A, B any](
	ctx PushAndPull[A, B],
	store *WindowInMemoryStore[B],
	wd WindowDescription,
	fm WindowFlushMode,
	td TriggerDescription,
	init B,
	merge func(x A, agg B) (B, error),
) error {
	// recovery from failure:
	// to avoid any double processing of data process should work only on data from last snapshot
	// load all windows after last snapshot and before last watermark to memory
	// for each window apply function
	// save window

	flush := MatchWindowFlushModeR1(
		fm,
		func(x *Discard) func(watermark *Watermark[A]) error {
			return func(watermark *Watermark[A]) error {
				where, err := TriggerDescriptionToWhere(td)
				if err != nil {
					return fmt.Errorf("projection.DoWindow: flush trigger description to whare: %w", err)
				}

				//where.Params[":key"] = schema.MkString(watermark.Key)
				where.Params[":watermark"] = schema.MkInt(watermark.EventTime)

				find := &schemaless.FindingRecords[schemaless.Record[*WindowRecord[B]]]{
					RecordType: store.recordType,
					Where:      where,
					Sort: []schemaless.SortField{
						{
							Field:      "Data.Window.End",
							Descending: false,
						},
						{
							Field:      "Data.Key",
							Descending: false,
						},
					},
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
						_, err = store.store.UpdateRecords(schemaless.Delete(records.Items...))
						if err != nil {
							return fmt.Errorf("projection.DoWindow: flush delete: %w", err)
						}
					}

					if records.HasNext() {
						find = records.Next
						continue
					}

					if err != nil {
						return fmt.Errorf("projection.DoWindow: flush push: %w", err)
					}

					return nil
				}
			}
		},
	)

	for {
		if IsWatermarkMarksEndOfStream(ctx.LastWatermark()) {
			log.Debugf("projection.DoWindow: pull: no more data in stream for all keys (exit)")
			return nil
		}

		item, err := ctx.PullIn()
		if err != nil {
			if errors.Is(err, stream.ErrNoMoreNewDataInStream) {
				log.Debugf("projection.DoWindow: pull: no more data in stream")
				continue
			}
			return fmt.Errorf("projection.DoWindow: pull: %w", err)
		}

		log.Debugf("projection.DoWindow: pull: %#v", item)

		err = MatchDataR1(
			item.Data,
			func(x *Record[A]) error {
				dataKey := x.Key
				offset := *item.Offset

				for _, w := range MkWindow(x.EventTime, wd) {
					windowID := MkWindowID(x.Key, w)
					windowRecord, err := store.Load(windowID)

					if err != nil {
						if errors.Is(err, ErrWindowNotFound) {
							result, err := merge(x.Data, init)
							if err != nil {
								return fmt.Errorf("projection.DoWindow: merge first key=%s window=%s: %w", dataKey, windowID, err)
							}

							err = store.Save(&WindowRecord[B]{
								Key:    x.Key,
								Window: w,
								Record: result,
								Offset: offset,
							})

							if err != nil {
								return fmt.Errorf("projection.DoWindow: save key=%s window=%s: %w", dataKey, windowID, err)
							}

							continue
						}

						return fmt.Errorf("projection.DoWindow: load key=%s window=%s: %w", dataKey, windowID, err)
					} else {
						cmp, err := stream.OffsetCompare(offset, windowRecord.Offset)
						if err != nil {
							return fmt.Errorf("projection.DoWindow: compare offset key=%s window=%s: %w", dataKey, windowID, err)
						}

						if cmp <= 0 {
							log.Warnf("projection.DoWindow: skip key=%s window=%s: offset=%s, windowOffset=%s", dataKey, windowID, offset, windowRecord.Offset)
							// we already processed this record
							continue
						}

						result, err := merge(x.Data, windowRecord.Record)
						if err != nil {
							return fmt.Errorf("projection.DoWindow: merge key=%s window=%s: %w", dataKey, windowID, err)
						}

						err = store.Save(&WindowRecord[B]{
							Key:    x.Key,
							Window: w,
							Record: result,
							Offset: offset,
						})
						if err != nil {
							return fmt.Errorf("projection.DoWindow: save key=%s window=%s: %w", dataKey, windowID, err)
						}
					}
				}

				return nil
			},
			func(x *Watermark[A]) error {
				err := flush(x)
				if err != nil {
					return fmt.Errorf("projection.DoWindow: flush on watermark: %w", err)
				}

				err = ctx.PushOut(&Watermark[B]{
					EventTime: x.EventTime,
				})
				if err != nil {
					return fmt.Errorf("projection.DoWindow: push watermark: %w", err)
				}

				err = ctx.AckWatermark(&x.EventTime)
				if err != nil {
					return fmt.Errorf("projection.DoWindow: ack watermark: %w", err)
				}

				return nil
			},
		)

		if err != nil {
			return fmt.Errorf("projection.DoWindow: merge data %T: %w", item, err)
		}

		err = ctx.AckOffset(item.Offset)
		if err != nil {
			return fmt.Errorf("projection.DoWindow: ack; %w ", err)
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
	_, err := w.store.UpdateRecords(update)

	if err != nil {
		return fmt.Errorf("projection.WindowInMemoryStore: Save: %w", err)
	}

	return nil
}
