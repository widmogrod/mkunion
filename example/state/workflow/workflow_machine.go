package workflow

import "github.com/widmogrod/mkunion/x/schema"

//go:generate mkunion -name=State
type (
	Plan struct {
		Command string
		Input   string
	}
	Running struct {
		Command string
		Input   string
	}
	Done struct {
		Command string
		Input   string
		Result  string
	}
	Fail struct {
		Command string
		Input   string
		Error   string
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
)

//go:generate mkunion -name=Expr
type (
	End struct {
		Result Reshaper
		Fail   Reshaper
	}
	Assign struct {
		Var string
		Val Expr
	}
	Apply struct {
		Name string
		Args []Reshaper
	}
	Choose struct {
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
