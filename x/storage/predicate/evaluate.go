package predicate

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"reflect"
)

func Evaluate[A any](predicate Predicate, data A, bind ParamBinds) bool {
	v := reflect.TypeOf(new(A)).Elem()
	original := shape.MkRefNameFromReflect(v)

	s, found := shape.LookupShape(original)
	if !found {
		panic(fmt.Errorf("predicate.Evaluate: shape.RefName not found %s; %w", v.String(), shape.ErrShapeNotFound))
	}

	s = shape.IndexWith(s, original)

	sdata := schema.FromGo[A](data)

	return EvaluateShape(predicate, s, sdata, bind)
}

func EvaluateShape(predicate Predicate, s shape.Shape, data schema.Schema, bind ParamBinds) bool {
	return MatchPredicateR1(
		predicate,
		func(x *And) bool {
			for _, p := range x.L {
				if !EvaluateShape(p, s, data, bind) {
					return false
				}
			}
			return true

		},
		func(x *Or) bool {
			for _, p := range x.L {
				if EvaluateShape(p, s, data, bind) {
					return true
				}
			}
			return false
		},
		func(x *Not) bool {
			return !EvaluateShape(x.P, s, data, bind)
		},
		func(x *Compare) bool {
			value, ok := GetValue(x.BindValue, bind, data)
			if !ok {
				return false
			}

			loc := x.Location
			_ = loc

			// Field value that is not set and equality is not about None is always false.
			fieldValue, _ := schema.GetShapeLocation(s, data, x.Location)
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

func EvaluateSchema(predicate Predicate, data schema.Schema, bind ParamBinds) bool {
	return MatchPredicateR1(
		predicate,
		func(x *And) bool {
			for _, p := range x.L {
				if !EvaluateSchema(p, data, bind) {
					return false
				}
			}
			return true

		},
		func(x *Or) bool {
			for _, p := range x.L {
				if EvaluateSchema(p, data, bind) {
					return true
				}
			}
			return false
		},
		func(x *Not) bool {
			return !EvaluateSchema(x.P, data, bind)
		},
		func(x *Compare) bool {
			value, ok := GetValue(x.BindValue, bind, data)
			if !ok {
				return false
			}

			// Field value that is not set and equality is not about None is always false.
			fieldValue := schema.GetSchema(data, x.Location)
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
			return schema.GetSchema(data, x.Location), true
		},
	)
}

func EvaluateEqualPrimitive(record schema.Schema, location string, value any) bool {
	return EvaluateSchema(
		&Compare{
			Location:  location,
			Operation: "=",
			BindValue: &BindValue{BindName: ":value"},
		},
		record,
		map[string]schema.Schema{
			":value": schema.FromPrimitiveGo(value),
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
