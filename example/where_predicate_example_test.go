package example

import (
	"testing"
)

//go:generate go run ../cmd/mkunion/main.go -name=WherePredicate -types=Eq,Gt,Lt,And,Or,Path -path=where_predicate_example_gen_test -packageName=example
type (
	Eq   struct{ V interface{} }
	Gt   struct{ V interface{} }
	Lt   struct{ V interface{} }
	And  []WherePredicate
	Or   []WherePredicate
	Path struct {
		Parts     []string
		Condition WherePredicate
	}
)

func TestPredicate(t *testing.T) {
	_ = And{
		&Path{
			Parts:     []string{"name"},
			Condition: &Eq{"bar"},
		},
	}
}
