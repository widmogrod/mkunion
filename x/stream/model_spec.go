package stream

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"math/rand"
	"testing"
)

// HappyPathSpec is a helper function to test the happy path of a stream
// assumption: all keys land in the same partition, so Pulling will result in the same order as Pushing
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

// IsolationSpec verifies that a stream behaves like a log of serialized
// messages (Kafka-like): what is pushed is captured at push time, and what
// is pulled is a copy. Mutating a caller's value after push, or a pulled
// item, must never change what the stream holds. The in-memory stream is
// the reference used in recovery tests, so it must not offer a shortcut
// (live pointers) that real brokers do not.
func IsolationSpec(t *testing.T, s Stream[[]string]) {
	t.Run("pushed data does not alias the caller's value", func(t *testing.T) {
		topicName := fmt.Sprintf("topic-%d", rand.Int63())
		item := &Item[[]string]{
			Topic: topicName,
			Key:   "key",
			Data:  []string{"original"},
		}
		if err := s.Push(item); err != nil {
			t.Fatalf("push should succeed, got %v", err)
		}

		item.Data[0] = "mutated-after-push"

		pulled, err := s.Pull(&FromBeginning{Topic: topicName})
		if err != nil {
			t.Fatalf("pull should succeed, got %v", err)
		}
		if diff := cmp.Diff([]string{"original"}, pulled.Data); diff != "" {
			t.Errorf("the stream must capture data at push time, got %s", diff)
		}
	})

	t.Run("pulled item does not alias the stream's log", func(t *testing.T) {
		topicName := fmt.Sprintf("topic-%d", rand.Int63())
		if err := s.Push(&Item[[]string]{Topic: topicName, Key: "key", Data: []string{"original"}}); err != nil {
			t.Fatalf("push should succeed, got %v", err)
		}

		first, err := s.Pull(&FromBeginning{Topic: topicName})
		if err != nil {
			t.Fatalf("pull should succeed, got %v", err)
		}
		first.Data[0] = "mutated-after-pull"
		first.Key = "mutated-key"

		again, err := s.Pull(&FromBeginning{Topic: topicName})
		if err != nil {
			t.Fatalf("pull should succeed, got %v", err)
		}
		if diff := cmp.Diff([]string{"original"}, again.Data); diff != "" {
			t.Errorf("mutating a pulled item must not rewrite the log, got %s", diff)
		}
		if diff := cmp.Diff("key", again.Key); diff != "" {
			t.Errorf("mutating a pulled item must not rewrite the log, got %s", diff)
		}
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
