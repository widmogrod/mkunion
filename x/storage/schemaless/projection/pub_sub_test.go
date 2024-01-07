package projection

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestPubSubTest_Subscribe(t *testing.T) {
	ctx := context.TODO()

	n := &DoLoad{}
	msg := Message{
		Offset: 0,
		Key:    "123",
		Item: &Item{
			Key:  "321",
			Data: schema.FromGo(1),
		},
		Watermark: nil,
	}
	msg2 := Message{
		Offset: 0,
		Key:    "312",
		Item: &Item{
			Key:  "sdadfad",
			Data: schema.FromGo(2),
		},
		Watermark: nil,
	}

	pubsub := NewPubSub[Node]()
	err := pubsub.Publish(ctx, n, Message{
		Offset: 123,
	})
	assert.ErrorIs(t, err, ErrPublishWithOffset)

	err = pubsub.Publish(ctx, n, msg)
	assert.NoError(t, err)
	err = pubsub.Publish(ctx, n, msg2)
	assert.NoError(t, err)

	assertCalled := func() func(result Message) error {
		order := 0

		asserts := []Message{msg, msg2}

		return func(result Message) error {
			defer func() { order++ }()

			if order >= len(asserts) {
				assert.Fail(t, "should not receive message", result)
			} else {
				assert.Equal(t, asserts[order].Item, result.Item)
				assert.Equal(t, asserts[order].Watermark, result.Watermark)
			}

			return nil
		}
	}

	// when producing is not marked as finished, Subcribe3 will wait for messages
	// we need to run it in a goroutine
	done := make(chan struct{})
	go func() {
		err := pubsub.Subscribe(context.Background(), n, 0, assertCalled())
		assert.NoError(t, err)
		done <- struct{}{}
	}()

	// but when we mark it as finished, it should return
	pubsub.Finish(context.Background(), n)
	err = pubsub.Publish(ctx, n, msg)
	assert.Error(t, err, ErrFinished)
	err = pubsub.Register(n)
	assert.Error(t, err, ErrFinished)

	// Consuming from finished producer must be possible
	err = pubsub.Subscribe(ctx, n, 0, assertCalled())
	assert.NoError(t, err)

	<-done
	close(done)
}
