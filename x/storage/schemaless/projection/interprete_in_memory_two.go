package projection

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
	"math"
	"sync"
	"time"
)

func NewInMemoryTwoInterpreter() *InMemoryTwoInterpreter {
	return &InMemoryTwoInterpreter{
		pubsub:           NewPubSubMultiChan[Node](),
		byKeys:           make(map[Node]map[string]Item),
		running:          make(map[Node]struct{}),
		stats:            NewStatsCollector(),
		bagItem:          NewInMemoryBagOf[Item](),
		bagWindowTrigger: NewInMemoryBagOf[*WindowTrigger](),
	}
}

type InMemoryTwoInterpreter struct {
	lock    sync.Mutex
	pubsub  PubSubForInterpreter[Node]
	byKeys  map[Node]map[string]Item
	running map[Node]struct{}
	status  ExecutionStatus
	// what differences between process time and event time
	// should answers question
	// - are there any events in the system, that a process should wait?
	watermark int64
	stats     StatsCollector

	bagItem          BagOf[Item]
	bagWindowTrigger BagOf[*WindowTrigger]
	windowBuffers    map[Node]*WindowBuffer
}

func (i *InMemoryTwoInterpreter) Run(ctx context.Context, nodes []Node) error {
	i.lock.Lock()
	if i.status != ExecutionStatusNew {
		i.lock.Unlock()
		return fmt.Errorf("interpreter.Run state %d %w", i.status, ErrInterpreterNotInNewState)
	}
	i.status = ExecutionStatusRunning
	i.lock.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	group := &ExecutionGroup{
		ctx:    ctx,
		cancel: cancel,
	}

	// Registering new nodes makes sure that, in case of non-deterministic concurrency
	// when goroutine want to subscribe to a node, it will be registered, even if it's not publishing yet
	for _, node := range nodes {
		err := i.pubsub.Register(node)
		if err != nil {
			i.lock.Lock()
			i.status = ExecutionStatusError
			i.lock.Unlock()

			return fmt.Errorf("interpreter.Run(1) %w", err)
		}
	}

	for _, node := range nodes {
		func(node Node) {
			group.Go(func() (err error) {
				return i.run(ctx, node)
			})
		}(node)
	}

	if err := group.Wait(); err != nil {
		i.lock.Lock()
		i.status = ExecutionStatusError
		i.lock.Unlock()

		return fmt.Errorf("interpreter.Run(2) %w", err)
	}

	i.lock.Lock()
	i.status = ExecutionStatusFinished
	i.lock.Unlock()

	return nil
}

//func (a *InMemoryTwoInterpreter) Process(x Item, returning func(Item)) error {
//	return MustMatchWindowFlushMode(
//		a.fm,
//		func(y *Accumulate) error {
//			key := ItemKeyWindow(x)
//			previous, err := a.bagItem.Get(key)
//
//			isError := err != nil && err != NotFound
//			isFound := err == nil
//			if isError {
//				panic(err)
//			}
//
//			if isFound {
//				x.P
//				return a.mapf.Process(x, func(item Item) {
//					z := Item{
//						Key:    item.Key,
//						Window: item.Window,
//						Data: schema.MkList(
//							previous.Data,
//							item.Data,
//						),
//						EventTime: item.EventTime,
//					}
//
//					err := a.mergef.Process(z, func(item Item) {
//						err := a.bagItem.Set(key, item)
//						if err != nil {
//							panic(err)
//						}
//
//						returning(item)
//					})
//					if err != nil {
//						panic(err)
//					}
//				})
//			}
//
//			return a.mapf.Process(x, func(item Item) {
//				err := a.bagItem.Set(key, item)
//				//printItem(item, "set")
//				if err != nil {
//					panic(err)
//				}
//				returning(item)
//			})
//		},
//		func(y *Discard) error {
//			return a.mapf.Process(x, returning)
//		},
//		func(y *AccumulatingAndRetracting) error {
//			key := ItemKeyWindow(x)
//			previous, err := a.bagItem.Get(key)
//			isError := err != nil && err != NotFound
//			isFound := err == nil
//			if isError {
//				panic(err)
//			}
//
//			if isFound {
//				return a.mapf.Process(x, func(item Item) {
//					z := Item{
//						Key:    item.Key,
//						Window: item.Window,
//						Data: schema.MkList(
//							previous.Data,
//							item.Data,
//						),
//						EventTime: item.EventTime,
//					}
//
//					err := a.mergef.Process(z, func(newAggregate Item) {
//						err := a.bagItem.Set(key, newAggregate)
//						if err != nil {
//							panic(err)
//						}
//
//						// operation is in one messages, as one or nothing principle
//						// which will help in transactional systems.
//						returning(Item{
//							Key: newAggregate.Key,
//							Data: PackRetractAndAggregate(
//								previous.Data,
//								newAggregate.Data,
//							),
//							EventTime: newAggregate.EventTime,
//							Window:    newAggregate.Window,
//							Type:      ItemRetractAndAggregate,
//						})
//					})
//					if err != nil {
//						panic(err)
//					}
//				})
//			}
//
//			return a.mapf.Process(x, func(item Item) {
//				err := a.bagItem.Set(key, item)
//				if err != nil {
//					panic(err)
//				}
//				returning(item) // emit aggregate
//			})
//		},
//	)
//}

