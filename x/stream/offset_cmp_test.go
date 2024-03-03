package stream

import (
	"testing"
)

func TestOffsetCompare(t *testing.T) {
	// KafkaOffsetCompare
	t.Run("KafkaOffsetCompare", func(t *testing.T) {
		a := mkOffsetFromKafkaTopicPartition(1, 1)
		b := mkOffsetFromKafkaTopicPartition(1, 2)
		SpecComparable(t, *a, *b)
	})

	// InMemoryOffsetCompare
	t.Run("InMemoryOffsetCompare", func(t *testing.T) {
		a := mkInMemoryOffsetFromInt(1)
		b := mkInMemoryOffsetFromInt(2)
		SpecComparable(t, *a, *b)
	})
}
