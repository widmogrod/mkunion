package shape

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The Swift the exporter writes for testasset/type_api.go is kept under
// testasset/swift and compared byte for byte. Set UPDATE_GOLDEN=1 to rewrite it.
func TestSwiftExport(t *testing.T) {
	inferred, err := InferFromFile("testasset/type_api.go")
	require.NoError(t, err)

	r := NewSwiftRenderer()
	for _, x := range inferred.RetrieveShapes() {
		r.AddShape(x)
	}
	files, err := r.Render()
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"Testasset.swift", "MkunionJSON.swift"}, keys(files))

	for name, content := range files {
		golden := filepath.Join("testasset", "swift", name)
		if os.Getenv("UPDATE_GOLDEN") == "1" {
			require.NoError(t, os.MkdirAll(filepath.Dir(golden), 0755))
			require.NoError(t, os.WriteFile(golden, []byte(content), 0644))
		}
		want, err := os.ReadFile(golden)
		require.NoError(t, err, "run with UPDATE_GOLDEN=1 to write %s", golden)
		assert.Equal(t, string(want), content, name)
	}
}

func keys(m map[string]string) []string {
	var result []string
	for k := range m {
		result = append(result, k)
	}
	return result
}

// Shapes the exporter does not support stop the export with an error that
// names them, instead of writing Swift that does not compile.
func TestSwiftExportRefusesWhatItCannotWrite(t *testing.T) {
	generic := &StructLike{
		Name: "Box", PkgName: "demo", PkgImportName: "example.com/demo",
		TypeParams: []TypeParam{{Name: "T", Type: &Any{}}},
		Fields:     []*FieldLike{{Name: "Value", Type: &RefName{Name: "T"}}},
	}
	instantiated := &StructLike{
		Name: "Holder", PkgName: "demo", PkgImportName: "example.com/demo",
		Fields: []*FieldLike{{Name: "Box", Type: &RefName{
			Name: "Box", PkgName: "demo", PkgImportName: "example.com/demo",
			Indexed: []Shape{&PrimitiveLike{Kind: &StringLike{}}},
		}}},
	}
	inline := &StructLike{
		Name: "Outer", PkgName: "demo", PkgImportName: "example.com/demo",
		Fields: []*FieldLike{{Name: "Inner", Type: &StructLike{
			Fields: []*FieldLike{{Name: "N", Type: &PrimitiveLike{Kind: &NumberLike{Kind: &Int{}}}}},
		}}},
	}
	empty := &UnionLike{Name: "Nothing", PkgName: "demo", PkgImportName: "example.com/demo"}

	r := NewSwiftRenderer()
	for _, x := range []Shape{generic, instantiated, inline, empty} {
		r.AddShape(x)
	}
	_, err := r.Render()
	require.Error(t, err)
	for _, want := range []string{
		"demo.Box has type parameters",
		"demo.Box is instantiated with type arguments",
		"demo.Outer.Inner is an inline struct",
		"demo.Nothing has no variants",
	} {
		assert.Contains(t, err.Error(), want)
	}
}

// Two Go packages with the same base name would land in the same Swift file.
func TestSwiftExportRefusesPackageNameClash(t *testing.T) {
	r := NewSwiftRenderer()
	r.AddShape(&StructLike{Name: "A", PkgName: "api", PkgImportName: "example.com/one/api"})
	r.AddShape(&StructLike{Name: "B", PkgName: "api", PkgImportName: "example.com/two/api"})
	_, err := r.Render()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "both render as Api.swift")
}

func TestSwiftNames(t *testing.T) {
	assert.Equal(t, "Api", swiftPkgName("api"))
	assert.Equal(t, "MyApp", swiftPkgName("my_app"))
	assert.Equal(t, "getInfo", swiftCaseName("GetInfo"))
	assert.Equal(t, "url", swiftCaseName("URL"))
	assert.Equal(t, "httpServer", swiftCaseName("HTTPServer"))
	assert.Equal(t, "k", swiftCaseName("K"))
	assert.Equal(t, "`default`", swiftCaseName("Default"))
	assert.Equal(t, "Type_", swiftTypeName("Type"))
}

// zeroOf and typeOf decide what a missing JSON value stands for and how a Go
// type is spelled in Swift, one reference kind at a time.
func TestSwiftZeroAndTypeOfReferences(t *testing.T) {
	r := NewSwiftRenderer()
	r.AddShape(&UnionLike{Name: "State", PkgName: "demo", PkgImportName: "example.com/demo",
		Variant: []Shape{&StructLike{Name: "On", PkgName: "demo", PkgImportName: "example.com/demo"}}})
	r.AddShape(&AliasLike{Name: "Names", PkgName: "demo", PkgImportName: "example.com/demo",
		Type: &ListLike{Element: &PrimitiveLike{Kind: &StringLike{}}}})
	r.AddShape(&StructLike{Name: "Plain", PkgName: "demo", PkgImportName: "example.com/demo"})
	w := &swiftWriter{r: r, pkg: r.packages["example.com/demo"]}

	ref := func(pkg, imp, name string) *RefName { return &RefName{Name: name, PkgName: pkg, PkgImportName: imp} }
	cases := []struct {
		name      string
		shape     Shape
		typ, zero string
	}{
		{"union", ref("demo", "example.com/demo", "State"), "State", "nil"},
		{"alias", ref("demo", "example.com/demo", "Names"), "Names", "[]"},
		{"struct", ref("demo", "example.com/demo", "Plain"), "Plain", "Plain()"},
		{"other package, not found", ref("other", "example.com/nowhere/other", "Thing"), "Other.Thing", "Other.Thing()"},
		{"time", ref("time", "time", "Time"), "Date", "MkunionJSON.zeroDate"},
		{"duration", ref("time", "time", "Duration"), "Int", "0"},
		{"builtin string", ref("", "", "string"), "String", `""`},
		{"builtin int", ref("", "", "int"), "Int", "0"},
		{"pointer", &PointerLike{Type: ref("demo", "example.com/demo", "Plain")}, "Plain?", "nil"},
		{"bytes", &ListLike{Element: &PrimitiveLike{Kind: &NumberLike{Kind: &UInt8{}}}}, "Data", "Data()"},
		{"map", &MapLike{Key: &PrimitiveLike{Kind: &NumberLike{Kind: &Int{}}}, Val: &PrimitiveLike{Kind: &BooleanLike{}}}, "[String: Bool]", "[:]"},
		{"any", &Any{}, "JSONValue", ".null"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.typ, w.typeOf(c.shape))
			assert.Equal(t, c.zero, w.zeroOf(c.shape))
		})
	}

	t.Run("generic instantiation and unknown builtin are errors", func(t *testing.T) {
		assert.Equal(t, ".null", w.zeroOf(&RefName{Name: "Box", PkgName: "demo", PkgImportName: "example.com/demo", Indexed: []Shape{&Any{}}}))
		assert.Equal(t, ".null", w.zeroOf(ref("", "", "error")))
		assert.Equal(t, "JSONValue", w.typeOf(ref("", "", "error")))
		assert.NotEmpty(t, r.errs)
	})
}
