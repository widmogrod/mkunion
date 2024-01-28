package stream

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestNewTypedStream(t *testing.T) {
	stream := NewInMemoryStream[schema.Schema](WithSystemTimeFixed(10))

	ints := NewTypedStreamTopic[int](stream, "ints")
	strings := NewTypedStreamTopic[string](stream, "strings")

	t.Run("Push to ints", func(t *testing.T) {
		err := ints.Push(&Item[int]{
			Topic: "ints",
			Key:   "1",
			Data:  1,
		})
		assert.NoError(t, err)

		t.Run("fail to push to incorrect topic", func(t *testing.T) {
			err = ints.Push(&Item[int]{
				Topic: "strings",
				Key:   "2",
				Data:  2,
			})
			assert.ErrorIs(t, err, ErrTypedTopicMismatch)
		})
	})

	t.Run("Push to strings", func(t *testing.T) {

		err := strings.Push(&Item[string]{
			Topic: "strings",
			Key:   "1",
			Data:  "hello",
		})
		assert.NoError(t, err)

		t.Run("fail to push to incorrect topic", func(t *testing.T) {
			err = strings.Push(&Item[string]{
				Topic: "ints",
				Key:   "2",
				Data:  "hello",
			})
			assert.ErrorIs(t, err, ErrTypedTopicMismatch)
		})
	})

	t.Run("Pull from ints", func(t *testing.T) {
		item, err := stream.Pull(&FromBeginning{
			Topic: "ints",
		})
		assert.NoError(t, err)

		expected := &Item[schema.Schema]{
			Topic:     "ints",
			Key:       "1",
			Data:      schema.MkInt(1),
			EventTime: MkEventTimeFromInt(10),
			Offset:    MkOffsetFromInt(0),
		}

		if diff := cmp.Diff(expected, item); diff != "" {
			t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
		}

		item, err = stream.Pull(&FromOffset{
			Topic:  "ints",
			Offset: item.Offset,
		})
		assert.ErrorIs(t, err, ErrNoMoreNewDataInStream)

	})

	t.Run("Pull from strings", func(t *testing.T) {
		item, err := stream.Pull(&FromBeginning{
			Topic: "strings",
		})
		assert.NoError(t, err)
		expected := &Item[schema.Schema]{
			Topic:     "strings",
			Key:       "1",
			Data:      schema.MkString("hello"),
			EventTime: MkEventTimeFromInt(10),
			Offset:    MkOffsetFromInt(0),
		}

		if diff := cmp.Diff(expected, item); diff != "" {
			t.Fatalf("Pull: diff: (-want +got)\n%s", diff)
		}
	})
}
