package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
)

var (
	strShape = &shape.PrimitiveLike{Kind: &shape.StringLike{}}
	numShape = &shape.PrimitiveLike{Kind: &shape.NumberLike{}}
)

func mkStructShape(name string, fields ...*shape.FieldLike) *shape.StructLike {
	return &shape.StructLike{
		Name:          name,
		PkgName:       "schematest",
		PkgImportName: "example.com/schematest",
		Fields:        fields,
	}
}

func mkField2(name string, typ shape.Shape) *shape.FieldLike {
	return &shape.FieldLike{Name: name, Type: typ}
}

// userShape models:
//
//	type User struct {
//	    Name    string
//	    Age     float64
//	    Tags    []string
//	    Attrs   map[string]float64
//	    Address Address // struct { City string }
//	}
func userShape() *shape.StructLike {
	return mkStructShape("User",
		mkField2("Name", strShape),
		mkField2("Age", numShape),
		mkField2("Tags", &shape.ListLike{Element: strShape}),
		mkField2("Attrs", &shape.MapLike{Val: numShape}),
		mkField2("Address", mkStructShape("Address",
			mkField2("City", strShape),
		)),
	)
}

func userData() Schema {
	return MkMap(
		MkField("Name", MkString("Ala")),
		MkField("Age", MkInt(42)),
		MkField("Tags", MkList(MkString("a"), MkString("b"))),
		MkField("Attrs", MkMap(MkField("height", MkFloat(1.7)))),
		MkField("Address", MkMap(MkField("City", MkString("Warsaw")))),
	)
}

func TestGetShapeLocation_Struct(t *testing.T) {
	useCases := map[string]struct {
		shape    shape.Shape
		data     Schema
		location string
		want     Schema
		found    bool
	}{
		"top level string field": {
			shape: userShape(), data: userData(),
			location: "Name",
			want:     MkString("Ala"), found: true,
		},
		"top level number field": {
			shape: userShape(), data: userData(),
			location: "Age",
			want:     MkInt(42), found: true,
		},
		"nested struct field": {
			shape: userShape(), data: userData(),
			location: "Address.City",
			want:     MkString("Warsaw"), found: true,
		},
		"field not present in shape": {
			shape: userShape(), data: userData(),
			location: "Nope",
			want:     nil, found: false,
		},
		"field in shape but absent in data": {
			shape: userShape(),
			data:  MkMap(MkField("Age", MkInt(1))),
			// Name is a valid field of the shape, but this record has no value for it
			location: "Name",
			want:     nil, found: false,
		},
		"data is not a map": {
			shape: userShape(),
			data:  MkList(MkInt(1)),
			// struct access requires *Map data
			location: "Name",
			want:     nil, found: false,
		},
		"nil data": {
			shape: userShape(), data: nil,
			location: "Name",
			want:     nil, found: false,
		},
		"list index in range": {
			shape: userShape(), data: userData(),
			location: "Tags[1]",
			want:     MkString("b"), found: true,
		},
		"list index out of range": {
			shape: userShape(), data: userData(),
			location: "Tags[2]",
			want:     nil, found: false,
		},
		"index into a non-list shape": {
			shape: userShape(), data: userData(),
			// Name is a string; indexing it resolves nothing
			location: "Name[0]",
			want:     nil, found: false,
		},
		"index into list shape but data is not a list": {
			shape: userShape(),
			data:  MkMap(MkField("Tags", MkString("not-a-list"))),
			location: "Tags[0]",
			want:     nil, found: false,
		},
		"map value by key": {
			shape: userShape(), data: userData(),
			location: "Attrs.height",
			want:     MkFloat(1.7), found: true,
		},
		"map key absent in data": {
			shape: userShape(), data: userData(),
			location: "Attrs.width",
			want:     nil, found: false,
		},
		"map shape with non-map data": {
			shape:    userShape(),
			data:     MkMap(MkField("Attrs", MkInt(1))),
			location: "Attrs.height",
			want:     nil, found: false,
		},
		// Documents lenient current behavior: field access on a primitive
		// ignores the field name and returns the primitive itself.
		"field access on a number returns the number": {
			shape: userShape(), data: userData(),
			location: "Age.Whatever",
			want:     MkInt(42), found: true,
		},
		"field access on a string returns the string": {
			shape: userShape(), data: userData(),
			location: "Name.Whatever",
			want:     MkString("Ala"), found: true,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got, _, found := GetShapeLocation(uc.shape, uc.data, uc.location)
			assert.Equal(t, uc.found, found)
			assert.Equal(t, uc.want, got)
		})
	}
}

