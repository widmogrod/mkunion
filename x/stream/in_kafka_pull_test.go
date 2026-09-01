package stream

import (
	"errors"
	"testing"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConsumer replays a scripted list of Poll events; nil entries
// simulate poll timeouts.
type fakeConsumer struct {
	events []kafka.Event
	polls  int

	subscribeErr error
	assignErr    error
	unassignErr  error

	subscribed []string
	assigned   [][]kafka.TopicPartition
	unassigns  int
	closed     bool
}

func (f *fakeConsumer) Subscribe(topic string, _ kafka.RebalanceCb) error {
	f.subscribed = append(f.subscribed, topic)
	return f.subscribeErr
}

func (f *fakeConsumer) Poll(_ int) kafka.Event {
	if f.polls >= len(f.events) {
		panic("fakeConsumer: Poll called after all scripted events were consumed")
	}
	e := f.events[f.polls]
	f.polls++
	return e
}

func (f *fakeConsumer) Assign(partitions []kafka.TopicPartition) error {
	f.assigned = append(f.assigned, partitions)
	return f.assignErr
}

func (f *fakeConsumer) Unassign() error {
	f.unassigns++
	return f.unassignErr
}

func (f *fakeConsumer) Close() error {
	f.closed = true
	return nil
}

func kafkaStreamWithFake(t *testing.T, fake *fakeConsumer) *KafkaStream[int] {
	t.Helper()
	k := NewKafkaStream[int](kafka.ConfigMap{}, kafka.ConfigMap{}, WithSystemTime)
	k.newConsumer = func(conf *kafka.ConfigMap) (kafkaConsumer, error) {
		return fake, nil
	}
	return k
}

func kafkaMessage(topic string, key string, value string, partition int32, offset kafka.Offset) *kafka.Message {
	return &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: partition,
			Offset:    offset,
		},
		Key:       []byte(key),
		Value:     []byte(value),
		Timestamp: time.Unix(0, 42),
	}
}

func TestKafkaPullFromBeginning(t *testing.T) {
	t.Run("empty topic is rejected", func(t *testing.T) {
		k := kafkaStreamWithFake(t, &fakeConsumer{})
		_, err := k.Pull(&FromBeginning{Topic: ""})
		assert.ErrorIs(t, err, ErrEmptyTopic)
	})

	t.Run("consumer initiation error propagates", func(t *testing.T) {
		k := NewKafkaStream[int](kafka.ConfigMap{}, kafka.ConfigMap{}, WithSystemTime)
		k.newConsumer = func(conf *kafka.ConfigMap) (kafkaConsumer, error) {
			return nil, errors.New("boom")
		}
		_, err := k.Pull(&FromBeginning{Topic: "t"})
		assert.ErrorContains(t, err, "on consumer initiation")
	})

	t.Run("subscribe error propagates and consumer closes", func(t *testing.T) {
		fake := &fakeConsumer{subscribeErr: errors.New("no broker")}
		k := kafkaStreamWithFake(t, fake)
		_, err := k.Pull(&FromBeginning{Topic: "t"})
		assert.ErrorContains(t, err, "subscribe")
		assert.True(t, fake.closed, "consumer must be closed on the error path")
	})

	t.Run("skips timeouts, kafka errors, and unknown events until a message", func(t *testing.T) {
		fake := &fakeConsumer{events: []kafka.Event{
			nil, // poll timeout
			kafka.NewError(kafka.ErrAllBrokersDown, "transient", false),
			kafka.OffsetsCommitted{}, // unknown event type is ignored
			kafkaMessage("t", "key-1", "7", 2, 9),
		}}
		k := kafkaStreamWithFake(t, fake)

		item, err := k.Pull(&FromBeginning{Topic: "t"})
		require.NoError(t, err)
		assert.Equal(t, "t", item.Topic)
		assert.Equal(t, "key-1", item.Key)
		assert.Equal(t, 7, item.Data)
		assert.Equal(t, EventTime(42), *item.EventTime)
		assert.Equal(t, Offset("k:2:9"), *item.Offset)
		assert.Equal(t, []string{"t"}, fake.subscribed)
	})

	t.Run("message that fails to unmarshal errors", func(t *testing.T) {
		fake := &fakeConsumer{events: []kafka.Event{
			kafkaMessage("t", "k", "not-an-int", 0, 0),
		}}
		k := kafkaStreamWithFake(t, fake)

		_, err := k.Pull(&FromBeginning{Topic: "t"})
		assert.ErrorContains(t, err, "unmarshal")
	})
}

