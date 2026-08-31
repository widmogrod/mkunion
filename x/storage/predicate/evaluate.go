package predicate

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"reflect"
	"sync"
)

// evaluateShapeCache holds indexed shapes keyed by reflect.Type,
// so repeated Evaluate calls skip the lookup and re-indexing.
var evaluateShapeCache sync.Map

func Evaluate[A any](predicate Predicate, data A, bind ParamBinds) bool {
	v := reflect.TypeOf(new(A)).Elem()

	var s shape.Shape
	if cached, ok := evaluateShapeCache.Load(v); ok {
		s = cached.(shape.Shape)
	} else {
		original := shape.MkRefNameFromReflect(v)

		found := false
		s, found = shape.LookupShape(original)
		if !found {
			panic(fmt.Errorf("predicate.Evaluate: shape.RefName not found %s; %w", v.String(), shape.ErrShapeNotFound))
		}

		s = shape.IndexWith(s, original)
		evaluateShapeCache.Store(v, s)
	}

	sdata := schema.FromGo[A](data)

	return EvaluateShape(predicate, s, sdata, bind)
}

// lookupLocation resolves a location within the evaluated data.
// It abstracts the difference between shape-aware and plain schema lookup.
type lookupLocation func(location string) (schema.Schema, bool)

func EvaluateShape(predicate Predicate, s shape.Shape, data schema.Schema, bind ParamBinds) bool {
	return evaluate(predicate, bind, func(location string) (schema.Schema, bool) {
		value, _, found := schema.GetShapeLocation(s, data, location)
		return value, found
	})
}

func EvaluateSchema(predicate Predicate, data schema.Schema, bind ParamBinds) bool {
	return evaluate(predicate, bind, func(location string) (schema.Schema, bool) {
		return schema.GetSchema(data, location)
	})
}

func evaluate(predicate Predicate, bind ParamBinds, lookup lookupLocation) bool {
	return MatchPredicateR1(
		predicate,
		func(x *And) bool {
			for _, p := range x.L {
				if !evaluate(p, bind, lookup) {
					return false
				}
			}

			return true
		},
		func(x *Or) bool {
			for _, p := range x.L {
				if evaluate(p, bind, lookup) {
					return true
				}
			}
			return false
		},
		func(x *Not) bool {
			return !evaluate(x.P, bind, lookup)
		},
		func(x *Compare) bool {
			value, found := getValue(x.BindValue, bind, lookup)
			if !found {
				return false
			}

			// Field value that is not set and equality is not about None is always false.
			fieldValue, found := lookup(x.Location)
			if !found {
				return false
			}

			cmp := schema.Compare(fieldValue, value)
			switch x.Operation {
			case "=", "==":
				return cmp == 0
			case "<":
				return cmp < 0
			case ">":
				return cmp > 0
			case "<=":
				return cmp <= 0
			case ">=":
				return cmp >= 0
			case "<>", "!=":
				return cmp != 0
			default:
				return false
			}
		},
	)
}

func GetValue(x Bindable, params ParamBinds, data schema.Schema) (schema.Schema, bool) {
	return getValue(x, params, func(location string) (schema.Schema, bool) {
		return schema.GetSchema(data, location)
	})
}

func getValue(x Bindable, params ParamBinds, lookup lookupLocation) (schema.Schema, bool) {
	return MatchBindableR2(
		x,
		func(x *BindValue) (schema.Schema, bool) {
			result, ok := params[x.BindName]
			return result, ok
		},
		func(x *Literal) (schema.Schema, bool) {
			return x.Value, true
		},
		func(x *Locatable) (schema.Schema, bool) {
			return lookup(x.Location)
		},
	)
}

func EvaluateEqual[A any](data A, location string, value any) bool {
	return Evaluate[A](
		&Compare{
			Location:  location,
			Operation: "=",
			BindValue: &BindValue{BindName: ":value"},
		},
		data,
		map[string]schema.Schema{
			":value": schema.FromGo(value),
		},
	)
}
