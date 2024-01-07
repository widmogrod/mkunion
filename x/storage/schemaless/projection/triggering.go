package projection

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"sync"
	"time"
)

//go:generate go run ../../../../cmd/mkunion/main.go
//go:generate go run ../../../../cmd/mkunion/main.go serde

// go:generate mkunion -name=TriggerType -variants=AtPeriod,AtWindowItemSize,AtWatermark
//
//go:tag mkunion:"TriggerType"
type (
	AtPeriod1         = AtPeriod
	AtWindowItemSize1 = AtWindowItemSize
	AtWatermark1      = AtWatermark
)

//go:tag mkunion:"TriggerDescription"
type (
	AtPeriod struct {
		Duration time.Duration
	}
	AtWindowItemSize struct {
		Number int
	}
	AtWatermark struct {
		Timestamp int64
	}
	AnyOf struct {
		Triggers []TriggerDescription
	}
	AllOf struct {
		Triggers []TriggerDescription
	}
)

//go:tag mkunion:"WindowFlushMode"
type (
	Accumulate struct {
		AllowLateArrival time.Duration
	}
	Discard                   struct{}
	AccumulatingAndRetracting struct {
		AllowLateArrival time.Duration
	}
)

type TriggerHandler struct {
	//wd WindowDescription
	td TriggerDescription

	wb *WindowBuffer

	wts BagOf[*WindowTrigger]

	lock sync.Mutex
}

var _ Handler = (*TriggerHandler)(nil)

func printTrigger(triggerType TriggerType) {
	MatchTriggerTypeR0(
		triggerType,
		func(x *AtPeriod) {
			fmt.Printf("AtPeriod(%v)", x.Duration)

		},
		func(x *AtWindowItemSize) {
			fmt.Printf("AtWindowItemSize(%v)", x.Number)
		},
		func(x *AtWatermark) {
			fmt.Printf("AtWatermark()")
		})
}

func (tm *TriggerHandler) Triggered(trigger TriggerType, returning func(Item)) error {
	tm.lock.Lock()
	defer tm.lock.Unlock()

	tm.wb.EachItemGroupedByWindow(func(group *ItemGroupedByWindow) {
		wt, err := tm.wts.Get(WindowKey(group.Window))
		isError := err != nil && err != NotFound
		isFound := err == nil

		if isError {
			panic(err)
		}

		if !isFound {
			wt = NewWindowTrigger(group.Window, tm.td)
			err = tm.wts.Set(WindowKey(group.Window), wt)
			if err != nil {
				panic(err)
			}
		}

		wt.ReceiveEvent(trigger)
		wt.ReceiveEvent(&AtWindowItemSize{Number: len(*group.Data)})

		if wt.ShouldTrigger() {
			returning(ToElement(group))
			tm.wb.RemoveItemGropedByWindow(group)
			wt.Reset()
		}
	})

	return nil
}

func (tm *TriggerHandler) Process(x Item, returning func(Item)) error {
	tm.lock.Lock()
	tm.wb.Append(x)
	tm.lock.Unlock()
	return tm.Triggered(&AtWindowItemSize{Number: -1}, returning)
}

func (tm *TriggerHandler) Retract(x Item, returning func(Item)) error {
	panic("implement me")
}

var NotFound = errors.New("not found")

type BagOf[A any] interface {
	Set(key string, value A) error
	Get(key string) (A, error)
	Del(key string) error
	Range(f func(key string, item A))
}

var _ BagOf[any] = (*InMemoryBagOf[any])(nil)

type InMemoryBagOf[A any] struct {
	m map[string]A
}

func NewInMemoryBagOf[A any]() *InMemoryBagOf[A] {
	return &InMemoryBagOf[A]{
		m: make(map[string]A),
	}
}

func (b *InMemoryBagOf[A]) Set(key string, value A) error {
	b.m[key] = value
	return nil
}

func (b *InMemoryBagOf[A]) Get(key string) (A, error) {
	if value, ok := b.m[key]; ok {
		return value, nil
	}

	var a A
	return a, NotFound
}

func (b *InMemoryBagOf[A]) Del(key string) error {
	delete(b.m, key)
	return nil
}

