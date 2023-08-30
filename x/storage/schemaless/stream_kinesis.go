package schemaless

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
	"sync"
	"time"
)

func NewKinesisStream(k *kinesis.Client, streamName string) *KinesisStream {
	ctx := context.Background()
	stream, err := k.DescribeStream(ctx, &kinesis.DescribeStreamInput{
		StreamName: aws.String(streamName),
		Limit:      aws.Int32(1000),
	})
	if err != nil {
		panic(err)
	}

	return &KinesisStream{
		kinesis:    k,
		stream:     stream,
		streamName: streamName,
	}
}

type KinesisStream struct {
	kinesis    *kinesis.Client
	stream     *kinesis.DescribeStreamOutput
	streamName string

	lock        sync.RWMutex
	subscribers []func(Change[schema.Schema])
	done        []chan struct{}
	once        sync.Once
}

func (s *KinesisStream) Pull() chan Change[schema.Schema] {
	result := make(chan Change[schema.Schema])

	ctx := context.Background()
	for _, shard := range s.stream.StreamDescription.Shards {
		var shardIterator *string = nil
		if shardIterator == nil {
			it := &kinesis.GetShardIteratorInput{
				ShardId:           shard.ShardId,
				StreamName:        aws.String(s.streamName),
				ShardIteratorType: types.ShardIteratorTypeLatest,
				//ShardIteratorType:      types.ShardIteratorTypeAtSequenceNumber,
				//StartingSequenceNumber: shard.SequenceNumberRange.StartingSequenceNumber,
			}
			iterator, err := s.kinesis.GetShardIterator(ctx, it)
			if err != nil {
				panic(err)
			}
			shardIterator = iterator.ShardIterator
		}

		go s.processShard(ctx, shardIterator, result)
	}

	return result
}

func (s *KinesisStream) processShard(ctx context.Context, shardIterator *string, resultC chan Change[schema.Schema]) {
	lastRequest := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			// continue
		}

		if diff := time.Now().Sub(lastRequest); diff < time.Second/5 {
			//log.Debugf("🗺Sleeping for %s", time.Second/5-diff)
			time.Sleep(time.Second/5 - diff)
		}

		lastRequest = time.Now()
		records, err := s.kinesis.GetRecords(ctx, &kinesis.GetRecordsInput{
			ShardIterator: shardIterator,
			Limit:         aws.Int32(100),
			//StreamARN:     shard.ShardId,
		})
		if err != nil {
			// check if error is ProvisionedThroughputExceededException
			// if so, sleep for 1 second and try again
			var ptee *types.ProvisionedThroughputExceededException
			if ok := errors.As(err, &ptee); ok {
				//log.Warnln("kinesis.GetRecords: SLEEP(B) ProvisionedThroughputExceededException:", err)
				time.Sleep(5 * time.Second)
				continue
			}

			log.Errorln("kinesis.GetRecords: ", err)
			panic(err)
		}

		for _, record := range records.Records {
			//log.Infoln("🗺Kaflka Stream Record:", string(record.Data))
			schemed, err := schema.FromJSON(record.Data)
			if err != nil {
				panic(err)
			}

			// potentially change, can be just state. And data pipeline can detect it
			// groub by  key
			// no initial state, not created
			// ther is state, and there is new - updated
			// there is state, with deleted flag set to true, delete
			// this implice that soft delete, or other options can happen.
			// but when is deleted, key could be closed? this would require some instruction
			// or maybe as with imposibility of distributed consensus,
			// data flush is important, windowing and triggers, etc?
			result := Change[schema.Schema]{
				Before:  nil,
				After:   nil,
				Deleted: false,
			}

			switch schema.AsDefault[string](schema.Get(schemed, "eventName"), "") {
			case "MODIFY":
				// has both NewImage and OldImage
				old := schema.Get(schemed, "dynamodb.OldImage")
				before, err := s.toTyped(old)
				if err != nil {
					panic(err)
				}
				result.Before = &before

				new := schema.Get(schemed, "dynamodb.NewImage")
				after, err := s.toTyped(new)
				if err != nil {
					panic(err)
				}
				result.After = &after

			case "INSERT":
				// has only NewImage
				new := schema.Get(schemed, "dynamodb.NewImage")
				after, err := s.toTyped(new)
				if err != nil {
					panic(err)
				}
				result.After = &after
			case "REMOVE":
				// has only OldImage
				old := schema.Get(schemed, "dynamodb.OldImage")
				before, err := s.toTyped(old)
				if err != nil {
					panic(err)
				}
				result.Before = &before
				result.Deleted = true

			default:
				panic(fmt.Errorf("unknown event name: %s", schema.AsDefault[string](schema.Get(schemed, "eventName"), "")))
			}

			resultC <- result
		}

		if records.NextShardIterator == nil {
			log.Infoln("🗺ShardIterator is nil, exiting")
			return
		}
		shardIterator = records.NextShardIterator
	}
}

func (s *KinesisStream) Subscribe(ctx context.Context, fromOffset int, f func(Change[schema.Schema])) error {
	done := make(chan struct{})

	//log.Errorf("🗺store.KinesisStream SUBSCRIBE")
	s.lock.Lock()
	s.subscribers = append(s.subscribers, f)
	s.done = append(s.done, done)
	s.lock.Unlock()

	<-done

	return nil
}

func (s *KinesisStream) Process() {
	defer func() {
		s.lock.RLock()
		for _, done := range s.done {
			done <- struct{}{}
		}
		s.lock.RUnlock()
	}()

	//log.Errorf("🗺store.KinesisStream PROCESS")
	//defer log.Errorf("🗺store.KinesisStream PROCESS END")
	for result := range s.Pull() {
		s.lock.RLock()
		//log.Errorf("🗺store.KinesisStream subscribers: %d %#v \n", len(s.subscribers), result)
		for _, f := range s.subscribers {
			f(result)
		}
		s.lock.RUnlock()
	}
}

func (s *KinesisStream) toTyped(record schema.Schema) (Record[schema.Schema], error) {
	normalised, err := schema.UnwrapDynamoDB(record)
	if err != nil {
		data, _ := schema.ToJSON(record)
		log.Errorln("🗺store.KinesisStream corrupted (1) record:", string(data), err)
		return Record[schema.Schema]{}, fmt.Errorf("store.KinesisStream unwrap DynamoDB record: %v; %w", record, err)
	}

	typed, err := schema.ToGoG[*Record[schema.Schema]](normalised, WithOnlyRecordSchemaOptions)
	if err != nil {
		data, _ := schema.ToJSON(record)
		log.Errorln("🗺store.KinesisStream corrupted (2) record:", string(data), err)
		return Record[schema.Schema]{}, fmt.Errorf("store.KinesisStream convert record: %v; %w", record, err)
	}

	return *typed, nil
}
