package predicate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
)

func strPrim() shape.Shape {
	return &shape.PrimitiveLike{Kind: &shape.StringLike{}}
}

// syntheticShape exercises resolver branches that the generated test types
// do not reach: pointers, aliases, Any, maps, json tags, unresolved refs.
func syntheticShape() shape.Shape {
	return &shape.StructLike{
		Name:    "Synthetic",
		PkgName: "fake",
		Fields: []*shape.FieldLike{
			{Name: "Ptr", Type: &shape.PointerLike{Type: strPrim()}},
			{Name: "Alias", Type: &shape.AliasLike{Name: "MyAlias", PkgName: "fake", Type: strPrim()}},
			{Name: "Anything", Type: &shape.Any{}},
			{Name: "Tags", Type: &shape.MapLike{Key: strPrim(), Val: strPrim()}},
			{Name: "Nums", Type: &shape.ListLike{Element: &shape.PrimitiveLike{Kind: &shape.NumberLike{}}}},
			{Name: "Unknown", Type: &shape.RefName{Name: "NotRegistered", PkgName: "fake", PkgImportName: "example.com/fake"}},
			{Name: "Renamed", Type: strPrim(), Tags: map[string]shape.Tag{"json": {Value: "renamed,omitempty"}}},
			{Name: "Hidden", Type: strPrim(), Tags: map[string]shape.Tag{"json": {Value: "-"}}},
		},
	}
}

func TestResolveJSONPaths_EdgeShapes(t *testing.T) {
	s := syntheticShape()

	ok := func(t *testing.T, location, want string) {
		t.Helper()
		paths, err := ResolveJSONPaths(s, location)
		require.NoError(t, err)
		require.Len(t, paths, 1)
		assert.Equal(t, want, paths[0].String())
	}

	fails := func(t *testing.T, location string) {
		t.Helper()
		_, err := ResolveJSONPaths(s, location)
		assert.Error(t, err, location)
	}

	ok(t, "Ptr", "Ptr")
	ok(t, "Alias", "Alias")
	ok(t, `Anything.Deep[3][*]`, "Anything.Deep[3][*]")
	ok(t, `Tags["key"]`, "Tags.key")
	ok(t, "Tags[*]", "Tags[*]")
	ok(t, "Nums[2]", "Nums[2]")
	ok(t, "Nums[*]", "Nums[*]")
	ok(t, "Unknown.Whatever[1]", "Unknown.Whatever[1]")
	ok(t, "Renamed", "renamed")
	ok(t, `Renamed`, "renamed")

	fails(t, "Hidden")          // json:"-" fields are not serialized
	fails(t, "Ptr.Deep")        // cannot descend into a primitive
	fails(t, "Nums.Field")      // list needs [index] or [*]
	fails(t, "Tags[5]")         // map needs a key or [*]
	fails(t, "Missing")         // unknown struct field
	fails(t, `Anything`[:0+3]+"thing2") // "Anything2" unknown field

	t.Run("json name lookup by tag name", func(t *testing.T) {
		paths, err := ResolveJSONPaths(s, "renamed")
		require.NoError(t, err)
		assert.Equal(t, "renamed", paths[0].String())
	})

	t.Run("struct with wildcard is an error", func(t *testing.T) {
		_, err := ResolveJSONPaths(s, "Tags")
		require.NoError(t, err)
		_, err = ResolveJSONPaths(&shape.StructLike{Name: "S", PkgName: "fake"}, "[*]")
		assert.Error(t, err)
	})
}

