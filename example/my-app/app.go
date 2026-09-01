package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/machine"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shapeopenai"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/jsonful"
	"github.com/widmogrod/mkunion/x/taskqueue"
	"github.com/widmogrod/mkunion/x/workflow"
)

// app wires the demo server: plain-JSON repositories with shape-aware
// queries, workflow dependencies, and the OpenAI client used by /message.
type app struct {
	oaic       *openai.Client
	statesRepo *jsonful.InMemoryRepository[workflow.State]
	flowsRepo  *jsonful.InMemoryRepository[workflow.Flow]
	di         *workflow.DI
	srv        *Service[workflow.Dependency, workflow.Command, workflow.State]
}

func newApp(oaic *openai.Client) *app {
	statesRepo, err := jsonful.NewInMemoryRepository[workflow.State]()
	if err != nil {
		log.Fatalf("failed to create states repository: %v", err)
	}
	flowsRepo, err := jsonful.NewInMemoryRepository[workflow.Flow]()
	if err != nil {
		log.Fatalf("failed to create flows repository: %v", err)
	}

	di := &workflow.DI{
		FindWorkflowF: func(flowID string) (*workflow.Flow, error) {
			record, err := flowsRepo.Get(flowID, "flow")
			if err != nil {
				return nil, err
			}

			return &record.Data, nil
		},
		FindFunctionF: func(funcID string) (workflow.Function, error) {
			if fn, ok := functions[funcID]; ok {
				return fn, nil
			}

			return nil, fmt.Errorf("function %s not found", funcID)
		},
		GenerateCallbackIDF: func() string {
			return fmt.Sprintf("callback_%d", rand.Int())
		},
		GenerateRunIDF: func() string {
			return fmt.Sprintf("run_id:%d", rand.Int())
		},
	}

	srv := NewService[workflow.Dependency, workflow.Command, workflow.State](
		"process",
		statesRepo,
		func(state workflow.State) *machine.Machine[workflow.Dependency, workflow.Command, workflow.State] {
			return workflow.NewMachine(di, state)
		},
		commandToWhere,
		func(state workflow.State) (string, bool) {
			return workflow.GetRunIDFromBaseState(state), true
		},
	)

	return &app{
		oaic:       oaic,
		statesRepo: statesRepo,
		flowsRepo:  flowsRepo,
		di:         di,
		srv:        srv,
	}
}

// commandToWhere maps commands that address existing runs to the query
// that finds their record; other commands create new records.
func commandToWhere(cmd workflow.Command) (*predicate.WherePredicates, bool) {
	switch cmd := cmd.(type) {
	case *workflow.StopSchedule:
		return predicate.MustWhere(`Data["workflow.Scheduled"].BaseState.RunOption["workflow.ScheduleRun"].ParentRunID = :runID`, predicate.ParamBinds{
			":runID": schema.MkString(cmd.ParentRunID),
		}, nil), true
	case *workflow.ResumeSchedule:
		return predicate.MustWhere(`Data["workflow.ScheduleStopped"].BaseState.RunOption["workflow.ScheduleRun"].ParentRunID = :runID`, predicate.ParamBinds{
			":runID": schema.MkString(cmd.ParentRunID),
		}, nil), true
	case *workflow.TryRecover:
		return predicate.MustWhere(`Data["workflow.Error"].BaseState.RunID = :runID`, predicate.ParamBinds{
			":runID": schema.MkString(cmd.RunID),
		}, nil), true
	}
	return nil, false
}

func (a *app) registerRoutes(e *echo.Echo) {
	e.POST("/message", TypedJSONRequest(a.handleMessage))
	e.POST("/func", a.handleFunc)
	e.POST("/flow", TypedJSONRequest(a.handleFlowCreate))
	e.GET("/flow/:id", a.handleFlowGet)
	e.POST("/flows", a.handleFlowsList)
	e.POST("/flows-updating", a.handleFlowsUpdating)
	e.POST("/states", a.handleStatesList)
	e.POST("/state-updating", a.handleStateUpdating)
	e.POST("/", TypedJSONRequest(a.handleCommand))
	e.POST("/workflow-to-str", a.handleWorkflowToStr)
	e.GET("/workflow-to-str-from-run/:id", a.handleWorkflowToStrFromRun)
	e.POST("/callback", TypedJSONRequest(a.handleCallback))
}

func chatTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: shapeopenai.ToOpenAIFunctionDefinition(
				"count_words",
				"count number of valid words in sentence",
				shape.FromGo(ListWorkflowsFn{}),
			),
		},
		{
			Type: openai.ToolTypeFunction,
			Function: shapeopenai.ToOpenAIFunctionDefinition(
				"refresh_flows",
				"refresh list of workflows visible to user on UI",
				shape.FromGo(RefreshFlows{}),
			),
		},
		{
			Type: openai.ToolTypeFunction,
			Function: shapeopenai.ToOpenAIFunctionDefinition(
				"refresh_states",
				"refresh list of states visible to user on UI",
				shape.FromGo(RefreshStates{}),
			),
		},
		{
			Type: openai.ToolTypeFunction,
			Function: shapeopenai.ToOpenAIFunctionDefinition(
				"generate_image",
				"generate image",
				shape.FromGo(GenerateImage{}),
			),
		},
	}
}

func (a *app) handleMessage(ctx context.Context, x ChatCMD) (ChatResult, error) {
	model := openai.GPT3Dot5Turbo1106
	tools := chatTools()

	var history []openai.ChatCompletionMessage
	history = append(history, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: x.(*UserMessage).Message,
	})

	result, err := a.oaic.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    model,
		Messages: history,
		Tools:    tools,
	})

	if err != nil {
		log.Errorf("failed to create chat completion: %v", err)
		return nil, err
	}

	history = append(history, result.Choices[0].Message)

	response := &ChatResponses{}
	response.Responses = append(response.Responses, &SystemResponse{
		Message:   result.Choices[0].Message.Content,
		ToolCalls: result.Choices[0].Message.ToolCalls,
	})

	for _, tool := range result.Choices[0].Message.ToolCalls {
		switch tool.Function.Name {
		case "refresh_states":
			records, err := a.statesRepo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
				RecordType: "process",
			})
			if err != nil {
				return nil, err
			}

			result, err := shared.JSONMarshal[schemaless.PageResult[schemaless.Record[workflow.State]]](records)
			if err != nil {
				log.Errorf("failed to convert to json: %v", err)
				return nil, err
			}

			history = append(history, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    string(result),
				ToolCallID: tool.ID,
			})

		default:
			history = append(history, openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    "not implemented",
				ToolCallID: tool.ID,
			})
		}
	}

	if len(result.Choices[0].Message.ToolCalls) > 0 {
		result2, err2 := a.oaic.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    model,
			Messages: history,
			Tools:    tools,
		})

		if err2 != nil {
			log.Errorf("failed to create chat completion2: %v", err2)
			for _, h := range history {
				log.Infof("history: %#+v \n", h)
			}
			return nil, err2
		}

		response.Responses = append(response.Responses, &SystemResponse{
			Message:   result2.Choices[0].Message.Content,
			ToolCalls: result2.Choices[0].Message.ToolCalls,
		})
	}

	log.Infof("result: %+v", result)
	return response, nil
}

func (a *app) handleFunc(c echo.Context) error {
	data, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Errorf("failed to read request body: %v", err)
		return err
	}

	x, err := shared.JSONUnmarshal[*workflow.FunctionInput](data)
	if err != nil {
		log.Errorf("failed to parse request body: %v", err)
		return err
	}

	fn, err := a.di.FindFunction(x.Name)
	if err != nil {
		return err
	}

	result, err := fn(x)
	if err != nil {
		return err
	}

	resultJSON, err := shared.JSONMarshal[*workflow.FunctionOutput](result)
	if err != nil {
		log.Errorf("failed to convert to json: %v", err)
		return err
	}

	return c.JSONBlob(http.StatusOK, resultJSON)
}

