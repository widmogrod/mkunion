package predicate

import (
	"github.com/widmogrod/mkunion/x/schema"
)

//go:generate go run ../../../cmd/mkunion/main.go
//go:generate go run ../../../cmd/mkunion/main.go serde

//go:tag mkunion:"Predicate"
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
		Location  string
		Operation string
		BindValue Bindable
	}
)

//go:tag mkunion:"Bindable"
type (
	BindValue struct{ BindName BindName }
	Literal   struct{ Value schema.Schema }
	Locatable struct{ Location string }
)

type BindName = string

//go:tag serde:"json"
type ParamBinds map[BindName]schema.Schema
