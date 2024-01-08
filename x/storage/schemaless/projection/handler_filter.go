package projection

import (
	"github.com/widmogrod/mkunion/x/storage/predicate"
)

var _ Handler = &FilterHandler{}

type FilterHandler struct {
	Where *predicate.WherePredicates
}

func (f *FilterHandler) Process(x Item, returning func(Item)) error {
	panic("not implemented")
	//if f.Where.Evaluate(x.Data) {
	//	returning(x)
	//}

	return nil
}

func (f *FilterHandler) Retract(x Item, returning func(Item)) error {
	panic("not implemented")
	//return f.Process(x, returning)
	return nil
}
