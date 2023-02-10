package schema

import (
	"fmt"
	"reflect"
	"strings"
)

type fromGoConfig struct {
	transformations    []GoRuleMatcher
	useDefaultRegistry bool
}

type fromGoConfigFunc func(*fromGoConfig)

func WithTransformationsFromRegistry(r *Registry) fromGoConfigFunc {
	return WithOnlyTheseTransformations(r.matchingRules...)
}

func WithOnlyTheseTransformations(transformations ...GoRuleMatcher) fromGoConfigFunc {
	return func(c *fromGoConfig) {
		c.useDefaultRegistry = false
		c.transformations = transformations
	}
}

func WithExtraTransformations(transformations ...GoRuleMatcher) fromGoConfigFunc {
	return func(c *fromGoConfig) {
		c.transformations = append(c.transformations, transformations...)
	}
}

//func WithUnionsWrappedAsPkgName() fromGoConfigFunc {
//	return func(c *fromGoConfig) {
//		c.transformations = append(c.transformations, UnionAsPkgName)
//	}
//}

func FromGo(x any, options ...fromGoConfigFunc) Schema {
	c := fromGoConfig{
		useDefaultRegistry: true,
	}
	for _, option := range options {
		option(&c)
	}

	if c.useDefaultRegistry {
		c.transformations = append(c.transformations, defaultRegistry.matchingRules...)
	}

	return goToSchema(x, &c)
}

func goToSchema(x any, c *fromGoConfig) Schema {
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

			for _, rule := range c.transformations {
				v, ok := rule.Transform(x, r)
				if ok {
					return v
				}
			}

			return r
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

func MustToGo(x Schema, options ...toGoConfigFunc) any {
	v, err := ToGo(x, options...)
	if err != nil {
		panic(err)
	}
	return v
}

type toGoConfigFunc func(c *toGoConfig)

func WithRulesFromRegistry(registry *Registry) toGoConfigFunc {
	return WithOnlyTheseRules(registry.matchingRules...)
}
func WithoutDefaultRegistry() toGoConfigFunc {
	return func(c *toGoConfig) {
		c.useDefaultRegistry = false
	}
}

func WithExtraRules(rules ...GoRuleMatcher) toGoConfigFunc {
	return func(c *toGoConfig) {
		c.rules = append(c.rules, rules...)
	}
}

func WithOnlyTheseRules(rules ...GoRuleMatcher) toGoConfigFunc {
	return func(c *toGoConfig) {
		c.useDefaultRegistry = false
		c.rules = rules
	}
}

func WithDefaultMaoDef(def TypeMapDefinition) toGoConfigFunc {
	return func(c *toGoConfig) {
		c.defaultMapDef = def
	}
}

func WithDefaultListDef(def TypeListDefinition) toGoConfigFunc {
	return func(c *toGoConfig) {
		c.defaultListDef = def
	}
}

var defaultToGoConfig = toGoConfig{
	defaultListDef:     &NativeList{},
	defaultMapDef:      &NativeMap{},
	useDefaultRegistry: true,
}

func ToGo(x Schema, options ...toGoConfigFunc) (any, error) {
	c := defaultToGoConfig
	for _, option := range options {
		option(&c)
	}

	if c.useDefaultRegistry {
		c.rules = append(c.rules, defaultRegistry.matchingRules...)
	}

	return schemaToGo(x, &c, nil)
}

var unionMap = &UnionMap{}

type toGoConfig struct {
	defaultListDef     TypeListDefinition
	defaultMapDef      TypeMapDefinition
	rules              []GoRuleMatcher
	useDefaultRegistry bool
}

func (c *toGoConfig) ListDefFor(x *List, path []string) TypeListDefinition {
	return c.defaultListDef
}
func (c *toGoConfig) MapDefFor(x *Map, path []string) TypeMapDefinition {
	for _, rule := range c.rules {
		if _, ok, _ := rule.UnwrapField(x); ok {
			return unionMap
		}
		if typeDef, ok := rule.MatchPath(path, x); ok {
			return typeDef
		}
	}

	return c.defaultMapDef
}

func schemaToGo(x Schema, c *toGoConfig, path []string) (any, error) {
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
					return nil, fmt.Errorf("schemaToGo: at path %s, at type %T, cause %w", strings.Join(path, "."), x, err)
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
					return nil, fmt.Errorf("schemaToGo: at path %s, at type %T, cause %w", strings.Join(path, "."), x, err)
				}
			}

			return build.Build(), nil
		})
}
