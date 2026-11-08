package breaking

import (
	"github.com/widmogrod/mkunion/x/shape"
)

// typeEqual reports whether two shapes read and write the same JSON.
// Renamed types count as equal; widthOnly is set when the only difference
// is an integer or float width, which is the same JSON number.
func (c *comparer) typeEqual(a, b shape.Shape) (equal bool, widthOnly bool) {
	switch x := a.(type) {
	case *shape.Any:
		_, ok := b.(*shape.Any)
		return ok, false
	case *shape.PrimitiveLike:
		y, ok := b.(*shape.PrimitiveLike)
		if !ok {
			return false, false
		}
		return primitiveEqual(x, y)
	case *shape.PointerLike:
		y, ok := b.(*shape.PointerLike)
		if !ok {
			return false, false
		}
		return c.typeEqual(x.Type, y.Type)
	case *shape.ListLike:
		y, ok := b.(*shape.ListLike)
		if !ok {
			return false, false
		}
		return c.typeEqual(x.Element, y.Element)
	case *shape.MapLike:
		y, ok := b.(*shape.MapLike)
		if !ok {
			return false, false
		}
		return c.bothEqual(x.Key, y.Key, x.Val, y.Val)
	case *shape.RefName:
		y, ok := b.(*shape.RefName)
		if !ok || !c.sameName(x.PkgName, x.Name, y.PkgName, y.Name) || len(x.Indexed) != len(y.Indexed) {
			return false, false
		}
		for i := range x.Indexed {
			if eq, _ := c.typeEqual(x.Indexed[i], y.Indexed[i]); !eq {
				return false, false
			}
		}
		return true, false
	case *shape.StructLike:
		y, ok := b.(*shape.StructLike)
		if !ok {
			return false, false
		}
		if x.Name != "" || y.Name != "" {
			return c.sameName(x.PkgName, x.Name, y.PkgName, y.Name), false
		}
		return c.fieldsEqual(x.Fields, y.Fields), false
	case *shape.UnionLike:
		y, ok := b.(*shape.UnionLike)
		return ok && c.sameName(x.PkgName, x.Name, y.PkgName, y.Name), false
	case *shape.AliasLike:
		y, ok := b.(*shape.AliasLike)
		return ok && c.sameName(x.PkgName, x.Name, y.PkgName, y.Name), false
	}
	return false, false
}

func (c *comparer) bothEqual(a1, b1, a2, b2 shape.Shape) (bool, bool) {
	if eq, _ := c.typeEqual(a1, b1); !eq {
		return false, false
	}
	if eq, _ := c.typeEqual(a2, b2); !eq {
		return false, false
	}
	return true, false
}

// fieldsEqual compares inline structs: same wire names and types in order.
func (c *comparer) fieldsEqual(a, b []*shape.FieldLike) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if jsonName(a[i]) != jsonName(b[i]) || a[i].Name != b[i].Name {
			return false
		}
		if eq, _ := c.typeEqual(a[i].Type, b[i].Type); !eq {
			return false
		}
	}
	return true
}

// sameName matches a type of the old snapshot with one of the new, through
// the package rename and the type renames found so far.
func (c *comparer) sameName(oldPkg, oldName, newPkg, newName string) bool {
	if oldPkg == c.old.pkg && newPkg == c.new.pkg {
		oldPkg = newPkg
	}
	if oldPkg != newPkg {
		return false
	}
	return c.renamed(oldName) == newName
}

func primitiveEqual(a, b *shape.PrimitiveLike) (equal bool, widthOnly bool) {
	an, aIsNumber := a.Kind.(*shape.NumberLike)
	bn, bIsNumber := b.Kind.(*shape.NumberLike)
	if !aIsNumber || !bIsNumber {
		return shape.ToGoTypeName(a) == shape.ToGoTypeName(b), false
	}
	if shape.NumberKindToGoName(an.Kind) == shape.NumberKindToGoName(bn.Kind) {
		return true, false
	}
	return false, isFloat(an.Kind) == isFloat(bn.Kind)
}

func isFloat(k shape.NumberKind) bool {
	switch k.(type) {
	case *shape.Float32, *shape.Float64:
		return true
	}
	return false
}

// describe is a type as the Go source spells it: []Chapter, *Series, time.Time.
func (c *comparer) describe(x shape.Shape) string {
	return shape.ToGoTypeName(x, shape.WithRootPkgName(c.new.pkg))
}

func jsonName(f *shape.FieldLike) string {
	return shape.TagGetValue(f.Tags, "json", f.Name)
}

func omitempty(f *shape.FieldLike) bool {
	return shape.TagHasOption(f.Tags, "json", "omitempty")
}

// wireFields are the fields a struct puts in its JSON: no f.Returns
// phantom, no `json:"-"`.
func wireFields(st *shape.StructLike) []*shape.FieldLike {
	var result []*shape.FieldLike
	for _, f := range st.Fields {
		if shape.IsPhantomField(f) || jsonName(f) == "-" {
			continue
		}
		result = append(result, f)
	}
	return result
}
