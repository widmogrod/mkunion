package workflow

import "github.com/widmogrod/mkunion/x/schema"

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
)

//go:generate mkunion -name=ActivityT
type (
	Start struct {
		Var String
	}
	Choose struct {
		If   Predicate
		Then Workflow
		Else Workflow
	}
	Assign struct {
		Var  String
		Flow Workflow
	}
	Invocation struct {
		T1 Fid
		T2 Reshaper
	}
)

//go:generate mkunion -name=Workflow
type (
	Activity struct {
		Id       AID
		Activity ActivityT
	}
	Transition struct {
		From Workflow
		To   Workflow
	}
)

//go:generate mkunion -name=Predicate
type (
	Eq struct {
		Left  Reshaper
		Right Reshaper
	}
	Exists struct {
		Path Path
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
		T1 Path
	}
	SetValue struct {
		T1 schema.Schema
	}
)

type (
	Fid    = string
	Path   = []string
	AID    = string
	String = string
)
