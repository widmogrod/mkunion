package predicate

import (
	"github.com/widmogrod/mkunion/x/schema"
)

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

// IsMatchAll reports whether p matches every record: an And with no
// sub-predicates, as returned by Where(""). Backends that translate
// predicates into native filters can skip such a predicate entirely.
func IsMatchAll(p Predicate) bool {
	x, ok := p.(*And)
	return ok && len(x.L) == 0
}

type BindName = string

//go:tag serde:"json"
type ParamBinds map[BindName]schema.Schema
