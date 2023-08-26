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

//go:generate mkunion -name=Command
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

//go:generate mkunion -name=State
type (
	NextOperation struct {
		StepID    string
		Result    schema.Schema
		BaseState *BaseState
	}
	Done struct {
		StepID    string
		Result    schema.Schema
		BaseState *BaseState
	}
	Fail struct {
		StepID    string
		Result    schema.Schema
		BaseState *BaseState
	}
	Error struct {
		StepID    string
		Code      string
		Reason    string
		Retried   int64
		BaseState *BaseState
	}
	Await struct {
		StepID     string
		CallbackID string
		Timeout    time.Duration
		BaseState  *BaseState
	}
)

type BaseState struct {
	Flow       Worflow
	Variables  map[string]schema.Schema
	ExprResult map[string]schema.Schema
}

//go:generate mkunion -name=ASTNode -variants=Flow,End,Assign,Apply,Choose,GetValue,SetValue

//go:generate mkunion -name=Worflow
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

//go:generate mkunion -name=Expr
type (
	End struct {
		ID     string
		Result Reshaper
		Fail   Reshaper
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

//go:generate mkunion -name=Predicate
type (
	Eq struct {
		Left  Reshaper
		Right Reshaper
	}
	Exists struct {
		Path []string
	}
	And struct {
		T1 Predicate
		T2 Predicate
	}
	Or struct {
		T1 Predicate
		T2 Predicate
	}
)

//go:generate mkunion -name=Reshaper
type (
	GetValue struct {
		Path string
	}
	SetValue struct {
		Value schema.Schema
	}
)
