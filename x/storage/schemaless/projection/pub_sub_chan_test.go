package projection

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestPubSubChan(t *testing.T) {
	psc := NewPubSubChan[string]()
	go psc.Process()

	var err = errors.New("foo err")

	done := make(chan struct{})

	go func() {
		defer close(done)
		err2 := psc.Subscribe(func(msg string) error {
			assert.Equal(t, "foo", msg)
			return err
		})
		assert.Equal(t, err2, err)
	}()

	start := time.Now()
	timeout := 1 * time.Second
	for !psc.HasSubscribers() {
		if time.Since(start) > timeout {
			t.Fatal("timeout waiting for subscribers")
		}
		// wait for subscribers
		time.Sleep(time.Millisecond * 100)
	}

	err3 := psc.Publish("foo")
	assert.NoError(t, err3)

	<-done
}

func TestPubSubChan_WaitReady(t *testing.T) {
	t.Run("WaitReady returns immediately after Process is called", func(t *testing.T) {
		ps := NewPubSubChan[Message]()

		// Start Process in background
		go ps.Process()

		// WaitReady should return quickly
		done := make(chan struct{})
		go func() {
			ps.WaitReady()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("WaitReady did not return in time")
		}
	})

	t.Run("WaitReady blocks until Process is called", func(t *testing.T) {
		ps := NewPubSubChan[Message]()

		ready := make(chan struct{})
		go func() {
			ps.WaitReady()
			close(ready)
		}()

		// Verify WaitReady is blocking
		select {
		case <-ready:
			t.Fatal("WaitReady returned before Process was called")
		case <-time.After(50 * time.Millisecond):
			// Expected - WaitReady should be blocking
		}

		// Now start Process
		go ps.Process()

		// WaitReady should now return
		select {
		case <-ready:
			// Success
		case <-time.After(100 * time.Millisecond):
			t.Fatal("WaitReady did not return after Process was called")
		}
	})

	t.Run("Multiple WaitReady calls return after Process", func(t *testing.T) {
		ps := NewPubSubChan[Message]()

		readyCount := 3
		ready := make([]chan struct{}, readyCount)

		// Start multiple WaitReady calls
		for i := 0; i < readyCount; i++ {
			ready[i] = make(chan struct{})
			go func(ch chan struct{}) {
				ps.WaitReady()
				close(ch)
			}(ready[i])
		}

		// Give goroutines time to start
		time.Sleep(10 * time.Millisecond)

		// Start Process
		go ps.Process()

		// All WaitReady calls should complete
		for i, ch := range ready {
			select {
			case <-ch:
				// Success
			case <-time.After(100 * time.Millisecond):
				t.Fatalf("WaitReady call %d did not return", i)
			}
		}
	})

	t.Run("WaitReady can be called multiple times after Process", func(t *testing.T) {
		ps := NewPubSubChan[Message]()

		// Start Process
		go ps.Process()

		// First call
		ps.WaitReady()

		// Second call should also work
		done := make(chan struct{})
		go func() {
			ps.WaitReady()
			close(done)
		}()

		select {
		case <-done:
			// Success
		case <-time.After(50 * time.Millisecond):
			t.Fatal("Second WaitReady call did not return")
		}
	})

	t.Run("Publish before Process with WaitReady", func(t *testing.T) {
		ps := NewPubSubChan[Message]()

		// Try to publish before Process starts
		msg := Message{Key: "test", Item: &Item{Key: "test"}}
		err := ps.Publish(msg)
		if err != nil {
			t.Fatalf("Publish failed: %v", err)
		}

		// Set up subscriber
		received := make(chan Message, 1)
		subscribed := make(chan struct{})
		go func() {
			close(subscribed)
			ps.Subscribe(func(m Message) error {
				received <- m
				return nil
			})
		}()

		// Wait for subscriber to be ready
		<-subscribed
		time.Sleep(10 * time.Millisecond)

		// Start Process
		go ps.Process()

		// Wait for Process to be ready
		ps.WaitReady()

		// Message should be received
		select {
		case m := <-received:
			if m.Key != msg.Key {
				t.Fatalf("Received wrong message: got %v, want %v", m, msg)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Message was not received")
		}
	})
}
