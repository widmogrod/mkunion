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
