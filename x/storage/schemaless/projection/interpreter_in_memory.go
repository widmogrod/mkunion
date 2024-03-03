package projection

import (
	"context"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"sync"
)

// import (
//
//	"context"
//	"fmt"
//	log "github.com/sirupsen/logrus"
//	"github.com/widmogrod/mkunion/x/schema"
//	"sync"
//
// )
//
//	func DefaultInMemoryInterpreter() *InMemoryInterpreter {
//		return &InMemoryInterpreter{
//			pubsub: NewPubSubMultiChan[Node](),
//			//pubsub:  NewPubSub[Node](),
//			byKeys:  make(map[Node]map[string]Item),
//			running: make(map[Node]struct{}),
//			stats:   NewStatsCollector(),
//		}
//	}
type ExecutionStatus int

const (
	ExecutionStatusNew ExecutionStatus = iota
	ExecutionStatusRunning
	ExecutionStatusError
	ExecutionStatusFinished
)

var (
	ErrInterpreterNotInNewState = fmt.Errorf("interpreter is not in new state")
)

type PubSubForInterpreter[T comparable] interface {
	Register(key T) error
	Publish(ctx context.Context, key T, msg Message) error
	Finish(ctx context.Context, key T)
	Subscribe(ctx context.Context, node T, fromOffset int, f func(Message) error) error
}

