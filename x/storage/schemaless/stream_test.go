package schemaless

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
	"time"
)

func TestAppendLog(t *testing.T) {
	ctx := context.TODO()
	schemaDef, found := shape.LookupShapeReflectAndIndex[Change[int]]()
	assert.True(t, found)
	log := NewAppendLog[int](schemaDef)

	done := make(chan struct{})
	go func() {
		err := log.Subscribe(ctx, 0, nil, func(c Change[int]) {
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
