package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sashabaranov/go-openai"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/machine"
	"github.com/widmogrod/mkunion/x/schema"
	_ "github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
	"github.com/widmogrod/mkunion/x/taskqueue"
	"github.com/widmogrod/mkunion/x/workflow"
	_ "github.com/widmogrod/mkunion/x/workflow"
	"io"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
)

// this command make sure that all types that are imported will have generated typescript mapping
//go:generate ../../cmd/mkunion/mkunion shape-export --language=typescript -o ./src/workflow

// this lines defines all types that should have typescript mapping generated by above command
type (
	Workflow       = workflow.Workflow
	State          = workflow.State
	Command        = workflow.Command
	Expr           = workflow.Expr
	Predicate      = workflow.Predicate
	Reshaper       = workflow.Reshaper
	Schema         = schema.Schema
	UpdateRecords  = schemaless.UpdateRecords[schemaless.Record[any]]
	FindRecords    = schemaless.FindingRecords[schemaless.Record[any]]
	PageResult     = schemaless.PageResult[schemaless.Record[any]]
	FunctionOutput = workflow.FunctionOutput
	FunctionInput  = workflow.FunctionInput
)

//go:tag mkunion:"ChatCMD"
type (
	UserMessage struct {
		Message string
	}
)

//go:tag mkunion:"ChatResult"
type (
	SystemResponse struct {
		//OrderID 	  string
		Message   string
		ToolCalls []openai.ToolCall
	}
	UserResponse struct {
		//OrderID 	  string
		Message string
	}
	ChatResponses struct {
		Responses []ChatResult
	}
)

type ListWorkflowsFn struct {
	Count    int      `desc:"total number of words in sentence"`
	Words    []string `desc:"list of words in sentence"`
	EnumTest string   `desc:"skip words" enum:"hello,world"`
}

