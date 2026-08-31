package schemaless

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
)

// fakeKinesis replays scripted GetRecords responses; after the script it
// keeps returning an empty page with a nil iterator, which ends the shard.
type fakeKinesis struct {
	describeErr error
	iteratorErr error
	records     []recordsReply
	calls       int
}

type recordsReply struct {
	out *kinesis.GetRecordsOutput
	err error
}

func (f *fakeKinesis) DescribeStream(_ context.Context, _ *kinesis.DescribeStreamInput, _ ...func(*kinesis.Options)) (*kinesis.DescribeStreamOutput, error) {
	if f.describeErr != nil {
		return nil, f.describeErr
	}
	return &kinesis.DescribeStreamOutput{
		StreamDescription: &types.StreamDescription{
			Shards: []types.Shard{{ShardId: aws.String("shard-1")}},
		},
	}, nil
}

func (f *fakeKinesis) GetShardIterator(_ context.Context, _ *kinesis.GetShardIteratorInput, _ ...func(*kinesis.Options)) (*kinesis.GetShardIteratorOutput, error) {
	if f.iteratorErr != nil {
		return nil, f.iteratorErr
	}
	return &kinesis.GetShardIteratorOutput{ShardIterator: aws.String("it-0")}, nil
}

func (f *fakeKinesis) GetRecords(_ context.Context, _ *kinesis.GetRecordsInput, _ ...func(*kinesis.Options)) (*kinesis.GetRecordsOutput, error) {
	if f.calls >= len(f.records) {
		return &kinesis.GetRecordsOutput{}, nil // nil NextShardIterator ends the shard
	}
	reply := f.records[f.calls]
	f.calls++
	return reply.out, reply.err
}

// dynamoImage is the DynamoDB-JSON form of a Record, the way DynamoDB
// streams deliver it through Kinesis.
func dynamoImage(id string) map[string]any {
	return map[string]any{
		"ID":      map[string]any{"S": id},
		"Type":    map[string]any{"S": "test"},
		"Version": map[string]any{"N": "1"},
		"Data": map[string]any{"M": map[string]any{
			"$type":         map[string]any{"S": "schema.String"},
			"schema.String": map[string]any{"S": "payload-" + id},
		}},
	}
}

func streamEvent(t *testing.T, eventName string, images map[string]any) types.Record {
	t.Helper()
	payload := map[string]any{
		"eventName": eventName,
		"dynamodb":  images,
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)
	return types.Record{Data: data}
}

func newTestKinesisStream(fake *fakeKinesis) *KinesisStream {
	s := NewKinesisStream(fake, "stream")
	s.throttleBackoff = time.Millisecond
	return s
}

func collectChanges(t *testing.T, s *KinesisStream, want int) []Change[schema.Schema] {
	t.Helper()
	out := s.Pull()
	var changes []Change[schema.Schema]
	timeout := time.After(5 * time.Second)
	for len(changes) < want {
		select {
		case c := <-out:
			changes = append(changes, c)
		case <-timeout:
			t.Fatalf("timed out after %d of %d changes", len(changes), want)
		}
	}
	return changes
}

