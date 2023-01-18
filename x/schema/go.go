package schema

import (
	"fmt"
	"log"
	"reflect"
)

func FromGo(x any, transformations ...TransformFunc) Schema {
	finalTransformations := append(defaultRegistry.transformations, transformations...)
	return goToSchema(x, finalTransformations...)
}

func goToSchema(x any, transformations ...TransformFunc) Schema {
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

	case []interface{}:
		var r = &List{}
		for _, v := range y {
			r.Items = append(r.Items, goToSchema(v, transformations...))
		}
		return r

	case map[string]any:
		var r = &Map{}
		for k, v := range y {
			r.Field = append(r.Field, Field{
				Name:  k,
				Value: goToSchema(v, transformations...),
			})
		}
		return r

	case reflect.Value:
		return goToSchema(y.Interface(), transformations...)

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
					Value: goToSchema(v.MapIndex(k), transformations...),
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
					Value: goToSchema(v.Field(i), transformations...),
				})
			}

			for _, transformation := range transformations {
				v, ok := transformation(x, r)
				if ok {
					return v
				}
			}

			return r
		}

		if v.Kind() == reflect.Slice {
			var r = &List{}
			for i := 0; i < v.Len(); i++ {
				r.Items = append(r.Items, goToSchema(v.Index(i), transformations...))
			}
			return r
		}
	}

	panic(fmt.Errorf("goToSchema: unsupported type: %T", x))
}

func ToGo(x Schema, rules ...RuleMatcher) any {
	finalRules := append(defaultRegistry.matchingRules, rules...)
	return schemaToGo(x, finalRules, nil)
}

func schemaToGo(x Schema, rules []RuleMatcher, path []any) any {
	return MustMatchSchema(
		x,
		func(x *None) any {
			return nil
		},
		func(x *Bool) any {
			return bool(*x)
		},
		func(x *Number) any {
			return float64(*x)
		},
		func(x *String) any {
			return string(*x)
		},
		func(x *List) any {
			var setter Setter = &NativeList{l: []any{}}
			for _, v := range x.Items {
				_ = setter.Set("value is ignored", schemaToGo(v, rules, append(path, "[*]")))
			}

			return setter.Get()
		},
		func(x *Map) any {
			var setters []Setter
			for _, rule := range rules {
				if y, ok, field := rule.UnwrapField(x); ok {
					return schemaToGo(y, rules, append(path, field))
				}

				newSetter, ok := rule.MatchPath(path, x)
				if ok {
					setters = append(setters, newSetter)
					break
				}
			}

			setters = append(setters, &NativeMap{m: map[string]interface{}{}})

			for _, setter := range setters {
				var err error
				for i := range x.Field {
					key := x.Field[i].Name
					value := x.Field[i].Value
					err = setter.Set(key, schemaToGo(value, rules, append(path, key)))
					if err != nil {
						break
					}
				}

				if err != nil {
					log.Println("schemaToGo: setter err. next looop. err:", err)
					continue
				}

				return setter.Get()
			}

			panic("reach unreachable!")
		})
}
