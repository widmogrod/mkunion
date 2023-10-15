package taskqueue

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/localstackutil"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
	"github.com/widmogrod/mkunion/x/workflow"
	"os"
	"testing"
	"time"
)

var functions = map[string]workflow.Function{
	"concat": func(body *workflow.FunctionInput) (*workflow.FunctionOutput, error) {
		args := body.Args
		a, ok := schema.As[string](args[0])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}
		b, ok := schema.As[string](args[1])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}

		return &workflow.FunctionOutput{
			Result: schema.MkString(a + b),
		}, nil
	},
}

func TestTaskQueue(t *testing.T) {
	schema.RegisterRules([]schema.RuleMatcher{
		schema.WhenPath([]string{"*", "BaseState"}, schema.UseStruct(workflow.BaseState{})),
	})
	schema.RegisterUnionTypes(
		workflow.StateSchemaDef(),
	)

	program := &workflow.Flow{
		Name: "hello_world_flow",
		Arg:  "input",
		Body: []workflow.Expr{
			&workflow.Assign{
				ID:    "assign1",
				VarOk: "res",
				Val: &workflow.Apply{ID: "apply1", Name: "concat", Args: []workflow.Reshaper{
					&workflow.SetValue{Value: schema.MkString("hello ")},
					&workflow.GetValue{Path: "input"},
				}},
			},
			&workflow.End{
				ID:     "end1",
				Result: &workflow.GetValue{Path: "res"},
			},
		},
	}

	di := &workflow.DI{
		FindWorkflowF: func(flowID string) (*workflow.Flow, error) {
			return program, nil
		},
		FindFunctionF: func(funcID string) (workflow.Function, error) {
			if fn, ok := functions[funcID]; ok {
				return fn, nil
			}

			return nil, fmt.Errorf("function %s not found", funcID)
		},
	}

	desc := &Description{
		Change: []string{"create", "update"},
		Entity: "process",
		//Filter: "Data.#.Retried > Data.#.Error.MaxRetries",
	}

	store := schemaless.NewInMemoryRepository()
	repo := typedful.NewTypedRepository[workflow.State](store)
	proc := &FunctionProcessor[workflow.State]{
		F: func(task Task[schemaless.Record[workflow.State]]) {
			t.Logf("task: %#v \n", task)
			work := workflow.NewMachine(di, task.Data.Data)
			//err := work.Handle(&workflow.Retry{})
			err := work.Handle(&workflow.Run{})
			if err != nil {
				t.Logf("err: %s", err)
				return
			}

			err = repo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
				ID:      task.Data.ID,
				Data:    work.State(),
				Version: task.Data.Version,
			}))

			if err != nil {
				if errors.Is(err, schemaless.ErrVersionConflict) {
					// make it configurable, but by default we should
					// just ignore conflicts, since that means we may have duplicate,
					// or some other process already update it.
					// assuming that queue is populated from stream of changes
					// it such case (there was update) new message with new version
					// will land in queue soon (if it pass selector)
					fmt.Println("version conflict, ignoring")
				} else {
					panic(err)
				}
			}
		},
	}

	//queue := NewInMemoryQueue[schemaless.Record[schema.Schema]]()

	awsconf, err := localstackutil.LoadLocalStackAwsConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	awssqs := sqs.NewFromConfig(awsconf)

	queueURL := os.Getenv("AWS_SQS_QUEUE_URL")
	if queueURL == "" {
		t.Skip(`Skipping test because:
- AWS_SQS_QUEUE_URL is not set. 
- Assuming SQS is not setup. 

To run this test, please set AWS_SQS_QUEUE_URL to the address of your AWS SQS instance, like:
	export AWS_SQS_QUEUE_URL=http://localhost:4566/000000000000/localstack-queue
`)
	}

	queue := NewSQSQueue(awssqs, queueURL)
	//queue := NewInMemoryQueue[schemaless.Record[schema.Schema]]()

	ctx := context.Background()
	tq2 := NewTaskQueue(desc, queue, store, proc)
	go func() {
		err := tq2.RunSelector(ctx)
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		err := tq2.RunProcessor(ctx)
		if err != nil {
			panic(err)
		}
	}()

	work := workflow.NewMachine(di, nil)
	err = work.Handle(&workflow.Run{
		Flow:  &workflow.FlowRef{FlowID: "hello_world_flow"},
		Input: schema.MkString("world"),
	})
	assert.NoError(t, err)

	newState := work.State()
	err = repo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
		ID:   "1",
		Type: "process",
		Data: newState,
	}))
	assert.NoError(t, err)

	time.Sleep(2 * time.Second)
}
