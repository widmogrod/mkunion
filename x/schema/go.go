package schema

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
	"reflect"
)

func IsPrimitive(x any) bool {
	switch x.(type) {
	case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, string, []byte:
		return true

	case *bool, *int, *int8, *int16, *int32, *int64, *uint, *uint8, *uint16, *uint32, *uint64, *float32, *float64, *string, *[]byte:
		return true
	}

	return false
}

func FromPrimitiveGo(x any) Schema {
	switch y := x.(type) {
	case bool:
		return MkBool(y)

	case int:
		return MkInt(int64(y))

	case int8:
		return MkInt(int64(y))

	case int16:
		return MkInt(int64(y))

	case int32:
		return MkInt(int64(y))

	case int64:
		return MkInt(int64(y))

	case uint:
		return MkUint(uint64(y))

	case uint8:
		return MkUint(uint64(y))

	case uint16:
		return MkUint(uint64(y))

	case uint32:
		return MkUint(uint64(y))

	case uint64:
		return MkUint(uint64(y))

	case float32:
		return MkFloat(float64(y))

	case float64:
		return MkFloat(y)

	case string:
		return MkString(y)

	case []byte:
		return MkBinary(y)

	case []any:
		result := List{}
		for _, item := range y {
			result = append(result, FromPrimitiveGo(item))
		}

		return &result

	case map[any]any:
		result := Map{}
		for key, value := range y {
			result[key.(string)] = FromPrimitiveGo(value)
		}

		return &result
	}

	if x == nil {
		return MkNone()
	}

	if y, ok := x.(Schema); ok {
		return y
	}

	panic(fmt.Errorf("schema.FromPrimitiveGo: unknown type %T", x))
}

func ToGoPrimitive(x Schema) (any, error) {
	return MatchSchemaR2(
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
		func(x *Binary) (any, error) {
			return []byte(*x), nil
		},
		func(x *List) (any, error) {
			result := []any{}
			for _, item := range *x {
				item, err := ToGoPrimitive(item)
				if err != nil {
					return nil, err
				}

				result = append(result, item)
			}

			return result, nil
		},
		func(x *Map) (any, error) {
			result := map[any]any{}
			for key, value := range *x {
				key, err := ToGoPrimitive(MkString(key))
				if err != nil {
					return nil, err
				}

				value, err := ToGoPrimitive(value)
				if err != nil {
					return nil, err
				}

				result[key] = value
			}

			return result, nil
		},
	)
}

func ToGoG[A any](x Schema) (res A, err error) {
	//defer func() {
	//	if r := recover(); r != nil {
	//		if e, ok := r.(error); ok {
	//			err = fmt.Errorf("schema.ToGoG: panic recover; %w", e)
	//		} else {
	//			err = fmt.Errorf("schema.ToGoG: panic recover; %#v", e)
	//		}
	//	}
	//}()

	res = ToGo[A](x)
	return
}

func ToGo[A any](x Schema) A {
	if IsPrimitive(new(A)) {
		value, err := ToGoPrimitive(x)
		if err != nil {
			panic(fmt.Errorf("schema.ToGo: primitive; %w", err))
		}

		var result A
		switch y := value.(type) {
		case float64:
			switch any(result).(type) {
			case int:
				return any(int(y)).(A)
			//case int8:
			//	return any(int8(y)).(A)
			//case int16:
			//	return any(int16(y)).(A)
			//case int32:
			//	return any(int32(y)).(A)
			//case int64:
			//	return any(int64(y)).(A)
			//case uint:
			//	return any(uint(y)).(A)
			//case uint8:
			//	return any(uint8(y)).(A)
			//case uint16:
			//	return any(uint16(y)).(A)
			//case uint32:
			//	return any(uint32(y)).(A)
			//case uint64:
			//	return any(uint64(y)).(A)
			//case float32:
			//	return any(float32(y)).(A)
			case float64:
				return any(float64(y)).(A)
			}
		}
	}

	v := reflect.TypeOf(new(A)).Elem()
	original := shape.MkRefNameFromReflect(v)

	s, found := shape.LookupShape(original)
	if !found {
		panic(fmt.Errorf("schema.FromGo: shape.RefName not found %s; %w", v.String(), shape.ErrShapeNotFound))
	}

	s = shape.IndexWith(s, original)

	value, err := ToGoReflect(s, x, v)
	if err != nil {
		panic(fmt.Errorf("schema.ToGo: %w", err))
	}

	return value.Interface().(A)
}

