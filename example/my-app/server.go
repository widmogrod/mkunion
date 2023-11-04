package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/machine"
	"github.com/widmogrod/mkunion/x/schema"
	_ "github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
	"github.com/widmogrod/mkunion/x/taskqueue"
	"github.com/widmogrod/mkunion/x/workflow"
	_ "github.com/widmogrod/mkunion/x/workflow"
	"io"
	"math/rand"
	"net/http"
)

func main() {
	schema.RegisterRules([]schema.RuleMatcher{
		schema.WhenPath([]string{"*", "BaseState"}, schema.UseStruct(workflow.BaseState{})),
		schema.WhenPath([]string{"*", "Await"}, schema.UseStruct(&workflow.ApplyAwaitOptions{})),
	})

	log.SetLevel(log.DebugLevel)

	store := schemaless.NewInMemoryRepository()
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

	srv := NewService[workflow.Command, workflow.State](
		"process",
		statesRepo,
		func(state workflow.State) *machine.Machine[workflow.Command, workflow.State] {
			return workflow.NewMachine(di, state)
		},
		func(cmd workflow.Command) (string, bool) {
			if cmd == nil {
				return "", false
			}

			switch cmd := cmd.(type) {
			case *workflow.StopSchedule:
				return cmd.RunID, true
			case *workflow.ResumeSchedule:
				return cmd.RunID, true
			default:
				return "", false
			}
		},
		func(state workflow.State) (string, bool) {
			return workflow.GetRunID(state), true
		},
	)

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))
	e.POST("/flow", func(c echo.Context) error {
		data, err := io.ReadAll(c.Request().Body)
		if err != nil {
			log.Errorf("failed to read request body: %v", err)
			return err
		}

		schemed, err := schema.FromJSON(data)
		if err != nil {
			log.Errorf("failed to parse request body: %v", err)
			return err
		}

		program, err := schema.ToGoG[workflow.Worflow](schemed)
		if err != nil {
			log.Errorf("failed to convert to command: %v", err)
			return err
		}

		flow, ok := program.(*workflow.Flow)
		if !ok {
			return errors.New("expected *workflow.Flow")
		}

		err = flowsRepo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.Flow]{
			ID:   flow.Name,
			Type: "flow",
			Data: *flow,
		}))

		if err != nil {
			log.Errorf("failed to save state: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, data)
	})
	e.GET("/flow/:id", func(c echo.Context) error {
		record, err := flowsRepo.Get(c.Param("id"), "flow")
		if err != nil {
			log.Errorf("failed to get flow: %v", err)
			return err
		}

		schemed := schema.FromGo(record)
		result, err := schema.ToJSON(schemed)
		if err != nil {
			if errors.As(err, &schemaless.ErrNotFound) {
				return c.JSONBlob(http.StatusNotFound, []byte(`{"error": "not found"}`))
			}

			log.Errorf("failed to convert to json: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, result)
	})
	e.GET("/flows", func(c echo.Context) error {
		records, err := flowsRepo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.Flow]]{
			RecordType: "flow",
			//Limit:      2,
		})
		if err != nil {
			log.Errorf("failed to get flowsRepo: %v", err)
			return err
		}

		schemed := schema.FromGo(records)
		result, err := schema.ToJSON(schemed)
		if err != nil {
			log.Errorf("failed to convert to json: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, result)
	})

	e.GET("/list", func(c echo.Context) error {
		records, err := statesRepo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
			RecordType: "process",
			//Where: predicate.MustWhere(
			//	"Type = :type AND Data.#.#.BaseState.Flow.#.Name = :name",
			//	predicate.ParamBinds{
			//		":type": schema.MkString("process"),
			//		":name": schema.MkString("hello_world"),
			//	},
			//),
		})
		if err != nil {
			return err
		}

		schemed := schema.FromGo(records)
		result, err := schema.ToJSON(schemed)
		if err != nil {
			log.Errorf("failed to convert to json: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, result)
	})

	e.POST("/", TypedRequest(func(cmd workflow.Command) (workflow.State, error) {
		return srv.CreateOrUpdate(cmd)
		//work := workflow.NewMachine(di, nil)
		//err := work.Handle(cmd)
		//if err != nil {
		//	log.Errorf("failed to handle command: %v", err)
		//	return nil, err
		//}
		//
		//newState := work.State()
		//err = repo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
		//	ID:   workflow.GetRunID(newState),
		//	Type: "process",
		//	Data: newState,
		//}))
		//if err != nil {
		//	log.Errorf("failed to save state: %v", err)
		//	return nil, err
		//}
		//
		//return newState, nil
	}))

	e.POST("/workflow-to-str", func(c echo.Context) error {
		data, err := io.ReadAll(c.Request().Body)
		if err != nil {
			log.Errorf("failed to read request body: %v", err)
			return err
		}

		schemed, err := schema.FromJSON(data)
		if err != nil {
			log.Errorf("failed to parse request body: %v", err)
			return err
		}

		program, err := schema.ToGoG[workflow.Worflow](schemed)
		if err != nil {
			log.Errorf("failed to convert to workflow: %v", err)
			return err
		}

		return c.String(http.StatusOK, workflow.ToStrWorkflow(program, 0))
	})

	e.POST("/callback", TypedRequest(func(cmd workflow.Command) (workflow.State, error) {
		callbackCMD, ok := cmd.(*workflow.Callback)
		if !ok {
			log.Errorf("expected callback command")
			return nil, errors.New("expected callback command")
		}

		// find callback id in database
		records, err := statesRepo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
			//RecordType: "process",
			Where: predicate.MustWhere(
				`Type = :type AND Data["workflow.Await"].CallbackID = :callbackID`,
				predicate.ParamBinds{
					":type":       schema.MkString("process"),
					":callbackID": schema.MkString(callbackCMD.CallbackID),
				},
			),
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
		err = work.Handle(cmd)
		if err != nil {
			log.Errorf("failed to handle command: %v", err)
			return nil, err
		}

		// save state
		newState := work.State()
		err = statesRepo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
			ID:      workflow.GetRunID(newState),
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

	proc := &taskqueue.FunctionProcessor[workflow.State]{
		F: func(task taskqueue.Task[schemaless.Record[workflow.State]]) {
			//log.Infof("data id: %#v \n", task.Data.ID)
			work := workflow.NewMachine(di, task.Data.Data)
			err := work.Handle(&workflow.Run{})
			if err != nil {
				log.Errorf("err: %s", err)
				return
			}

			newState := work.State()
			//log.Infof("newState: %T", newState)

			saving := []schemaless.Record[workflow.State]{
				{
					ID:      task.Data.ID,
					Data:    newState,
					Type:    task.Data.Type,
					Version: task.Data.Version,
				},
			}

			if next := workflow.ScheduleNext(newState, di); next != nil {
				work := workflow.NewMachine(di, nil)
				err := work.Handle(next)
				if err != nil {
					log.Infof("err: %s", err)
					return
				}

				//log.Infof("next id=%s", workflow.GetRunID(work.State()))

				saving = append(saving, schemaless.Record[workflow.State]{
					ID:   workflow.GetRunID(work.State()),
					Type: task.Data.Type,
					Data: work.State(),
				})
			}

			err = statesRepo.UpdateRecords(schemaless.Save(saving...))
			if err != nil {
				if errors.Is(err, schemaless.ErrVersionConflict) {
					// make it configurable, but by default we should
					// just ignore conflicts, since that means we may have duplicate,
					// or some other process already update it.
					// assuming that queue is populated from stream of changes
					// it such case (there was update) new message with new version
					// will land in queue soon (if it pass selector)
					log.Warnf("version conflict, ignoring: %s", err.Error())
				} else {
					panic(err)
				}
			}
		},
	}

	// there can be few process,
	// - timeout out workflow (command to timeout)
	// - retry workflow (command to retry)
	// - run workflow (command to run)
	// - callback workflow (command to callback)
	// - terminate workflow (command to terminate)
	// - complete workflow (command to complete)

	desc := &taskqueue.Description{
		Change: []string{"create"},
		Entity: "process",
		//Filter: `Data[*]["workflow.Scheduled"].RunOption["workflow.DelayRun"].DelayBySeconds > 0 AND Version = 1`,
		Filter: `Data[*]["workflow.Scheduled"].ExpectedRunTimestamp <= :now 
AND Data[*]["workflow.Scheduled"].ExpectedRunTimestamp > 0
`,
	}
	queue := taskqueue.NewInMemoryQueue[schemaless.Record[schema.Schema]]()
	stream := store.AppendLog()
	ctx := context.Background()
	tq2 := taskqueue.NewTaskQueue(desc, queue, store, stream, proc)

	//go func() {
	//	err := tq2.RunCDC(ctx)
	//	if err != nil {
	//		panic(err)
	//	}
	//}()

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

	e.Logger.Fatal(e.Start(":8080"))

}

func TypedRequest[A, B any](handle func(x A) (B, error)) func(c echo.Context) error {
	return func(c echo.Context) error {
		data, err := io.ReadAll(c.Request().Body)
		if err != nil {
			log.Errorf("failed to read request body: %v", err)
			return err
		}

		schemed, err := schema.FromJSON(data)
		if err != nil {
			log.Errorf("failed to parse request body: %v", err)
			return err
		}

		in, err := schema.ToGoG[A](schemed)
		if err != nil {
			log.Errorf("failed to convert to command: %v", err)
			return err
		}

		out, err := handle(in)
		if err != nil {
			return err
		}

		if _, ok := any(out).(B); !ok {
			var b B
			return fmt.Errorf("TypedRequest: expected %T, got %T", b, out)
		}

		schemed = schema.FromGo(out)
		result, err := schema.ToJSON(schemed)
		if err != nil {
			log.Errorf("failed to convert to json: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, result)
	}
}

func NewService[CMD any, State any](
	recordType string,
	statesRepo *typedful.TypedRepoWithAggregator[State, any],
	newMachine func(state State) *machine.Machine[CMD, State],
	extractID func(CMD) (string, bool),
	extractIDFromState func(State) (string, bool),
) *Service[CMD, State] {
	return &Service[CMD, State]{
		repo:                  statesRepo,
		extractIDFromCommandF: extractID,
		recordType:            recordType,
		newMachine:            newMachine,
		extractIDFromStateF:   extractIDFromState,
	}
}

type Service[CMD any, State any] struct {
	repo                  *typedful.TypedRepoWithAggregator[State, any]
	extractIDFromCommandF func(CMD) (string, bool)
	extractIDFromStateF   func(State) (string, bool)
	recordType            string
	newMachine            func(state State) *machine.Machine[CMD, State]
}

func (service *Service[CMD, State]) CreateOrUpdate(cmd CMD) (res State, err error) {
	version := uint16(0)
	recordID, foundAndUpdate := service.extractIDFromCommandF(cmd)
	if foundAndUpdate {
		record, err := service.repo.Get(recordID, service.recordType)
		if err != nil {
			return res, err
		}

		res = record.Data
		version = record.Version
	}

	work := service.newMachine(res)
	err = work.Handle(cmd)
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

	err = service.repo.UpdateRecords(schemaless.Save(schemaless.Record[State]{
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
