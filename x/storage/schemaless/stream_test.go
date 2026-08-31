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

func newTestLog(t *testing.T) *AppendLog[int] {
	t.Helper()
	schemaDef, found := shape.LookupShapeReflectAndIndex[Change[int]]()
	assert.True(t, found)
	return NewAppendLog[int](schemaDef)
}

func TestAppendLogCloseUnblocksEmptySubscribe(t *testing.T) {
	log := newTestLog(t)

	errCh := make(chan error, 1)
	go func() {
		errCh <- log.Subscribe(context.Background(), 0, nil, func(c Change[int]) {})
	}()

	// let the subscriber park on the empty log
	time.Sleep(50 * time.Millisecond)
	log.Close()

	select {
	case err := <-errCh:
		assert.NoError(t, err, "closing the log must end the subscription cleanly")
	case <-time.After(2 * time.Second):
		assert.Fail(t, "Close must unblock a subscriber waiting on an empty log")
	}
}

func TestAppendLogSubscribeCtxCancelOnEmptyLog(t *testing.T) {
	log := newTestLog(t)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- log.Subscribe(ctx, 0, nil, func(c Change[int]) {})
	}()

	// let the subscriber park on the empty log
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		assert.Fail(t, "ctx cancellation must unblock a waiting subscriber")
	}
}

func TestAppendLogSubscribeCtxCancelAtTail(t *testing.T) {
	log := newTestLog(t)
	log.Push(Change[int]{After: &Record[int]{ID: "1", Data: 1}})

	ctx, cancel := context.WithCancel(context.Background())
	received := make(chan struct{}, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- log.Subscribe(ctx, 0, nil, func(c Change[int]) {
			received <- struct{}{}
		})
	}()

	<-received
	// the subscriber is now parked at the tail waiting for more messages
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		assert.Fail(t, "ctx cancellation must unblock a subscriber parked at the tail")
	}
}

func TestAppendLogAssignsOffsets(t *testing.T) {
	log := newTestLog(t)
	log.Push(Change[int]{After: &Record[int]{ID: "1", Data: 1}})
	log.Push(Change[int]{After: &Record[int]{ID: "2", Data: 2}})
	log.Push(Change[int]{After: &Record[int]{ID: "3", Data: 3}})
	log.Close()

	collect := func(fromOffset int) ([]int, error) {
		var got []int
		err := log.Subscribe(context.Background(), fromOffset, nil, func(c Change[int]) {
			got = append(got, c.Offset)
		})
		return got, err
	}

	all, err := collect(0)
	assert.NoError(t, err)
	assert.Equal(t, []int{0, 1, 2}, all, "each appended change gets the next offset")

	fromSecond, err := collect(1)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2}, fromSecond, "subscription resumes from the given offset")

	fromLast, err := collect(2)
	assert.NoError(t, err)
	assert.Equal(t, []int{2}, fromLast, "the last offset is also addressable")
}

// Run with -race: Close used to write `closed` under a read lock,
// racing with subscribers that read it under the same read lock.
func TestAppendLogCloseIsRaceFree(t *testing.T) {
	log := newTestLog(t)

	errCh := make(chan error, 1)
	go func() {
		errCh <- log.Subscribe(context.Background(), 0, nil, func(c Change[int]) {})
	}()

	log.Push(Change[int]{After: &Record[int]{ID: "1", Data: 1}})
	// let the subscriber consume and park at the tail
	time.Sleep(50 * time.Millisecond)
	log.Close()

	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		assert.Fail(t, "Close must end the subscription")
	}
}
