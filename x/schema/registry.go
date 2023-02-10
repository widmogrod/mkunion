package schema

import (
	"fmt"
	"reflect"
)

var defaultRegistry *Registry

func init() {
	defaultRegistry = NewRegistry()
}

func RegisterRules(xs []GoRuleMatcher) {
	defaultRegistry.RegisterRules(xs)
}

func MustDefineUnion[A any](xs ...A) *UnionVariants[A] {
	result := UnionVariants[A]{
		unique:      make(map[string]struct{}),
		pathToUnion: make(map[string]*StructDefinition),
	}
	for _, x := range xs {
		t := reflect.TypeOf(x)
		if _, ok := result.unique[t.Elem().Name()]; ok {
			panic(fmt.Errorf("union variant %s already defined %T", t.Elem().Name(), x))
		}
		result.variants = append(result.variants, x)
		result.reflections = append(result.reflections, t)
		result.unique[t.Elem().Name()] = struct{}{}
	}
	return &result
}

func RegisterUnionTypes[A any](x *UnionVariants[A]) {
	defaultRegistry.RegisterRules([]GoRuleMatcher{x})
}

func NewRegistry() *Registry {
	return &Registry{
		rules: nil,
	}
}

type Registry struct {
	rules []GoRuleMatcher
}

func (r *Registry) RegisterRules(xs []GoRuleMatcher) {
	r.rules = append(r.rules, xs...)
}
