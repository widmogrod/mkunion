package taskqueue

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

type fakeSQS struct {
	sendIn  *sqs.SendMessageInput
	sendErr error

	receiveOut *sqs.ReceiveMessageOutput
	receiveErr error

	deleteIn  *sqs.DeleteMessageBatchInput
	deleteErr error
}

func (f *fakeSQS) SendMessage(_ context.Context, params *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	f.sendIn = params
	return &sqs.SendMessageOutput{}, f.sendErr
}

func (f *fakeSQS) ReceiveMessage(_ context.Context, params *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return f.receiveOut, f.receiveErr
}

func (f *fakeSQS) DeleteMessageBatch(_ context.Context, params *sqs.DeleteMessageBatchInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageBatchOutput, error) {
	f.deleteIn = params
	return &sqs.DeleteMessageBatchOutput{}, f.deleteErr
}

func sqsRecord(id string) rec {
	return rec{ID: id, Type: "process", Data: schema.MkString("payload")}
}

func TestSQSQueuePush(t *testing.T) {
	t.Run("marshals the record and dedupes on task id", func(t *testing.T) {
		fake := &fakeSQS{}
		queue := NewSQSQueue(fake, "http://queue")

		data := sqsRecord("1")
		err := queue.Push(context.Background(), Task[rec]{ID: "t-1", Data: &data})
		require.NoError(t, err)

		require.NotNil(t, fake.sendIn)
		assert.Equal(t, "http://queue", *fake.sendIn.QueueUrl)
		assert.Equal(t, "t-1", *fake.sendIn.MessageDeduplicationId)
		assert.Nil(t, fake.sendIn.MessageGroupId)
		assert.Contains(t, *fake.sendIn.MessageBody, `"ID":"1"`)
	})

	t.Run("group id from task meta", func(t *testing.T) {
		fake := &fakeSQS{}
		queue := NewSQSQueue(fake, "http://queue")

		data := sqsRecord("1")
		err := queue.Push(context.Background(), Task[rec]{
			ID:   "t-1",
			Data: &data,
			Meta: map[string]string{"SQS.MessageGroupId": "group-1"},
		})
		require.NoError(t, err)
		require.NotNil(t, fake.sendIn.MessageGroupId)
		assert.Equal(t, "group-1", *fake.sendIn.MessageGroupId)
	})

	t.Run("empty group id in meta is ignored", func(t *testing.T) {
		fake := &fakeSQS{}
		queue := NewSQSQueue(fake, "http://queue")

		data := sqsRecord("1")
		err := queue.Push(context.Background(), Task[rec]{
			ID:   "t-1",
			Data: &data,
			Meta: map[string]string{"SQS.MessageGroupId": ""},
		})
		require.NoError(t, err)
		assert.Nil(t, fake.sendIn.MessageGroupId)
	})

	t.Run("task without data sends an empty body", func(t *testing.T) {
		fake := &fakeSQS{}
		queue := NewSQSQueue(fake, "http://queue")

		err := queue.Push(context.Background(), Task[rec]{ID: "t-1"})
		require.NoError(t, err)
		assert.Equal(t, "", *fake.sendIn.MessageBody)
	})

	t.Run("send failure propagates", func(t *testing.T) {
		fake := &fakeSQS{sendErr: errors.New("throttled")}
		queue := NewSQSQueue(fake, "http://queue")

		data := sqsRecord("1")
		err := queue.Push(context.Background(), Task[rec]{ID: "t-1", Data: &data})
		assert.ErrorContains(t, err, "throttled")
	})
}

func TestSQSQueuePop(t *testing.T) {
	t.Run("messages decode into tasks with receipt handles", func(t *testing.T) {
		body, err := shared.JSONMarshal[rec](sqsRecord("1"))
		require.NoError(t, err)

		fake := &fakeSQS{receiveOut: &sqs.ReceiveMessageOutput{
			Messages: []types.Message{{
				MessageId:     aws.String("m-1"),
				ReceiptHandle: aws.String("rh-1"),
				Body:          aws.String(string(body)),
			}},
		}}
		queue := NewSQSQueue(fake, "http://queue")

		tasks, err := queue.Pop(context.Background())
		require.NoError(t, err)
		require.Len(t, tasks, 1)
		assert.Equal(t, "m-1", tasks[0].ID)
		assert.Equal(t, "rh-1", tasks[0].Meta["SQS.ReceiptHandle"])
		require.NotNil(t, tasks[0].Data)
		assert.Equal(t, "1", tasks[0].Data.ID)
	})

	t.Run("receive failure propagates", func(t *testing.T) {
		fake := &fakeSQS{receiveErr: errors.New("gone")}
		queue := NewSQSQueue(fake, "http://queue")

		_, err := queue.Pop(context.Background())
		assert.ErrorContains(t, err, "gone")
	})

	t.Run("malformed message body errors", func(t *testing.T) {
		fake := &fakeSQS{receiveOut: &sqs.ReceiveMessageOutput{
			Messages: []types.Message{{
				MessageId:     aws.String("m-1"),
				ReceiptHandle: aws.String("rh-1"),
				Body:          aws.String("{broken"),
			}},
		}}
		queue := NewSQSQueue(fake, "http://queue")

		_, err := queue.Pop(context.Background())
		assert.ErrorContains(t, err, "JSONUnmarshal")
	})
}

func TestSQSQueueDelete(t *testing.T) {
	t.Run("empty task list is a no-op", func(t *testing.T) {
		fake := &fakeSQS{}
		queue := NewSQSQueue(fake, "http://queue")

		require.NoError(t, queue.Delete(context.Background(), nil))
		assert.Nil(t, fake.deleteIn)
	})

	t.Run("deletes by receipt handle", func(t *testing.T) {
		fake := &fakeSQS{}
		queue := NewSQSQueue(fake, "http://queue")

		err := queue.Delete(context.Background(), []Task[rec]{
			{ID: "t-1", Meta: map[string]string{"SQS.ReceiptHandle": "rh-1"}},
		})
		require.NoError(t, err)
		require.NotNil(t, fake.deleteIn)
		require.Len(t, fake.deleteIn.Entries, 1)
		assert.Equal(t, "rh-1", *fake.deleteIn.Entries[0].ReceiptHandle)
	})

	t.Run("missing receipt handle is rejected", func(t *testing.T) {
		queue := NewSQSQueue(&fakeSQS{}, "http://queue")

		err := queue.Delete(context.Background(), []Task[rec]{{ID: "t-1"}})
		assert.ErrorContains(t, err, "missing SQS.ReceiptHandle")
	})

	t.Run("delete failure propagates", func(t *testing.T) {
		fake := &fakeSQS{deleteErr: errors.New("gone")}
		queue := NewSQSQueue(fake, "http://queue")

		err := queue.Delete(context.Background(), []Task[rec]{
			{ID: "t-1", Meta: map[string]string{"SQS.ReceiptHandle": "rh-1"}},
		})
		assert.ErrorContains(t, err, "gone")
	})
}

// keep the schemaless import used even if cases change
var _ = schemaless.Record[schema.Schema]{}