//	type InMemoryInterpreter struct {
//		lock    sync.Mutex
//		pubsub  PubSubForInterpreter[Node]
//		byKeys  map[Node]map[string]Item
//		running map[Node]struct{}
//		status  ExecutionStatus
//		// what differences between process time and event time
//		// should answers question
//		// - are there any events in the system, that a process should wait?
//		watermark int64
//		stats     StatsCollector
//	}
//
//	func (i *InMemoryInterpreter) Run(ctx context.Context, nodes []Node) error {
//		i.lock.Lock()
//		if i.status != ExecutionStatusNew {
//			i.lock.Unlock()
//			return fmt.Errorf("interpreter.Run state %d %w", i.status, ErrInterpreterNotInNewState)
//		}
//		i.status = ExecutionStatusRunning
//		i.lock.Unlock()
//
//		ctx, cancel := context.WithCancel(ctx)
//		group := &ExecutionGroup{
//			ctx:    ctx,
//			cancel: cancel,
//		}
//
//		// Registering new nodes makes sure that, in case of non-deterministic concurrency
//		// when goroutine want to subscribe to a node, it will be registered, even if it's not publishing yet
//		for _, node := range nodes {
//			err := i.pubsub.Register(node)
//			if err != nil {
//				i.lock.Lock()
//				i.status = ExecutionStatusError
//				i.lock.Unlock()
//
//				return fmt.Errorf("interpreter.Run(1) %w", err)
//			}
//		}
//
//		for _, node := range nodes {
//			func(node Node) {
//				group.Go(func() (err error) {
//					return i.run(ctx, node)
//				})
//			}(node)
//		}
//
//		if err := group.Wait(); err != nil {
//			i.lock.Lock()
//			i.status = ExecutionStatusError
//			i.lock.Unlock()
//
//			return fmt.Errorf("interpreter.Run(2) %w", err)
//		}
//
//		i.lock.Lock()
//		i.status = ExecutionStatusFinished
//		i.lock.Unlock()
//
//		return nil
//	}
//
//	func (i *InMemoryInterpreter) run(ctx context.Context, dag Node) error {
//		if dag == nil {
//			//panic("fix nodes that are nil! fix dag builder!")
//			return nil
//		}
//
//		// TODO introduce parallelism for Item - key groups
//		// bounded to some number of goroutines, that can be configured
//		// and that can be used to limit memory usage
//
//		// TODO introduce merge window triggers, and triggers in general so that
//		// - RepositorySink can be used with batches
//		// - LiveSelect in TicTacToe game, can show progress in game after reload, not through streaming updates, but by sending final state - debounce?
//
//		/*
//
//				parallelize.Window().DoWindow().Log()
//				Parallelize by key groups
//				- group by key
//				- for each key group, run in parallel
//
//				i: (a, 1) (b, 2) (c, 3) (d, 4) (a, 5) (b, 6) (c, 7) (d, 8)
//			    Window(i, +1)
//				o: (a, 2) (b, 3) (c, 4) (d, 5) (a, 6) (b, 7) (c, 8) (d, 9)
//
//				(a, 1) (a, 5)
//				(b, 2) (b, 6)
//				(c, 3) (c, 7)
//				(d, 4) (d, 8)
//
//		*/
//
//		return MustMatchNode(
//			dag,
//			func(x *Window) error {
//				log.Debugln("Window: Start ", i.str(x))
//				var lastOffset int = 0
//
//				err := i.pubsub.Subscribe(
//					ctx,
//					x.Input,
//					lastOffset,
//					func(msg Message) error {
//						lastOffset = msg.Offset
//						log.Debugln("Window: ", i.str(x), msg.Item != nil, msg.Watermark != nil)
//						log.Debugf("✉️: %+v %s\n", msg, i.str(x))
//						switch true {
//						case msg.Item != nil && msg.Watermark == nil,
//							msg.Item != nil && msg.Watermark != nil && !x.Ctx.ShouldRetract():
//
//							err := x.OnMap.Process(*msg.Item, func(item Item) {
//								i.stats.Incr(fmt.Sprintf("map[%s].returning.aggregate", x.Ctx.Name()), 1)
//
//								err := i.pubsub.Publish(ctx, x, Message{
//									Key:  item.Key,
//									Item: &item,
//								})
//								if err != nil {
//									panic(err)
//								}
//							})
//							if err != nil {
//								panic(err)
//							}
//
//						case msg.Item != nil && msg.Watermark != nil && x.Ctx.ShouldRetract():
//							buff := NewDual()
//							err := x.OnMap.Process(*msg.Item, buff.ReturningAggregate)
//							if err != nil {
//								panic(err)
//							}
//							err = x.OnMap.Retract(*msg.Watermark, buff.ReturningRetract)
//							if err != nil {
//								panic(err)
//							}
//
//							if !buff.IsValid() {
//								panic("Window(1); asymmetry " + i.str(x))
//							}
//
//							for _, msg := range buff.List() {
//								i.stats.Incr(fmt.Sprintf("map[%s].returning.aggregate", x.Ctx.Name()), 1)
//								i.stats.Incr(fmt.Sprintf("map[%s].returning.retract", x.Ctx.Name()), 1)
//
//								err := i.pubsub.Publish(ctx, x, *msg)
//								if err != nil {
//									panic(err)
//								}
//							}
//
//						case msg.Item == nil && msg.Watermark != nil && x.Ctx.ShouldRetract():
//							err := x.OnMap.Retract(*msg.Watermark, func(item Item) {
//
//								i.stats.Incr(fmt.Sprintf("map[%s].returning.aggregate", x.Ctx.Name()), 1)
//
//								err := i.pubsub.Publish(ctx, x, Message{
//									Key:       item.Key,
//									Watermark: &item,
//								})
//								if err != nil {
//									panic(err)
//								}
//							})
//							if err != nil {
//								panic(err)
//							}
//
//						case msg.Item == nil && msg.Watermark != nil && !x.Ctx.ShouldRetract():
//							log.Debugln("ignored retraction", i.str(x))
//
//						default:
//							panic("not implemented Window(3); " + i.str(x) + " " + ToStrMessage(msg))
//						}
//
//						log.Debugln("√", i.str(x))
//
//						return nil
//					},
//				)
//				if err != nil {
//					return fmt.Errorf("interpreter.Window(1) %w", err)
//				}
//
//				log.Debugln("Window: Finish", i.str(x))
//				i.pubsub.Finish(ctx, x)
//
//				return nil
//			},
//			func(x *DoWindow) error {
//				var lastOffset int = 0
//				prev := make(map[string]*Item)
//
//				err := i.pubsub.Subscribe(
//					ctx,
//					x.Input,
//					lastOffset,
//					func(msg Message) error {
//						lastOffset = msg.Offset
//
//						if msg.Watermark == nil && msg.Item == nil {
//							panic("message has not Aggretate nor Watermark. not implemented (1)")
//						}
//
//						log.Debugln("DoWindow 👯: ", i.str(x), msg.Item != nil, msg.Watermark != nil)
//
//						if _, ok := prev[msg.Key]; ok {
//							base := prev[msg.Key]
//
//							// TODO: retraction and aggregatoin don't happen in transactional way, even if message has both operations
//							// this is a problem, because if retraction fails, then aggregation will be lost
//							if msg.Watermark != nil && x.Ctx.ShouldRetract() {
//								log.Debugln("❌retracting in merge", i.str(x))
//								retract := Item{
//									Key:  msg.Key,
//									Data: schema.MkList(base.Data, msg.Watermark.Data),
//								}
//
//								if err := x.OnMap.Retract(retract, func(item Item) {
//
//									i.stats.Incr(fmt.Sprintf("merge[%s].returning.retract", x.Ctx.Name()), 1)
//
//									base = &item
//									err := i.pubsub.Publish(ctx, x, Message{
//										Key:       msg.Key,
//										Watermark: &item,
//									})
//									if err != nil {
//										panic(err)
//									}
//								}); err != nil {
//									panic(err)
//								}
//							}
//
//							if msg.Item != nil {
//								log.Debugln("✅aggregate in merge", i.str(x))
//								merge := Item{
//									Key:  msg.Key,
//									Data: schema.MkList(base.Data, msg.Item.Data),
//								}
//								err := x.OnMap.Process(merge, func(item Item) {
//									i.stats.Incr(fmt.Sprintf("merge[%s].returning.aggregate", x.Ctx.Name()), 1)
//
//									p := base
//									base = &item
//									// TODO: In feature, we should make better decision whenever send retractions or not.
//									// For now, we always send retractions, they don't have to be treated as retraction by the receiver.
//									// But, this has penalty related to throughput, and latency, and for some applications, it is not acceptable.
//									err := i.pubsub.Publish(ctx, x, Message{
//										Key:       msg.Key,
//										Item:      &item,
//										Watermark: p,
//									})
//									if err != nil {
//										panic(err)
//									}
//								})
//								if err != nil {
//									panic(err)
//								}
//							}
//
//							prev[msg.Key] = base
//
//						} else {
//							if msg.Watermark != nil {
//								panic("no previous state, and requesing retracting. not implemented (2)" + ToStrMessage(msg))
//							}
//
//							i.stats.Incr(fmt.Sprintf("merge[%s].returning.aggregate", x.Ctx.Name()), 1)
//
//							prev[msg.Key] = msg.Item
//							err := i.pubsub.Publish(ctx, x, Message{
//								Key:  msg.Key,
//								Item: msg.Item,
//							})
//							if err != nil {
//								return fmt.Errorf("interpreter.DoWindow(1) %w", err)
//							}
//						}
//
//						return nil
//					},
//				)
//				if err != nil {
//					return fmt.Errorf("interpreter.DoWindow(1) %w", err)
//				}
//
//				//for _, item := range prev {
//				//	err := i.pubsub.Publish(ctx, x, Message{
//				//		Key:       item.Key,
//				//		Item: item,
//				//	})
//				//	if err != nil {
//				//		return fmt.Errorf("interpreter.DoWindow(2) %w", err)
//				//	}
//				//}
//
//				log.Debugln("DoWindow: Finish", i.str(x))
//				i.pubsub.Finish(ctx, x)
//
//				return nil
//			},
//			func(x *DoLoad) error {
//				var err error
//				log.Debugln("DoLoad: Start", i.str(x))
//				err = x.OnLoad.Process(Item{}, func(item Item) {
//					if err != nil {
//						return
//					}
//
//					//if item.EventTime == 0 {
//					//	item.EventTime = time.Now().UnixNano()
//					//}
//					//
//					//// calculate watermark
//					//if item.EventTime > i.watermark {
//					//	i.watermark = item.EventTime
//					//}
//
//					i.stats.Incr(fmt.Sprintf("load[%s].returning", x.Ctx.Name()), 1)
//
//					err = i.pubsub.Publish(ctx, x, Message{
//						Key:       item.Key,
//						Item:      &item,
//						Watermark: nil,
//					})
//				})
//
//				if err != nil {
//					return fmt.Errorf("interpreter.DoLoad(1) %w", err)
//				}
//
//				log.Debugln("DoLoad: Finish", i.str(x))
//				i.pubsub.Finish(ctx, x)
//
//				return nil
//			},
//			func(x *DoJoin) error {
//				lastOffset := make([]int, len(x.Input))
//				for idx, _ := range x.Input {
//					lastOffset[idx] = 0
//				}
//
//				group := ExecutionGroup{ctx: ctx}
//
//				for idx := range x.Input {
//					func(idx int) {
//						group.Go(func() error {
//							return i.pubsub.Subscribe(
//								ctx,
//								x.Input[idx],
//								lastOffset[idx],
//								func(msg Message) error {
//									lastOffset[idx] = msg.Offset
//
//									i.stats.Incr(fmt.Sprintf("join[%s].returning", x.Ctx.Name()), 1)
//
//									// join streams and publish
//									err := i.pubsub.Publish(ctx, x, Message{
//										Key:       msg.Key,
//										Item:      msg.Item,
//										Watermark: msg.Watermark,
//									})
//
//									if err != nil {
//										return fmt.Errorf("interpreter.DoJoin(1) %w", err)
//									}
//
//									return nil
//								},
//							)
//						})
//					}(idx)
//				}
//
//				if err := group.Wait(); err != nil {
//					return fmt.Errorf("interpreter.DoJoin(1) %w", err)
//				}
//
//				log.Debugln("DoJoin: Finish", i.str(x))
//				i.pubsub.Finish(ctx, x)
//
//				return nil
//			},
//		)
//	}
//func (i *InMemoryInterpreter) str(x Node) string {
//	return ToStr(x)
//}
//
//func (i *InMemoryInterpreter) StatsSnapshotAndReset() Stats {
//	return i.stats.SnapshotFrom()
//}

