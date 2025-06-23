package projection

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync"
)

func NewPubSubMultiChan[T comparable]() *PubSubMulti[T] {
	return &PubSubMulti[T]{
		multi: make(map[T]PubSubSingler[Message]),
		lock:  &sync.RWMutex{},
		new: func() PubSubSingler[Message] {
			return NewPubSubChan[Message]()
		},
		finished: make(map[T]bool),
	}
}

type PubSubSingler[T comparable] interface {
	Publish(msg T) error
	Process()
	Subscribe(f func(T) error) error
	Close()
	WaitReady()
}

var _ PubSubForInterpreter[any] = (*PubSubMulti[any])(nil)

type PubSubMulti[T comparable] struct {
	multi    map[T]PubSubSingler[Message]
	lock     *sync.RWMutex
	new      func() PubSubSingler[Message]
	finished map[T]bool
}

func (p *PubSubMulti[T]) Register(key T) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if _, ok := p.multi[key]; ok {
		return nil
		//return fmt.Errorf("PubSubMulti.registerRec: key %s already registered", key)
	}

	ps := p.new()
	p.multi[key] = ps

	// Start the Process goroutine immediately upon registration
	// This ensures the channel is ready to receive messages before any publishing occurs
	go ps.Process()

	return nil
}

func (p *PubSubMulti[T]) Publish(ctx context.Context, key T, msg Message) error {
	log.Debugf("PublishMulti: key=%v msg=%v", key, msg)
	select {
	case <-ctx.Done():
		return fmt.Errorf("PubSubMulti.Publish: key=%#v ctx=%s %w", key, ctx.Err(), ErrContextDone)
	default:
		// continue
	}

	if msg.Offset != 0 {
		return fmt.Errorf("PubSubMulti.Publish: key=%#v %w", key, ErrPublishWithOffset)
	}

	p.lock.RLock()
	if _, ok := p.finished[key]; ok {
		p.lock.RUnlock()
		return fmt.Errorf("PubSubMulti.Publish: key=%#v %w", key, ErrFinished)
	}
	p.lock.RUnlock()

	if _, ok := p.multi[key]; !ok {
		return fmt.Errorf("PubSubMulti.Publish: key=%#v not registered", key)
	}

	return p.multi[key].Publish(msg)
}

func (p *PubSubMulti[T]) Finish(ctx context.Context, key T) {
	err := p.Publish(ctx, key, Message{finished: true})
	if err != nil {
		panic(err)
	}
	p.lock.Lock()
	p.finished[key] = true
	p.lock.Unlock()

	//p.multi[key].Close()
}

func (p *PubSubMulti[T]) Subscribe(ctx context.Context, key T, fromOffset int, f func(Message) error) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("PubSubMulti.Subscribe %s %w", ctx.Err(), ErrContextDone)
	default:
	}

	p.lock.RLock()
	if _, ok := p.finished[key]; ok {
		p.lock.RUnlock()
		return fmt.Errorf("PubSubMulti.Subscribe: key=%#v %w", key, ErrFinished)
	}
	p.lock.RUnlock()

	p.lock.RLock()
	if _, ok := p.multi[key]; !ok {
		p.lock.RUnlock()
		return fmt.Errorf("PubSubMulti.Subscribe: key %T not registered", key)
	}
	p.lock.RUnlock()

	return p.multi[key].Subscribe(f)
}

// WaitReady waits for all registered pubsubs to be ready to receive messages
func (p *PubSubMulti[T]) WaitReady() {
	p.lock.RLock()
	pubsubs := make([]PubSubSingler[Message], 0, len(p.multi))
	for _, ps := range p.multi {
		pubsubs = append(pubsubs, ps)
	}
	p.lock.RUnlock()

	// Wait for all Process goroutines to be ready
	for _, ps := range pubsubs {
		ps.WaitReady()
	}
}
