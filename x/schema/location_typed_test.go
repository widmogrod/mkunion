// package schema_test breaks the import cycle: these tests exercise
// TypedLocation the way typedful repositories use it, with the real
// Record[T] shapes from x/storage/schemaless.
package schema_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
	"github.com/widmogrod/mkunion/x/workflow"
)

// notRegistered must never gain a generated shape - the constructor
// tests rely on it being unknown to the registry.
//
//go:tag shape:"-"
type notRegistered struct{ A int }

func recordOfSchemaShape(t *testing.T) shape.Shape {
	t.Helper()
	encodedAs, found := shape.LookupShapeReflectAndIndex[schemaless.Record[schema.Schema]]()
	require.True(t, found)
	return encodedAs
}

func typedUserLocation(t *testing.T) *schema.TypedLocation {
	t.Helper()
	loc, err := schema.NewTypedLocationWithEncoded[schemaless.Record[typedful.User]](recordOfSchemaShape(t))
	require.NoError(t, err)
	return loc
}

// A typed Record[User] is stored as Record[schema.Schema]; locations into
// the typed Data must gain the schema.* wrappers of the encoded form,
// while fields that encode identically (ID, Type) stay untouched.
func TestTypedLocationWrapsEncodedFields(t *testing.T) {
	loc := typedUserLocation(t)

	useCases := map[string]struct {
		in   string
		want string
	}{
		"typed string field": {"Data.Name", `Data["schema.Map"].Name["schema.String"]`},
		"typed number field": {"Data.Age", `Data["schema.Map"].Age["schema.Number"]`},
		"identically encoded ID":   {"ID", "ID"},
		"identically encoded Type": {"Type", "Type"},
		"bare diverging field stays bare": {"Data", "Data"},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got, err := loc.WrapLocationStr(uc.in)
			require.NoError(t, err)
			assert.Equal(t, uc.want, got)
		})
	}
}

func TestTypedLocationWrapsUnionData(t *testing.T) {
	loc, err := schema.NewTypedLocationWithEncoded[schemaless.Record[schema.Location]](recordOfSchemaShape(t))
	require.NoError(t, err)

	t.Run("variant field gains both wrappers", func(t *testing.T) {
		got, err := loc.WrapLocationStr(`Data["schema.LocationField"].Name`)
		require.NoError(t, err)
		assert.Equal(t, `Data["schema.Map"]["schema.LocationField"]["schema.Map"].Name["schema.String"]`, got)
	})

	t.Run("$type resolves to a wrapped string", func(t *testing.T) {
		got, err := loc.WrapLocationStr(`Data["$type"]`)
		require.NoError(t, err)
		assert.Equal(t, `Data["schema.Map"]["$type"]["schema.String"]`, got)
	})
}

func TestTypedLocationWrapsCollections(t *testing.T) {
	wrapFor := func(t *testing.T, loc *schema.TypedLocation, in string) string {
		t.Helper()
		got, err := loc.WrapLocationStr(in)
		require.NoError(t, err)
		return got
	}

	t.Run("list field indexed inside a struct", func(t *testing.T) {
		loc, err := schema.NewTypedLocationWithEncoded[schemaless.Record[workflow.Flow]](recordOfSchemaShape(t))
		require.NoError(t, err)
		got := wrapFor(t, loc, "Data.Body[0]")
		assert.Equal(t, `Data["schema.Map"].Body["schema.List"][0]["schema.Map"]`, got)
	})

	// mirrors the predicates the demo server uses against workflow states,
	// e.g. Data["workflow.Done"].BaseState.Variables.input
	t.Run("union variant, struct, and map segments", func(t *testing.T) {
		loc, err := schema.NewTypedLocationWithEncoded[schemaless.Record[workflow.State]](recordOfSchemaShape(t))
		require.NoError(t, err)
		got := wrapFor(t, loc, `Data["workflow.Done"].BaseState.Variables.input`)
		assert.Equal(t, `Data["schema.Map"]["workflow.Done"]["schema.Map"].BaseState["schema.Map"].Variables["schema.Map"].input["schema.Map"]`, got)
	})

	// Documents a current limitation: an index directly under the typed
	// Data field (e.g. Data[0] for Record[[]string]) is not implemented
	// at the encodedAs layer and panics.
	t.Run("index directly under Data panics", func(t *testing.T) {
		loc, err := schema.NewTypedLocationWithEncoded[schemaless.Record[[]string]](recordOfSchemaShape(t))
		require.NoError(t, err)
		assert.Panics(t, func() {
			_, _ = loc.WrapLocationStr("Data[0]")
		})
	})
}

func TestTypedLocationRejects(t *testing.T) {
	loc := typedUserLocation(t)

	t.Run("unparsable location errors", func(t *testing.T) {
		_, err := loc.WrapLocationStr(`Data[`)
		assert.Error(t, err)
	})

	t.Run("unknown top-level field panics", func(t *testing.T) {
		assert.Panics(t, func() {
			_, _ = loc.WrapLocationStr("Nope")
		})
	})

	t.Run("unknown typed field panics", func(t *testing.T) {
		assert.Panics(t, func() {
			_, _ = loc.WrapLocationStr("Data.Nope")
		})
	})

	t.Run("index into a struct panics", func(t *testing.T) {
		assert.Panics(t, func() {
			_, _ = loc.WrapLocationStr("Data[0]")
		})
	})
}

