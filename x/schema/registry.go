package schema

import (
	"fmt"
	"reflect"
)

var defaultRegistry *Registry

func init() {
	defaultRegistry = NewRegistry()
}

func RegisterTransformations(xs []GoRuleMatcher) {
	defaultRegistry.RegisterRules(xs)
}

func RegisterRules(xs []GoRuleMatcher) {
	defaultRegistry.RegisterRules(xs)
}

func MustDefineUnion[A any](xs ...A) *UnionVariants[A] {
	result := UnionVariants[A]{
		unique: make(map[string]struct{}),
	}
	for _, x := range xs {
		t := reflect.TypeOf(x)
		if _, ok := result.unique[t.Name()]; ok {
			panic(fmt.Sprintf("union variant %s already defined", t.Name()))
		}
		result.variants = append(result.variants, x)
		result.reflections = append(result.reflections, t)
		result.unique[t.Name()] = struct{}{}
	}
	return &result
}

func RegisterUnionTypes[A any](x *UnionVariants[A]) {
	defaultRegistry.RegisterUnionType(x)
}

type UnionVariants[A any] struct {
	variants    []A
	reflections []reflect.Type
	unique      map[string]struct{}
}

func (u *UnionVariants[A]) IsPartOf(x any) bool {
	_, ok := x.(A)
	return ok
}

//func (u *UnionVariants[A]) TransformFunc() []TransformFunc {
//	var result []TransformFunc
//	for i, variant := range u.variants {
//		result = append(result, WrapStruct(variant, u.reflections[i].Name()))
//	}
//	return result
//}

func (u *UnionVariants[A]) Rules() []GoRuleMatcher {
	var result []GoRuleMatcher
	for i, variant := range u.variants {
		result = append(result, UnwrapStruct(variant, u.reflections[i].Name()))
	}
	return result
}

type Unioner interface {
	IsPartOf(x any) bool
	Rules() []GoRuleMatcher
	//TransformFunc() []TransformFunc
}

func NewRegistry() *Registry {
	return &Registry{
		//transformations: nil,
		matchingRules: nil,
	}
}

type Registry struct {
	//transformations []TransformFunc
	matchingRules []GoRuleMatcher
	unionTypes    []Unioner
}

//func (r *Registry) RegisterTransformations(xs []TransformFunc) {
//	r.transformations = append(r.transformations, xs...)
//}

func (r *Registry) RegisterRules(xs []GoRuleMatcher) {
	r.matchingRules = append(r.matchingRules, xs...)
}

func (r *Registry) RegisterUnionType(u Unioner) {
	r.unionTypes = append(r.unionTypes, u)

	//r.RegisterTransformations(u.TransformFunc())
	r.RegisterRules(u.Rules())
}
