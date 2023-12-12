package schema

import (
	"fmt"
	"reflect"
	"strings"
)

var (
	defaultListDef = &NativeList{}
	defaultMapDef  = &NativeMap{}
	unionMap       = &UnionMap{}
)

type WellDefinedFromToStrategy[T any] struct {
	ToSchema func(x T) Schema
	ToGo     func(x Schema) T
}

type goConfig struct {
	defaultListDef                  TypeListDefinition
	defaultMapDef                   TypeMapDefinition
	localRules                      []RuleMatcher
	registry                        *Registry
	useRegistry                     bool
	unionFormatter                  UnionFormatFunc
	activeBuilder                   any
	localWellDefinedTypesConversion map[string]WellDefinedFromToStrategy[any]
}

func (c *goConfig) ListDefFor(x *List, path []string) TypeListDefinition {
	return c.defaultListDef
}

func (c *goConfig) formatter() UnionFormatFunc {
	if c.unionFormatter == nil {
		return c.registry.unionFormatter
	}

	return c.unionFormatter
}

func (c *goConfig) MapDefFor(x *Map, path []string) TypeMapDefinition {
	for _, rule := range c.localRules {
		if typeDef, ok := rule.MapDefFor(x, path, c); ok {
			return typeDef
		}
	}

	if c.useRegistry && c.registry != nil {
		for _, rule := range c.registry.rules {
			if typeDef, ok := rule.MapDefFor(x, path, c); ok {
				return typeDef
			}
		}
	}

	return c.defaultMapDef
}

func (c *goConfig) Transform(x any, r Schema) Schema {
	for _, rule := range c.localRules {
		v, ok := rule.SchemaToUnionType(x, r, c)
		if ok {
			return v
		}
	}

	if c.useRegistry {
		for _, rule := range c.registry.rules {
			v, ok := rule.SchemaToUnionType(x, r, c)
			if ok {
				return v
			}
		}
	}

	return r
}

func (c *goConfig) variantName(r reflect.Type) string {
	return c.formatter()(r)
}

func (c *goConfig) typeName(r reflect.Type) string {
	return c.formatter()(r)
}

func (c *goConfig) RegisterStrategy(r reflect.Type, x WellDefinedFromToStrategy[any]) {
	if c.localWellDefinedTypesConversion == nil {
		c.localWellDefinedTypesConversion = make(map[string]WellDefinedFromToStrategy[any])
	}

	name := c.typeName(r)
	c.localWellDefinedTypesConversion[name] = x
}

func (c *goConfig) WellDefinedTypeToSchema(x any) Schema {
	r := reflect.TypeOf(x)
	name := c.typeName(r)
	if wellDefined, ok := c.localWellDefinedTypesConversion[name]; ok {
		return wellDefined.ToSchema(x)
	}

	if c.useRegistry {
		if wellDefined, ok := c.registry.wellDefinedTypesConversion[name]; ok {
			return wellDefined.ToSchema(x)
		}
	}

	return nil
}

func (c *goConfig) WellDefinedTypeToGo(x Schema, r reflect.Type) any {
	name := c.typeName(r)
	if wellDefined, ok := c.localWellDefinedTypesConversion[name]; ok {
		return wellDefined.ToGo(x)
	}

	if c.useRegistry {
		if wellDefined, ok := c.registry.wellDefinedTypesConversion[name]; ok {
			return wellDefined.ToGo(x)
		}
	}

	return nil
}

type goConfigFunc func(c *goConfig)

func WithWellDefinedTypeConversion[T any](from func(T) Schema, to func(Schema) T) goConfigFunc {
	return func(c *goConfig) {
		var t T

		r := reflect.TypeOf(t)
		c.RegisterStrategy(r, NewWellDefinedFromToStrategy(from, to))
	}
}

// NewWellDefinedFromToStrategy assumption is that from and to functions are symmetrical
// and that function works on values, not pointers (to reduce implementation complexity)
// and if there is need to do conversion on pointers, then it should be done in "runtime" wrapper (here)
func NewWellDefinedFromToStrategy[T any](from func(T) Schema, to func(Schema) T) WellDefinedFromToStrategy[any] {
	return WellDefinedFromToStrategy[any]{
		ToSchema: func(x any) Schema {
			if y, ok := x.(T); ok {
				return from(y)
			}

			if y, ok := x.(*T); ok {
				// check if value can be dereference, not using reflection
				if y != nil {
					return from(*y)
				} else {
					return MkNone()
				}
			}

			panic(fmt.Errorf("schema.NewWellDefinedFromToStrategy: invalid type %T", x))
		},
		ToGo: func(x Schema) any {
			return any(to(x))
		},
	}
}

func WithRulesFromRegistry(registry *Registry) goConfigFunc {
	return func(c *goConfig) {
		c.useRegistry = true
		c.registry = registry
	}
}

func WithoutDefaultRegistry() goConfigFunc {
	return func(c *goConfig) {
		c.useRegistry = false
	}
}

func WithExtraRules(rules ...RuleMatcher) goConfigFunc {
	return func(c *goConfig) {
		c.localRules = append(c.localRules, rules...)
	}
}

