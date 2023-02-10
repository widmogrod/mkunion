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

type goConfig struct {
	defaultListDef TypeListDefinition
	defaultMapDef  TypeMapDefinition
	localRules     []GoRuleMatcher
	registry       *Registry
	useRegistry    bool
}

func (c *goConfig) ListDefFor(x *List, path []string) TypeListDefinition {
	return c.defaultListDef
}

func (c *goConfig) MapDefFor(x *Map, path []string) TypeMapDefinition {
	for _, rule := range c.localRules {
		if typeDef, ok := rule.MapDefFor(x, path); ok {
			return typeDef
		}
	}

	if c.useRegistry && c.registry != nil {
		for _, rule := range c.registry.rules {
			if typeDef, ok := rule.MapDefFor(x, path); ok {
				return typeDef
			}
		}
	}

	return c.defaultMapDef
}

func (c *goConfig) Transform(x any, r *Map) Schema {
	for _, rule := range c.localRules {
		v, ok := rule.SchemaToUnionType(x, r)
		if ok {
			return v
		}
	}

	if c.useRegistry {
		for _, rule := range c.registry.rules {
			v, ok := rule.SchemaToUnionType(x, r)
			if ok {
				return v
			}
		}
	}

	return r
}

type goConfigFunc func(c *goConfig)

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

func WithExtraRules(rules ...GoRuleMatcher) goConfigFunc {
	return func(c *goConfig) {
		c.localRules = append(c.localRules, rules...)
	}
}

func WithOnlyTheseRules(rules ...GoRuleMatcher) goConfigFunc {
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
	switch y := x.(type) {
	case nil:
		return &None{}

	case bool:
		return (*Bool)(&y)
	case *bool:
		if y == nil {
			return &None{}
		} else {
			return (*Bool)(y)
		}

	case string:
		return MkString(y)
	case *string:
		if y == nil {
			return &None{}
		} else {
			return MkString(*y)
		}

	case float64:
		v := Number(y)
		return &v
	case *float64:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case float32:
		v := Number(y)
		return &v
	case *float32:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case int:
		v := Number(y)
		return &v
	case *int:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case int8:
		v := Number(y)
		return &v
	case *int8:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case int16:
		v := Number(y)
		return &v
	case *int16:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case int32:
		v := Number(y)
		return &v
	case *int32:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case int64:
		v := Number(y)
		return &v
	case *int64:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case uint:
		v := Number(y)
		return &v
	case *uint:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case uint8:
		v := Number(y)
		return &v
	case *uint8:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case uint16:
		v := Number(y)
		return &v
	case *uint16:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case uint32:
		v := Number(y)
		return &v

	case *uint32:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case uint64:
		v := Number(y)
		return &v
	case *uint64:
		if y == nil {
			return &None{}
		} else {
			v := Number(*y)
			return &v
		}

	case []any:
		var r = &List{}
		for _, v := range y {
			r.Items = append(r.Items, goToSchema(v, c))
		}
		return r

	case map[string]any:
		var r = &Map{}
		for k, v := range y {
			r.Field = append(r.Field, Field{
				Name:  k,
				Value: goToSchema(v, c),
			})
		}
		return r

	case reflect.Value:
		return goToSchema(y.Interface(), c)

	default:
		v := reflect.ValueOf(x)

		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return &None{}
			}
			v = v.Elem()
		}

		if v.Kind() == reflect.Map {
			var r = &Map{}
			for _, k := range v.MapKeys() {
				r.Field = append(r.Field, Field{
					Name:  k.String(),
					Value: goToSchema(v.MapIndex(k), c),
				})
			}
			return r
		}

		if v.Kind() == reflect.Struct {
			var r = &Map{}
			for i := 0; i < v.NumField(); i++ {
				if !v.Type().Field(i).IsExported() {
					continue
				}

				name, ok := v.Type().Field(i).Tag.Lookup("name")
				if !ok {
					name = v.Type().Field(i).Name
				}

				r.Field = append(r.Field, Field{
					Name:  name,
					Value: goToSchema(v.Field(i), c),
				})
			}

			return c.Transform(x, r)
		}

		if v.Kind() == reflect.Slice {
			var r = &List{}
			for i := 0; i < v.Len(); i++ {
				r.Items = append(r.Items, goToSchema(v.Index(i), c))
			}
			return r
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
			return nil, nil
		},
		func(x *Bool) (any, error) {
			return bool(*x), nil
		},
		func(x *Number) (any, error) {
			return float64(*x), nil
		},
		func(x *String) (any, error) {
			return string(*x), nil
		},
		func(x *List) (any, error) {
			build := c.ListDefFor(x, path).NewListBuilder()
			for _, v := range x.Items {
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
			for _, field := range x.Field {
				value, err := schemaToGo(field.Value, c, append(path, field.Name))
				if err != nil {
					return nil, err
				}

				err = build.Set(field.Name, value)
				if err != nil {
					return nil, fmt.Errorf("schema.schemaToGo: at path %s, at type %T, cause %w", strings.Join(path, "."), x, err)
				}
			}

			return build.Build(), nil
		})
}
