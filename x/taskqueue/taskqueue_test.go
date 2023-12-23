package taskqueue

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
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
		GenerateRunIDF: func() string {
			return "run_id" + time.Now().String()
		},
	}

	// every time we have workflow with error,
	// that was not retried more than MaxRetries option
	// we will try recover it
	desc := &Description{
		Change: []string{"create", "update"},
		Entity: "process",
		//Filter: `Data[*]["workflow.Error"].Retried < Data["workflow.Error"].BaseState.MaxRetries`,
		//Filter: `Data #= "workflow.Error" AND Data[*].Retried < Data[*].BaseState.MaxRetries`,
	}

	// every time we have delayed workflow,
	// that is ready to run, we will run it
	desc = &Description{
		Change: []string{"create"},
		Entity: "process",
		Filter: `Data["workflow.Scheduled"].ExpectedRunTimestamp <= :now 
AND Data["workflow.Scheduled"].ExpectedRunTimestamp > 0
AND Version = 1`,
		//Filter: `Data[*]["workflow.Scheduled"].RunOption["workflow.DelayRun"].DelayBySeconds > 0`,
		//Filter: `Data #= "workflow.Scheduled" && Data[*].RunOption.Delayed > 0`,
	}

	store := schemaless.NewInMemoryRepository[schema.Schema]()
	stream := typedful.NewTypedAppendLog[workflow.State](store.AppendLog())

	repo := typedful.NewTypedRepository[workflow.State](store)
	proc := &FunctionProcessor[schemaless.Record[workflow.State]]{
		F: func(task Task[schemaless.Record[workflow.State]]) {
			//t.Logf("task id: %s \n", task.ID)
			t.Logf("data id: %s \n", task.Data.ID)
			t.Logf("version: %d \n", task.Data.Version)
			work := workflow.NewMachine(di, task.Data.Data)
			err := work.Handle(&workflow.Run{})
			//err := work.Handle(&workflow.TryRecover{})
			if err != nil {
				t.Logf("err: %s", err)
				return
			}

			newState := work.State()
			//d, _ := schema.ToJSON(schema.FromPrimitiveGo(newState))
			//t.Logf("newState: %s", string(d))

			saving := []schemaless.Record[workflow.State]{
				{
					ID:      task.Data.ID,
					Data:    newState,
					Type:    task.Data.Type,
					Version: task.Data.Version,
				},
			}

			if next := workflow.ScheduleNext(newState, di); next != nil {
				//d, _ := schema.ToJSON(schema.FromPrimitiveGo(next))
				//t.Logf("next: %s", string(d))
				work := workflow.NewMachine(di, nil)
				err := work.Handle(next)
				if err != nil {
					t.Logf("err: %s", err)
					return
				}

				t.Logf("next id=%s", workflow.GetRunID(work.State()))
				//d, _ = schema.ToJSON(schema.FromPrimitiveGo(work.State()))
				//t.Logf("nextState: %s", string(d))

				saving = append(saving, schemaless.Record[workflow.State]{
					ID:   workflow.GetRunID(work.State()),
					Type: task.Data.Type,
					Data: work.State(),
				})
			}

			err = repo.UpdateRecords(schemaless.Save(saving...))
			if err != nil {
				if errors.Is(err, schemaless.ErrVersionConflict) {
					// make it configurable, but by default we should
					// just ignore conflicts, since that means we may have duplicate,
					// or some other process already update it.
					// assuming that queue is populated from stream of changes
					// it such case (there was update) new message with new version
					// will land in queue soon (if it pass selector)
					t.Log("version conflict, ignoring")
					t.Logf("err: %s", err)
				} else {
					panic(err)
				}
			}
		},
	}

	//awsconf, err := localstackutil.LoadLocalStackAwsConfig(context.Background())
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//awssqs := sqs.NewFromConfig(awsconf)

	queueURL := os.Getenv("AWS_SQS_QUEUE_URL")
	if queueURL == "" {
		t.Skip(`Skipping test because:
- AWS_SQS_QUEUE_URL is not set. 
- Assuming SQS is not setup. 

To run this test, please set AWS_SQS_QUEUE_URL to the address of your AWS SQS instance, like:
	export AWS_SQS_QUEUE_URL=http://localhost:4566/000000000000/localstack-queue
`)
	}

	//queue := NewSQSQueue(awssqs, queueURL)
	queue := NewInMemoryQueue[schemaless.Record[workflow.State]]()

	ctx := context.Background()
	tq2 := NewTaskQueue[workflow.State](desc, queue, repo, stream, proc)
	go func() {
		err := tq2.RunSelector(ctx)
		if err != nil {
			panic(err)
		}
	}()

	//go func() {
	//	err := tq2.RunCDC(ctx)
	//	if err != nil {
	//		panic(err)
	//	}
	//}()

	go func() {
		err := tq2.RunProcessor(ctx)
		if err != nil {
			panic(err)
		}
	}()

	work := workflow.NewMachine(di, nil)
	err := work.Handle(&workflow.Run{
		//RunOption: &workflow.DelayRun{
		//	DelayBySeconds: int64(1 * time.Second),
		//},
		RunOption: &workflow.ScheduleRun{
			Interval: "@every 0s",
		},
		Flow:  &workflow.FlowRef{FlowID: "hello_world_flow"},
		Input: schema.MkString("world"),
	})
	assert.NoError(t, err)

	newState := work.State()
	fmt.Printf("newState: %#v\n", newState)
	err = repo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
		ID:   workflow.GetRunID(newState),
		Type: "process",
		Data: newState,
	}))
	assert.NoError(t, err)

	time.Sleep(5 * time.Second)
}
