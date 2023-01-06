package schema

import (
	"encoding/json"
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
	//panic(fmt.Sprintf("GoToSchema:unknown type x=%setter", x))
}

//func SchemaToGo(x Schema, rules []Rule) any {
//	return MustMatchSchema(
//		x,
//		func(x *Value) any {
//			return x.V
//		}, func(x *List) any {
//			var r []any
//			for _, v := range x.Items {
//				r = append(r, SchemaToGo(v, rules))
//			}
//			return r
//		}, func(x *Map) any {
//			var isMap bool
//			var r any = make(map[string]interface{})
//
//			for i := range rules {
//				if setter, ok := rules[i].(*TopLevel); ok {
//					rt := reflect.TypeOf(setter.setter)
//					if rt.Kind() == reflect.Ptr {
//						panic(fmt.Sprintf("RegisterName. Registred type must not be a pointer, but given %setter", setter.setter))
//					}
//					r = reflect.New(rt).Elem()
//					isMap = true
//					break
//				}
//			}
//
//			for i := range x.path {
//				key := x.path[i].Name
//				value := x.path[i].Value
//
//				if !isMap {
//					r.(map[string]interface{})[key] = SchemaToGo(value, rules)
//				} else {
//					f := r.(reflect.Value).FieldByName(key)
//					if f.IsValid() && f.CanSet() {
//						f.Set(reflect.ValueOf(SchemaToGo(value, rules)))
//					}
//				}
//			}
//
//			if isMap {
//				return r.(reflect.Value).Interface()
//			}
//			return r
//		})
//}

func SchemaToGo(x Schema, rules []RuleMatcher, path []any) any {
	return MustMatchSchema(
		x,
		func(x *Value) any {
			return x.V
		},
		func(x *List) any {
			var setter Setter = &NativeList{l: []any{}}
			for _, v := range x.Items {
				_ = setter.Set("value is ignored", SchemaToGo(v, rules, append(path, "[*]")))
			}

			return setter.Get()
		},
		func(x *Map) any {
			var setter Setter = &NativeMap{m: map[string]interface{}{}}
			for _, rule := range rules {
				newSetter, ok := rule.Match(path)
				if ok {
					setter = newSetter
					break
				}
			}

			for i := range x.Field {
				key := x.Field[i].Name
				value := x.Field[i].Value
				_ = setter.Set(key, SchemaToGo(value, rules, append(path, key)))
			}

			return setter.Get()
		})
}
