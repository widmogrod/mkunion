package workflow

import (
	"github.com/widmogrod/mkunion/x/schema"
	"time"
)

type Execution struct {
	FlowID    string
	Status    State
	Location  string
	StartTime int64
	EndTime   int64
	Variables map[string]schema.Schema
}

//go:generate go run ../../../cmd/mkunion/main.go -name=Command
type (
	Run struct {
		//FlowID string
		Flow  Worflow
		Input schema.Schema
	}
	Callback struct {
		CallbackID string
		//Flow       Worflow
		Result schema.Schema
		//Fail       schema.Schema
	}
	//Retry struct {
	//	StepID string
	//}
)

//go:generate go run ../../../cmd/mkunion/main.go -name=State
type (
	NextOperation struct {
		Result    schema.Schema
		BaseState BaseState
	}
	Done struct {
		Result    schema.Schema
		BaseState BaseState
	}
	Error struct {
		Code      string
		Reason    string
		Retried   int64
		BaseState BaseState
	}
	Await struct {
		CallbackID string
		Timeout    time.Duration
		BaseState  BaseState
	}
)

type BaseState struct {
	Flow       Worflow
	StepID     string
	Variables  map[string]schema.Schema
	ExprResult map[string]schema.Schema
}

//go:generate go run ../../../cmd/mkunion/main.go -name=Worflow
type (
	Flow struct {
		Name string // name of the flow
		Arg  string // name of the argument, which will hold the input to the flow
		Body []Expr
	}
	FlowRef struct {
		FlowID string
	}
)

//go:generate go run ../../../cmd/mkunion/main.go -name=Expr
type (
	End struct {
		ID     string
		Result Reshaper
	}
	Assign struct {
		ID  string
		Var string
		Val Expr
	}
	Apply struct {
		ID    string
		Name  string
		Args  []Reshaper
		Await *ApplyAwaitOptions
	}
	Choose struct {
		ID   string
		If   Predicate
		Then []Expr
		Else []Expr
	}
)

type ApplyAwaitOptions struct {
	Timeout time.Duration
}

//go:generate go run ../../../cmd/mkunion/main.go -name=Reshaper
type (
	GetValue struct {
		Path string
	}
	SetValue struct {
		Value schema.Schema
	}
)

//go:generate go run ../../../cmd/mkunion/main.go -name=Predicate
type (
	And struct {
		L []Predicate
	}
	Or struct {
		L []Predicate
	}
	Not struct {
		P Predicate
	}
	Compare struct {
		Operation string
		Left      Reshaper
		Right     Reshaper
	}
)
