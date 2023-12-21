package schema

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/shape"
	"reflect"
	"strconv"
	"strings"
)

func As[A int | int8 | int16 | int32 | int64 |
	uint | uint8 | uint16 | uint32 | uint64 |
	float32 | float64 |
	bool | string | []byte](x Schema) (A, bool) {
	var def A
	if x == nil {
		if any(def) == nil {
			return def, true
		}
		return def, false
	}

	return MustMatchSchemaR2(
		x,
		func(x *None) (A, bool) {
			if any(def) == nil {
				return def, true
			}
			return def, false
		},
		func(x *Bool) (A, bool) {
			switch any(def).(type) {
			case bool:
				return any(bool(*x)).(A), true
			}

			return def, false
		},
		func(x *Number) (A, bool) {
			switch any(def).(type) {
			case float32:
				return any(float32(*x)).(A), true
			case float64:
				return any(float64(*x)).(A), true
			case int:
				return any(int(*x)).(A), true
			case int8:
				return any(int8(*x)).(A), true
			case int16:
				return any(int16(*x)).(A), true
			case int32:
				return any(int32(*x)).(A), true
			case int64:
				return any(int64(*x)).(A), true
			case uint:
				return any(uint(*x)).(A), true
			case uint8:
				return any(uint8(*x)).(A), true
			case uint16:
				return any(uint16(*x)).(A), true
			case uint32:
				return any(uint32(*x)).(A), true
			case uint64:
				return any(uint64(*x)).(A), true
			}
			return def, false
		},
		func(x *String) (A, bool) {
			switch any(def).(type) {
			case string:
				return any(string(*x)).(A), true
			case []byte:
				return any([]byte((*x))).(A), true
			case float64:
				v, err := strconv.ParseFloat(string(*x), 64)
				if err != nil {
					return def, false
				}
				return any(v).(A), true
			case float32:
				v, err := strconv.ParseFloat(string(*x), 32)
				if err != nil {
					return def, false
				}
				return any(float32(v)).(A), true
			case int:
				v, err := strconv.Atoi(string(*x))
				if err != nil {
					return def, false
				}
				return any(v).(A), true
			case int8:
				v, err := strconv.ParseInt(string(*x), 10, 8)
				if err != nil {
					return def, false
				}
				return any(int8(v)).(A), true
			case int16:
				v, err := strconv.ParseInt(string(*x), 10, 16)
				if err != nil {
					return def, false
				}
				return any(int16(v)).(A), true
			case int32:
				v, err := strconv.ParseInt(string(*x), 10, 32)
				if err != nil {
					return def, false
				}
				return any(int32(v)).(A), true
			case int64:
				v, err := strconv.ParseInt(string(*x), 10, 64)
				if err != nil {
					return def, false
				}
				return any(v).(A), true
			}

			return def, false
		},
		func(x *Binary) (A, bool) {
			switch any(def).(type) {
			case []byte:
				return any([]byte(*x)).(A), true
			case string:
				return any(string(*x)).(A), true
			}

			return def, false
		},
		func(x *List) (A, bool) {
			return def, false
		},
		func(x *Map) (A, bool) {
			return def, false
		})
}

func AsDefault[A int | int8 | int16 | int32 | int64 |
	uint | uint8 | uint16 | uint32 | uint64 |
	float32 | float64 |
	bool | string | []byte](x Schema, def A) A {

	res, ok := As[A](x)
	if ok {
		return res
	}

	return def
}

func GetSchema(data Schema, location string) Schema {
	path, err := ParseLocation(location)
	if err != nil {
		log.Warnf("schema.GetSchema: failed to parse location: %s", err)
		return nil
	}

	return GetSchemaLocation(data, path)
}

func GetSchemaLocation(data Schema, locations []Location) Schema {
	for {
		if len(locations) == 0 {
			return data
		}

		location := locations[0]
		locations = locations[1:]

		data, locations = MustMatchLocationR2(
			location,
			func(x *LocationField) (Schema, []Location) {
				mapData, ok := data.(*Map)
				if !ok {
					return nil, locations
				}

				if value, ok := (*mapData)[x.Name]; ok {
					return value, locations
				}

				return nil, locations
			},
			func(x *LocationIndex) (Schema, []Location) {
				listData, ok := data.(*List)
				if ok && len(*listData) > x.Index {
					return (*listData)[x.Index], locations
				}

				return nil, locations
			},
			func(x *LocationAnything) (Schema, []Location) {
				switch y := data.(type) {
				case *List:
					for _, item := range *y {
						newData := GetSchemaLocation(item, locations)
						if newData != nil {
							return newData, nil
						}
					}

					return nil, locations

				case *Map:
					for _, value := range *y {
						newData := GetSchemaLocation(value, locations)
						if newData != nil {
							return newData, nil
						}
					}
				}

				return nil, locations
			},
		)

		if data == nil {
			return data
		}
	}
}

