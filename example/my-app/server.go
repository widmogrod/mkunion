package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
	_ "github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
	"github.com/widmogrod/mkunion/x/workflow"
	_ "github.com/widmogrod/mkunion/x/workflow"
	"io"
	"math/rand"
	"net/http"
	"strconv"
)

var program = &workflow.Flow{
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

var di = &workflow.DI{
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

func main() {
	schema.RegisterRules([]schema.RuleMatcher{
		schema.WhenPath([]string{"*", "BaseState"}, schema.UseStruct(workflow.BaseState{})),
	})

	log.SetLevel(log.DebugLevel)

	store := schemaless.NewInMemoryRepository()
	repo := typedful.NewTypedRepository[workflow.State](store)

	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
	}))
	e.GET("/list", func(c echo.Context) error {
		records, err := repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[workflow.State]]{
			RecordType: "process",
			//Limit:      2,
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
		err = repo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
			ID:   strconv.Itoa(rand.Int()),
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
	e.Logger.Fatal(e.Start(":8080"))

}