func TestGetShapeLocation_ResolvedShape(t *testing.T) {
	t.Run("leaf keeps its declared shape", func(t *testing.T) {
		_, s, found := GetShapeLocation(userShape(), userData(), "Address.City")
		assert.True(t, found)
		assert.Equal(t, strShape, s)
	})
	t.Run("map access resolves to the value shape", func(t *testing.T) {
		_, s, found := GetShapeLocation(userShape(), userData(), "Attrs.height")
		assert.True(t, found)
		assert.Equal(t, numShape, s)
	})
}

func TestGetShapeLocation_AliasAndPointer(t *testing.T) {
	alias := &shape.AliasLike{
		Name:          "UserAlias",
		PkgName:       "schematest",
		PkgImportName: "example.com/schematest",
		Type:          userShape(),
	}
	t.Run("alias delegates to the aliased struct", func(t *testing.T) {
		got, _, found := GetShapeLocation(alias, userData(), "Name")
		assert.True(t, found)
		assert.Equal(t, MkString("Ala"), got)
	})
	t.Run("alias to a missing field", func(t *testing.T) {
		_, _, found := GetShapeLocation(alias, userData(), "Nope")
		assert.False(t, found)
	})

	ptr := &shape.PointerLike{Type: userShape()}
	t.Run("pointer is transparent for field access", func(t *testing.T) {
		got, _, found := GetShapeLocation(ptr, userData(), "Age")
		assert.True(t, found)
		assert.Equal(t, MkInt(42), got)
	})
}

var _ = func() bool {
	shape.Register(mkStructShape("RefTarget",
		mkField2("Value", strShape),
	))
	return true
}()

func refTargetRef() *shape.RefName {
	return &shape.RefName{
		Name:          "RefTarget",
		PkgName:       "schematest",
		PkgImportName: "example.com/schematest",
	}
}

func TestGetShapeLocation_RefName(t *testing.T) {
	holder := mkStructShape("Holder",
		mkField2("Ref", refTargetRef()),
	)
	data := MkMap(
		MkField("Ref", MkMap(MkField("Value", MkString("resolved")))),
	)

	t.Run("ref resolves through the registry", func(t *testing.T) {
		got, _, found := GetShapeLocation(holder, data, "Ref.Value")
		assert.True(t, found)
		assert.Equal(t, MkString("resolved"), got)
	})
	t.Run("unregistered ref resolves nothing", func(t *testing.T) {
		missing := mkStructShape("Holder2",
			mkField2("Ref", &shape.RefName{
				Name:          "NeverRegistered",
				PkgName:       "schematest",
				PkgImportName: "example.com/schematest",
			}),
		)
		_, _, found := GetShapeLocation(missing, data, "Ref.Value")
		assert.False(t, found)
	})
}

func unionShape() *shape.UnionLike {
	return &shape.UnionLike{
		Name:          "Shape2D",
		PkgName:       "schematest",
		PkgImportName: "example.com/schematest",
		Variant: []shape.Shape{
			mkStructShape("Circle", mkField2("Radius", numShape)),
			mkStructShape("Square", mkField2("Side", numShape)),
		},
	}
}

// unionData mirrors how mkunion serialises a union value:
//
//	{"$type": "schematest.Circle", "schematest.Circle": {"Radius": 10}}
func unionData() Schema {
	return MkMap(
		MkField("$type", MkString("schematest.Circle")),
		MkField("schematest.Circle", MkMap(MkField("Radius", MkInt(10)))),
	)
}

// wrapShape puts the union under a struct field, because a location
// path cannot start with a bracket access.
func wrapShape() *shape.StructLike {
	return mkStructShape("Wrap", mkField2("Data", unionShape()))
}

func wrapData() Schema {
	return MkMap(MkField("Data", unionData()))
}