type RefreshStates struct{}
type RefreshFlows struct{}
type GenerateImage struct {
	Width  int `desc:"width of image as int between 50 and 500"`
	Height int `desc:"height of image as int between 50 and 500"`
}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func main() {
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// ... rest of the program ...

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	store := schemaless.NewInMemoryRepository[schema.Schema]()
	statesRepo := typedful.NewTypedRepository[workflow.State](store)
	flowsRepo := typedful.NewTypedRepository[workflow.Flow](store)

	var di = &workflow.DI{
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
			return "callback_id"
		},
		GenerateRunIDF: func() string {
			return fmt.Sprintf("run_id:%d", rand.Int())
		},
	}

	oaic := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	srv := NewService[workflow.Dependency, workflow.Command, workflow.State](
		"process",
		statesRepo,
		func(state workflow.State) *machine.Machine[workflow.Dependency, workflow.Command, workflow.State] {
			return workflow.NewMachine(di, state)
		},
		func(cmd workflow.Command) (*predicate.WherePredicates, bool) {
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
		},
		func(state workflow.State) (string, bool) {
			return workflow.GetRunIDFromBaseState(state), true
		},
	)

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))

	e.POST("/message", TypedJSONRequest(
		func(ctx context.Context, x ChatCMD) (ChatResult, error) {
			model := openai.GPT3Dot5Turbo1106
			tools := []openai.Tool{
				{
					Type: openai.ToolTypeFunction,
					Function: shape.ToOpenAIFunctionDefinition(
						"count_words",
						"count number of valid words in sentence",
						shape.FromGo(ListWorkflowsFn{}),
					),
				},
				{
					Type: openai.ToolTypeFunction,
					Function: shape.ToOpenAIFunctionDefinition(
						"refresh_flows",
						"refresh list of workflows visible to user on UI",
						shape.FromGo(RefreshFlows{}),
					),
				},
				{
					Type: openai.ToolTypeFunction,
					Function: shape.ToOpenAIFunctionDefinition(
						"refresh_states",
						"refresh list of states visible to user on UI",
						shape.FromGo(RefreshStates{}),
					),
				},
				{
					Type: openai.ToolTypeFunction,
					Function: shape.ToOpenAIFunctionDefinition(
						"generate_image",
						"generate image",
						shape.FromGo(GenerateImage{}),
					),
				},
			}

			var history []openai.ChatCompletionMessage
			history = append(history, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: x.(*UserMessage).Message,
			})

			result, err := oaic.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
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
					records, err := statesRepo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
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
				result2, err2 := oaic.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
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
		},
	))

	e.POST("/func", func(c echo.Context) error {
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

		fn, err := di.FindFunction(x.Name)
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
	})

	e.POST("/flow", TypedJSONRequest(
		func(ctx context.Context, x workflow.Workflow) (workflow.Workflow, error) {
			flow, ok := x.(*workflow.Flow)
			if !ok {
				return nil, errors.New("expected *workflow.Flow")
			}

			_, err := flowsRepo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.Flow]{
				ID:   flow.Name,
				Type: "flow",
				Data: *flow,
			}))

			if err != nil {
				log.Errorf("POST /flow: failed to save flow: %v", err)
				return nil, err
			}

			return flow, nil
		},
	))

	e.GET("/flow/:id", func(c echo.Context) error {
		record, err := flowsRepo.Get(c.Param("id"), "flow")
		if err != nil {
			log.Errorf("failed to get flow: %v", err)
			return err
		}

		result, err := shared.JSONMarshal[workflow.Flow](record.Data)
		if err != nil {
			if errors.Is(err, schemaless.ErrNotFound) {
				return c.JSONBlob(http.StatusNotFound, []byte(`{"error": "not found"}`))
			}

			log.Errorf("failed to convert to json: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, result)
	})

	e.POST("/flows", func(c echo.Context) error {
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

		records, err := flowsRepo.FindingRecords(query)
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
	})

	e.POST("/flows-updating", func(c echo.Context) error {
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

		_, err = flowsRepo.UpdateRecords(updating)
		if err != nil {
			log.Errorf("failed to update records: %v", err)
			return err
		}

		return c.NoContent(http.StatusOK)
	})

	e.POST("/states", func(c echo.Context) error {
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
		records, err := statesRepo.FindingRecords(query)
		if err != nil {
			return err
		}

		result, err := shared.JSONMarshal[schemaless.PageResult[schemaless.Record[workflow.State]]](records)
		if err != nil {
			log.Errorf("failed to convert to json: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, result)
	})

	e.POST("/state-updating", func(c echo.Context) error {
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

		_, err = statesRepo.UpdateRecords(updating)
		if err != nil {
			log.Errorf("failed to update records: %v", err)
			return err
		}

		return c.NoContent(http.StatusOK)
	})

	e.POST("/", TypedJSONRequest(
		func(ctx context.Context, cmd workflow.Command) (workflow.State, error) {
			return srv.CreateOrUpdate(ctx, cmd)
		}))

	e.POST("/workflow-to-str", func(c echo.Context) error {
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
	})

	e.GET("/workflow-to-str-from-run/:id", func(c echo.Context) error {
		runID := c.Param("id")

		state, err := srv.StateByID(runID)
		if err != nil {
			log.Errorf("workflow-to-str-from-run: id=%s failed to get state: %v", runID, err)
			return err
		}

		program, err := workflow.GetFlowFromState(state, di)
		if err != nil {
			log.Errorf("workflow-to-str-from-run: id=%s failed to get flow: %v", runID, err)
			return err
		}

		return c.String(http.StatusOK, workflow.ToStrWorkflow(program, workflow.ToStrContextFromState(state)))
	})

	e.POST("/callback", TypedJSONRequest(
		func(ctx context.Context, cmd workflow.Command) (workflow.State, error) {
			callbackCMD, ok := cmd.(*workflow.Callback)
			if !ok {
				log.Errorf("expected callback command")
				return nil, errors.New("expected callback command")
			}

			// find callback id in database
			records, err := statesRepo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
				//RecordType: "process",
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
				log.Errorf("state, with callbackID not found")
				return nil, errors.New("state, with callbackID not found")
			}

			state := records.Items[0]
			log.Infof("state: %+v", state)

			// apply command
			work := workflow.NewMachine(di, state.Data)
			err = work.Handle(ctx, cmd)
			if err != nil {
				log.Errorf("failed to handle command: %v", err)
				return nil, err
			}

			// save state
			newState := work.State()
			_, err = statesRepo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
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
		}))

	procScheduled, descScheduled := backgroundScheduled(di, statesRepo)
	queue := taskqueue.NewInMemoryQueue[schemaless.Record[workflow.State]]()
	stream := typedful.NewTypedAppendLog[workflow.State](store.AppendLog())
	taskScheduled := taskqueue.NewTaskQueue[workflow.State](descScheduled, queue, statesRepo, stream, procScheduled)

	go func() {
		err := taskScheduled.RunSelector(ctx)
		if err != nil {
			panic(err)
		}
	}()
	go func() {
		err := taskScheduled.RunProcessor(ctx)
		if err != nil {
			panic(err)
		}
	}()

	procRetry, descRetry := backgroundRetry(di, statesRepo)
	queueRetry := taskqueue.NewInMemoryQueue[schemaless.Record[workflow.State]]()
	streamRetry := typedful.NewTypedAppendLog[workflow.State](store.AppendLog())
	taskRetry := taskqueue.NewTaskQueue[workflow.State](descRetry, queueRetry, statesRepo, streamRetry, procRetry)

	go func() {
		err := taskRetry.RunCDC(ctx)
		if err != nil {
			panic(err)
		}
	}()
	go func() {
		err := taskRetry.RunProcessor(ctx)
		if err != nil {
			panic(err)
		}
	}()

	procTimeout, descTimeout := backgroundTimeout(di, statesRepo)
	queueTimeout := taskqueue.NewInMemoryQueue[schemaless.Record[workflow.State]]()
	streamTimeout := typedful.NewTypedAppendLog[workflow.State](store.AppendLog())
	taskTimeout := taskqueue.NewTaskQueue[workflow.State](descTimeout, queueTimeout, statesRepo, streamTimeout, procTimeout)

	go func() {
		err := taskTimeout.RunSelector(ctx)
		if err != nil {
			panic(err)
		}
	}()
	go func() {
		err := taskTimeout.RunProcessor(ctx)
		if err != nil {
			panic(err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Infof("shutting down server")
		if err := e.Shutdown(ctx); err != nil {
			log.Errorf("failed to shutdown server: %v", err)
		}
	}()

	if err := e.Start(":8080"); err != nil {
		if err == http.ErrServerClosed {
			log.Infof("server closed")
		} else {
			log.Errorf("failed to start server: %v", err)
		}
	}

	log.Infof("exiting")
}

func TypedJSONRequest[A, B any](handle func(ctx context.Context, x A) (B, error)) func(c echo.Context) error {
	return func(c echo.Context) error {
		data, err := io.ReadAll(c.Request().Body)
		if err != nil {
			log.Errorf("TypedJSONRequest: failed to read request body: %v", err)
			return err
		}

		in, err := shared.JSONUnmarshal[A](data)
		if err != nil {
			log.Errorf("TypedJSONRequest: failed to parse request body: %v", err)
			return err
		}

		out, err := handle(c.Request().Context(), in)
		if err != nil {
			return err
		}

		if _, ok := any(out).(B); !ok {
			var b B
			return fmt.Errorf("TypedJSONRequest: TypedRequest: expected %T, got %T", b, out)
		}

		result, err := shared.JSONMarshal[B](out)
		if err != nil {
			log.Errorf("TypedJSONRequest: failed to convert to json: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, result)
	}
}

func NewService[Dep any, CMD any, State any](
	recordType string,
	statesRepo *typedful.TypedRepoWithAggregator[State, any],
	newMachine func(state State) *machine.Machine[Dep, CMD, State],
	extractWhere func(CMD) (*predicate.WherePredicates, bool),
	extractIDFromState func(State) (string, bool),
) *Service[Dep, CMD, State] {
	return &Service[Dep, CMD, State]{
		repo:                     statesRepo,
		extractWhereFromCommandF: extractWhere,
		recordType:               recordType,
		newMachine:               newMachine,
		extractIDFromStateF:      extractIDFromState,
	}
}

type Service[Dep any, CMD any, State any] struct {
	repo                     *typedful.TypedRepoWithAggregator[State, any]
	extractWhereFromCommandF func(CMD) (*predicate.WherePredicates, bool)
	extractIDFromStateF      func(State) (string, bool)
	recordType               string
	newMachine               func(state State) *machine.Machine[Dep, CMD, State]
}

func (service *Service[Dep, CMD, State]) CreateOrUpdate(ctx context.Context, cmd CMD) (res State, err error) {
	version := uint16(0)
	recordID := ""
	where, foundAndUpdate := service.extractWhereFromCommandF(cmd)
	if foundAndUpdate {
		records, err := service.repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[State]]{
			RecordType: service.recordType,
			Where:      where,
			Limit:      1,
		})
		if err != nil {
			return res, err
		}
		if len(records.Items) == 0 {
			return res, fmt.Errorf("expected at least one record")
		}

		record := records.Items[0]
		res = record.Data
		version = record.Version
		recordID = record.ID
	}

	work := service.newMachine(res)
	err = work.Handle(ctx, cmd)
	if err != nil {
		log.Errorf("failed to handle command: %v", err)
		return res, err
	}

	newState := work.State()
	if !foundAndUpdate {
		saveId, ok := service.extractIDFromStateF(newState)
		if !ok {
			return res, fmt.Errorf("expected recordID in state")
		}
		recordID = saveId
	}

	if recordID == "" {
		return res, fmt.Errorf("expected recordID in state")
	}

	_, err = service.repo.UpdateRecords(schemaless.Save(schemaless.Record[State]{
		ID:      recordID,
		Type:    service.recordType,
		Data:    newState,
		Version: version,
	}))

	if err != nil {
		return res, err
	}

	return newState, nil
}

func (service *Service[Dep, CMD, State]) StateByID(id string) (res State, err error) {
	record, err := service.repo.Get(id, service.recordType)
	if err != nil {
		err = fmt.Errorf("service.Service.StateByID(%s) err=%w", id, err)
		return
	}

	return record.Data, nil
}
