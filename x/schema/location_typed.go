package schema

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
)

func NewTypedLocation[T any]() (*TypedLocation, error) {
	s, found := shape.LookupShapeReflectAndIndex[T]()
	if !found {
		return nil, fmt.Errorf("typedful.NewTypedLocation: shape not found %T; %w", *new(T), shape.ErrShapeNotFound)
	}

	return &TypedLocation{
		shape: s,
	}, nil
}
func NewTypedLocationWithEncoded[T any](encodedAs shape.Shape) (*TypedLocation, error) {
	s, found := shape.LookupShapeReflectAndIndex[T]()
	if !found {
		return nil, fmt.Errorf("typedful.NewTypedLocation: shape not found %T; %w", *new(T), shape.ErrShapeNotFound)
	}

	return &TypedLocation{
		shape:     s,
		encodedAs: encodedAs,
	}, nil
}

type TypedLocation struct {
	shape     shape.Shape
	encodedAs shape.Shape
}

func (location *TypedLocation) ShapeDef() shape.Shape {
	return location.shape
}

func (location *TypedLocation) EncodedAs() shape.Shape {
	return location.encodedAs
}

func (location *TypedLocation) WrapLocationStr(field string) (string, error) {
	loc, err := ParseLocation(field)
	if err != nil {
		return "", fmt.Errorf("typedful.WrapLocationStr: %w", err)
	}

	loc, err = location.WrapLocation(loc)
	if err != nil {
		return "", fmt.Errorf("typedful.WrapLocationStr: %w", err)
	}

	return LocationToStr(loc), nil
}

func (location *TypedLocation) WithEncodedAs(encodedAs shape.Shape) *TypedLocation {
	return &TypedLocation{
		shape:     location.shape,
		encodedAs: encodedAs,
	}
}

func (location *TypedLocation) WrapLocation(loc []Location) ([]Location, error) {
	if location.encodedAs == nil {
		loc = location.wrapLocationShapeAware(loc, location.shape, false)
	} else {
		var err error
		loc, err = location.WrapLocationEncodedAs(loc, location.shape, location.encodedAs, false)
		if err != nil {
			return nil, fmt.Errorf("typedful.WrapLocation: %w", err)
		}
	}

	return loc, nil
}

//go:tag mkmatch:"MatchDifference"
type MatchDifference[A, B shape.Shape] interface {
	StructLikes(x *shape.StructLike, y *shape.StructLike)
	UnionLikes(x *shape.UnionLike, y *shape.UnionLike)
	RightRef(x shape.Shape, y *shape.RefName)
	LeftRef(x *shape.RefName, y shape.Shape)
	Finally(x, y shape.Shape)
}

