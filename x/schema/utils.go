package schema

import (
	"strconv"
	"strings"
)

func As[A int | float64 | bool | string](x Schema, def A) A {
	if x == nil {
		return def
	}

	return MustMatchSchema(
		x,
		func(x *None) A {
			return def
		},
		func(x *Bool) A {
			switch any(def).(type) {
			case bool:
				return any(bool(*x)).(A)
			}

			return def
		},
		func(x *Number) A {
			switch any(def).(type) {
			case float64:
				return any(float64(*x)).(A)
			case int:
				return any(int(*x)).(A)
			}

			return def
		},
		func(x *String) A {
			switch any(def).(type) {
			case string:
				return any(string(*x)).(A)
			}

			return def
		},
		func(x *List) A {
			return def
		},
		func(x *Map) A {
			return def
		})
}

func Get(data Schema, location string) Schema {
	path := strings.Split(location, ".")
	for _, p := range path {
		if p == "" {
			return nil
		}

		if strings.HasPrefix(p, "[") && strings.HasSuffix(p, "]") {
			idx := strings.TrimPrefix(p, "[")
			idx = strings.TrimSuffix(idx, "]")
			i, err := strconv.Atoi(idx)
			if err != nil {
				return nil
			}

			listData, ok := data.(*List)
			if ok && len(listData.Items) > i {
				data = listData.Items[i]
				continue
			}

			return nil
		}

		mapData, ok := data.(*Map)
		if !ok {
			return nil
		}

		var found bool
		for _, item := range mapData.Field {
			if item.Name == p {
				found = true
				data = item.Value
				break
			}
		}

		if !found {
			return nil
		}
	}

	return data
}

func Reduce[A any](data Schema, init A, fn func(Schema, A) A) A {
	if data == nil {
		return init
	}

	return MustMatchSchema(
		data,
		func(x *None) A {
			return init
		},
		func(x *Bool) A {
			return fn(x, init)
		},
		func(x *Number) A {
			return fn(x, init)
		},
		func(x *String) A {
			return fn(x, init)
		},
		func(x *List) A {
			for _, y := range x.Items {
				init = fn(y, init)
			}

			return init
		},
		func(x *Map) A {
			for _, y := range x.Field {
				init = fn(y.Value, init)
			}

			return init
		},
	)
}
