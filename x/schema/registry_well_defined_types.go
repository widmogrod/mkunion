package schema

import "time"

func init() {
	RegisterWellDefinedTypesConversion[time.Duration](
		func(x time.Duration) Schema {
			return MkInt(int(x))
		},
		func(x Schema) time.Duration {
			if v, ok := x.(*Number); ok {
				return time.Duration(int64(v.N))
			}

			panic("invalid type")
		},
	)
	RegisterWellDefinedTypesConversion[time.Time](
		func(x time.Time) Schema {
			return MkString(x.Format(time.RFC3339Nano))
		},
		func(x Schema) time.Time {
			if v, ok := x.(*String); ok {
				t, _ := time.Parse(time.RFC3339Nano, v.S)
				return t
			}

			panic("invalid type")
		},
	)
}
