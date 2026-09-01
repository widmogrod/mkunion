package projection

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
)

func publishN(t *testing.T, p *PubSubSingle, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		require.NoError(t, p.Publish(context.Background(), Message{
			Key:  "k",
			Item: &Item{Key: "k", Data: schema.MkInt(int64(i))},
		}))
	}
}

// subscribeCollect drains the log until Finish and returns received offsets.
func subscribeCollect(t *testing.T, p *PubSubSingle, fromOffset int) ([]int, error) {
	t.Helper()
	var got []int
	var mu sync.Mutex
	done := make(chan error, 1)
	go func() {
		done <- p.Subscribe(context.Background(), fromOffset, func(msg Message) error {
			mu.Lock()
			got = append(got, msg.Offset)
			mu.Unlock()
			return nil
		})
	}()
	p.Finish()
	select {
	case err := <-done:
		return got, err
	case <-time.After(5 * time.Second):
		t.Fatal("subscribe did not finish")
		return nil, nil
	}
}

func TestPubSubSingleOffsets(t *testing.T) {
	t.Run("offset 0 replays from the beginning", func(t *testing.T) {
		p := NewPubSubSingle()
		publishN(t, p, 3)
		got, err := subscribeCollect(t, p, 0)
		require.NoError(t, err)
		assert.Equal(t, []int{0, 1, 2}, got)
	})

	t.Run("offset -1 starts at the last message", func(t *testing.T) {
		p := NewPubSubSingle()
		publishN(t, p, 3)
		got, err := subscribeCollect(t, p, -1)
		require.NoError(t, err)
		assert.Equal(t, []int{2}, got)
	})

	t.Run("a specific offset resumes there", func(t *testing.T) {
		p := NewPubSubSingle()
		publishN(t, p, 3)
		got, err := subscribeCollect(t, p, 1)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, got)
	})

	t.Run("unknown offset errors", func(t *testing.T) {
		p := NewPubSubSingle()
		publishN(t, p, 2)
		_, err := subscribeCollect(t, p, 99)
		assert.ErrorContains(t, err, "offset not found")
	})

	t.Run("finished empty log returns immediately", func(t *testing.T) {
		p := NewPubSubSingle()
		p.Finish()
		err := p.Subscribe(context.Background(), 0, func(Message) error {
			t.Fatal("nothing to deliver")
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("handler error stops the subscription", func(t *testing.T) {
		p := NewPubSubSingle()
		publishN(t, p, 1)
		err := p.Subscribe(context.Background(), 0, func(Message) error {
			return errors.New("handler boom")
		})
		assert.ErrorIs(t, err, ErrHandlerReturnErr)
	})

	t.Run("cancelled context stops the subscription", func(t *testing.T) {
		p := NewPubSubSingle()
		publishN(t, p, 1)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := p.Subscribe(ctx, 0, func(Message) error { return nil })
		assert.ErrorIs(t, err, ErrContextDone)
	})
}

func TestPubSubSinglePublishRejections(t *testing.T) {
	t.Run("publish with an explicit offset is rejected", func(t *testing.T) {
		p := NewPubSubSingle()
		err := p.Publish(context.Background(), Message{Offset: 7})
		assert.ErrorIs(t, err, ErrPublishWithOffset)
	})

	t.Run("publish after finish is rejected", func(t *testing.T) {
		p := NewPubSubSingle()
		p.Finish()
		err := p.Publish(context.Background(), Message{})
		assert.ErrorIs(t, err, ErrFinished)
	})

	t.Run("publish with a cancelled context is rejected", func(t *testing.T) {
		p := NewPubSubSingle()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := p.Publish(ctx, Message{})
		assert.ErrorIs(t, err, ErrContextDone)
	})
}
