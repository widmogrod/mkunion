package stream

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/shared"
	"sync"
	"time"
)

func init() {
	RegisterOffsetCompare("k", KafkaOffsetCompare)
}

func NewKafkaStream[A any](
	consumerConfig kafka.ConfigMap,
	producerConfig kafka.ConfigMap,
	systemTime func() EventTime,
) *KafkaStream[A] {
	return &KafkaStream[A]{
		consumerConfig: consumerConfig,
		producerConfig: producerConfig,
		consumer:       &sync.Map{},
		systemTime:     systemTime,
		pullTimeoutMs:  100,
	}
}

type KafkaStream[A any] struct {
	consumerConfig kafka.ConfigMap
	consumer       *sync.Map

	producerConfig kafka.ConfigMap
	producer       *kafka.Producer
	systemTime     func() EventTime
	pullTimeoutMs  int
}

var _ Stream[any] = (*KafkaStream[any])(nil)

func (k *KafkaStream[A]) Push(x *Item[A]) error {
	if x.Topic == "" {
		return ErrEmptyTopic
	}
	if x.Key == "" {
		return ErrEmptyKey
	}
	if x.Offset.IsSet() {
		return ErrOffsetSetOnPush
	}

	data, err := shared.JSONMarshal[A](x.Data)
	if err != nil {
		return fmt.Errorf("stream.KafkaStream.Push: marshaling err; %w", err)
	}

	if k.producer == nil {
		p, err := kafka.NewProducer(&k.producerConfig)
		if err != nil {
			return fmt.Errorf("stream.KafkaStream.Push: on producer initiation; %w", err)
		}
		k.producer = p
	}

	delivery := make(chan kafka.Event, 1)
	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:       &x.Topic,
			Partition:   kafka.PartitionAny,
			Offset:      kafka.OffsetBeginning,
			Metadata:    nil,
			Error:       nil,
			LeaderEpoch: nil,
		},
		Value: data,
		Key:   []byte(x.Key),
	}

	msg = k.ensureEventTime(msg, x)

	err = k.producer.Produce(msg, delivery)
	if err != nil {
		return fmt.Errorf("stream.KafkaStream.Push: %w", err)
	}

	select {
	case ev := <-delivery:
		fmt.Printf("stream.KafkaStream.Push: delivered %s\n", ev.String())
	}

	return nil
}

func (k *KafkaStream[A]) Pull(fromOffset PullCMD) (*Item[A], error) {
	return MatchPullCMDR2(
		fromOffset,
		func(x *FromBeginning) (*Item[A], error) {
			if x.Topic == "" {
				return nil, ErrEmptyTopic
			}

			conf := k.consumerConfig
			consumer, err := kafka.NewConsumer(&conf)
			if err != nil {
				return nil, fmt.Errorf("stream.KafkaStream.Pull: on consumer initiation; %w", err)
			}
			defer consumer.Close()

			err = consumer.Subscribe(x.Topic, nil)
			if err != nil {
				return nil, fmt.Errorf("stream.KafkaStream.Pull: subscribe %w", err)
			}

			for {
				event := consumer.Poll(k.pullTimeoutMs)
				if event == nil {
					continue
				}

				switch e := event.(type) {
				case *kafka.Message:
					data, err := shared.JSONUnmarshal[A](e.Value)
					if err != nil {
						return nil, fmt.Errorf("stream.KafkaStream.Pull(FromBeginning): unmarshal %w", err)
					}

					result := &Item[A]{
						Topic:     *e.TopicPartition.Topic,
						Key:       string(e.Key),
						Data:      data,
						EventTime: MkEventTimeFromInt(e.Timestamp.UnixNano()),
						Offset:    mkOffsetFromKafkaTopicPartition(e.TopicPartition.Partition, e.TopicPartition.Offset),
					}

					return result, nil

				case kafka.Error:
					log.Errorf("stream.KafkaStream.Pull(FromBeginning): %v", e)

				default:
					fmt.Printf("Ignored: %v\n", e)
				}
			}
		},
		func(x *FromOffset) (*Item[A], error) {
			if x.Topic == "" {
				return nil, ErrEmptyTopic
			}

			log.Infof("stream.KafkaStream.Pull(FromOffset): %v", x)
			consumer, err := k.consumerForTopicAndPartition(x.Topic, x.Offset)
			if err != nil {
				return nil, err
			}

			if err != nil {
				return nil, fmt.Errorf("stream.KafkaStream.Pull(FromOffset): %w", err)
			}

			for {
				event := consumer.Poll(k.pullTimeoutMs)
				if event == nil {
					log.Printf("stream.KafkaStream.Pull(FromOffset): no event")

					continue
				}
				switch e := event.(type) {
				case *kafka.Message:
					data, err := shared.JSONUnmarshal[A](e.Value)
					if err != nil {
						return nil, fmt.Errorf("stream.KafkaStream.Pull(FromOffset): unmarshal %w", err)
					}

					result := &Item[A]{
						Topic:     *e.TopicPartition.Topic,
						Key:       string(e.Key),
						Data:      data,
						EventTime: MkEventTimeFromInt(e.Timestamp.UnixNano()),
						Offset:    mkOffsetFromKafkaTopicPartition(e.TopicPartition.Partition, e.TopicPartition.Offset),
					}

					return result, nil

				case kafka.AssignedPartitions:
					// Handle new assignments
					fmt.Println("AssignedPartitions:", e.Partitions)
					// Here, implement your custom logic for partition assignment.
					// For this example, we will just use the assigned partitions as is.
					err := consumer.Assign(e.Partitions)
					if err != nil {
						return nil, fmt.Errorf("stream.KafkaStream.Pull(FromOffset): assign %w", err)
					}

				case kafka.RevokedPartitions:
					// Handle partition revocation
					fmt.Println("RevokedPartitions:", e.Partitions)
					err := consumer.Unassign()
					if err != nil {
						return nil, fmt.Errorf("stream.KafkaStream.Pull(FromOffset): unassign %w", err)
					}

				case kafka.Error:
					// Errors should generally be considered
					// informational, the client will try to
					// automatically recover.
					// But in this example we choose to terminate
					// the application if all brokers are down.
					log.Errorf("stream.KafkaStream.Pull(FromOffset): %v", e)
					//return nil, fmt.Errorf("stream.KafkaStream.Pull(FromOffset): %w", e)
				}
			}
		},
	)
}

