package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInjectPkgImportName(t *testing.T) {
	inject := InjectPkgImportName(map[string]string{
		"known": "example.com/known",
	})

	t.Run("fills the import name for every taggable shape kind", func(t *testing.T) {
		ref := &RefName{Name: "T", PkgName: "known"}
		structLike := &StructLike{Name: "S", PkgName: "known"}
		unionLike := &UnionLike{Name: "U", PkgName: "known"}
		aliasLike := &AliasLike{Name: "A", PkgName: "known"}

		for _, s := range []Shape{ref, structLike, unionLike, aliasLike} {
			inject(s)
		}

		assert.Equal(t, "example.com/known", ref.PkgImportName)
		assert.Equal(t, "example.com/known", structLike.PkgImportName)
		assert.Equal(t, "example.com/known", unionLike.PkgImportName)
		assert.Equal(t, "example.com/known", aliasLike.PkgImportName)
	})

	t.Run("existing import names are never overwritten", func(t *testing.T) {
		ref := &RefName{Name: "T", PkgName: "known", PkgImportName: "example.com/original"}
		inject(ref)
		assert.Equal(t, "example.com/original", ref.PkgImportName)
	})

	t.Run("unknown package stays empty", func(t *testing.T) {
		ref := &RefName{Name: "T", PkgName: "unknown"}
		inject(ref)
		assert.Equal(t, "", ref.PkgImportName)
	})

	t.Run("shapes without a package name are untouched", func(t *testing.T) {
		ref := &RefName{Name: "T"}
		inject(ref)
		assert.Equal(t, "", ref.PkgImportName)
	})

	t.Run("unhandled shapes are a no-op", func(t *testing.T) {
		p := &PrimitiveLike{Kind: &StringLike{}}
		assert.NotPanics(t, func() { inject(p) })
	})
}

func TestInjectPkgName(t *testing.T) {
	inject := InjectPkgName("mypkg")

	t.Run("fills the package name for every taggable shape kind", func(t *testing.T) {
		ref := &RefName{Name: "T"}
		structLike := &StructLike{Name: "S"}
		unionLike := &UnionLike{Name: "U"}
		aliasLike := &AliasLike{Name: "A"}

		for _, s := range []Shape{ref, structLike, unionLike, aliasLike} {
			inject(s)
		}

		assert.Equal(t, "mypkg", ref.PkgName)
		assert.Equal(t, "mypkg", structLike.PkgName)
		assert.Equal(t, "mypkg", unionLike.PkgName)
		assert.Equal(t, "mypkg", aliasLike.PkgName)
	})

	t.Run("existing package names are never overwritten", func(t *testing.T) {
		ref := &RefName{Name: "T", PkgName: "original"}
		inject(ref)
		assert.Equal(t, "original", ref.PkgName)
	})

	t.Run("unhandled shapes are a no-op", func(t *testing.T) {
		assert.NotPanics(t, func() { inject(&Any{}) })
	})
}

func TestInjectDotImportResolver(t *testing.T) {
	inject := InjectDotImportResolver(func(name string) (string, string, bool) {
		if name == "Known" {
			return "known", "example.com/known", true
		}
		return "", "", false
	})

	t.Run("resolves unqualified refs", func(t *testing.T) {
		ref := &RefName{Name: "Known"}
		inject(ref)
		assert.Equal(t, "known", ref.PkgName)
		assert.Equal(t, "example.com/known", ref.PkgImportName)
	})

	t.Run("unresolvable refs stay unqualified", func(t *testing.T) {
		ref := &RefName{Name: "Mystery"}
		inject(ref)
		assert.Equal(t, "", ref.PkgName)
	})

	t.Run("qualified refs are untouched", func(t *testing.T) {
		ref := &RefName{Name: "Known", PkgName: "other", PkgImportName: "example.com/other"}
		inject(ref)
		assert.Equal(t, "other", ref.PkgName)
		assert.Equal(t, "example.com/other", ref.PkgImportName)
	})
}