func FromGo[A any](x A) Schema {
	if IsPrimitive(x) {
		return FromPrimitiveGo(x)
	}

	s, found := shape.LookupShapeReflectAndIndex[A]()
	if !found {
		panic(fmt.Errorf("schema.FromGo: shape.RefName not found for %T; %w", *new(A), shape.ErrShapeNotFound))
	}

	return FromGoReflect(s, reflect.ValueOf(x))
}

func FromGoReflect(xschema shape.Shape, yreflect reflect.Value) Schema {
	return shape.MatchShapeR1(
		xschema,
		func(x *shape.Any) Schema {
			panic("schema.FromGoReflect: not implemented shape.Any to Schema")
		},
		func(x *shape.RefName) Schema {
			y, found := shape.LookupShape(x)
			if !found {
				panic(fmt.Errorf("schema.FromGoReflect: shape.RefName not found %s; %w",
					shape.ToGoTypeName(x, shape.WithPkgImportName()),
					shape.ErrShapeNotFound))
			}

			y = shape.IndexWith(y, x)

			return FromGoReflect(y, yreflect)
		},
		func(x *shape.PointerLike) Schema {
			if yreflect.IsNil() {
				return MkNone()
			}

			return FromGoReflect(x.Type, yreflect.Elem())
		},
		func(x *shape.AliasLike) Schema {
			return FromGoReflect(x.Type, yreflect)
		},
		func(x *shape.PrimitiveLike) Schema {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) Schema {
					if yreflect.Kind() != reflect.Bool {
						panic(fmt.Errorf("schema.FromGoReflect: shape.BooleanLike expected reflect.Bool, got %s", yreflect.String()))
					}

					return MkBool(yreflect.Bool())
				},
				func(x *shape.StringLike) Schema {
					if yreflect.Kind() != reflect.String {
						panic(fmt.Errorf("schema.FromGoReflect: shape.StringLike expected reflect.String, got %s", yreflect.String()))
					}

					return MkString(yreflect.String())
				},
				func(x *shape.NumberLike) Schema {
					return shape.MatchNumberKindR1(
						x.Kind,
						func(x *shape.UInt) Schema {
							return MkUint(yreflect.Uint())
						},
						func(x *shape.UInt8) Schema {
							return MkUint(yreflect.Uint())
						},
						func(x *shape.UInt16) Schema {
							return MkUint(yreflect.Uint())
						},
						func(x *shape.UInt32) Schema {
							return MkUint(yreflect.Uint())
						},
						func(x *shape.UInt64) Schema {
							return MkUint(yreflect.Uint())
						},
						func(x *shape.Int) Schema {
							return MkInt(yreflect.Int())
						},
						func(x *shape.Int8) Schema {
							return MkInt(yreflect.Int())
						},
						func(x *shape.Int16) Schema {
							return MkInt(yreflect.Int())
						},
						func(x *shape.Int32) Schema {
							return MkInt(yreflect.Int())
						},
						func(x *shape.Int64) Schema {
							return MkInt(yreflect.Int())
						},
						func(x *shape.Float32) Schema {
							return MkFloat(yreflect.Float())
						},
						func(x *shape.Float64) Schema {
							return MkFloat(yreflect.Float())
						},
					)
				},
			)
		},
		func(x *shape.ListLike) Schema {
			if yreflect.Kind() != reflect.Slice {
				panic(fmt.Errorf("schema.FromGoReflect: shape.ListLike expected reflect.Slice, got %s", yreflect.Kind().String()))
			}

			// optimisation for []byte, otherwise it would iterate byte by byte!
			if shape.IsBinary(x) {
				return MkBinary(yreflect.Bytes())
			}

			result := List{}
			for i := 0; i < yreflect.Len(); i++ {
				result = append(result, FromGoReflect(x.Element, yreflect.Index(i)))
			}

			return &result
		},
		func(x *shape.MapLike) Schema {
			if yreflect.Kind() != reflect.Map {
				panic(fmt.Errorf("schema.FromGoReflect: shape.MapLike expected reflect.Map, got %s", yreflect.String()))
			}

			result := Map{}
			for _, key := range yreflect.MapKeys() {
				result[key.String()] = FromGoReflect(x.Val, yreflect.MapIndex(key))
			}

			return &result
		},
		func(x *shape.StructLike) Schema {
			if yreflect.Kind() == reflect.Ptr {
				yreflect = yreflect.Elem()
			}
			if yreflect.Kind() != reflect.Struct {
				panic(fmt.Errorf("schema.FromGoReflect: shape.StructLike expected reflect.Struct, got %s", yreflect.String()))
			}

			result := Map{}
			for _, field := range x.Fields {
				fieldReflect := yreflect.FieldByName(field.Name)
				if !fieldReflect.IsValid() {
					continue
				}

				result[field.Name] = FromGoReflect(field.Type, fieldReflect)
			}

			return &result
		},
		func(x *shape.UnionLike) Schema {
			if yreflect.IsNil() {
				return MkNone()
			}

			// find which variant is set
			refNameOriginal := shape.MkRefNameFromReflect(yreflect.Elem().Type())
			reflectedTypeName := shape.ToGoTypeName(
				shape.IndexWith(refNameOriginal, refNameOriginal),
				shape.WithPkgImportName(),
			)

			for _, variant := range x.Variant {
				s := shape.IndexWith(variant, refNameOriginal)
				variantName := shape.ToGoTypeName(
					s,
					shape.WithPkgImportName(),
					shape.WithInstantiation(),
				)
				if variantName == reflectedTypeName {
					if yreflect.Kind() == reflect.Interface {
						yreflect = yreflect.Elem()
					}
					if yreflect.Kind() == reflect.Ptr {
						yreflect = yreflect.Elem()
					}

					variantShort := shape.ToGoTypeName(s)
					return MkMap(
						MkField(
							"$type",
							MkString(variantShort),
						),
						MkField(
							variantShort,
							FromGoReflect(variant, yreflect),
						),
					)
				}
			}

			panic(fmt.Errorf("schema.FromGoReflect: shape.UnionLike %s not found",
				shape.ToGoFullTypeNameFromReflect(yreflect.Type())))
		},
	)
}

