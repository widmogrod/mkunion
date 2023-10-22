package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
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
	e.POST("/", func(c echo.Context) error {
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

		cmd, err := schema.ToGoG[workflow.Command](schemed)
		if err != nil {
			log.Errorf("failed to convert to command: %v", err)
			return err
		}

		work := workflow.NewMachine(di, nil)
		err = work.Handle(cmd)
		if err != nil {
			log.Errorf("failed to handle command: %v", err)
			return err
		}

		newState := work.State()
		err = statesRepo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
			ID:   workflow.GetRunID(newState),
			Type: "process",
			Data: newState,
		}))
		if err != nil {
			log.Errorf("failed to save state: %v", err)
			return err
		}

		schemed = schema.FromGo(newState)
		result, err := schema.ToJSON(schemed)
		if err != nil {
			log.Errorf("failed to convert to json: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, result)
	})

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

	e.POST("/callback", func(c echo.Context) error {
		// validate payload
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

		cmd, err := schema.ToGoG[workflow.Command](schemed)
		if err != nil {
			log.Errorf("failed to convert to command: %v", err)
			return err
		}

		callbackCMD, ok := cmd.(*workflow.Callback)
		if !ok {
			log.Errorf("expected callback command")
			return errors.New("expected callback command")
		}

		// find callback id in database
		records, err := statesRepo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
			//RecordType: "process",
			Where: predicate.MustWhere(
				"Type = :type AND Data.#.#.CallbackID = :callbackID",
				predicate.ParamBinds{
					":type":       schema.MkString("process"),
					":callbackID": schema.MkString(callbackCMD.CallbackID),
				},
			),
			Limit: 1,
		})
		if err != nil {
			log.Errorf("failed to find callback id: %v", err)
			return err
		}

		if len(records.Items) == 0 {
			log.Errorf("state, with callbackID not found")
			return errors.New("state, with callbackID not found")
		}

		state := records.Items[0]
		log.Infof("state: %+v", state)

		// apply command
		work := workflow.NewMachine(di, state.Data)
		err = work.Handle(cmd)
		if err != nil {
			log.Errorf("failed to handle command: %v", err)
			return err
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
			return err
		}

		schemed = schema.FromGo(newState)
		result, err := schema.ToJSON(schemed)
		if err != nil {
			log.Errorf("failed to convert to json: %v", err)
			return err
		}

		return c.JSONBlob(http.StatusOK, result)
	})

	proc := &taskqueue.FunctionProcessor[workflow.State]{
		F: func(task taskqueue.Task[schemaless.Record[workflow.State]]) {
			log.Infof("task: %#v \n", task)
			work := workflow.NewMachine(di, task.Data.Data)
			//err := work.Handle(&workflow.Retry{})
			err := work.Handle(&workflow.Run{})
			if err != nil {
				log.Errorf("err: %s", err)
				return
			}

			err = statesRepo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
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
					log.Warnf("version conflict, ignoring")
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
		Change: []string{"create", "update"},
		Entity: "process",
		//Filter: "Data.#.Retried > Data.#.Error.MaxRetries",
	}
	queue := taskqueue.NewInMemoryQueue[schemaless.Record[schema.Schema]]()

	ctx := context.Background()
	tq2 := taskqueue.NewTaskQueue(desc, queue, store, proc)
	_ = ctx
	_ = tq2
	//go func() {
	//	err := tq2.RunSelector(ctx)
	//	if err != nil {
	//		panic(err)
	//	}
	//}()
	//
	//go func() {
	//	err := tq2.RunProcessor(ctx)
	//	if err != nil {
	//		panic(err)
	//	}
	//}()

	e.Logger.Fatal(e.Start(":8080"))

}
