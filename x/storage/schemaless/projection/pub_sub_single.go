package projection

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"sync"
)

func NewPubSubSingle() *PubSubSingle {
	lock := sync.RWMutex{}
	return &PubSubSingle{
		lock:      &lock,
		cond:      sync.NewCond(lock.RLocker()),
		appendLog: list.New(),
		finished:  false,
	}
}

type PubSubSingle struct {
	lock      *sync.RWMutex
	cond      *sync.Cond
	appendLog *list.List
	finished  bool
}

func (p *PubSubSingle) Publish(ctx context.Context, msg Message) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("pubsubsingle.Publish: ctx=%s %w", ctx.Err(), ErrContextDone)
	default:
		// continue
	}

	if msg.Offset != 0 {
		return fmt.Errorf("pubsubsingle.Publish: %w", ErrPublishWithOffset)
	}

	p.lock.Lock()
	defer p.lock.Unlock()
	if p.finished {
		return fmt.Errorf("pubsubsingle.Publish: %w", ErrFinished)
	}

	msg.Offset = p.appendLog.Len()
	p.appendLog.PushBack(msg)
	p.cond.Broadcast()
	return nil
}

// Finish is called when a node won't publish any more messages
func (p *PubSubSingle) Finish() {
	p.lock.Lock()
	p.finished = true
	p.lock.Unlock()

	p.cond.Broadcast()
}

func (p *PubSubSingle) Subscribe(ctx context.Context, fromOffset int, f func(Message) error) error {
	var prev *list.Element = nil

	// Until, there is no messages, wait
	p.cond.L.Lock()
	for p.appendLog.Len() == 0 && !p.finished {
		p.cond.Wait()
	}
	if p.appendLog.Len() == 0 && p.finished {
		p.cond.L.Unlock()
		return nil
	}

	// Select the offset to start reading messages from
	switch fromOffset {
	case 0:
		prev = p.appendLog.Front()
	case -1:
		prev = p.appendLog.Back()
	default:
		for e := p.appendLog.Front(); e != nil; e = e.Next() {
			prev = e
			if e.Value.(Message).Offset == fromOffset {
				break
			}
		}

		if prev == p.appendLog.Back() {
			p.cond.L.Unlock()
			return errors.New("offset not found")
		}
	}
	p.cond.L.Unlock()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("pubsubsingle.Subscribe %s %w", ctx.Err(), ErrContextDone)

		default:
			msg := prev.Value.(Message)

			err := f(msg)
			if err != nil {
				return fmt.Errorf("pubsubsingle.Subscribe %s %w", err, ErrHandlerReturnErr)
			}

			// Wait for new changes to be available
			p.cond.L.Lock()
			for prev.Next() == nil && !p.finished {
				p.cond.Wait()
			}
			if prev.Next() == nil && p.finished {
				p.cond.L.Unlock()
				return nil
			}

			prev = prev.Next()
			p.cond.L.Unlock()
		}
	}
}