func ToGoReflect(xshape shape.Shape, ydata Schema, zreflect reflect.Type) (reflect.Value, error) {
	if IsNone(ydata) {
		return reflect.Zero(zreflect), nil
	}

	return shape.MatchShapeR2(
		xshape,
		func(x *shape.Any) (reflect.Value, error) {
			panic("not implemented")
		},
		func(x *shape.RefName) (reflect.Value, error) {
			newShape, found := shape.LookupShape(x)
			if !found {
				return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.RefName not found %#v; %w", x, shape.ErrShapeNotFound)
			}

			newShape = shape.IndexWith(newShape, x)

			return ToGoReflect(newShape, ydata, zreflect)
		},
		func(x *shape.PointerLike) (reflect.Value, error) {
			return ToGoReflect(x.Type, ydata, zreflect)
		},
		func(x *shape.AliasLike) (reflect.Value, error) {
			value, err := ToGoReflect(x.Type, ydata, zreflect)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.AliasLike %s; %w", x.Name, err)
			}

			if zreflect.Kind() == reflect.Ptr {
				if value.CanAddr() {
					return value.Addr(), nil
				} else {
					ptr := reflect.New(value.Type())
					ptr.Elem().Set(value)
					return ptr.Convert(zreflect), nil
				}
			}

			return value.Convert(zreflect), nil
		},
		func(x *shape.PrimitiveLike) (reflect.Value, error) {
			return shape.MatchPrimitiveKindR2(
				x.Kind,
				func(x *shape.BooleanLike) (reflect.Value, error) {
					data, ok := ydata.(*Bool)
					if !ok {
						return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.BooleanLike expected *Bool, got %T", ydata)
					}

					return reflect.ValueOf(bool(*data)), nil
				},
				func(x *shape.StringLike) (reflect.Value, error) {
					data, ok := ydata.(*String)
					if !ok {
						return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.StringLike expected *String, got %T", ydata)
					}
					return reflect.ValueOf(string(*data)), nil
				},
				func(x *shape.NumberLike) (reflect.Value, error) {
					data, ok := ydata.(*Number)
					if !ok {
						return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.NumberLike expected *Number, got %T", ydata)
					}

					if nil == x.Kind {
						return reflect.ValueOf(int(*data)).Convert(zreflect), nil
					}

					return shape.MatchNumberKindR2(
						x.Kind,
						func(x *shape.UInt) (reflect.Value, error) {
							return reflect.ValueOf(uint(*data)), nil
						},
						func(x *shape.UInt8) (reflect.Value, error) {
							return reflect.ValueOf(uint8(*data)), nil
						},
						func(x *shape.UInt16) (reflect.Value, error) {
							return reflect.ValueOf(uint16(*data)), nil
						},
						func(x *shape.UInt32) (reflect.Value, error) {
							return reflect.ValueOf(uint32(*data)), nil
						},
						func(x *shape.UInt64) (reflect.Value, error) {
							return reflect.ValueOf(uint64(*data)), nil
						},
						func(x *shape.Int) (reflect.Value, error) {
							return reflect.ValueOf(int(*data)), nil
						},
						func(x *shape.Int8) (reflect.Value, error) {
							return reflect.ValueOf(int8(*data)), nil
						},
						func(x *shape.Int16) (reflect.Value, error) {
							return reflect.ValueOf(int16(*data)), nil
						},
						func(x *shape.Int32) (reflect.Value, error) {
							return reflect.ValueOf(int32(*data)), nil
						},
						func(x *shape.Int64) (reflect.Value, error) {
							return reflect.ValueOf(int64(*data)), nil
						},
						func(x *shape.Float32) (reflect.Value, error) {
							return reflect.ValueOf(float32(*data)), nil
						},
						func(x *shape.Float64) (reflect.Value, error) {
							return reflect.ValueOf(float64(*data)), nil
						},
					)
				},
			)
		},

		func(x *shape.ListLike) (reflect.Value, error) {
			switch data := ydata.(type) {
			case *Binary:
				// optimisation for []byte, otherwise it would iterate byte by byte!
				if shape.IsBinary(x) {
					return reflect.ValueOf(*data), nil
				}

				return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.ListLike detected *schema.Binary but list shape is not like []uint8, got []%s", shape.ToGoTypeName(x.Element))

			case *List:
				if zreflect.Kind() != reflect.Slice {
					return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.ListLike expected reflect.Slice, got %s", zreflect.String())
				}

				if len(*data) == 0 {
					return reflect.Zero(zreflect), nil
				}

				result := reflect.MakeSlice(zreflect, len(*data), len(*data))
				for i, item := range *data {
					dest, err := ToGoReflect(x.Element, item, zreflect.Elem())
					if err != nil {
						return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.ListLike; %w", err)
					}

					result.Index(i).Set(dest)
				}

				return result, nil
			}

			return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.ListLike expected *List, got %T", ydata)
		},
		func(x *shape.MapLike) (reflect.Value, error) {
			data, ok := ydata.(*Map)
			if !ok {
				return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.MapLike expected *Map, got %T", ydata)
			}

			if zreflect.Kind() == reflect.Ptr {
				zreflect = zreflect.Elem()
			}

			if zreflect.Kind() != reflect.Map {
				return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.MapLike expected reflect.Map, got %s", zreflect.String())
			}

			if len(*data) == 0 {
				return reflect.Zero(zreflect), nil
			}

			result := reflect.MakeMap(zreflect)
			for key, value := range *data {
				dest, err := ToGoReflect(x.Val, value, zreflect.Elem())
				if err != nil {
					return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.MapLike; %w", err)
				}

				result.SetMapIndex(reflect.ValueOf(key), dest)
			}

			return result, nil
		},
		func(x *shape.StructLike) (reflect.Value, error) {
			data, ok := ydata.(*Map)
			if !ok {
				return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.StructLike expected *Map, got %T", ydata)
			}

			wasPointer := false
			if zreflect.Kind() == reflect.Ptr {
				wasPointer = true
				zreflect = zreflect.Elem()
			}

			if zreflect.Kind() != reflect.Struct {
				return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.StructLike expected reflect.Struct, got %d", zreflect.Kind())
			}

			result := reflect.New(zreflect).Elem()
			for _, field := range x.Fields {
				value, ok := (*data)[field.Name]
				if !ok {
					continue
				}

				fieldValue, ok := zreflect.FieldByName(field.Name)
				if !ok {
					return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: field %s not found", field.Name)
				}

				dest, err := ToGoReflect(field.Type, value, fieldValue.Type)
				if err != nil {
					return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: field %s; %w", field.Name, err)
				}

				result.FieldByName(field.Name).Set(dest)
			}

			if wasPointer {
				return result.Addr(), nil
			}

			return result, nil
		},
		func(x *shape.UnionLike) (reflect.Value, error) {
			data, ok := ydata.(*Map)
			if !ok {
				return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.UnionLike expected *Map, got %T", ydata)
			}

			if zreflect.Kind() != reflect.Interface {
				return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.UnionLike expected reflect.Interface, got %T", zreflect)
			}

			for _, variant := range x.Variant {
				variantName := shape.ToGoTypeName(variant)
				_, found := (*data)[variantName]
				if found {
					// zreflect is interface, so we need to find the actual type
					fullPkgName := shape.ToGoTypeName(variant, shape.WithPkgImportName())
					typ, found := shared.TypeRegistryLoad(fullPkgName)
					if !found {
						return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.UnionLike %s not found %s", x.Name, fullPkgName)
					}

					typR := reflect.TypeOf(typ)

					return ToGoReflect(variant, (*data)[variantName], typR)
				}
			}

			return reflect.Value{}, fmt.Errorf("schema.ToGoReflect: shape.UnionLike %s not found at all %#v", x.Name, data)
		},
	)
}
