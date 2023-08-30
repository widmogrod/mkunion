package predicate

import "github.com/widmogrod/mkunion/x/schema"

func Evaluate(predicate Predicate, data schema.Schema, bind ParamBinds) bool {
	return MustMatchPredicate(
		predicate,
		func(x *And) bool {
			for _, p := range x.L {
				if !Evaluate(p, data, bind) {
					return false
				}
			}
			return true

		},
		func(x *Or) bool {
			for _, p := range x.L {
				if Evaluate(p, data, bind) {
					return true
				}
			}
			return false
		},
		func(x *Not) bool {
			return !Evaluate(x.P, data, bind)
		},
		func(x *Compare) bool {
			value, ok := bind[x.BindValue]
			if !ok {
				return false
			}

			// Field value that is not set and equality is not about None is always false.
			fieldValue := schema.Get(data, x.Location)
			cmp := schema.Compare(fieldValue, value)
			switch x.Operation {
			case "=":
				return cmp == 0
			case "<":
				return cmp < 0
			case ">":
				return cmp > 0
			case "<=":
				return cmp <= 0
			case ">=":
				return cmp >= 0
			case "<>":
				return cmp != 0
			default:
				return false
			}
		},
	)
}

func EvaluateEqual(record schema.Schema, location string, value any) bool {
	return Evaluate(
		&Compare{
			Location:  location,
			Operation: "=",
			BindValue: ":value",
		},
		record,
		map[string]schema.Schema{
			":value": schema.FromGo(value),
		},
	)
}
