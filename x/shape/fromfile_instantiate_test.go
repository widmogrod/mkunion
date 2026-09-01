package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstantiateTypeThatAreOvershadowByTypeParam(t *testing.T) {
	str := &PrimitiveLike{Kind: &StringLike{}}
	num := &PrimitiveLike{Kind: &NumberLike{}}
	replacement := map[string]Shape{"A": str, "B": num}

	sub := func(s Shape) Shape {
		return InstantiateTypeThatAreOvershadowByTypeParam(s, replacement)
	}

	t.Run("type param refs are replaced", func(t *testing.T) {
		assert.Equal(t, str, sub(&RefName{Name: "A"}))
	})

	t.Run("other refs keep their identity and substitute their args", func(t *testing.T) {
		got := sub(&RefName{
			Name: "ListOf", PkgName: "p", PkgImportName: "example.com/p",
			Indexed: []Shape{&RefName{Name: "A"}},
		})
		assert.Equal(t, &RefName{
			Name: "ListOf", PkgName: "p", PkgImportName: "example.com/p",
			Indexed: []Shape{str},
		}, got)
	})

	t.Run("primitives and any pass through", func(t *testing.T) {
		assert.Equal(t, str, sub(str))
		assert.Equal(t, &Any{}, sub(&Any{}))
	})

	t.Run("pointers, lists, and maps substitute recursively", func(t *testing.T) {
		assert.Equal(t,
			&PointerLike{Type: str},
			sub(&PointerLike{Type: &RefName{Name: "A"}}))

		assert.Equal(t,
			&ListLike{Element: str},
			sub(&ListLike{Element: &RefName{Name: "A"}}))

		assert.Equal(t,
			&MapLike{Key: str, Val: num},
			sub(&MapLike{Key: &RefName{Name: "A"}, Val: &RefName{Name: "B"}}))
	})

	t.Run("alias substitutes its target and marks substituted params", func(t *testing.T) {
		got := sub(&AliasLike{
			Name: "Alias", PkgName: "p",
			Type:       &RefName{Name: "A"},
			TypeParams: []TypeParam{{Name: "A", Type: &Any{}}, {Name: "C", Type: &Any{}}},
		})
		alias, ok := got.(*AliasLike)
		assert.True(t, ok)
		assert.Equal(t, str, alias.Type)
		assert.Equal(t, str, alias.TypeParams[0].Type, "substituted param records its replacement")
		assert.Equal(t, &Any{}, alias.TypeParams[1].Type, "unrelated params stay untouched")
	})

	t.Run("struct fields substitute", func(t *testing.T) {
		got := sub(&StructLike{
			Name: "S", PkgName: "p",
			Fields: []*FieldLike{{Name: "F", Type: &RefName{Name: "A"}}},
		})
		s, ok := got.(*StructLike)
		assert.True(t, ok)
		assert.Equal(t, str, s.Fields[0].Type)
	})

	t.Run("union variants substitute", func(t *testing.T) {
		got := sub(&UnionLike{
			Name: "U", PkgName: "p",
			Variant: []Shape{
				&StructLike{
					Name: "V", PkgName: "p",
					Fields: []*FieldLike{{Name: "F", Type: &RefName{Name: "B"}}},
				},
			},
		})
		u, ok := got.(*UnionLike)
		assert.True(t, ok)
		v := u.Variant[0].(*StructLike)
		assert.Equal(t, num, v.Fields[0].Type)
	})
}
