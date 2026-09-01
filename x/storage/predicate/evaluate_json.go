package predicate

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
)

// JSONEvaluator evaluates predicates against a decoded JSON document
// (the result of json.Unmarshal into any) of a value stored in the mkunion
// JSON encoding. Locations are user-facing and Go-shaped (Data.Age,
// Data["testutil.Branch"].Name, Data["$type"]); the shape given at construction
// resolves them into document paths once, then evaluation is pure map and
// slice walking - no reflection.
type JSONEvaluator struct {
	shape shape.Shape
	paths map[string][]JSONPath
}

func NewJSONEvaluator(s shape.Shape) *JSONEvaluator {
	return &JSONEvaluator{
		shape: s,
		paths: make(map[string][]JSONPath),
	}
}

// ResolvePaths resolves and memoises a location. Exposed so callers can
// validate query locations up front and reuse paths for sorting.
func (e *JSONEvaluator) ResolvePaths(location string) ([]JSONPath, error) {
	if paths, ok := e.paths[location]; ok {
		return paths, nil
	}
	paths, err := ResolveJSONPaths(e.shape, location)
	if err != nil {
		return nil, err
	}
	e.paths[location] = paths
	return paths, nil
}

func (e *JSONEvaluator) Evaluate(p Predicate, doc any, bind ParamBinds) (bool, error) {
	return MatchPredicateR2(
		p,
		func(x *And) (bool, error) {
			for _, sub := range x.L {
				ok, err := e.Evaluate(sub, doc, bind)
				if err != nil {
					return false, err
				}
				if !ok {
					return false, nil
				}
			}
			return true, nil
		},
		func(x *Or) (bool, error) {
			for _, sub := range x.L {
				ok, err := e.Evaluate(sub, doc, bind)
				if err != nil {
					return false, err
				}
				if ok {
					return true, nil
				}
			}
			return false, nil
		},
		func(x *Not) (bool, error) {
			ok, err := e.Evaluate(x.P, doc, bind)
			if err != nil {
				return false, err
			}
			return !ok, nil
		},
		func(x *Compare) (bool, error) {
			value, found, err := e.bindValue(x.BindValue, bind, doc)
			if err != nil {
				return false, err
			}
			if !found {
				return false, nil
			}

			paths, err := e.ResolvePaths(x.Location)
			if err != nil {
				return false, err
			}

			// A location can name alternatives (union variants) and
			// wildcards; the compare holds when any candidate satisfies it.
			for _, path := range paths {
				for _, candidate := range LookupJSONPath(doc, path) {
					cmp, comparable := CompareJSONValues(candidate, value)
					if !comparable {
						continue
					}
					ok, err := applyOperation(x.Operation, cmp)
					if err != nil {
						return false, err
					}
					if ok {
						return true, nil
					}
				}
			}
			return false, nil
		},
	)
}

func (e *JSONEvaluator) bindValue(x Bindable, params ParamBinds, doc any) (any, bool, error) {
	return MatchBindableR3(
		x,
		func(x *BindValue) (any, bool, error) {
			value, ok := params[x.BindName]
			if !ok {
				return nil, false, fmt.Errorf("predicate.JSONEvaluator: missing bind parameter %s", x.BindName)
			}
			return SchemaToJSONValue(value), true, nil
		},
		func(x *Literal) (any, bool, error) {
			return SchemaToJSONValue(x.Value), true, nil
		},
		func(x *Locatable) (any, bool, error) {
			paths, err := e.ResolvePaths(x.Location)
			if err != nil {
				return nil, false, err
			}
			for _, path := range paths {
				if values := LookupJSONPath(doc, path); len(values) > 0 {
					return values[0], true, nil
				}
			}
			return nil, false, nil
		},
	)
}

// LookupJSONPath walks a decoded JSON document along path and returns every
// value the path selects; [*] steps can select many.
func LookupJSONPath(doc any, path JSONPath) []any {
	current := []any{doc}
	for _, step := range path {
		var next []any
		for _, value := range current {
			switch {
			case step.Field != nil:
				if m, ok := value.(map[string]any); ok {
					if v, ok := m[*step.Field]; ok {
						next = append(next, v)
					}
				}
			case step.Index != nil:
				if l, ok := value.([]any); ok && *step.Index >= 0 && *step.Index < len(l) {
					next = append(next, l[*step.Index])
				}
			case step.Any:
				switch v := value.(type) {
				case []any:
					next = append(next, v...)
				case map[string]any:
					for _, item := range v {
						next = append(next, item)
					}
				}
			}
		}
		if len(next) == 0 {
			return nil
		}
		current = next
	}
	return current
}

// SchemaToJSONValue converts a schema.Schema bind value into the
// representation encoding/json produces when unmarshalling into any,
// so it compares naturally against document values.
func SchemaToJSONValue(s schema.Schema) any {
	switch x := s.(type) {
	case nil, *schema.None:
		return nil
	case *schema.Bool:
		return bool(*x)
	case *schema.Number:
		return float64(*x)
	case *schema.String:
		return string(*x)
	case *schema.Binary:
		return base64.StdEncoding.EncodeToString(*x)
	case *schema.List:
		result := make([]any, 0, len(*x))
		for _, item := range *x {
			result = append(result, SchemaToJSONValue(item))
		}
		return result
	case *schema.Map:
		result := make(map[string]any, len(*x))
		for key, value := range *x {
			result[key] = SchemaToJSONValue(value)
		}
		return result
	}
	return nil
}

// CompareJSONValues orders two decoded JSON values of the same kind.
// Values of different kinds are not comparable; note this makes both
// `=` and `!=` false for them, mirroring SQL's treatment of NULL.
func CompareJSONValues(a, b any) (int, bool) {
	if a == nil || b == nil {
		if a == nil && b == nil {
			return 0, true
		}
		return 0, false
	}

	switch av := a.(type) {
	case float64:
		bv, ok := b.(float64)
		if !ok {
			return 0, false
		}
		switch {
		case av < bv:
			return -1, true
		case av > bv:
			return 1, true
		default:
			return 0, true
		}
	case string:
		bv, ok := b.(string)
		if !ok {
			return 0, false
		}
		return strings.Compare(av, bv), true
	case bool:
		bv, ok := b.(bool)
		if !ok {
			return 0, false
		}
		switch {
		case av == bv:
			return 0, true
		case !av:
			return -1, true
		default:
			return 1, true
		}
	}
	return 0, false
}

func applyOperation(operation string, cmp int) (bool, error) {
	switch operation {
	case "=", "==":
		return cmp == 0, nil
	case "<":
		return cmp < 0, nil
	case ">":
		return cmp > 0, nil
	case "<=":
		return cmp <= 0, nil
	case ">=":
		return cmp >= 0, nil
	case "<>", "!=":
		return cmp != 0, nil
	default:
		return false, fmt.Errorf("predicate.JSONEvaluator: unknown operation %q", operation)
	}
}