//func ToStrMessage(msg Message) string {
//	return fmt.Sprintf("Message{Key: %s, Watermark: %s, Item: %s}",
//		msg.Key,
//		//ToStrItem(msg.Watermark),
//		ToStrItem(msg.Item))
//}

func ToStrItem(item *Item) string {
	if item == nil {
		return "nil"
	}
	bytes, err := shared.JSONMarshal[schema.Schema](item.Data)

	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("Item{Key: %s, Data: %s}",
		item.Key, string(bytes))
}

func ToStr(x Node) string {
	return MatchNodeR1(
		x,
		func(x *DoWindow) string {
			return fmt.Sprintf("map(%sv)", x.Ctx.Name())
		},
		func(x *DoMap) string {
			return fmt.Sprintf("merge(%sv)", x.Ctx.Name())
		},
		func(x *DoLoad) string {
			return fmt.Sprintf("DoLoad(%s)", x.Ctx.Name())
		},
		func(x *DoJoin) string {
			return fmt.Sprintf("join(%s)", x.Ctx.Name())
		},
	)
}

type ExecutionGroup struct {
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
	err    error
	once   sync.Once
}

func (g *ExecutionGroup) Go(f func() error) {
	g.wg.Add(1)

	started := make(chan struct{})
	go func() {
		defer g.wg.Done()

		select {
		case <-g.ctx.Done():
			// signal that goroutine has started
			close(started)
			if err := g.ctx.Err(); err != nil {
				g.once.Do(func() {
					g.err = err
					if g.cancel != nil {
						g.cancel()
					}
				})
			}

		default:
			// signal that goroutine has started
			close(started)
			err := f()
			if err != nil {
				g.once.Do(func() {
					g.err = err
					if g.cancel != nil {
						g.cancel()
					}
				})
			}
		}
	}()

	<-started
}

func (g *ExecutionGroup) Wait() error {
	g.wg.Wait()
	if g.cancel != nil {
		g.cancel()
	}
	return nil
}
