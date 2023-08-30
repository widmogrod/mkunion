package predicate

import "github.com/widmogrod/mkunion/x/schema"

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
		Location  string
		Operation string
		BindValue BindValue
	}
)

type BindValue = string
type ParamBinds map[BindValue]schema.Schema
