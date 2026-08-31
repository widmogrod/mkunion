package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testStr = &PrimitiveLike{Kind: &StringLike{}}
	testNum = &PrimitiveLike{Kind: &NumberLike{}}
)

func refIn(pkg, imp, name string, indexed ...Shape) *RefName {
	return &RefName{Name: name, PkgName: pkg, PkgImportName: imp, Indexed: indexed}
}

func TestExtractPkgImportNames(t *testing.T) {
	useCases := map[string]struct {
		in   Shape
		want map[string]string
	}{
		"any yields nothing":       {&Any{}, nil},
		"primitive yields nothing": {testStr, nil},
		"ref yields its package": {
			refIn("other", "example.com/other", "T"),
			map[string]string{"other": "example.com/other"},
		},
		"ref without package yields empty": {
			&RefName{Name: "T"},
			map[string]string{},
		},
		"indexed ref includes type arguments": {
			refIn("other", "example.com/other", "T",
				refIn("third", "example.com/third", "U")),
			map[string]string{
				"other": "example.com/other",
				"third": "example.com/third",
			},
		},
		"pointer unwraps": {
			&PointerLike{Type: refIn("other", "example.com/other", "T")},
			map[string]string{"other": "example.com/other"},
		},
		"list element": {
			&ListLike{Element: refIn("other", "example.com/other", "T")},
			map[string]string{"other": "example.com/other"},
		},
		"map key and value": {
			&MapLike{
				Key: refIn("k", "example.com/k", "K"),
				Val: refIn("v", "example.com/v", "V"),
			},
			map[string]string{"k": "example.com/k", "v": "example.com/v"},
		},
		"alias includes its own package, its target, and type params": {
			&AliasLike{
				Name: "A", PkgName: "a", PkgImportName: "example.com/a",
				Type: refIn("t", "example.com/t", "T"),
				TypeParams: []TypeParam{
					{Name: "P", Type: refIn("p", "example.com/p", "P")},
				},
			},
			map[string]string{
				"a": "example.com/a",
				"t": "example.com/t",
				"p": "example.com/p",
			},
		},
		"struct includes fields and type params": {
			&StructLike{
				Name: "S", PkgName: "s", PkgImportName: "example.com/s",
				TypeParams: []TypeParam{
					{Name: "P", Type: refIn("p", "example.com/p", "P")},
				},
				Fields: []*FieldLike{
					{Name: "F", Type: refIn("f", "example.com/f", "F")},
				},
			},
			map[string]string{
				"s": "example.com/s",
				"p": "example.com/p",
				"f": "example.com/f",
			},
		},
		"union includes variants": {
			&UnionLike{
				Name: "U", PkgName: "u", PkgImportName: "example.com/u",
				Variant: []Shape{
					&StructLike{Name: "V", PkgName: "v", PkgImportName: "example.com/v"},
				},
			},
			map[string]string{"u": "example.com/u", "v": "example.com/v"},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.want, ExtractPkgImportNames(uc.in))
		})
	}
}

func TestExtractPkgImportNamesForTypeInitialisation(t *testing.T) {
	t.Run("struct fields are ignored, unlike ExtractPkgImportNames", func(t *testing.T) {
		s := &StructLike{
			Name: "S", PkgName: "s", PkgImportName: "example.com/s",
			Fields: []*FieldLike{
				{Name: "F", Type: refIn("f", "example.com/f", "F")},
			},
		}
		assert.Equal(t,
			map[string]string{"s": "example.com/s"},
			ExtractPkgImportNamesForTypeInitialisation(s))
	})

	t.Run("everything else matches ExtractPkgImportNames", func(t *testing.T) {
		shapes := []Shape{
			&Any{},
			testNum,
			refIn("other", "example.com/other", "T",
				refIn("third", "example.com/third", "U")),
			&PointerLike{Type: refIn("o", "example.com/o", "T")},
			&ListLike{Element: refIn("o", "example.com/o", "T")},
			&MapLike{
				Key: refIn("k", "example.com/k", "K"),
				Val: refIn("v", "example.com/v", "V"),
			},
			&AliasLike{
				Name: "A", PkgName: "a", PkgImportName: "example.com/a",
				Type: refIn("t", "example.com/t", "T"),
				TypeParams: []TypeParam{
					{Name: "P", Type: refIn("p", "example.com/p", "P")},
				},
			},
			&UnionLike{
				Name: "U", PkgName: "u", PkgImportName: "example.com/u",
				Variant: []Shape{
					// variant without fields: identical either way
					&StructLike{Name: "V", PkgName: "v", PkgImportName: "example.com/v"},
				},
			},
		}
		for _, s := range shapes {
			assert.Equal(t, ExtractPkgImportNames(s), ExtractPkgImportNamesForTypeInitialisation(s), "%T", s)
		}
	})
}