// Direct calls with hand-built shapes pin the contract of every
// mismatch branch: divergences the wrapper cannot reconcile panic.
func TestWrapLocationEncodedAsMismatches(t *testing.T) {
	loc := typedUserLocation(t)

	str := &shape.PrimitiveLike{Kind: &shape.StringLike{}}
	num := &shape.PrimitiveLike{Kind: &shape.NumberLike{}}
	structA := &shape.StructLike{
		Name: "A", PkgName: "x", PkgImportName: "example.com/x",
		Fields: []*shape.FieldLike{{Name: "F", Type: str}},
	}
	structB := &shape.StructLike{Name: "B", PkgName: "x", PkgImportName: "example.com/x"}
	unionU := &shape.UnionLike{
		Name: "U", PkgName: "x", PkgImportName: "example.com/x",
		Variant: []shape.Shape{structA},
	}
	unionV := &shape.UnionLike{Name: "V", PkgName: "x", PkgImportName: "example.com/x"}

	field := func(name string) []schema.Location {
		return []schema.Location{&schema.LocationField{Name: name}}
	}

	t.Run("empty locations resolve to nothing", func(t *testing.T) {
		got, err := loc.WrapLocationEncodedAs(nil, structA, structA, false)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("same union resolves its variant", func(t *testing.T) {
		got, err := loc.WrapLocationEncodedAs(field("x.A"), unionU, unionU, false)
		require.NoError(t, err)
		assert.Equal(t, field("x.A"), got)
	})

	panics := map[string]func(){
		"different structs": func() {
			_, _ = loc.WrapLocationEncodedAs(field("F"), structA, structB, false)
		},
		"field not found in struct": func() {
			_, _ = loc.WrapLocationEncodedAs(field("Nope"), structA, structA, false)
		},
		"different unions that are not schema": func() {
			_, _ = loc.WrapLocationEncodedAs(field("F"), unionU, unionV, false)
		},
		"$type on the same union": func() {
			_, _ = loc.WrapLocationEncodedAs(field("$type"), unionU, unionU, false)
		},
		"unknown variant of the same union": func() {
			_, _ = loc.WrapLocationEncodedAs(field("x.Zzz"), unionU, unionU, false)
		},
		"primitive divergence that is not schema": func() {
			_, _ = loc.WrapLocationEncodedAs(field("F"), str, num, false)
		},
		"unresolvable right ref": func() {
			_, _ = loc.WrapLocationEncodedAs(field("F"), structA, &shape.RefName{
				Name: "Missing", PkgName: "x", PkgImportName: "example.com/x",
			}, false)
		},
		"unresolvable left ref": func() {
			_, _ = loc.WrapLocationEncodedAs(field("F"), &shape.RefName{
				Name: "Missing", PkgName: "x", PkgImportName: "example.com/x",
			}, structA, false)
		},
		"index location": func() {
			_, _ = loc.WrapLocationEncodedAs([]schema.Location{&schema.LocationIndex{Index: 0}}, structA, structA, false)
		},
		"anything location": func() {
			_, _ = loc.WrapLocationEncodedAs([]schema.Location{&schema.LocationAnything{}}, structA, structA, false)
		},
	}
	for name, fn := range panics {
		t.Run(name+" panics", func(t *testing.T) {
			assert.Panics(t, fn)
		})
	}
}

func TestTypedLocationConstructors(t *testing.T) {
	t.Run("unregistered type errors", func(t *testing.T) {
		_, err := schema.NewTypedLocation[notRegistered]()
		assert.ErrorIs(t, err, shape.ErrShapeNotFound)

		_, err = schema.NewTypedLocationWithEncoded[notRegistered](nil)
		assert.ErrorIs(t, err, shape.ErrShapeNotFound)
	})

	t.Run("registered type resolves and exposes its shape", func(t *testing.T) {
		loc, err := schema.NewTypedLocation[schemaless.Record[typedful.User]]()
		require.NoError(t, err)
		assert.NotNil(t, loc.ShapeDef())
		assert.Nil(t, loc.EncodedAs())

		withEncoded := loc.WithEncodedAs(recordOfSchemaShape(t))
		assert.NotNil(t, withEncoded.EncodedAs())
	})

	t.Run("without encoding, index into a struct panics", func(t *testing.T) {
		loc, err := schema.NewTypedLocation[schemaless.Record[typedful.User]]()
		require.NoError(t, err)
		assert.Panics(t, func() {
			_, _ = loc.WrapLocationStr("Data[0]")
		})
	})

	t.Run("without encoding, index into a union panics", func(t *testing.T) {
		loc, err := schema.NewTypedLocation[schemaless.Record[schema.Location]]()
		require.NoError(t, err)
		assert.Panics(t, func() {
			_, _ = loc.WrapLocationStr("Data[0]")
		})
	})

	t.Run("empty location wraps to empty", func(t *testing.T) {
		loc := typedUserLocation(t)
		got, err := loc.WrapLocation(nil)
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