func (location *TypedLocation) WrapLocationEncodedAs(loc []Location, s0, s1 shape.Shape, wrap bool) ([]Location, error) {
	if len(loc) == 0 {
		return nil, nil
	}

	return MatchLocationR2(
		loc[0],
		func(l0 *LocationField) ([]Location, error) {
			return MatchDifferenceR2(
				s0, s1,
				func(x0 *shape.StructLike, x1 *shape.StructLike) ([]Location, error) {
					sameName := x0.Name == x1.Name
					samePackage := x0.PkgImportName == x1.PkgImportName

					if !(sameName && samePackage) {
						panic("not the same struct")
					}

					for i, field := range x0.Fields {
						if field.Name == l0.Name {
							result, err := location.WrapLocationEncodedAs(loc[1:], x0.Fields[i].Type, x1.Fields[i].Type, true)
							if err != nil {
								return nil, fmt.Errorf("typedful.WrapLocationEncodedAs: %w", err)
							}

							return append(
								append([]Location{l0}),
								result...,
							), nil
						}
					}

					panic("field not found")
				},
				func(x0 *shape.UnionLike, x1 *shape.UnionLike) ([]Location, error) {
					sameName := x0.Name == x1.Name
					samePackage := x0.PkgImportName == x1.PkgImportName

					if !(sameName && samePackage) {
						if IsShapeASchema(x1) {
							return append(
								location.wrapCond([]Location{}, wrap, x0),
								location.wrapLocationShapeAware(loc, x0, wrap)...,
							), nil
						}

						panic("not the same union")
					}

					if l0.Name == "$type" {
						panic("not implemented")
					}

					for i, variant := range x0.Variant {
						if shape.ToGoTypeName(variant) == l0.Name {
							result, err := location.WrapLocationEncodedAs(loc[1:], variant, x1.Variant[i], wrap)
							if err != nil {
								return nil, fmt.Errorf("typedful.WrapLocationEncodedAs: %w", err)
							}

							return append(
								append([]Location{l0}),
								result...,
							), nil
						}
					}

					panic("not implemented")
				},
				func(x0 shape.Shape, x1 *shape.RefName) ([]Location, error) {
					sch1, found := shape.LookupShape(x1)
					if !found {
						panic(fmt.Errorf("typedful.WrapLocationEncodedAs: shape.RefName not found %s; %w", x1.Name, shape.ErrShapeNotFound))
					}

					sch1 = shape.IndexWith(sch1, x1)
					return location.WrapLocationEncodedAs(loc, x0, sch1, wrap)
				},
				func(x0 *shape.RefName, x1 shape.Shape) ([]Location, error) {
					sch0, found := shape.LookupShape(x0)
					if !found {
						panic(fmt.Errorf("typedful.WrapLocationEncodedAs: shape.RefName not found %s; %w", x0.Name, shape.ErrShapeNotFound))
					}

					sch0 = shape.IndexWith(sch0, x0)
					return location.WrapLocationEncodedAs(loc, sch0, x1, wrap)
				},
				func(x0 shape.Shape, x1 shape.Shape) ([]Location, error) {
					if IsShapeASchema(x1) {
						return append(
							location.wrapCond([]Location{}, wrap, x0),
							location.wrapLocationShapeAware(loc, x0, wrap)...,
						), nil
					}

					panic("not implemented")
				},
			)
		},
		func(x *LocationIndex) ([]Location, error) {
			panic("not implemented")
		},
		func(x *LocationAnything) ([]Location, error) {
			panic("not implemented")
		},
	)
}

