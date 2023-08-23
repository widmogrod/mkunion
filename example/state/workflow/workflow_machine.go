package workflow

import (
	"github.com/widmogrod/mkunion/x/schema"
	"time"
)

type Execution struct {
	FlowID    string
	Status    Status
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
	Schedule struct {
		//FlowID string
		Flow  Worflow
		Delay time.Duration
		Input schema.Schema
	}
	Callback struct {
		StepID string
		Result schema.Schema
		Fail   schema.Schema
	}
	Retry struct {
		StepID string
	}
)

//go:generate mkunion -name=Status
type (
	Start struct {
		FlowID string
		Input  schema.Schema
	}
	Result struct {
		StepID string
		Result schema.Schema
	}
	Done struct {
		StepID string
		Result schema.Schema
	}
	Fail struct {
		StepID string
		Result schema.Schema
	}
	Error struct {
		StepID  string
		Code    string
		Reason  string
		Retried int64
	}
	Await struct {
		Timeout time.Duration
	}
)

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
		ID   string
		Name string
		Args []Reshaper
	}
	Choose struct {
		ID   string
		If   Predicate
		Then []Expr
		Else []Expr
	}
)

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