func (b *InMemoryBagOf[A]) Range(f func(key string, item A)) {
	for k, v := range b.m {
		f(k, v)
	}
}

type AccumulateDiscardRetractHandler struct {
	fm     WindowFlushMode
	mapf   Handler
	mergef Handler

	bag BagOf[Item]
}

var _ Handler = (*AccumulateDiscardRetractHandler)(nil)

func printItem(x Item, sx ...string) {
	data, _ := shared.JSONMarshal[schema.Schema](x.Data)
	fmt.Println(fmt.Sprintf("Item(%v)", sx), x.Key, x.Window, string(data), x.EventTime)
}

func (a *AccumulateDiscardRetractHandler) Process(x Item, returning func(Item)) error {
	return MatchWindowFlushModeR1(
		a.fm,
		func(y *Accumulate) error {
			key := KeyedWindowKey(ToKeyedWindowFromItem(&x))
			previous, err := a.bag.Get(key)

			isError := err != nil && err != NotFound
			isFound := err == nil
			if isError {
				panic(err)
			}

			if isFound {
				return a.mapf.Process(x, func(item Item) {
					z := Item{
						Key:    item.Key,
						Window: item.Window,
						Data: schema.MkList(
							previous.Data,
							item.Data,
						),
						EventTime: item.EventTime,
					}

					err := a.mergef.Process(z, func(item Item) {
						err := a.bag.Set(key, item)
						if err != nil {
							panic(err)
						}

						returning(item)
					})
					if err != nil {
						panic(err)
					}
				})
			}

			return a.mapf.Process(x, func(item Item) {
				err := a.bag.Set(key, item)
				if err != nil {
					panic(err)
				}
				returning(item)
			})
		},
		func(y *Discard) error {
			return a.mapf.Process(x, returning)
		},
		func(y *AccumulatingAndRetracting) error {
			key := KeyedWindowKey(ToKeyedWindowFromItem(&x))
			previous, err := a.bag.Get(key)
			isError := err != nil && err != NotFound
			isFound := err == nil
			if isError {
				panic(err)
			}

			if isFound {
				return a.mapf.Process(x, func(item Item) {
					z := Item{
						Key:    item.Key,
						Window: item.Window,
						Data: schema.MkList(
							previous.Data,
							item.Data,
						),
						EventTime: item.EventTime,
					}

					err := a.mergef.Process(z, func(newAggregate Item) {
						err := a.bag.Set(key, newAggregate)
						if err != nil {
							panic(err)
						}

						// operation is in one messages, as one or nothing principle
						// which will help in transactional systems.
						returning(Item{
							Key: newAggregate.Key,
							Data: PackRetractAndAggregate(
								previous.Data,
								newAggregate.Data,
							),
							EventTime: newAggregate.EventTime,
							Window:    newAggregate.Window,
							Type:      ItemRetractAndAggregate,
						})
					})
					if err != nil {
						panic(err)
					}
				})
			}

			return a.mapf.Process(x, func(item Item) {
				err := a.bag.Set(key, item)
				if err != nil {
					panic(err)
				}
				returning(item) // emit aggregate
			})
		},
	)
}

func (a *AccumulateDiscardRetractHandler) Retract(x Item, returning func(Item)) error {
	//TODO implement me
	panic("implement me")
}

type (
	WindowBufferSignaler interface {
		SignalWindowCreated(kw *KeyedWindow)
		SignalWindowDeleted(kw *KeyedWindow)
		SignalWindowSizeReached(kw *KeyedWindow, size int)
	}

	WatermarkSignaler interface {
		SignalWatermark(timestamp int64)
	}

	TimeSignaler interface {
		SignalDuration(duration time.Duration)
	}
)

func NewTriggerManager(td TriggerDescription) *TriggerManager {
	tm := &TriggerManager{
		td:             td,
		windowTriggers: NewInMemoryBagOf[*WindowTrigger](),
		keyedWindows:   NewInMemoryBagOf[*KeyedWindow](),
	}

	return tm
}

