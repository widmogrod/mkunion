package schema

import (
	"reflect"
	"strings"
)

var defaultRegistry *Registry

func init() {
	defaultRegistry = NewRegistry()

	RegisterUnionTypes(SchemaSchemaDef())
}

func RegisterRules(xs []RuleMatcher) {
	defaultRegistry.RegisterRules(xs)
}

type UnionFormatFunc func(t reflect.Type) string

func FormatUnionNameUsingFullName(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		return t.Elem().PkgPath() + "." + t.Elem().Name()
	}
	return t.PkgPath() + "." + t.Name()
}

func FormatUnionNameUsingTypeName(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	}
	return t.Name()
}
func FormatUnionNameUsingTypeNameWithPackage(t reflect.Type) string {
	// remove information about pointer types, eg. *ast.Ast -> ast.Ast
	return strings.TrimLeft(t.String(), "*")
}

func SetDefaultUnionTypeFormatter(f UnionFormatFunc) {
	defaultRegistry.SetUnionTypeFormatter(f)
}

func RegisterUnionTypes[A any](x *UnionVariants[A]) {
	defaultRegistry.RegisterRules([]RuleMatcher{x})
}

func UnionOf(t reflect.Type) (reflect.Type, []reflect.Type, bool) {
	return defaultRegistry.UnionOf(t)
}

func NewRegistry() *Registry {
	return &Registry{
		rules:          nil,
		unionFormatter: FormatUnionNameUsingTypeNameWithPackage,
	}
}

type Registry struct {
	rules          []RuleMatcher
	unionFormatter func(t reflect.Type) string
}

func (r *Registry) RegisterRules(xs []RuleMatcher) {
	r.rules = append(r.rules, xs...)
}

func (r *Registry) SetUnionTypeFormatter(f UnionFormatFunc) {
	r.unionFormatter = f
}

func (r *Registry) UnionOf(t reflect.Type) (reflect.Type, []reflect.Type, bool) {
	for _, x := range r.rules {
		if y, ok := x.(UnionInformationRule); ok {
			if y.IsUnionOrUnionType(t) {
				return y.UnionType(), y.VariantsTypes(), true
			}
		}
	}

	return nil, nil, false
}