func TestResolveJSONPaths_UnionEdges(t *testing.T) {
	union := &shape.UnionLike{
		Name:    "Choice",
		PkgName: "fake",
		Variant: []shape.Shape{
			&shape.StructLike{Name: "A", PkgName: "fake", Fields: []*shape.FieldLike{{Name: "X", Type: strPrim()}}},
			&shape.StructLike{Name: "B", PkgName: "fake", Fields: []*shape.FieldLike{{Name: "Y", Type: strPrim()}}},
		},
	}
	root := &shape.StructLike{
		Name: "Root", PkgName: "fake",
		Fields: []*shape.FieldLike{{Name: "C", Type: union}},
	}

	t.Run("wildcard over variants with shared descent", func(t *testing.T) {
		paths, err := ResolveJSONPaths(root, "C[*].X")
		require.NoError(t, err)
		require.Len(t, paths, 1)
		assert.Equal(t, "C.fake.A.X", paths[0].String())
	})

	t.Run("wildcard alone lists every variant", func(t *testing.T) {
		paths, err := ResolveJSONPaths(root, "C[*]")
		require.NoError(t, err)
		assert.Len(t, paths, 2)
	})

	t.Run("index into union is an error", func(t *testing.T) {
		_, err := ResolveJSONPaths(root, "C[5]")
		assert.Error(t, err)
	})

	t.Run("descend below $type is an error", func(t *testing.T) {
		_, err := ResolveJSONPaths(root, `C["$type"].X`)
		assert.Error(t, err)
	})

	t.Run("wildcard with unmatched descent is an error", func(t *testing.T) {
		_, err := ResolveJSONPaths(root, "C[*].Nope")
		assert.Error(t, err)
	})

	t.Run("variant name helpers", func(t *testing.T) {
		assert.Equal(t, "fake.A", JSONVariantName(&shape.PointerLike{Type: union.Variant[0]}))
		assert.Equal(t, "fake.Choice", JSONVariantName(union))
		assert.Equal(t, "", JSONVariantName(strPrim()))
		assert.Equal(t, "A", variantBareName(&shape.PointerLike{Type: union.Variant[0]}))
		assert.Equal(t, "", variantBareName(strPrim()))
		assert.Equal(t, "fake.Alias", JSONVariantName(&shape.AliasLike{Name: "Alias", PkgName: "fake"}))
		assert.Equal(t, "fake.Ref", JSONVariantName(&shape.RefName{Name: "Ref", PkgName: "fake"}))
	})
}

func TestSchemaToJSONValue(t *testing.T) {
	assert.Nil(t, SchemaToJSONValue(nil))
	assert.Nil(t, SchemaToJSONValue(schema.MkNone()))
	assert.Equal(t, true, SchemaToJSONValue(schema.MkBool(true)))
	assert.Equal(t, 1.5, SchemaToJSONValue(schema.MkFloat(1.5)))
	assert.Equal(t, "x", SchemaToJSONValue(schema.MkString("x")))
	assert.Equal(t, "aGk=", SchemaToJSONValue(schema.MkBinary([]byte("hi"))))
	assert.Equal(t, []any{"a", 2.0}, SchemaToJSONValue(schema.MkList(schema.MkString("a"), schema.MkInt(2))))
	assert.Equal(t, map[string]any{"k": "v"}, SchemaToJSONValue(schema.MkMap(schema.MkField("k", schema.MkString("v")))))
}

func TestCompareJSONValues(t *testing.T) {
	check := func(a, b any, want int, comparable bool) {
		t.Helper()
		got, ok := CompareJSONValues(a, b)
		assert.Equal(t, comparable, ok)
		if comparable {
			assert.Equal(t, want, got)
		}
	}
	check(nil, nil, 0, true)
	check(nil, "x", 0, false)
	check(1.0, 2.0, -1, true)
	check(2.0, 1.0, 1, true)
	check(1.0, 1.0, 0, true)
	check(1.0, "x", 0, false)
	check("a", "b", -1, true)
	check("a", 1.0, 0, false)
	check(true, true, 0, true)
	check(false, true, -1, true)
	check(true, false, 1, true)
	check(true, "x", 0, false)
	check([]any{}, []any{}, 0, false)
}

func TestEvaluate_UnknownOperation(t *testing.T) {
	eval := NewJSONEvaluator(syntheticShape())
	_, err := eval.Evaluate(&Compare{
		Location:  "Ptr",
		Operation: "~",
		BindValue: &Literal{Value: schema.MkString("x")},
	}, map[string]any{"Ptr": "x"}, nil)
	assert.Error(t, err)
}

func TestEvaluate_LocatableAndMissingBind(t *testing.T) {
	eval := NewJSONEvaluator(syntheticShape())
	doc := map[string]any{"Ptr": "x", "Alias": "x"}

	t.Run("locatable compares two fields", func(t *testing.T) {
		got, err := eval.Evaluate(&Compare{
			Location:  "Ptr",
			Operation: "=",
			BindValue: &Locatable{Location: "Alias"},
		}, doc, nil)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("locatable with missing value is false", func(t *testing.T) {
		got, err := eval.Evaluate(&Compare{
			Location:  "Ptr",
			Operation: "=",
			BindValue: &Locatable{Location: "Renamed"},
		}, doc, nil)
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("locatable with bad location errors", func(t *testing.T) {
		_, err := eval.Evaluate(&Compare{
			Location:  "Ptr",
			Operation: "=",
			BindValue: &Locatable{Location: "Missing"},
		}, doc, nil)
		assert.Error(t, err)
	})

	t.Run("missing bind param errors", func(t *testing.T) {
		_, err := eval.Evaluate(&Compare{
			Location:  "Ptr",
			Operation: "=",
			BindValue: &BindValue{BindName: ":gone"},
		}, doc, ParamBinds{})
		assert.Error(t, err)
	})
}
