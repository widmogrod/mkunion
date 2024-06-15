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

type TypedLocation struct {
	shape shape.Shape
}

func (location *TypedLocation) ShapeDef() shape.Shape {
	return location.shape
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

func (location *TypedLocation) WrapLocation(loc []Location) ([]Location, error) {
	loc = location.wrapLocationShapeAware(loc, location.shape)
	return loc, nil
}

func (location *TypedLocation) wrapLocationShapeAware(loc []Location, s shape.Shape) []Location {
	if len(loc) == 0 {
		return loc
	}

	return MatchLocationR1(
		loc[0],
		func(x *LocationField) []Location {
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

					return location.wrapLocationShapeAware(loc, s)
				},
				func(x *shape.PointerLike) []Location {
					return location.wrapLocationShapeAware(loc, x.Type)
				},
				func(y *shape.AliasLike) []Location {
					panic("not implemented")
				},
				func(x *shape.PrimitiveLike) []Location {
					panic("not implemented")
				},
				func(y *shape.ListLike) []Location {
					panic("not implemented")
				},
				func(y *shape.MapLike) []Location {
					panic("not implemented")
				},
				func(y *shape.StructLike) []Location {
					for _, field := range y.Fields {
						if field.Name == x.Name {
							result := location.wrapLocationShapeAware(loc[1:], field.Type)
							return append(
								append([]Location{x}, location.shapeToSchemaName(field.Type)...),
								result...,
							)
						}
					}

					panic(fmt.Errorf("wrapLocationShapeAware: field %s not found in struct %s", x.Name, y.Name))
				},
				func(y *shape.UnionLike) []Location {
					if x.Name == "$type" {
						return append([]Location{x}, location.shapeToSchemaName(&shape.PrimitiveLike{
							Kind: &shape.StringLike{},
						})...)
					}

					for _, variant := range y.Variant {
						if shape.ToGoTypeName(variant) == x.Name {
							result := location.wrapLocationShapeAware(loc[1:], variant)
							return append(
								append([]Location{x}, location.shapeToSchemaName(variant)...),
								result...,
							)
						}
					}

					panic("not implemented")
				},
			)
		},
		func(x *LocationIndex) []Location {
			panic("not implemented")
		},
		func(x *LocationAnything) []Location {
			panic("not implemented")
		},
	)
}

func (location *TypedLocation) shapeToSchemaName(x shape.Shape) []Location {
	return shape.MatchShapeR1(
		x,
		func(x *shape.Any) []Location {
			panic("not implemented")
		},
		func(x *shape.RefName) []Location {
			s, found := shape.LookupShape(x)
			if !found {
				panic(fmt.Errorf("shapeToSchemaName: shape.RefName not found %s; %w", x.Name, shape.ErrShapeNotFound))
			}

			s = shape.IndexWith(s, x)

			return location.shapeToSchemaName(s)
		},
		func(x *shape.PointerLike) []Location {
			return location.shapeToSchemaName(x.Type)
		},
		func(x *shape.AliasLike) []Location {
			panic("not implemented")
		},
		func(x *shape.PrimitiveLike) []Location {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) []Location {
					return []Location{
						&LocationField{
							Name: "schema.Boolean",
						},
					}
				},
				func(x *shape.StringLike) []Location {
					return []Location{
						&LocationField{
							Name: "schema.String",
						},
					}
				},
				func(x *shape.NumberLike) []Location {
					return []Location{
						&LocationField{
							Name: "schema.Number",
						},
					}
				},
			)
		},
		func(x *shape.ListLike) []Location {
			panic("not implemented")
		},
		func(x *shape.MapLike) []Location {
			panic("not implemented")
		},
		func(x *shape.StructLike) []Location {
			return []Location{
				&LocationField{
					Name: "schema.Map",
				},
			}
		},
		func(x *shape.UnionLike) []Location {
			return []Location{
				&LocationField{
					Name: "schema.Map",
				},
			}
		},
	)
}