func (a *app) handleFlowCreate(ctx context.Context, x workflow.Workflow) (workflow.Workflow, error) {
	flow, ok := x.(*workflow.Flow)
	if !ok {
		return nil, errors.New("expected *workflow.Flow")
	}

	_, err := a.flowsRepo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.Flow]{
		ID:   flow.Name,
		Type: "flow",
		Data: *flow,
	}))

	if err != nil {
		log.Errorf("POST /flow: failed to save flow: %v", err)
		return nil, err
	}

	return flow, nil
}

func (a *app) handleFlowGet(c echo.Context) error {
	record, err := a.flowsRepo.Get(c.Param("id"), "flow")
	if err != nil {
		if errors.Is(err, schemaless.ErrNotFound) {
			return c.JSONBlob(http.StatusNotFound, []byte(`{"error": "not found"}`))
		}

		log.Errorf("failed to get flow: %v", err)
		return err
	}

	result, err := shared.JSONMarshal[workflow.Flow](record.Data)
	if err != nil {
		log.Errorf("failed to convert to json: %v", err)
		return err
	}

	return c.JSONBlob(http.StatusOK, result)
}

func (a *app) handleFlowsList(c echo.Context) error {
	data, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Errorf("failed to read request body: %v", err)
		return err
	}

	query, err := shared.JSONUnmarshal[schemaless.FindingRecords[schemaless.Record[workflow.Flow]]](data)
	if err != nil {
		log.Warnf("failed to parse query: %v", err)
		query = schemaless.FindingRecords[schemaless.Record[workflow.Flow]]{}
	}

	query.RecordType = "flow"

	records, err := a.flowsRepo.FindingRecords(query)
	if err != nil {
		log.Errorf("failed to get flowsRepo: %v", err)
		return err
	}

	result, err := shared.JSONMarshal[schemaless.PageResult[schemaless.Record[workflow.Flow]]](records)
	if err != nil {
		log.Errorf("failed to convert to json: %v", err)
		return err
	}

	return c.JSONBlob(http.StatusOK, result)
}

func (a *app) handleFlowsUpdating(c echo.Context) error {
	data, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Errorf("failed to read request body: %v", err)
		return err
	}

	updating, err := shared.JSONUnmarshal[schemaless.UpdateRecords[schemaless.Record[workflow.Flow]]](data)
	if err != nil {
		log.Errorf("failed to parse body: %v", err)
		return err
	}

	_, err = a.flowsRepo.UpdateRecords(updating)
	if err != nil {
		log.Errorf("failed to update records: %v", err)
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (a *app) handleStatesList(c echo.Context) error {
	data, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Errorf("failed to read request body: %v", err)
		return err
	}

	query, err := shared.JSONUnmarshal[schemaless.FindingRecords[schemaless.Record[workflow.State]]](data)
	if err != nil {
		log.Warnf("failed to parse query: %v", err)
		query = schemaless.FindingRecords[schemaless.Record[workflow.State]]{}
	}

	query.RecordType = "process"
	records, err := a.statesRepo.FindingRecords(query)
	if err != nil {
		return err
	}

	result, err := shared.JSONMarshal[schemaless.PageResult[schemaless.Record[workflow.State]]](records)
	if err != nil {
		log.Errorf("failed to convert to json: %v", err)
		return err
	}

	return c.JSONBlob(http.StatusOK, result)
}

func (a *app) handleStateUpdating(c echo.Context) error {
	data, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Errorf("failed to read request body: %v", err)
		return err
	}

	updating, err := shared.JSONUnmarshal[schemaless.UpdateRecords[schemaless.Record[workflow.State]]](data)
	if err != nil {
		log.Errorf("failed to parse body: %v", err)
		return err
	}

	_, err = a.statesRepo.UpdateRecords(updating)
	if err != nil {
		log.Errorf("failed to update records: %v", err)
		return err
	}

	return c.NoContent(http.StatusOK)
}

func (a *app) handleCommand(ctx context.Context, cmd workflow.Command) (workflow.State, error) {
	return a.srv.CreateOrUpdate(ctx, cmd)
}

func (a *app) handleWorkflowToStr(c echo.Context) error {
	data, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Errorf("failed to read request body: %v", err)
		return err
	}

	program, err := workflow.WorkflowFromJSON(data)
	if err != nil {
		log.Errorf("failed to convert to workflow: %v", err)
		return err
	}

	return c.String(http.StatusOK, workflow.ToStrWorkflow(program, nil))
}

