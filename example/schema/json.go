package schema

import (
	"encoding/json"
	"log"
	"reflect"
)

func JsonToSchema(data []byte) (Schema, error) {
	var x any
	if err := json.Unmarshal(data, &x); err != nil {
		return nil, err
	}

	return GoToSchema(x), nil
}

func GoToSchema(x any) Schema {
	switch y := x.(type) {
	case []interface{}:
		var r = &List{}
		for _, v := range y {
			r.Items = append(r.Items, GoToSchema(v))
		}
		return r

	case map[string]any:
		var r = &Map{}
		for k, v := range y {
			r.Field = append(r.Field, Field{
				Name:  k,
				Value: GoToSchema(v),
			})
		}
		return r

	default:
		v := reflect.ValueOf(x)
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() == reflect.Map {
			m := make(map[string]any)
			for _, k := range v.MapKeys() {
				m[k.String()] = v.MapIndex(k).Interface()
			}
			return GoToSchema(m)
		}

		if v.Kind() == reflect.Struct {
			m := make(map[string]any)
			for i := 0; i < v.NumField(); i++ {
				name, ok := v.Type().Field(i).Tag.Lookup("name")
				if !ok {
					name = v.Type().Field(i).Name
				}
				m[name] = v.Field(i).Interface()
			}
			return GoToSchema(m)
		}
	}

	return &Value{V: x}
}

func SchemaToGo(x Schema, rules ...RuleMatcher) any {
	return SchemaToGoWithPath(x, rules, nil)
}

func SchemaToGoWithPath(x Schema, rules []RuleMatcher, path []any) any {
	return MustMatchSchema(
		x,
		func(x *Value) any {
			return x.V
		},
		func(x *List) any {
			var setter Setter = &NativeList{l: []any{}}
			for _, v := range x.Items {
				_ = setter.Set("value is ignored", SchemaToGoWithPath(v, rules, append(path, "[*]")))
			}

			return setter.Get()
		},
		func(x *Map) any {
			var setters []Setter
			for _, rule := range rules {
				newSetter, ok := rule.Match(path, x)
				if ok {
					setters = append(setters, newSetter)
				}
			}

			setters = append(setters, &NativeMap{m: map[string]interface{}{}})

			for _, setter := range setters {
				var err error
				for i := range x.Field {
					key := x.Field[i].Name
					value := x.Field[i].Value
					err = setter.Set(key, SchemaToGoWithPath(value, rules, append(path, key)))
					if err != nil {
						break
					}
				}

				if err != nil {
					log.Println("SchemaToGoWithPath: setter err. next looop. err:", err)
					continue
				}

				return setter.Get()
			}

			panic("reach unreachable!")
		})
}