func TestKinesisProcessShard(t *testing.T) {
	t.Run("INSERT, MODIFY, and REMOVE map to changes", func(t *testing.T) {
		fake := &fakeKinesis{records: []recordsReply{
			{out: &kinesis.GetRecordsOutput{
				NextShardIterator: aws.String("it-1"),
				Records: []types.Record{
					streamEvent(t, "INSERT", map[string]any{"NewImage": dynamoImage("1")}),
					streamEvent(t, "MODIFY", map[string]any{
						"OldImage": dynamoImage("1"),
						"NewImage": dynamoImage("2"),
					}),
					streamEvent(t, "REMOVE", map[string]any{"OldImage": dynamoImage("2")}),
				},
			}},
		}}
		s := newTestKinesisStream(fake)

		changes := collectChanges(t, s, 3)

		insert := changes[0]
		require.NotNil(t, insert.After)
		assert.Nil(t, insert.Before)
		assert.False(t, insert.Deleted)
		assert.Equal(t, "1", insert.After.ID)
		assert.Equal(t, schema.MkString("payload-1"), insert.After.Data)

		modify := changes[1]
		require.NotNil(t, modify.Before)
		require.NotNil(t, modify.After)
		assert.Equal(t, "1", modify.Before.ID)
		assert.Equal(t, "2", modify.After.ID)

		remove := changes[2]
		require.NotNil(t, remove.Before)
		assert.Nil(t, remove.After)
		assert.True(t, remove.Deleted)
	})

	t.Run("throughput throttling retries instead of failing", func(t *testing.T) {
		fake := &fakeKinesis{records: []recordsReply{
			{err: &types.ProvisionedThroughputExceededException{}},
			{out: &kinesis.GetRecordsOutput{
				NextShardIterator: aws.String("it-1"),
				Records: []types.Record{
					streamEvent(t, "INSERT", map[string]any{"NewImage": dynamoImage("1")}),
				},
			}},
		}}
		s := newTestKinesisStream(fake)

		changes := collectChanges(t, s, 1)
		assert.Equal(t, "1", changes[0].After.ID)
		assert.GreaterOrEqual(t, fake.calls, 2, "the throttled call must be retried")
	})

	t.Run("shard ends when the iterator is exhausted", func(t *testing.T) {
		fake := &fakeKinesis{}
		s := newTestKinesisStream(fake)

		done := make(chan struct{})
		go func() {
			s.processShard(context.Background(), aws.String("it-0"), make(chan Change[schema.Schema], 1))
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("processShard did not stop on a nil NextShardIterator")
		}
	})

	t.Run("context cancellation stops the shard", func(t *testing.T) {
		// an endless supply of empty pages keeps the loop alive
		fake := &fakeKinesis{}
		s := newTestKinesisStream(fake)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan struct{})
		go func() {
			s.processShard(ctx, aws.String("it-0"), make(chan Change[schema.Schema], 1))
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("processShard ignored the cancelled context")
		}
	})

	t.Run("unknown event names panic", func(t *testing.T) {
		fake := &fakeKinesis{records: []recordsReply{
			{out: &kinesis.GetRecordsOutput{
				Records: []types.Record{
					streamEvent(t, "MYSTERY", map[string]any{}),
				},
			}},
		}}
		s := newTestKinesisStream(fake)

		assert.Panics(t, func() {
			s.processShard(context.Background(), aws.String("it-0"), make(chan Change[schema.Schema], 1))
		})
	})

	t.Run("malformed record payload panics", func(t *testing.T) {
		fake := &fakeKinesis{records: []recordsReply{
			{out: &kinesis.GetRecordsOutput{
				Records: []types.Record{{Data: []byte("{broken")}},
			}},
		}}
		s := newTestKinesisStream(fake)

		assert.Panics(t, func() {
			s.processShard(context.Background(), aws.String("it-0"), make(chan Change[schema.Schema], 1))
		})
	})

	t.Run("non-throttling client error panics", func(t *testing.T) {
		fake := &fakeKinesis{records: []recordsReply{
			{err: errors.New("aws down")},
		}}
		s := newTestKinesisStream(fake)

		assert.Panics(t, func() {
			s.processShard(context.Background(), aws.String("it-0"), make(chan Change[schema.Schema], 1))
		})
	})
}

func TestNewKinesisStreamDescribeFailure(t *testing.T) {
	assert.Panics(t, func() {
		NewKinesisStream(&fakeKinesis{describeErr: errors.New("no stream")}, "stream")
	})
}

func TestKinesisToTypedErrors(t *testing.T) {
	s := newTestKinesisStream(&fakeKinesis{})

	t.Run("value that is not DynamoDB-wrapped errors", func(t *testing.T) {
		_, err := s.toTyped(schema.MkInt(1))
		assert.ErrorContains(t, err, "unwrap DynamoDB")
	})

	t.Run("wrapped value that is not a record errors", func(t *testing.T) {
		// valid DynamoDB wrapping, but Version has the wrong type
		bad := schema.MkMap(
			schema.MkField("ID", schema.MkMap(
				schema.MkField("S", schema.MkString("1")),
			)),
			schema.MkField("Version", schema.MkMap(
				schema.MkField("S", schema.MkString("not-a-number")),
			)),
		)
		_, err := s.toTyped(bad)
		assert.ErrorContains(t, err, "convert record")
	})
}