func TestKafkaPullFromOffset(t *testing.T) {
	fromOffset := func(offset string) *FromOffset {
		o := Offset(offset)
		return &FromOffset{Topic: "t", Offset: &o}
	}

	t.Run("empty topic is rejected", func(t *testing.T) {
		k := kafkaStreamWithFake(t, &fakeConsumer{})
		_, err := k.Pull(&FromOffset{Topic: ""})
		assert.ErrorIs(t, err, ErrEmptyTopic)
	})

	t.Run("malformed offset is rejected", func(t *testing.T) {
		k := kafkaStreamWithFake(t, &fakeConsumer{})
		_, err := k.Pull(fromOffset("garbage"))
		assert.ErrorContains(t, err, "parse offset")
	})

	t.Run("resumes after the given offset", func(t *testing.T) {
		fake := &fakeConsumer{events: []kafka.Event{
			kafkaMessage("t", "k", "1", 3, 8),
		}}
		k := kafkaStreamWithFake(t, fake)

		item, err := k.Pull(fromOffset("k:3:7"))
		require.NoError(t, err)
		assert.Equal(t, Offset("k:3:8"), *item.Offset)

		require.Len(t, fake.assigned, 1)
		require.Len(t, fake.assigned[0], 1)
		assert.Equal(t, int32(3), fake.assigned[0][0].Partition)
		assert.Equal(t, kafka.Offset(8), fake.assigned[0][0].Offset, "assignment starts one past the stored offset")
	})

	t.Run("consumer is cached per topic and partition", func(t *testing.T) {
		fake := &fakeConsumer{events: []kafka.Event{
			kafkaMessage("t", "k", "1", 3, 8),
			kafkaMessage("t", "k", "2", 3, 9),
		}}
		created := 0
		k := NewKafkaStream[int](kafka.ConfigMap{}, kafka.ConfigMap{}, WithSystemTime)
		k.newConsumer = func(conf *kafka.ConfigMap) (kafkaConsumer, error) {
			created++
			return fake, nil
		}

		_, err := k.Pull(fromOffset("k:3:7"))
		require.NoError(t, err)
		_, err = k.Pull(fromOffset("k:3:8"))
		require.NoError(t, err)

		assert.Equal(t, 1, created, "same topic+partition must reuse the consumer")
	})

	t.Run("rebalance events are handled before the message", func(t *testing.T) {
		partitions := []kafka.TopicPartition{{Partition: 3}}
		fake := &fakeConsumer{events: []kafka.Event{
			kafka.AssignedPartitions{Partitions: partitions},
			kafka.RevokedPartitions{Partitions: partitions},
			nil,
			kafka.NewError(kafka.ErrAllBrokersDown, "transient", false),
			kafkaMessage("t", "k", "1", 3, 8),
		}}
		k := kafkaStreamWithFake(t, fake)

		item, err := k.Pull(fromOffset("k:3:7"))
		require.NoError(t, err)
		assert.Equal(t, 1, item.Data)
		// one assignment from the offset seek plus one from the rebalance
		assert.Len(t, fake.assigned, 2)
		assert.Equal(t, 1, fake.unassigns)
	})

	t.Run("assign error during rebalance propagates", func(t *testing.T) {
		fake := &fakeConsumer{
			events: []kafka.Event{
				kafka.AssignedPartitions{Partitions: []kafka.TopicPartition{{Partition: 3}}},
			},
		}
		k := kafkaStreamWithFake(t, fake)
		// the initial Assign (offset seek) succeeds; make the next one fail
		_, err := k.consumerForTopicAndPartition("t", ptrOffset("k:3:7"))
		require.NoError(t, err)
		fake.assignErr = errors.New("assign failed")

		_, err = k.Pull(fromOffset("k:3:7"))
		assert.ErrorContains(t, err, "assign")
	})

	t.Run("unassign error during rebalance propagates", func(t *testing.T) {
		fake := &fakeConsumer{
			unassignErr: errors.New("unassign failed"),
			events: []kafka.Event{
				kafka.RevokedPartitions{},
			},
		}
		k := kafkaStreamWithFake(t, fake)
		_, err := k.Pull(fromOffset("k:3:7"))
		assert.ErrorContains(t, err, "unassign")
	})

	t.Run("unmarshal error propagates", func(t *testing.T) {
		fake := &fakeConsumer{events: []kafka.Event{
			kafkaMessage("t", "k", "nope", 3, 8),
		}}
		k := kafkaStreamWithFake(t, fake)
		_, err := k.Pull(fromOffset("k:3:7"))
		assert.ErrorContains(t, err, "unmarshal")
	})
}

func ptrOffset(s string) *Offset {
	o := Offset(s)
	return &o
}

func TestKafkaOffsetCompare(t *testing.T) {
	t.Run("orders offsets within a partition", func(t *testing.T) {
		cmp, err := KafkaOffsetCompare(Offset("k:1:5"), Offset("k:1:3"))
		require.NoError(t, err)
		assert.Positive(t, cmp)

		cmp, err = KafkaOffsetCompare(Offset("k:1:3"), Offset("k:1:5"))
		require.NoError(t, err)
		assert.Negative(t, cmp)

		cmp, err = KafkaOffsetCompare(Offset("k:1:5"), Offset("k:1:5"))
		require.NoError(t, err)
		assert.Zero(t, cmp)
	})

	t.Run("different partitions are not comparable", func(t *testing.T) {
		_, err := KafkaOffsetCompare(Offset("k:1:5"), Offset("k:2:5"))
		assert.ErrorIs(t, err, ErrOffsetNotComparable)
	})

	t.Run("malformed offsets error", func(t *testing.T) {
		_, err := KafkaOffsetCompare(Offset("garbage"), Offset("k:1:5"))
		assert.ErrorIs(t, err, ErrParsingOffsetParser)

		_, err = KafkaOffsetCompare(Offset("k:1:5"), Offset("garbage"))
		assert.ErrorIs(t, err, ErrParsingOffsetParser)
	})
}

