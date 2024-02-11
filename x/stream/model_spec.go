package stream

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"math/rand"
	"testing"
)

// HappyPathSpec is a helper function to test the happy path of a stream
func HappyPathSpec[A any](t *testing.T, s Stream[A], gen func() A) {
	t.Run("Push to stream single value", func(t *testing.T) {
		topicName := fmt.Sprintf("topic-%d", rand.Int63())
		keyName := fmt.Sprintf("key-%d", rand.Int63())

		t.Logf("topicName: %s", topicName)
		t.Logf("keyName: %s", keyName)

		item := &Item[A]{
			Topic: topicName,
			Key:   keyName,
			Data:  gen(),
		}

		err := s.Push(item)
		if err != nil {
			t.Fatalf("expected nil, got %v", err)
		}

		t.Run("and then pull from stream", func(t *testing.T) {
			value, err := s.Pull(&FromBeginning{
				Topic: topicName,
			})
			if err != nil {
				t.Fatalf("expected nil, got %v", err)
			}

			ItemPushAndPullEqualSpec(t, item, value)
		})
	})

	t.Run("Push to stream few values", func(t *testing.T) {
		topicName := fmt.Sprintf("topic-%d", rand.Int63())
		keyName := fmt.Sprintf("key-%d", rand.Int63())

		t.Logf("topicName: %s", topicName)
		t.Logf("keyName: %s", keyName)

		var items []*Item[A]
		for i := 0; i < 10; i++ {
			item := &Item[A]{
				Topic: topicName,
				Key:   keyName,
				Data:  gen(),
			}
			items = append(items, item)
		}

		for i, item := range items {
			err := s.Push(item)
			if err != nil {
				t.Fatalf("publishing message %d should succeed, got %v", i, err)
			}
		}

		t.Run("and then pull from stream", func(t *testing.T) {
			var pullCmd PullCMD = &FromBeginning{
				Topic: topicName,
			}
			for i, item := range items {
				t.Logf("pullCmd: %+#v", pullCmd)
				value, err := s.Pull(pullCmd)
				if err != nil {
					t.Fatalf("pulling message %d should succeed, got %v", i, err)
				}

				if !ItemPushAndPullEqualSpec(t, item, value) {
					break
				}

				pullCmd = &FromOffset{
					Topic:  topicName,
					Offset: value.Offset,
				}
			}
		})

	})
}

// ItemPushAndPullEqualSpec is a helper function to test if the pushed and pulled items are equal
func ItemPushAndPullEqualSpec[A any](t *testing.T, pushed, pulled *Item[A]) bool {
	if pushed == nil {
		t.Fatalf("expected pushed item to not be nil")
	}
	if pulled == nil {
		t.Fatalf("expected pulled item to not be nil")
	}

	return t.Run("the same pulled", func(t *testing.T) {
		if diff := cmp.Diff(pushed.Topic, pulled.Topic); diff != "" {
			t.Errorf("expected topic to be the same, got %s", diff)
		}
		if diff := cmp.Diff(pushed.Key, pulled.Key); diff != "" {
			t.Errorf("expected key to be the same, got %s", diff)
		}
		if diff := cmp.Diff(pushed.Data, pulled.Data); diff != "" {
			t.Errorf("expected data to be the same, got %s", diff)
		}

		if pulled.Offset == nil {
			t.Errorf("expected pulled value to have offset set")
		}
		if pulled.EventTime == nil {
			t.Errorf("expected pulled value to have event time set")
		}
	})
}
