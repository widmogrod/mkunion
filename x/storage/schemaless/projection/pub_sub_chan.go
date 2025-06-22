package projection

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/shared"
	"sync"
	"sync/atomic"
)

type subscriber[T any] struct {
	inputs      chan T
	f           func(T) error
	done        chan error
	once        sync.Once
	processOnce sync.Once
	finished    chan struct{}
}

func (s *subscriber[T]) Close() {
	s.CloseWithErr(nil)
}

func (s *subscriber[T]) CloseWithErr(err error) {
	s.once.Do(func() {
		// close inputs channel to signal that no more messages will be sent
		close(s.inputs)

		// wait for all background invocations to finish
		<-s.finished
		close(s.finished)

		// send potential error to done channel
		s.done <- err
		close(s.done)
	})
}

func (s *subscriber[T]) Process() {
	s.processOnce.Do(func() {
		for msg := range s.inputs {
			if s.Invoke(msg) {
				break
			}
		}

		s.finished <- struct{}{}
	})
}

func (s *subscriber[T]) Invoke(msg T) bool {
	err := s.f(msg)
	if err != nil {
		s.CloseWithErr(err)
		return true
	}
	return false
}

func NewPubSubChan[T any]() *PubSubChan[T] {
	return &PubSubChan[T]{
		lock:        &sync.RWMutex{},
		channel:     make(chan T, 1000),
		subscribers: nil,

		closed: make(chan struct{}),
	}
}

type PubSubChan[T any] struct {
	lock        *sync.RWMutex
	channel     chan T
	subscribers []*subscriber[T]
	isClosed    atomic.Bool
	once        sync.Once

	closed chan struct{}
}

func (s *PubSubChan[T]) Publish(msg T) error {
	if msg2, ok := any(msg).(Message); ok {
		if msg2.finished {
			s.Close()
			return nil
		}
	}

	if s.isClosed.Load() {
		return fmt.Errorf("PubSubChan.Publish: channel is closed %w", ErrFinished)
	}
	s.channel <- msg
	return nil
}

func (s *PubSubChan[T]) Process() {
	var length int

	defer func() {
		s.lock.RLock()
		for _, sub := range s.subscribers {
			sub.Close()
		}
		s.lock.RUnlock()

		s.closed <- struct{}{}
	}()

	for msg := range s.channel {
		s.lock.RLock()

		length = len(s.subscribers)
		switch length {
		case 0:
			data, _ := shared.JSONMarshal[any](msg)
			log.Warn("PubSubChan.Process: no subscribers but get message: ",
				length, ",", string(data))

		// optimisation, when there is only one subscriber, we can invoke it directly
		case 1:
			s.subscribers[0].Invoke(msg)
		default:
			for _, sub := range s.subscribers {
				sub.inputs <- msg
			}
		}
		s.lock.RUnlock()
	}
}

func (s *PubSubChan[T]) Subscribe(f func(T) error) error {
	if s.isClosed.Load() {
		return fmt.Errorf("PubSubChan.Subscribe: channel is closed %w", ErrFinished)
	}

	sub := &subscriber[T]{
		f:        f,
		done:     make(chan error),
		inputs:   make(chan T, 1000),
		finished: make(chan struct{}),
	}

	go sub.Process()

	s.lock.Lock()
	s.subscribers = append(s.subscribers, sub)
	s.lock.Unlock()

	err := <-sub.done

	s.lock.Lock()
	newSubscribers := make([]*subscriber[T], 0, len(s.subscribers)-1)
	for _, su := range s.subscribers {
		if su != sub {
			newSubscribers = append(newSubscribers, su)
		}
	}
	s.subscribers = newSubscribers
	s.lock.Unlock()

	return err
}

func (s *PubSubChan[T]) Close() {
	s.once.Do(func() {
		s.isClosed.Store(true)
		close(s.channel)
		<-s.closed
	})
}

func (s *PubSubChan[T]) HasSubscribers() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.subscribers) > 0
}