// fakeProducer acknowledges every message on the delivery channel.
type fakeProducer struct {
	produceErr error
	messages   []*kafka.Message
}

func (f *fakeProducer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	if f.produceErr != nil {
		return f.produceErr
	}
	f.messages = append(f.messages, msg)
	deliveryChan <- msg
	return nil
}

func TestKafkaPushValidation(t *testing.T) {
	k := NewKafkaStream[int](kafka.ConfigMap{}, kafka.ConfigMap{}, WithSystemTime)

	t.Run("empty topic", func(t *testing.T) {
		err := k.Push(&Item[int]{Key: "k"})
		assert.ErrorIs(t, err, ErrEmptyTopic)
	})
	t.Run("empty key", func(t *testing.T) {
		err := k.Push(&Item[int]{Topic: "t"})
		assert.ErrorIs(t, err, ErrEmptyKey)
	})
	t.Run("offset must not be set", func(t *testing.T) {
		err := k.Push(&Item[int]{Topic: "t", Key: "k", Offset: ptrOffset("k:0:1")})
		assert.ErrorIs(t, err, ErrOffsetSetOnPush)
	})
}

func TestKafkaPushDelivery(t *testing.T) {
	t.Run("delivered message carries key, value, and event time", func(t *testing.T) {
		fake := &fakeProducer{}
		k := NewKafkaStream[int](kafka.ConfigMap{}, kafka.ConfigMap{}, WithSystemTime)
		k.newProducer = func(conf *kafka.ConfigMap) (kafkaProducer, error) {
			return fake, nil
		}

		et := EventTime(99)
		err := k.Push(&Item[int]{Topic: "t", Key: "k", Data: 7, EventTime: &et})
		require.NoError(t, err)

		require.Len(t, fake.messages, 1)
		msg := fake.messages[0]
		assert.Equal(t, []byte("k"), msg.Key)
		assert.Equal(t, []byte("7"), msg.Value)
		assert.Equal(t, time.Unix(0, 99), msg.Timestamp)
	})

	t.Run("producer is reused between pushes", func(t *testing.T) {
		created := 0
		k := NewKafkaStream[int](kafka.ConfigMap{}, kafka.ConfigMap{}, WithSystemTime)
		k.newProducer = func(conf *kafka.ConfigMap) (kafkaProducer, error) {
			created++
			return &fakeProducer{}, nil
		}

		require.NoError(t, k.Push(&Item[int]{Topic: "t", Key: "k", Data: 1}))
		require.NoError(t, k.Push(&Item[int]{Topic: "t", Key: "k", Data: 2}))
		assert.Equal(t, 1, created)
	})

	t.Run("producer initiation error propagates", func(t *testing.T) {
		k := NewKafkaStream[int](kafka.ConfigMap{}, kafka.ConfigMap{}, WithSystemTime)
		k.newProducer = func(conf *kafka.ConfigMap) (kafkaProducer, error) {
			return nil, errors.New("no broker")
		}

		err := k.Push(&Item[int]{Topic: "t", Key: "k", Data: 1})
		assert.ErrorContains(t, err, "on producer initiation")
	})

	t.Run("produce error propagates", func(t *testing.T) {
		k := NewKafkaStream[int](kafka.ConfigMap{}, kafka.ConfigMap{}, WithSystemTime)
		k.newProducer = func(conf *kafka.ConfigMap) (kafkaProducer, error) {
			return &fakeProducer{produceErr: errors.New("queue full")}, nil
		}

		err := k.Push(&Item[int]{Topic: "t", Key: "k", Data: 1})
		assert.ErrorContains(t, err, "queue full")
	})
}

func TestEnsureEventTime(t *testing.T) {
	fixed := EventTime(1000)
	k := NewKafkaStream[int](kafka.ConfigMap{}, kafka.ConfigMap{}, func() EventTime {
		return fixed
	})

	t.Run("missing event time uses system time", func(t *testing.T) {
		msg := k.ensureEventTime(&kafka.Message{}, &Item[int]{})
		assert.Equal(t, time.Unix(0, 1000), msg.Timestamp)
		assert.Equal(t, kafka.TimestampCreateTime, msg.TimestampType)
	})

	t.Run("explicit event time wins", func(t *testing.T) {
		et := EventTime(7)
		msg := k.ensureEventTime(&kafka.Message{}, &Item[int]{EventTime: &et})
		assert.Equal(t, time.Unix(0, 7), msg.Timestamp)
	})
}