func WithOnlyTheseRules(rules ...RuleMatcher) goConfigFunc {
	return func(c *goConfig) {
		c.useRegistry = false
		c.localRules = rules
	}
}

func WithDefaultMaoDef(def TypeMapDefinition) goConfigFunc {
	return func(c *goConfig) {
		c.defaultMapDef = def
	}
}

func WithDefaultListDef(def TypeListDefinition) goConfigFunc {
	return func(c *goConfig) {
		c.defaultListDef = def
	}
}
func WithUnionFormatter(f UnionFormatFunc) goConfigFunc {
	return func(c *goConfig) {
		c.unionFormatter = f
	}
}

func FromGo(x any, options ...goConfigFunc) Schema {
	c := goConfig{
		useRegistry: true,
		registry:    defaultRegistry,
	}
	for _, option := range options {
		option(&c)
	}

	return goToSchema(x, &c)
}

func goToSchema(x any, c *goConfig) Schema {
	if to, ok := x.(Marshaler); ok {
		schemed, err := to.MarshalSchema()
		if err != nil {
			panic(err)
		}
		return schemed
	}

	switch y := x.(type) {
	case Schema:
		return c.Transform(x, y)

	case nil:
		return &None{}

	case bool:
		return MkBool(y)
	case *bool:
		if y == nil {
			return &None{}
		} else {
			return MkBool(*y)
		}

	case string:
		return MkString(y)
	case *string:
		if y == nil {
			return &None{}
		} else {
			return MkString(*y)
		}

	case []byte:
		return &Binary{B: y}
	case *[]byte:
		if y == nil {
			return &None{}
		} else {
			return &Binary{B: *y}
		}

	case float64:
		return MkFloat(y)
	case *float64:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(*y)
		}

	case float32:
		return MkFloat(float64(y))
	case *float32:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case int:
		return MkFloat(float64(y))
	case *int:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case int8:
		return MkFloat(float64(y))
	case *int8:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case int16:
		return MkFloat(float64(y))
	case *int16:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case int32:
		return MkFloat(float64(y))
	case *int32:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case int64:
		return MkFloat(float64(y))

	case *int64:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case uint:
		return MkFloat(float64(y))
	case *uint:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case uint8:
		return MkFloat(float64(y))
	case *uint8:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case uint16:
		return MkFloat(float64(y))
	case *uint16:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case uint32:
		return MkFloat(float64(y))
	case *uint32:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case uint64:
		return MkFloat(float64(y))
	case *uint64:
		if y == nil {
			return &None{}
		} else {
			return MkFloat(float64(*y))
		}

	case []any:
		var r = List{}
		for _, v := range y {
			r = append(r, goToSchema(v, c))
		}
		return &r

	case map[string]any:
		var r = make(Map)
		for k, v := range y {
			r[k] = goToSchema(v, c)
		}
		return &r

	case reflect.Value:
		return goToSchema(y.Interface(), c)

	default:
		if definedType := c.WellDefinedTypeToSchema(x); definedType != nil {
			return definedType
		}

		v := reflect.ValueOf(x)

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return &None{}
			}
			v = v.Elem()
		}

		if v.Kind() == reflect.Map {
			var r = make(Map)
			for _, k := range v.MapKeys() {
				r[k.String()] = goToSchema(v.MapIndex(k), c)
			}
			return &r
		}

		if v.Kind() == reflect.Struct {
			var r = make(Map)
			for i := 0; i < v.NumField(); i++ {
				if !v.Type().Field(i).IsExported() {
					continue
				}

				name, ok := v.Type().Field(i).Tag.Lookup("name")
				if !ok {
					name = v.Type().Field(i).Name
				}

				r[name] = goToSchema(v.Field(i), c)
			}

			return c.Transform(x, &r)
		}

		if v.Kind() == reflect.Slice {
			var r = List{}
			for i := 0; i < v.Len(); i++ {
				r = append(r, goToSchema(v.Index(i), c))
			}
			return &r
		}
	}

	panic(fmt.Errorf("goToSchema: unsupported type: %T", x))
}

func MustToGo(x Schema, options ...goConfigFunc) any {
	v, err := ToGo(x, options...)
	if err != nil {
		panic(err)
	}
	return v
}

func ToGo(x Schema, options ...goConfigFunc) (any, error) {
	c := goConfig{
		defaultListDef: defaultListDef,
		defaultMapDef:  defaultMapDef,
		useRegistry:    true,
		registry:       defaultRegistry,
	}
	for _, option := range options {
		option(&c)
	}

	return schemaToGo(x, &c, nil)
}

