package projection

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/stream"
	"math"
)

type Window struct {
	Start int64
	End   int64
}

type WindowKey = string

type WindowByKey[A any] struct {
	Window *Window
	Record A
}

func WindowToKey(w *Window) WindowKey {
	return fmt.Sprintf("%d-%d", w.Start, w.End)
}

func MkWindow(x EventTime) []*Window {
	result := []*Window{
		{
			Start: math.MinInt64,
			End:   math.MaxInt64,
		},
	}

	return result
}

func WindowToRecord[A any](key string, window WindowByKey[A]) *Record[A] {
	return &Record[A]{
		Key:       key,
		Data:      window.Record,
		EventTime: &window.Window.Start,
	}
}

func DoWindow[A, B any](
	ctx PushAndPull[A, B],
	merge func(x A, agg B) (B, error),
) error {
	// group by key
	// each group of keys, group by window
	// for each window apply function

	byKey := make(map[string]map[WindowKey]WindowByKey[B])

	flush := func() error {
		for key, byWindow := range byKey {
			for _, window := range byWindow {
				err := ctx.PushOut(WindowToRecord(key, window))
				if err != nil {
					return fmt.Errorf("projection.DoWindow: push: %w", err)
				}
			}
		}

		return nil
	}

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
				if byKey[dataKey] == nil {
					byKey[dataKey] = make(map[WindowKey]WindowByKey[B])
				}

				for _, w := range MkWindow(*x.EventTime) {
					windowKey := WindowToKey(w)
					window, ok := byKey[dataKey][windowKey]
					if !ok {
						var zero B
						result, err := merge(x.Data, zero)
						if err != nil {
							return fmt.Errorf("projection.DoWindow: merge first key=%s window=%s: %w", dataKey, windowKey, err)
						}

						byKey[dataKey][windowKey] = WindowByKey[B]{
							Window: w,
							Record: result,
						}
					} else {
						result, err := merge(x.Data, window.Record)
						if err != nil {
							return fmt.Errorf("projection.DoWindow: merge key=%s window=%s: %w", dataKey, windowKey, err)
						}

						window.Record = result
						byKey[dataKey][windowKey] = window
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