func TestGetShapeLocation_Union(t *testing.T) {
	t.Run("$type is accessible as a string", func(t *testing.T) {
		got, s, found := GetShapeLocation(wrapShape(), wrapData(), `Data["$type"]`)
		assert.True(t, found)
		assert.Equal(t, MkString("schematest.Circle"), got)
		assert.Equal(t, strShape, s)
	})
	t.Run("field inside the active variant", func(t *testing.T) {
		got, _, found := GetShapeLocation(wrapShape(), wrapData(), `Data["schematest.Circle"].Radius`)
		assert.True(t, found)
		assert.Equal(t, MkInt(10), got)
	})
	t.Run("field of a variant that is not active", func(t *testing.T) {
		_, _, found := GetShapeLocation(wrapShape(), wrapData(), `Data["schematest.Square"].Side`)
		assert.False(t, found)
	})
	t.Run("data is not a map", func(t *testing.T) {
		_, _, found := GetShapeLocation(wrapShape(), MkMap(MkField("Data", MkInt(1))), `Data["$type"]`)
		assert.False(t, found)
	})

	t.Run("second variant resolves too", func(t *testing.T) {
		data := MkMap(MkField("Data", MkMap(
			MkField("$type", MkString("schematest.Square")),
			MkField("schematest.Square", MkMap(MkField("Side", MkInt(4)))),
		)))
		got, _, found := GetShapeLocation(wrapShape(), data, `Data["schematest.Square"].Side`)
		assert.True(t, found)
		assert.Equal(t, MkInt(4), got)
	})
	t.Run("key that is no variant of the union is not found", func(t *testing.T) {
		data := MkMap(MkField("Data", MkMap(
			MkField("junk", MkInt(1)),
		)))
		_, _, found := GetShapeLocation(wrapShape(), data, `Data["junk"]`)
		assert.False(t, found)
	})
	t.Run("terminal access to a variant returns its sub-map", func(t *testing.T) {
		got, s, found := GetShapeLocation(wrapShape(), wrapData(), `Data["schematest.Circle"]`)
		assert.True(t, found)
		assert.Equal(t, MkMap(MkField("Radius", MkInt(10))), got)
		assert.Equal(t, unionShape().Variant[0], s)
	})
}

func TestGetShapeLocation_Anything(t *testing.T) {
	useCases := map[string]struct {
		shape    shape.Shape
		data     Schema
		location string
		want     Schema
		found    bool
	}{
		"anything on string leaf": {
			shape: userShape(), data: userData(),
			location: "Name[*]",
			want:     MkString("Ala"), found: true,
		},
		"anything on number leaf": {
			shape: userShape(), data: userData(),
			location: "Age[*]",
			want:     MkInt(42), found: true,
		},
		"anything searches map values": {
			shape: userShape(), data: userData(),
			location: "Attrs[*]",
			want:     MkFloat(1.7), found: true,
		},
		"anything searches list elements": {
			shape:    mkStructShape("Wrap", mkField2("Items", &shape.ListLike{Element: userShape()})),
			data:     MkMap(MkField("Items", MkList(userData()))),
			location: "Items[*].Name",
			want:     MkString("Ala"), found: true,
		},
		"anything searches union variants": {
			shape: wrapShape(),
			data:  wrapData(),
			// resolves through the serialised union without naming the variant
			location: "Data[*].Radius",
			want:     MkInt(10), found: true,
		},
		"anything searches struct fields": {
			shape:    mkStructShape("Wrap", mkField2("Data", userShape())),
			data:     MkMap(MkField("Data", MkMap(MkField("City", MkString("Warsaw"))))),
			location: "Data[*].City",
			want:     MkString("Warsaw"), found: true,
		},
		// The three cases below used to panic instead of reporting not-found
		// when a [*] search exhausted a list, union, or the data lacked
		// a match; they must behave like the map/struct cases.
		"anything on empty list finds nothing": {
			shape:    mkStructShape("Wrap", mkField2("Items", &shape.ListLike{Element: strShape})),
			data:     MkMap(MkField("Items", MkList())),
			location: "Items[*]",
			want:     nil, found: false,
		},
		"anything on list without a match finds nothing": {
			shape:    mkStructShape("Wrap", mkField2("Items", &shape.ListLike{Element: userShape()})),
			data:     MkMap(MkField("Items", MkList(MkMap(MkField("Other", MkInt(1)))))),
			location: "Items[*].Name",
			want:     nil, found: false,
		},
		"anything on union without an active variant finds nothing": {
			shape:    wrapShape(),
			data:     MkMap(MkField("Data", MkMap(MkField("$type", MkString("schematest.Circle"))))),
			location: "Data[*].Radius",
			want:     nil, found: false,
		},
		"anything through a ref with mismatched data finds nothing": {
			shape:    mkStructShape("Wrap", mkField2("Ref", refTargetRef())),
			data:     MkMap(MkField("Ref", MkInt(1))),
			location: "Ref[*].Nope",
			want:     nil, found: false,
		},
		"anything on list shape with non-list data": {
			shape:    mkStructShape("Wrap", mkField2("Items", &shape.ListLike{Element: strShape})),
			data:     MkMap(MkField("Items", MkString("not-a-list"))),
			location: "Items[*].Nope",
			want:     nil, found: false,
		},
		"anything through an unregistered ref finds nothing": {
			shape: mkStructShape("Wrap", mkField2("Ref", &shape.RefName{
				Name:          "NeverRegistered",
				PkgName:       "schematest",
				PkgImportName: "example.com/schematest",
			})),
			data:     MkMap(MkField("Ref", MkInt(1))),
			location: "Ref[*]",
			want:     nil, found: false,
		},
		"anything through an alias resolves": {
			shape: mkStructShape("Wrap", mkField2("A", &shape.AliasLike{
				Name: "UserAlias", PkgName: "schematest",
				PkgImportName: "example.com/schematest",
				Type:          userShape(),
			})),
			data:     MkMap(MkField("A", userData())),
			location: "A[*].Name",
			want:     MkString("Ala"), found: true,
		},
		"anything through an alias without a match finds nothing": {
			shape: mkStructShape("Wrap", mkField2("A", &shape.AliasLike{
				Name: "UserAlias", PkgName: "schematest",
				PkgImportName: "example.com/schematest",
				Type:          userShape(),
			})),
			data:     MkMap(MkField("A", userData())),
			location: "A[*].Nope",
			want:     nil, found: false,
		},
		// a terminal [*] matches whatever value sits at that location,
		// even when it does not match the declared shape
		"terminal anything matches any value": {
			shape:    mkStructShape("Wrap", mkField2("Ref", refTargetRef())),
			data:     MkMap(MkField("Ref", MkInt(1))),
			location: "Ref[*]",
			want:     MkInt(1), found: true,
		},
		"anything on string shape with non-string data": {
			shape:    userShape(),
			data:     MkMap(MkField("Name", MkInt(1))),
			location: "Name[*]",
			want:     nil, found: false,
		},
		"anything on number shape with non-number data": {
			shape:    userShape(),
			data:     MkMap(MkField("Age", MkString("old"))),
			location: "Age[*]",
			want:     nil, found: false,
		},
		"anything on empty map finds nothing": {
			shape:    userShape(),
			data:     MkMap(MkField("Attrs", MkMap())),
			location: "Attrs[*]",
			want:     nil, found: false,
		},
		"anything on union shape with non-map data": {
			shape:    wrapShape(),
			data:     MkMap(MkField("Data", MkString("not-a-map"))),
			location: "Data[*]",
			want:     nil, found: false,
		},
		"anything with mismatched data": {
			shape:    mkStructShape("Wrap", mkField2("Attrs", &shape.MapLike{Val: strShape})),
			data:     MkMap(MkField("Attrs", MkList(MkString("a")))),
			location: "Attrs[*]",
			want:     nil, found: false,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got, _, found := GetShapeLocation(uc.shape, uc.data, uc.location)
			assert.Equal(t, uc.found, found)
			assert.Equal(t, uc.want, got)
		})
	}
}

