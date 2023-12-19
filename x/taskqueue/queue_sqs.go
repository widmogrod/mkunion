package taskqueue

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

func NewSQSQueue(c *sqs.Client, queueURL string) *SQSQueue[schemaless.Record[schema.Schema]] {
	return &SQSQueue[schemaless.Record[schema.Schema]]{
		client:   c,
		queueURL: queueURL,
	}
}

// SQSQueue is a queue that uses AWS SQS as a backend.
type SQSQueue[T any] struct {
	client   *sqs.Client
	queueURL string
}

var _ Queuer[any] = (*SQSQueue[any])(nil)

func (queue *SQSQueue[T]) Push(ctx context.Context, task Task[T]) error {
	body, err := shared.JSONMarshal[T](task.Data)
	if err != nil {
		return fmt.Errorf("sqsQueue.Push: JSONMarshal=%w", err)
	}

	bodyStr := string(body)

	var messageGroupId *string
	if groupId, ok := task.Meta["SQS.MessageGroupId"]; ok {
		if groupId != "" {
			messageGroupId = &groupId
		}
	}

	msg := &sqs.SendMessageInput{
		MessageBody:            &bodyStr,
		QueueUrl:               &queue.queueURL,
		MessageGroupId:         messageGroupId,
		MessageDeduplicationId: &task.ID,
	}

	output, err := queue.client.SendMessage(ctx, msg)
	if err != nil {
		return fmt.Errorf("sqsQueue.Push: SendMessage=%w", err)
	}

	_ = output
	_ = output.MessageId
	_ = output.SequenceNumber

	return nil
}

func (queue *SQSQueue[T]) Pop(ctx context.Context) ([]Task[T], error) {
	output, err := queue.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:                &queue.queueURL,
		ReceiveRequestAttemptId: nil,
	})
	if err != nil {
		return nil, fmt.Errorf("sqsQueue.Pop: ReceiveMessage=%w", err)
	}

	var tasks []Task[T]
	for _, message := range output.Messages {
		data, err := shared.JSONUnmarshal[T]([]byte(*message.Body))
		if err != nil {
			return nil, fmt.Errorf("sqsQueue.Pop: JSONUnmarshal=%w", err)
		}

		task := Task[T]{
			ID:   *message.MessageId,
			Data: data,
			Meta: map[string]string{
				"SQS.ReceiptHandle": *message.ReceiptHandle,
			},
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (queue *SQSQueue[T]) Delete(ctx context.Context, tasks []Task[schemaless.Record[schema.Schema]]) error {
	if len(tasks) == 0 {
		return nil
	}

	var entries []types.DeleteMessageBatchRequestEntry
	for _, task := range tasks {
		receiptHandle, ok := task.Meta["SQS.ReceiptHandle"]
		if !ok {
			return fmt.Errorf("sqsQueue.Delete: missing SQS.ReceiptHandle in taskID=%s", task.ID)
		}
		entries = append(entries, types.DeleteMessageBatchRequestEntry{
			Id:            &task.ID,
			ReceiptHandle: &receiptHandle,
		})
	}
	_, err := queue.client.DeleteMessageBatch(ctx, &sqs.DeleteMessageBatchInput{
		Entries:  entries,
		QueueUrl: &queue.queueURL,
	})
	if err != nil {
		return fmt.Errorf("sqsQueue.Delete: DeleteMessageBatch=%w", err)
	}

	return nil
}