func (a *app) handleWorkflowToStrFromRun(c echo.Context) error {
	runID := c.Param("id")

	state, err := a.srv.StateByID(runID)
	if err != nil {
		log.Errorf("workflow-to-str-from-run: id=%s failed to get state: %v", runID, err)
		return err
	}

	program, err := workflow.GetFlowFromState(state, a.di)
	if err != nil {
		log.Errorf("workflow-to-str-from-run: id=%s failed to get flow: %v", runID, err)
		return err
	}

	return c.String(http.StatusOK, workflow.ToStrWorkflow(program, workflow.ToStrContextFromState(state)))
}

func (a *app) handleCallback(ctx context.Context, cmd workflow.Command) (workflow.State, error) {
	callbackCMD, ok := cmd.(*workflow.Callback)
	if !ok {
		log.Errorf("expected callback command")
		return nil, errors.New("expected callback command")
	}

	// find callback id in database
	records, err := a.statesRepo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
		Where: predicate.MustWhere(`Type = :type AND Data["workflow.Await"].CallbackID = :callbackID`, predicate.ParamBinds{
			":type":       schema.MkString("process"),
			":callbackID": schema.MkString(callbackCMD.CallbackID),
		}, nil),
		Limit: 1,
	})
	if err != nil {
		log.Errorf("failed to find callback id: %v", err)
		return nil, err
	}

	if len(records.Items) == 0 {
		log.Errorf("state, with callbackID %s not found", callbackCMD.CallbackID)
		return nil, errors.New("state, with callbackID not found")
	}

	state := records.Items[0]
	log.Infof("state: %+v", state)

	// apply command
	work := workflow.NewMachine(a.di, state.Data)
	err = work.Handle(ctx, cmd)
	if err != nil {
		log.Errorf("failed to handle command: %v", err)
		return nil, err
	}

	// save state
	newState := work.State()
	_, err = a.statesRepo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
		ID:      workflow.GetRunIDFromBaseState(newState),
		Type:    "process",
		Data:    newState,
		Version: state.Version,
	}))
	if err != nil {
		log.Errorf("failed to save state: %v", err)
		return nil, err
	}

	return newState, nil
}

// startBackgroundTasks launches the scheduled, retry, and timeout task
// queues; each goroutine panics on error, as main always did.
func (a *app) startBackgroundTasks(ctx context.Context) {
	runOrPanic := func(fn func(ctx context.Context) error) {
		go func() {
			if err := fn(ctx); err != nil {
				panic(err)
			}
		}()
	}

	procScheduled, descScheduled := backgroundScheduled(a.di, a.statesRepo)
	queue := taskqueue.NewInMemoryQueue[schemaless.Record[workflow.State]]()
	stream := a.statesRepo.AppendLog()
	taskScheduled := taskqueue.NewTaskQueue[workflow.State](descScheduled, queue, a.statesRepo, stream, procScheduled)
	runOrPanic(taskScheduled.RunSelector)
	runOrPanic(taskScheduled.RunProcessor)

	procRetry, descRetry := backgroundRetry(a.di, a.statesRepo)
	queueRetry := taskqueue.NewInMemoryQueue[schemaless.Record[workflow.State]]()
	streamRetry := a.statesRepo.AppendLog()
	taskRetry := taskqueue.NewTaskQueue[workflow.State](descRetry, queueRetry, a.statesRepo, streamRetry, procRetry)
	runOrPanic(taskRetry.RunCDC)
	runOrPanic(taskRetry.RunProcessor)

	procTimeout, descTimeout := backgroundTimeout(a.di, a.statesRepo)
	queueTimeout := taskqueue.NewInMemoryQueue[schemaless.Record[workflow.State]]()
	streamTimeout := a.statesRepo.AppendLog()
	taskTimeout := taskqueue.NewTaskQueue[workflow.State](descTimeout, queueTimeout, a.statesRepo, streamTimeout, procTimeout)
	runOrPanic(taskTimeout.RunSelector)
	runOrPanic(taskTimeout.RunProcessor)
}
