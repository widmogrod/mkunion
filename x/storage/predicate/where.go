package predicate

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"golang.org/x/exp/slices"
	"strings"
)

//go:tag serde:"json"
type WherePredicates struct {
	Predicate Predicate
	Params    ParamBinds
	Shape     shape.Shape
}

func (w *WherePredicates) Evaluate(data schema.Schema) bool {
	if w.Shape == nil {
		return EvaluateSchema(w.Predicate, data, w.Params)
	} else {
		return EvaluateShape(w.Predicate, w.Shape, data, w.Params)
	}
}

type WhereOpt struct {
	AllowExtraParams bool
	WithShapeDef     shape.Shape
}

var DefaultWhereOpt = WhereOpt{
	AllowExtraParams: false,
}

func Where(query string, params ParamBinds, opts *WhereOpt) (*WherePredicates, error) {
	if opts == nil {
		opts = &DefaultWhereOpt
	}

	if query == "" {
		return nil, nil
	}

	predicates, err := Parse(query)
	if err != nil {
		return nil, err
	}

	var missingParams, extraParams []string

	paramsInPredicate := bindValuesFromPredicate(predicates, nil)
	for _, param := range paramsInPredicate {
		if _, ok := params[param]; !ok {
			missingParams = append(missingParams, param)
		}
	}

	// find params that are in params but not in predicate
	for param := range params {
		if !slices.Contains(paramsInPredicate, param) {
			extraParams = append(extraParams, param)
		}
	}

	if (len(extraParams) > 0 && !opts.AllowExtraParams) ||
		len(missingParams) > 0 {
		message := strings.Builder{}
		if len(missingParams) > 0 {
			message.WriteString(fmt.Sprintf(`missing params: "%s"`, strings.Join(missingParams, `", "`)))
		}
		if len(extraParams) > 0 {
			if message.Len() > 0 {
				message.WriteString(", ")
			}
			message.WriteString(fmt.Sprintf(`extra params: "%s"`, strings.Join(extraParams, `", "`)))
		}

		return nil, fmt.Errorf(message.String())
	}

	return &WherePredicates{
		Predicate: predicates,
		Params:    params,
		Shape:     opts.WithShapeDef,
	}, nil
}

func MustWhere(query string, params ParamBinds, opts *WhereOpt) *WherePredicates {
	where, err := Where(query, params, opts)
	if err != nil {
		panic(err)
	}
	return where
}

func bindValuesFromPredicate(predicate Predicate, params []string) []string {
	return MatchPredicateR1(
		predicate,
		func(x *And) []string {
			for _, p := range x.L {
				params = bindValuesFromPredicate(p, params)
			}
			return params
		},
		func(x *Or) []string {
			for _, p := range x.L {
				params = bindValuesFromPredicate(p, params)
			}
			return params
		},
		func(x *Not) []string {
			return bindValuesFromPredicate(x.P, params)
		},
		func(x *Compare) []string {
			if bind, ok := x.BindValue.(*BindValue); ok {
				return append(params, bind.BindName)
			}

			return params
		},
	)
}