type TriggerManager struct {
	td TriggerDescription

	windowTriggers BagOf[*WindowTrigger]
	keyedWindows   BagOf[*KeyedWindow]

	triggerWindow func(w *KeyedWindow)
}

var _ WindowBufferSignaler = (*TriggerManager)(nil)
var _ WatermarkSignaler = (*TriggerManager)(nil)
var _ TimeSignaler = (*TriggerManager)(nil)

func (tm *TriggerManager) SignalWindowCreated(kw *KeyedWindow) {
	err := tm.windowTriggers.Set(KeyedWindowKey(kw), NewWindowTrigger(kw.Window, tm.td))
	if err != nil {
		panic(err)
	}
	err = tm.keyedWindows.Set(KeyedWindowKey(kw), kw)
	if err != nil {
		panic(err)
	}
}

func (tm *TriggerManager) SignalWindowDeleted(kw *KeyedWindow) {
	err := tm.windowTriggers.Del(KeyedWindowKey(kw))
	if err != nil {
		panic(err)
	}
	err = tm.keyedWindows.Del(KeyedWindowKey(kw))
	if err != nil {
		panic(err)
	}
}

func (tm *TriggerManager) SignalWindowSizeReached(kw *KeyedWindow, size int) {
	wt, err := tm.windowTriggers.Get(KeyedWindowKey(kw))
	if err != nil {
		panic(err)
	}

	wt.ReceiveEvent(&AtWindowItemSize{
		Number: size,
	})
	if wt.ShouldTrigger() {
		tm.triggerWindow(kw)
		wt.Reset()
	}
}

func (tm *TriggerManager) WhenTrigger(f func(w *KeyedWindow)) {
	if tm.triggerWindow != nil {
		panic("trigger window already set")
	}

	tm.triggerWindow = f
}

func (tm *TriggerManager) SignalDuration(duration time.Duration) {
	tm.windowTriggers.Range(func(key string, wt *WindowTrigger) {
		wt.ReceiveEvent(&AtPeriod{
			Duration: duration,
		})
		if wt.ShouldTrigger() {
			kw, err := tm.keyedWindows.Get(key)
			if err != nil {
				panic(err)
			}
			tm.triggerWindow(kw)
			wt.Reset()
		}
	})
}

func (tm *TriggerManager) SignalWatermark(timestamp int64) {
	tm.windowTriggers.Range(func(key string, wt *WindowTrigger) {
		wt.ReceiveEvent(&AtWatermark{
			Timestamp: timestamp,
		})
		if wt.ShouldTrigger() {
			kw, err := tm.keyedWindows.Get(key)
			if err != nil {
				panic(err)
			}
			tm.triggerWindow(kw)
			wt.Reset()
		}
	})
}

func NewTimeTicker() *Tickers {
	return &Tickers{
		tickers: map[TriggerDescription]*time.Ticker{},
	}
}

type Tickers struct {
	tickers map[TriggerDescription]*time.Ticker
}

func (t *Tickers) Register(td TriggerDescription, ts TimeSignaler) {
	MatchTriggerDescriptionR0(
		td,
		func(x *AtPeriod) {
			go func() {
				t.tickers[td] = time.NewTicker(x.Duration)
				for range t.tickers[td].C {
					ts.SignalDuration(x.Duration)
				}
			}()
		},
		func(x *AtWindowItemSize) {},
		func(x *AtWatermark) {},
		func(x *AnyOf) {
			for _, td := range x.Triggers {
				t.Register(td, ts)
			}
		},
		func(x *AllOf) {
			for _, td := range x.Triggers {
				t.Register(td, ts)
			}
		},
	)
}

func (t *Tickers) Unregister(td TriggerDescription) {
	MatchTriggerDescriptionR0(
		td,
		func(x *AtPeriod) {
			if ticker, ok := t.tickers[td]; ok {
				ticker.Stop()
				delete(t.tickers, td)
			}
		},
		func(x *AtWindowItemSize) {},
		func(x *AtWatermark) {},
		func(x *AnyOf) {
			for _, td := range x.Triggers {
				t.Unregister(td)
			}
		},
		func(x *AllOf) {
			for _, td := range x.Triggers {
				t.Unregister(td)
			}
		},
	)
}