func (k *KafkaStream[A]) consumerForTopic(topic Topic) (*kafka.Consumer, error) {
	key := fmt.Sprintf("t:%s", topic)
	value, ok := k.consumer.Load(key)
	if ok {
		return value.(*kafka.Consumer), nil
	}

	consumer, err := kafka.NewConsumer(&k.consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("stream.KafkaStream.Pull: on consumer initiation; %w", err)
	}

	err = consumer.Subscribe(topic, nil)
	if err != nil {
		return nil, fmt.Errorf("stream.KafkaStream.Pull: subscribe %w", err)
	}

	k.consumer.Store(key, consumer)
	return consumer, nil
}

func (k *KafkaStream[A]) consumerForTopicAndPartition(topic Topic, offset *Offset) (*kafka.Consumer, error) {
	kpartition, koffset, err := parseOffsetToKafka(offset)
	if err != nil {
		return nil, fmt.Errorf("stream.KafkaStream.consumerForTopicAndPartition: parse offset %w", err)
	}

	key := fmt.Sprintf("tp:%s:%d", topic, kpartition)
	value, ok := k.consumer.Load(key)
	if ok {
		return value.(*kafka.Consumer), nil
	}

	consumer, err := kafka.NewConsumer(&k.consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("stream.KafkaStream.consumerForTopicAndPartition: on consumer initiation (%s); %w", key, err)
	}

	err = consumer.Assign([]kafka.TopicPartition{
		{
			Topic:     &topic,
			Partition: kpartition,
			Offset:    koffset + 1,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("stream.KafkaStream.consumerForTopicAndPartition: Assign (%s); %w", key, err)
	}

	k.consumer.Store(key, consumer)
	return consumer, nil
}

func (k *KafkaStream[A]) ensureEventTime(msg *kafka.Message, x *Item[A]) *kafka.Message {
	if x.EventTime == nil {
		msg.Timestamp = time.Unix(0, k.systemTime())
		msg.TimestampType = kafka.TimestampCreateTime
	} else {
		msg.Timestamp = time.Unix(0, *x.EventTime)
		msg.TimestampType = kafka.TimestampCreateTime
	}

	return msg
}

func KafkaOffsetCompare(a, b Offset) (int8, error) {
	partitionA, offsetA, err := parseOffsetToKafka(&a)
	if err != nil {
		return 0, fmt.Errorf("stream.KafkaOffsetCompare: offset at first position; %w; %w", err, ErrParsingOffsetParser)
	}

	partitionB, offsetB, err := parseOffsetToKafka(&b)
	if err != nil {
		return 0, fmt.Errorf("stream.KafkaOffsetCompare: offset at second position; %w; %w", err, ErrParsingOffsetParser)
	}

	if partitionA != partitionB {
		return 0, fmt.Errorf("stream.KafkaOffsetCompare: partition mismatch; %w; %w", err, ErrOffsetNotComparable)
	}

	return int8(offsetA - offsetB), nil
}

func mkOffsetFromKafkaTopicPartition(partition int32, offset kafka.Offset) *Offset {
	result := Offset(fmt.Sprintf("k:%d:%d", partition, offset))
	return &result
}

func parseOffsetToKafka(offset *Offset) (int32, kafka.Offset, error) {
	var partition int32
	var offsetValue kafka.Offset
	_, err := fmt.Sscanf(string(*offset), "k:%d:%d", &partition, &offsetValue)
	if err != nil {
		return 0, 0, fmt.Errorf("stream.parseOffsetToKafka: %w", err)
	}

	return partition, offsetValue, nil
}
