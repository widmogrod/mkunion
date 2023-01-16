package schema

func Reduce[A any](x Schema, f func(x Schema, agg A) A, init A) A {
	return MustMatchSchema(
		x,
		func(x *None) A {
			return f(x, init)
		},
		func(x *Bool) A {
			return f(x, init)
		},
		func(x *Number) A {
			return f(x, init)
		},
		func(x *String) A {
			return f(x, init)
		},
		func(x *List) A {
			for _, item := range x.Items {
				init = Reduce(item, f, init)
			}
			return init
		},
		func(x *Map) A {
			for _, item := range x.Field {
				init = Reduce(item.Value, f, init)
			}
			return init
		},
	)
}
