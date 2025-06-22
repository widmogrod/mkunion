package projection

import (
	"context"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"sync"
)

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
	WaitReady()
}

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