func Get[A any](data A, location string) (Schema, shape.Shape) {
	v := reflect.TypeOf(new(A)).Elem()
	original := shape.MkRefNameFromReflect(v)

	s, found := shape.LookupShape(original)
	if !found {
		panic(fmt.Errorf("schema.GetLocation: shape.RefName not found %s", v.String()))
	}

	s = shape.IndexWith(s, original.Indexed)

	sdata := FromGo[A](data)

	return GetShapeLocation(s, sdata, location)
}

func GetShapeLocation(s shape.Shape, data Schema, location string) (Schema, shape.Shape) {
	l, err := ParseLocation(location)
	if err != nil {
		panic(fmt.Errorf("schema.GetLocation: failed to parse location: %s", err))
	}

	return GetShapeSchemaLocation(s, data, l)
}

type locres struct {
	data  Schema
	loc   []Location
	shape shape.Shape
}

func GetShapeSchemaLocation(s shape.Shape, data Schema, locations []Location) (Schema, shape.Shape) {
	for {
		if len(locations) == 0 {
			return data, s
		}

		location := locations[0]
		locations = locations[1:]

		res := MustMatchLocation(
			location,
			func(x *LocationField) *locres {
				switch y := s.(type) {
				case *shape.StructLike:
					mapData, ok := data.(*Map)
					if !ok {
						return nil
					}

					for _, field := range y.Fields {
						if field.Name == x.Name {
							fieldValue, ok := (*mapData)[x.Name]
							if !ok {
								return nil
							}

							return &locres{
								data:  fieldValue,
								loc:   locations,
								shape: field.Type,
							}
						}
					}
				case *shape.AliasLike:
					res, sch := GetShapeSchemaLocation(y.Type, data, append([]Location{x}, locations...))
					if res != nil {
						return &locres{
							data:  res,
							loc:   locations,
							shape: sch,
						}
					}

					return nil

				case *shape.MapLike:
					mapData, ok := data.(*Map)
					if !ok {
						return nil
					}

					value, ok := (*mapData)[x.Name]
					if !ok {
						return nil
					}

					return &locres{
						data:  value,
						loc:   locations,
						shape: y.Val,
					}

				case *shape.UnionLike:
					mapData, ok := data.(*Map)
					if !ok {
						return nil
					}

					_, ok = (*mapData)[x.Name]
					if !ok {
						return nil
					}

					for _, variant := range y.Variant {
						fieldName := shape.ToGoTypeName(variant)
						if x.Name != fieldName {
							continue
						}

						fieldValue, ok := (*mapData)[fieldName]
						if !ok {
							continue
						}

						res, sch := GetShapeSchemaLocation(variant, fieldValue, locations)
						if res != nil {
							return &locres{
								data:  res,
								loc:   locations,
								shape: sch,
							}
						}
					}

					return nil

				case *shape.RefName:
					ss, found := shape.LookupShape(y)
					if !found {
						return nil
					}

					res, sch := GetShapeSchemaLocation(ss, data, append([]Location{x}, locations...))
					if res != nil {
						return &locres{
							data:  res,
							loc:   locations,
							shape: sch,
						}
					}

				case *shape.NumberLike:
					numData, ok := data.(*Number)
					if !ok {
						return nil
					}

					return &locres{
						data:  numData,
						loc:   locations,
						shape: s,
					}

				case *shape.StringLike:
					strData, ok := data.(*String)
					if !ok {
						return nil
					}

					return &locres{
						data:  strData,
						loc:   locations,
						shape: s,
					}

				default:
					panic(fmt.Errorf("schema.GetShapeSchemaLocation: unknown field access %s with shape %#v", x.Name, s))
				}

				return nil
			},
			func(x *LocationIndex) *locres {
				switch y := s.(type) {
				case *shape.ListLike:
					listData, ok := data.(*List)
					if ok && len(*listData) > x.Index {
						return &locres{
							data:  (*listData)[x.Index],
							loc:   locations,
							shape: y.Element,
						}
					}
				}

				return nil
			},
			func(x *LocationAnything) *locres {
				switch y := s.(type) {
				case *shape.StringLike:
					strData, ok := data.(*String)
					if !ok {
						return nil
					}

					return &locres{
						data:  strData,
						shape: s,
						loc:   locations,
					}

				case *shape.NumberLike:
					numData, ok := data.(*Number)
					if !ok {
						return nil
					}

					return &locres{
						data:  numData,
						shape: s,
						loc:   locations,
					}

				case *shape.MapLike:
					mapData, ok := data.(*Map)
					if !ok {
						return nil
					}

					for _, value := range *mapData {
						res, sch := GetShapeSchemaLocation(y.Val, value, locations)
						if res != nil {
							return &locres{
								data:  res,
								loc:   locations,
								shape: sch,
							}
						}
					}

					return nil

				case *shape.UnionLike:
					mapData, ok := data.(*Map)
					if !ok {
						return nil
					}

					for _, variant := range y.Variant {
						fieldName := shape.ToGoTypeName(variant)
						fieldValue, ok := (*mapData)[fieldName]
						if !ok {
							continue
						}

						res, sch := GetShapeSchemaLocation(variant, fieldValue, locations)
						if res != nil {
							return &locres{
								data:  res,
								loc:   locations,
								shape: sch,
							}
						}
					}

				case *shape.ListLike:
					listData, ok := data.(*List)
					if !ok {
						return nil
					}

					for _, item := range *listData {
						res, sch := GetShapeSchemaLocation(y.Element, item, locations)
						if res != nil {
							return &locres{
								data:  res,
								loc:   locations,
								shape: sch,
							}
						}
					}

				case *shape.RefName:
					ss, found := shape.LookupShape(y)
					if !found {
						return nil
					}

					res, sch := GetShapeSchemaLocation(ss, data, append([]Location{x}, locations...))
					if res != nil {
						return &locres{
							data:  res,
							loc:   locations,
							shape: sch,
						}
					}

				case *shape.AliasLike:
					res, sch := GetShapeSchemaLocation(y.Type, data, locations)
					if res != nil {
						return &locres{
							data:  res,
							loc:   locations,
							shape: sch,
						}
					}

					return nil
				}

				panic(fmt.Errorf("schema.GetShapeSchemaLocation: unknown anything access %#v with shape %#v", x, s))
			},
		)

		if res == nil {
			return nil, nil
		}

		data = res.data
		s = res.shape
		locations = res.loc

		if data == nil {
			return data, s
		}
	}
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
		func(x *Binary) A {
			return fn(x, init)
		},
		func(x *List) A {
			for _, y := range *x {
				init = fn(y, init)
			}

			return init
		},
		func(x *Map) A {
			for _, value := range *x {
				init = fn(value, init)
			}

			return init
		},
	)
}

