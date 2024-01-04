package typedful

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
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

func (location *TypedLocation) WrapLocationStr(field string) (string, error) {
	loc, err := schema.ParseLocation(field)
	if err != nil {
		return "", fmt.Errorf("typedful.WrapLocationStr: %w", err)
	}

	loc, err = location.WrapLocation(loc)
	if err != nil {
		return "", fmt.Errorf("typedful.WrapLocationStr: %w", err)
	}

	return schema.LocationToStr(loc), nil
}

func (location *TypedLocation) WrapLocation(loc []schema.Location) ([]schema.Location, error) {
	loc = location.wrapLocationShapeAware(loc, location.shape)
	return loc, nil
}

func (location *TypedLocation) wrapLocationShapeAware(loc []schema.Location, s shape.Shape) []schema.Location {
	if len(loc) == 0 {
		return loc
	}

	return schema.MatchLocationR1(
		loc[0],
		func(x *schema.LocationField) []schema.Location {
			return shape.MatchShapeR1(
				s,
				func(y *shape.Any) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.RefName) []schema.Location {
					s, ok := shape.LookupShape(y)
					if !ok {
						panic(fmt.Errorf("wrapLocationShapeAware: shape.RefName not found %s; %w", y.Name, shape.ErrShapeNotFound))
					}

					return location.wrapLocationShapeAware(loc, s)
				},
				func(x *shape.PointerLike) []schema.Location {
					return location.wrapLocationShapeAware(loc, x.Type)
				},
				func(y *shape.AliasLike) []schema.Location {
					panic("not implemented")
				},
				func(x *shape.PrimitiveLike) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.ListLike) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.MapLike) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.StructLike) []schema.Location {
					for _, field := range y.Fields {
						if field.Name == x.Name {
							result := location.wrapLocationShapeAware(loc[1:], field.Type)
							return append(
								append([]schema.Location{x}, location.shapeToSchemaName(field.Type)...),
								result...,
							)
						}
					}

					panic(fmt.Errorf("wrapLocationShapeAware: field %s not found in struct %s", x.Name, y.Name))
				},
				func(y *shape.UnionLike) []schema.Location {
					for _, variant := range y.Variant {
						if shape.ToGoTypeName(variant) == x.Name {
							result := location.wrapLocationShapeAware(loc[1:], variant)
							return append(
								append([]schema.Location{x}, location.shapeToSchemaName(variant)...),
								result...,
							)
						}
					}

					panic("not implemented")
				},
			)
		},
		func(x *schema.LocationIndex) []schema.Location {
			panic("not implemented")
		},
		func(x *schema.LocationAnything) []schema.Location {
			panic("not implemented")
		},
	)
}

func (location *TypedLocation) shapeToSchemaName(x shape.Shape) []schema.Location {
	return shape.MatchShapeR1(
		x,
		func(x *shape.Any) []schema.Location {
			panic("not implemented")
		},
		func(x *shape.RefName) []schema.Location {
			s, found := shape.LookupShape(x)
			if !found {
				panic(fmt.Errorf("shapeToSchemaName: shape.RefName not found %s; %w", x.Name, shape.ErrShapeNotFound))
			}
			return location.shapeToSchemaName(s)
		},
		func(x *shape.PointerLike) []schema.Location {
			return location.shapeToSchemaName(x.Type)
		},
		func(x *shape.AliasLike) []schema.Location {
			panic("not implemented")
		},
		func(x *shape.PrimitiveLike) []schema.Location {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) []schema.Location {
					return []schema.Location{
						&schema.LocationField{
							Name: "schema.Boolean",
						},
					}
				},
				func(x *shape.StringLike) []schema.Location {
					return []schema.Location{
						&schema.LocationField{
							Name: "schema.String",
						},
					}
				},
				func(x *shape.NumberLike) []schema.Location {
					return []schema.Location{
						&schema.LocationField{
							Name: "schema.Number",
						},
					}
				},
			)
		},
		func(x *shape.ListLike) []schema.Location {
			panic("not implemented")
		},
		func(x *shape.MapLike) []schema.Location {
			panic("not implemented")
		},
		func(x *shape.StructLike) []schema.Location {
			return []schema.Location{
				&schema.LocationField{
					Name: "schema.Map",
				},
			}
		},
		func(x *shape.UnionLike) []schema.Location {
			return []schema.Location{
				&schema.LocationField{
					Name: "schema.Map",
				},
			}
		},
	)
}