func TestGetShapeSchemaLocation_Terminals(t *testing.T) {
	t.Run("empty locations return input as-is with the found flag", func(t *testing.T) {
		data, s, found := GetShapeSchemaLocation(userShape(), userData(), nil, true)
		assert.True(t, found)
		assert.Equal(t, userData(), data)
		assert.Equal(t, userShape(), s)
	})
	t.Run("empty locations preserve found=false", func(t *testing.T) {
		_, _, found := GetShapeSchemaLocation(userShape(), userData(), nil, false)
		assert.False(t, found)
	})
	t.Run("nil data with remaining locations is not found", func(t *testing.T) {
		data, s, found := GetShapeSchemaLocation(userShape(), nil, []Location{&LocationField{Name: "Name"}}, true)
		assert.False(t, found)
		assert.Nil(t, data)
		assert.Nil(t, s)
	})
}

// These used to panic ("unknown field access" / "unknown anything access").
// The mismatches are reachable from user-supplied query paths — for example
// a [*] search walks every struct field, including list fields — so they
// must report not-found, never crash.
func TestGetShapeLocation_ShapeMismatchesDoNotPanic(t *testing.T) {
	t.Run("field access on a list shape is not found", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_, _, found := GetShapeLocation(&shape.ListLike{Element: strShape}, MkList(), "Name")
			assert.False(t, found)
		})
	})
	t.Run("anything access on a pointer shape is not found", func(t *testing.T) {
		s := mkStructShape("W", mkField2("Ptr", &shape.PointerLike{Type: strShape}))
		data := MkMap(MkField("Ptr", MkString("x")))
		assert.NotPanics(t, func() {
			_, _, found := GetShapeLocation(s, data, "Ptr[*]")
			assert.False(t, found)
		})
	})
	t.Run("anything search over a struct with a list field is not found when nothing matches", func(t *testing.T) {
		s := mkStructShape("Wrap", mkField2("U", userShape()))
		data := MkMap(MkField("U", userData()))
		assert.NotPanics(t, func() {
			_, _, found := GetShapeLocation(s, data, "U[*].DoesNotExist")
			assert.False(t, found)
		})
	})
}