func (location *TypedLocation) wrapLocationShapeAware(loc []Location, s shape.Shape, wrap bool) []Location {
	if len(loc) == 0 {
		return loc
	}

	return MatchLocationR1(
		loc[0],
		func(x *LocationField) []Location {
			return shape.MatchShapeR1(
				s,
				func(y *shape.Any) []Location {
					return location.wrapCond([]Location{x}, wrap, y)
				},
				func(y *shape.RefName) []Location {
					s, ok := shape.LookupShape(y)
					if !ok {
						panic(fmt.Errorf("wrapLocationShapeAware: shape.RefName not found %s; %w", y.Name, shape.ErrShapeNotFound))
					}

					s = shape.IndexWith(s, y)

					return location.wrapLocationShapeAware(loc, s, wrap)
				},
				func(y *shape.PointerLike) []Location {
					panic("not implemented")
				},
				func(y *shape.AliasLike) []Location {
					panic("not implemented")
				},
				func(y *shape.PrimitiveLike) []Location {
					panic("not implemented")
					//return shape.MatchPrimitiveKindR1(
					//	y.Kind,
					//	func(z *shape.BooleanLike) []Location {
					//		return append([]Location{
					//			&LocationField{Name: "schema.Boolean"},
					//		})
					//	},
					//	func(z *shape.StringLike) []Location {
					//		return append([]Location{
					//			&LocationField{Name: "schema.String"},
					//		})
					//	},
					//	func(z *shape.NumberLike) []Location {
					//		return append([]Location{
					//			&LocationField{Name: "schema.Number"},
					//		})
					//	},
					//)
				},
				func(y *shape.ListLike) []Location {
					panic("not implemented")
				},
				func(y *shape.MapLike) []Location {
					return append(
						location.wrapCond([]Location{x}, wrap, y.Val),
						location.wrapLocationShapeAware(loc[1:], y.Val, wrap)...,
					)
				},
				func(y *shape.StructLike) []Location {
					for _, field := range y.Fields {
						if field.Name == x.Name {
							result := location.wrapLocationShapeAware(loc[1:], field.Type, wrap)
							return append(
								location.wrapCond([]Location{x}, wrap, field.Type),
								result...,
							)
						}
					}

					panic(fmt.Errorf("wrapLocationShapeAware: field %s not found in struct %s", x.Name, y.Name))
				},
				func(y *shape.UnionLike) []Location {
					if IsShapeASchema(y) {
						panic("not implemented")
					}
					if x.Name == "$type" {
						return location.wrapCond([]Location{x}, wrap, &shape.PrimitiveLike{
							Kind: &shape.StringLike{},
						})
					}

					for _, variant := range y.Variant {
						if shape.ToGoTypeName(variant) == x.Name {
							result := location.wrapLocationShapeAware(loc[1:], variant, wrap)
							return append(
								location.wrapCond([]Location{x}, wrap, variant),
								result...,
							)
						}
					}

					panic("not implemented")
				},
			)
		},
		func(x *LocationIndex) []Location {
			return shape.MatchShapeR1(
				s,
				func(y *shape.Any) []Location {
					panic("not implemented")
				},
				func(y *shape.RefName) []Location {
					s, ok := shape.LookupShape(y)
					if !ok {
						panic(fmt.Errorf("wrapLocationShapeAware: shape.RefName not found %s; %w", y.Name, shape.ErrShapeNotFound))
					}

					s = shape.IndexWith(s, y)

					return location.wrapLocationShapeAware(loc, s, wrap)
				},
				func(y *shape.PointerLike) []Location {
					return location.wrapLocationShapeAware(loc, y.Type, wrap)
				},
				func(y *shape.AliasLike) []Location {
					panic("not implemented")
				},
				func(y *shape.PrimitiveLike) []Location {
					panic("not implemented")
				},
				func(y *shape.ListLike) []Location {
					return append(
						location.wrapCond([]Location{x}, wrap, y.Element),
						location.wrapLocationShapeAware(loc[1:], y.Element, wrap)...,
					)
				},
				func(y *shape.MapLike) []Location {
					panic("not implemented")
				},
				func(y *shape.StructLike) []Location {
					panic("wrapLocationShapeAware: index not supported in struct")
				},
				func(y *shape.UnionLike) []Location {
					panic("wrapLocationShapeAware: index not supported in union")
				},
			)
		},
		func(x *LocationAnything) []Location {
			panic("not implemented")
		},
	)
}

func (location *TypedLocation) wrapCond(result []Location, wrap bool, s shape.Shape) []Location {
	if wrap {
		return shape.MatchShapeR1(
			s,
			func(x *shape.Any) []Location {
				return result
			},
			func(x *shape.RefName) []Location {
				s, ok := shape.LookupShape(x)
				if !ok {
					panic(fmt.Errorf("wrapLocationShapeAware: shape.RefName not found %s; %w", x.Name, shape.ErrShapeNotFound))
				}

				s = shape.IndexWith(s, x)
				return location.wrapCond(result, wrap, s)
			},
			func(x *shape.PointerLike) []Location {
				return location.wrapCond(result, wrap, x.Type)
			},
			func(x *shape.AliasLike) []Location {
				return location.wrapCond(result, wrap, x.Type)
			},
			func(x *shape.PrimitiveLike) []Location {
				return shape.MatchPrimitiveKindR1(
					x.Kind,
					func(x *shape.BooleanLike) []Location {
						return append(
							result,
							&LocationField{
								Name: "schema.Bool",
							},
						)
					},
					func(x *shape.StringLike) []Location {
						return append(
							result,
							&LocationField{
								Name: "schema.String",
							},
						)

					},
					func(x *shape.NumberLike) []Location {
						return append(
							result,
							&LocationField{
								Name: "schema.Number",
							},
						)
					},
				)
			},
			func(x *shape.ListLike) []Location {
				return append(
					result,
					&LocationField{
						Name: "schema.List",
					},
				)
			},
			func(x *shape.MapLike) []Location {
				return append(
					result,
					&LocationField{
						Name: "schema.Map",
					},
				)
			},
			func(x *shape.StructLike) []Location {
				return append(
					result,
					&LocationField{
						Name: "schema.Map",
					},
				)
			},
			func(x *shape.UnionLike) []Location {
				return append(
					result,
					&LocationField{
						Name: "schema.Map",
					},
				)
			},
		)
	}

	return result
}