func Compare(a, b Schema) int {
	if a == nil {
		a = none
	}
	if b == nil {
		b = none
	}

	return MustMatchSchema(
		a,
		func(x *None) int {
			switch b.(type) {
			case *None:
				return 0
			}

			return -1
		},
		func(x *Bool) int {
			switch y := b.(type) {
			case *None:
				return 1
			case *Bool:
				if *x == *y {
					return 0
				}
				if *x {
					return 1
				}
				return -1
			}

			return -1
		},
		func(x *Number) int {
			switch y := b.(type) {
			case *None, *Bool:
				return 1
			case *Number:
				return int(*x - *y)
			}

			return -1
		},
		func(x *String) int {
			switch y := b.(type) {
			case *None, *Bool, *Number:
				return 1
			case *String:
				return strings.Compare(string(*x), string(*y))
			}

			return -1
		},
		func(x *Binary) int {
			switch y := b.(type) {
			case *None, *Bool, *Number, *String:
				return 1
			case *Binary:
				return bytes.Compare(*x, *y)
			}

			return -1
		},
		func(x *List) int {
			switch y := b.(type) {
			case *None, *Bool, *Number, *String, *Binary:
				return 1
			case *List:
				if len(*x) == len(*y) {
					for i := range *x {
						cmp := Compare((*x)[i], (*y)[i])
						if cmp != 0 {
							return cmp
						}
					}
					return 0
				}
				if len(*x) > len(*y) {
					return 1
				}

				return -1
			}

			return -1

		},
		func(x *Map) int {
			switch y := b.(type) {
			case *None, *Bool, *Number, *String, *Binary, *List:
				return 1
			case *Map:
				if len(*x) == len(*y) {
					for xName, xField := range *x {
						var found bool
						for yName, yField := range *y {
							if xName == yName {
								found = true
								cmp := Compare(xField, yField)
								if cmp != 0 {
									return cmp
								}
								break
							}
						}
						if !found {
							return -1
						}
					}
					return 0
				}

				if len(*x) > len(*y) {
					return 1
				}

				return -1
			}

			return -1
		},
	)
}
