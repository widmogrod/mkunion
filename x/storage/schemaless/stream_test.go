package schemaless

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestAppendLog(t *testing.T) {
	ctx := context.TODO()
	log := NewAppendLog[int]()

	done := make(chan struct{})
	go func() {
		err := log.Subscribe(ctx, 0, func(c Change[int]) {
			done <- struct{}{}
		})
		assert.NoError(t, err)
	}()

	log.Push(Change[int]{
		Before: nil,
		After:  &Record[int]{ID: "123", Data: 1},
	})

	select {
	case <-done:
		// ok
	case <-time.After(100 * time.Millisecond):
		assert.Fail(t, "subscription should receive message")
	}
}