func (i *InMemoryTwoInterpreter) run(ctx context.Context, dag Node) error {
	if dag == nil {
		//panic("fix nodes that are nil! fix dag builder!")
		return nil
	}

	// TODO introduce parallelism for Item - key groups
	// bounded to some number of goroutines, that can be configured
	// and that can be used to limit memory usage

	// TODO introduce merge window triggers, and triggers in general so that
	// - RepositorySink can be used with batches
	// - LiveSelect in TicTacToe game, can show progress in game after reload, not through streaming updates, but by sending final state - debounce?

	/*

			parallelize.Window().DoWindow().Log()
			Parallelize by key groups
			- group by key
			- for each key group, run in parallel

			i: (a, 1) (b, 2) (c, 3) (d, 4) (a, 5) (b, 6) (c, 7) (d, 8)
		    Window(i, +1)
			o: (a, 2) (b, 3) (c, 4) (d, 5) (a, 6) (b, 7) (c, 8) (d, 9)

			(a, 1) (a, 5)
			(b, 2) (b, 6)
			(c, 3) (c, 7)
			(d, 4) (d, 8)

	*/

	return MatchNodeR1(
		dag,
		func(x *DoWindow) error {
			log.Debugln("DoWindow: Start ", i.str(x))
			var lastOffset int = 0

			trigger := NewTriggerManager(x.Ctx.td)

			timeTickers := NewTimeTicker()
			timeTickers.Register(x.Ctx.td, trigger)
			defer timeTickers.Unregister(x.Ctx.td)

			wb := NewWindowBuffer(x.Ctx.wd, trigger)
			returning := func(item Item) error {
				key := KeyedWindowKey(ToKeyedWindowFromItem(&item))
				key = KeyWithNamespace(key, x.Ctx.Name())

				return i.pubsub.Publish(ctx, x, Message{
					Key:  item.Key,
					Item: &item,
				})
			}

			trigger.WhenTrigger(func(kw *KeyedWindow) {
				wb.EachKeyedWindow(kw, func(group *ItemGroupedByWindow) {
					err := returning(ToElement(group))
					if err != nil {
						panic(err)
					}
					wb.RemoveItemGropedByWindow(group)
				})
			})
			err := i.pubsub.Subscribe(
				ctx,
				x.Input,
				lastOffset,
				func(msg Message) error {
					if msg.Item != nil {
						log.Info("DoWindow: buffer msg", msg)
						z := *msg.Item
						wb.Append(z)
					} else if msg.Watermark != nil {
						log.Info("DoWindow: watermark", msg)
						trigger.SignalWatermark(*msg.Watermark)

						// forward watermark
						err := i.pubsub.Publish(ctx, x, Message{
							Key:       msg.Key,
							Watermark: msg.Watermark,
						})
						if err != nil {
							panic(err)
						}
					} else {
						panic("DoWindow: unknown message type")
					}

					return nil
				},
			)
			if err != nil {
				return fmt.Errorf("interpreter.Window(1) %w", err)
			}

			// Trigger final window flush with max watermark
			trigger.SignalWatermark(math.MaxInt64)

			log.Debugln("DoWindow: Finish", i.str(x))
			i.pubsub.Finish(ctx, x)

			return nil
		},
		func(x *DoMap) error {
			log.Debugln("DoMap: Start ", i.str(x))
			var lastOffset int = 0

			err := i.pubsub.Subscribe(
				ctx,
				x.Input,
				lastOffset,
				func(msg Message) error {
					if msg.Item != nil {
						log.Info("DoMap buffer msg", msg)
						z := *msg.Item

						if z.Type == ItemRetractAndAggregate {
							return x.OnMap.Retract(z, func(item Item) {
								err := i.pubsub.Publish(ctx, x, Message{
									Key:  item.Key,
									Item: &item,
								})
								if err != nil {
									panic(err)
								}
							})
						}

						// If the item doesn't have a window, it means it hasn't been windowed yet
						// Just pass it through without accumulation logic (only windows groups can be accumulated)
						if z.Window == nil {
							return x.OnMap.Process(z, func(item Item) {
								err := i.pubsub.Publish(ctx, x, Message{
									Key:  item.Key,
									Item: &item,
								})
								if err != nil {
									panic(err)
								}
							})
						}

						key := KeyedWindowKey(ToKeyedWindowFromItem(&z))
						key = KeyWithNamespace(key, x.Ctx.Name())

						return MatchWindowFlushModeR1(
							x.Ctx.fm,
							func(y *Accumulate) error {
								previous, err := i.bagItem.Get(key)

								isError := err != nil && err != NotFound
								isFound := err == nil
								if isError {
									panic(err)
								}

								var item2 Item
								if isFound {
									item2 = Item{
										Key:    z.Key,
										Window: z.Window,
										Data: schema.MkMap(
											schema.MkField("Previous", previous.Data),
											schema.MkField("Current", z.Data),
										),
										EventTime: z.EventTime,
									}
								} else {
									item2 = Item{
										Key:    z.Key,
										Window: z.Window,
										Data: schema.MkMap(
											schema.MkField("Current", z.Data),
										),
										EventTime: z.EventTime,
									}
								}

								return x.OnMap.Process(item2, func(item Item) {
									err := i.bagItem.Set(key, item)
									if err != nil {
										panic(err)
									}

									err = i.pubsub.Publish(ctx, x, Message{
										Key:  item.Key,
										Item: &item,
									})
									if err != nil {
										panic(err)
									}
								})
							},
							func(y *Discard) error {
								return x.OnMap.Process(z, func(item Item) {
									err := i.pubsub.Publish(ctx, x, Message{
										Key:  item.Key,
										Item: &item,
									})
									if err != nil {
										panic(err)
									}
								})
							},
							func(y *AccumulatingAndRetracting) error {
								previous, err := i.bagItem.Get(key)

								isError := err != nil && err != NotFound
								isFound := err == nil
								if isError {
									panic(err)
								}

								log.Errorln("DoMap: AccumulatingAndRetracting ", key)
								log.Errorln("DoMap: AccumulatingAndRetracting ", isError, isFound)

								var item2 Item
								if isFound {
									item2 = Item{
										Key:    z.Key,
										Window: z.Window,
										Data: schema.MkMap(
											schema.MkField("Previous", previous.Data),
											schema.MkField("Current", z.Data),
										),
										EventTime: z.EventTime,
									}
								} else {
									item2 = Item{
										Key:    z.Key,
										Window: z.Window,
										Data: schema.MkMap(
											schema.MkField("Current", z.Data),
										),
										EventTime: z.EventTime,
									}
								}

								if isFound {
									return x.OnMap.Process(item2, func(newAggregate Item) {
										err := i.bagItem.Set(key, newAggregate)
										if err != nil {
											panic(err)
										}

										err = i.pubsub.Publish(ctx, x, Message{
											Key: newAggregate.Key,
											Item: &Item{
												Key: newAggregate.Key,
												Data: PackRetractAndAggregate(
													previous.Data,
													newAggregate.Data,
												),
												EventTime: newAggregate.EventTime,
												Window:    newAggregate.Window,
												Type:      ItemRetractAndAggregate,
											},
										})

										if err != nil {
											panic(err)
										}
									})
								}

								return x.OnMap.Process(item2, func(item Item) {
									err := i.bagItem.Set(key, item)
									if err != nil {
										panic(err)
									}

									err = i.pubsub.Publish(ctx, x, Message{
										Key:  item.Key,
										Item: &item,
									})
									if err != nil {
										panic(err)
									}
								})
							},
						)
					} else if msg.Watermark != nil {
						log.Info("DoMap: watermark", msg)

						// forward watermark
						err := i.pubsub.Publish(ctx, x, Message{
							Key:       msg.Key,
							Watermark: msg.Watermark,
						})
						if err != nil {
							panic(err)
						}
					} else {
						panic("DoMap: unknown message type")
					}

					return nil
				},
			)
			if err != nil {
				return fmt.Errorf("interpreter.Map(1) %w", err)
			}

			log.Debugln("DoMap: Finish", i.str(x))
			i.pubsub.Finish(ctx, x)

			return nil
		},
		func(x *DoLoad) error {
			var err error
			log.Debugln("DoLoad: Start", i.str(x))
			err = x.OnLoad.Process(Item{}, func(item Item) {
				if err != nil {
					return
				}

				if item.EventTime == 0 {
					item.EventTime = time.Now().UnixNano()
				}

				//// calculate watermark
				//if item.EventTime > i.watermark {
				//	i.watermark = item.EventTime
				//}

				i.stats.Incr(fmt.Sprintf("load[%s].returning", x.Ctx.Name()), 1)

				err = i.pubsub.Publish(ctx, x, Message{
					Key:  item.Key,
					Item: &item,
				})
			})

			if err != nil {
				return fmt.Errorf("interpreter.DoLoad(1) %w", err)
			}

			var mi int64 = math.MaxInt64
			err = i.pubsub.Publish(ctx, x, Message{
				Key:       "none",
				Watermark: &mi,
			})

			log.Debugln("DoLoad: Finish", i.str(x))
			i.pubsub.Finish(ctx, x)

			return nil
		},
		func(x *DoJoin) error {
			lastOffset := make([]int, len(x.Input))
			for idx, _ := range x.Input {
				lastOffset[idx] = 0
			}

			group := ExecutionGroup{ctx: ctx}

			for idx := range x.Input {
				func(idx int) {
					group.Go(func() error {
						return i.pubsub.Subscribe(
							ctx,
							x.Input[idx],
							lastOffset[idx],
							func(msg Message) error {
								lastOffset[idx] = msg.Offset

								i.stats.Incr(fmt.Sprintf("join[%s].returning", x.Ctx.Name()), 1)

								// join streams and publish
								err := i.pubsub.Publish(ctx, x, Message{
									Key:       msg.Key,
									Item:      msg.Item,
									Watermark: msg.Watermark,
								})

								if err != nil {
									return fmt.Errorf("interpreter.DoJoin(1) %w", err)
								}

								return nil
							},
						)
					})
				}(idx)
			}

			if err := group.Wait(); err != nil {
				return fmt.Errorf("interpreter.DoJoin(1) %w", err)
			}

			log.Debugln("DoJoin: Finish", i.str(x))
			i.pubsub.Finish(ctx, x)

			return nil
		},
	)
}

func (i *InMemoryTwoInterpreter) str(x Node) string {
	return ToStr(x)
}

func (i *InMemoryTwoInterpreter) StatsSnapshotAndReset() Stats {
	return i.stats.Snapshot()
}