func schemaToGo(x Schema, c *goConfig, path []string) (any, error) {
	return MustMatchSchemaR2(
		x,
		func(x *None) (any, error) {
			// it means that schema was serialised, and when being deserialized,
			// leaf variants, needs to be returned as is.
			// serialize(Struct{K: V}) == deserialize(Map(K=>V))
			// serialize(String(abc)) == deserialize(UnionMap(String => abc)
			// serialize(List(Map{k=>v})) == deserialize(UnionMap(List => Map{k => v}))
			if _, ok := c.activeBuilder.(*UnionMap); ok {
				return x, nil
			}

			return nil, nil
		},
		func(x *Bool) (any, error) {
			if _, ok := c.activeBuilder.(*UnionMap); ok {
				return x, nil
			}

			return bool(*x), nil
		},
		func(x *Number) (any, error) {
			if _, ok := c.activeBuilder.(*UnionMap); ok {
				return x, nil
			}

			return float64(*x), nil
		},
		func(x *String) (any, error) {
			if _, ok := c.activeBuilder.(*UnionMap); ok {
				return x, nil
			}

			return string(*x), nil
		},
		func(x *Binary) (any, error) {
			if _, ok := c.activeBuilder.(*UnionMap); ok {
				return x, nil
			}

			return x.B, nil
		},
		func(x *List) (any, error) {
			build := c.ListDefFor(x, path).NewListBuilder()
			for _, v := range *x {
				c.activeBuilder = build
				value, err := schemaToGo(v, c, append(path, "[*]"))
				if err != nil {
					return nil, err
				}

				err = build.Append(value)
				if err != nil {
					return nil, fmt.Errorf("schema.schemaToGo: at path %s, at type %T, cause %w", strings.Join(path, "."), x, err)
				}
			}

			return build.Build(), nil
		},
		func(x *Map) (any, error) {
			build := c.MapDefFor(x, path).NewMapBuilder()

			// If the builder can process raw map schemas, do so.
			// this is optional, and is only used for the Unmarshaler.
			if b, ok := build.(mapBuilderCanProcessRawMapSchema); ok {
				return b.BuildFromMapSchema(x)
			}

			if inject, ok := build.(wellDefinedSupported); ok {
				inject.WithWellDefinedTypesConversion(c.WellDefinedTypeToGo)
			}

			for key, value := range *x {
				c.activeBuilder = build
				value, err := schemaToGo(value, c, append(path, key))
				if err != nil {
					return nil, err
				}

				err = build.Set(key, value)
				if err != nil {
					return nil, fmt.Errorf("schema.schemaToGo: at path %s, at type %T, cause %w", strings.Join(path, "."), x, err)
				}
			}

			return build.Build(), nil
		})
}

// ToGoG converts a Schema to a Go value of the given type.
func ToGoG[A any](x Schema, options ...goConfigFunc) (A, error) {
	var a A
	var result any
	var err error

	if x == nil {
		return a, nil
	}

	switch any(a).(type) {
	case int:
		result = AsDefault[int](x, any(a).(int))
	case int8:
		result = AsDefault[int8](x, any(a).(int8))
	case int16:
		result = AsDefault[int16](x, any(a).(int16))
	case int32:
		result = AsDefault[int32](x, any(a).(int32))
	case int64:
		result = AsDefault[int64](x, any(a).(int64))
	case uint:
		result = AsDefault[uint](x, any(a).(uint))
	case uint8:
		result = AsDefault[uint8](x, any(a).(uint8))
	case uint16:
		result = AsDefault[uint16](x, any(a).(uint16))
	case uint32:
		result = AsDefault[uint32](x, any(a).(uint32))
	case uint64:
		result = AsDefault[uint64](x, any(a).(uint64))
	case float32:
		result = AsDefault[float32](x, any(a).(float32))
	case float64:
		result = AsDefault[float64](x, any(a).(float64))
	case string:
		result = AsDefault[string](x, any(a).(string))
	case bool:
		result = AsDefault[bool](x, any(a).(bool))
	case []byte:
		result = AsDefault[[]byte](x, any(a).([]byte))
	default:
		// Short circuit, if expected type is already a schema
		// and when you wonder why such strange type assertion?
		//
		//		var a A
		//		if _, ok := a.(Schema); ok {
		//			return x.(A), nil
		//		}
		//
		// above code results in error:
		//
		// 		invalid operation: cannot use type assertion on type parameter value a (variable of type A constrained by any)
		//
		// To avoid this, we need to do cast a to interface{} first.
		//
		//		if _, ok := any(a).(Schema) {...}
		//
		// But doing it that way, it removes all types that could be interfaces.
		// For example, generic type "ToGoG" can be set as "Schema", which is interface
		// and doing type conversion like this, will result in ok == false
		//
		// 		var a Schema  // in code is var a A
		//  	_, ok := any(a).(Schema)
		//
		// reason is that Schema is interface, and any(a) results interface{}
		// which is not the same as Schema.
		//
		// so to preserve original type, we do pointer type assertion
		if _, ok := any((*A)(nil)).(*Schema); ok {
			return x.(A), nil
		}

		if any(a) == nil {
			result, err = ToGo(x, options...)
		} else {
			options := append(options, WithExtraRules(WhenPath(nil, UseStruct(a))))
			result, err = ToGo(x, options...)
		}

		if err != nil {
			var a A
			return a, fmt.Errorf("schema.ToGoG[%T] schema conversion failed. %w", any((*A)(nil)), err)
		}
	}

	typed, ok := result.(A)
	if !ok {
		var a A
		return a, fmt.Errorf("schema.ToGoG[%T] type assertion failed on type %T", any((*A)(nil)), result)
	}

	return typed, nil
}
