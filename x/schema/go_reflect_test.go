package schema

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shape"
)

func mkNumShape(kind shape.NumberKind) *shape.PrimitiveLike {
	return &shape.PrimitiveLike{Kind: &shape.NumberLike{Kind: kind}}
}

var boolShape = &shape.PrimitiveLike{Kind: &shape.BooleanLike{}}

// personShape mirrors personGo below; the field types carry explicit
// number kinds so both directions of the reflection conversion work.
type personGo struct {
	Name string
	Age  int64
}

func personShape() *shape.StructLike {
	return mkStructShape("Person",
		mkField2("Name", strShape),
		mkField2("Age", mkNumShape(&shape.Int64{})),
	)
}

func TestFromGoReflect_Primitives(t *testing.T) {
	useCases := map[string]struct {
		kind  shape.NumberKind
		value any
		want  Schema
	}{
		"uint":    {&shape.UInt{}, uint(7), MkUint(7)},
		"uint8":   {&shape.UInt8{}, uint8(7), MkUint(7)},
		"uint16":  {&shape.UInt16{}, uint16(7), MkUint(7)},
		"uint32":  {&shape.UInt32{}, uint32(7), MkUint(7)},
		"uint64":  {&shape.UInt64{}, uint64(7), MkUint(7)},
		"int":     {&shape.Int{}, int(-7), MkInt(-7)},
		"int8":    {&shape.Int8{}, int8(-7), MkInt(-7)},
		"int16":   {&shape.Int16{}, int16(-7), MkInt(-7)},
		"int32":   {&shape.Int32{}, int32(-7), MkInt(-7)},
		"int64":   {&shape.Int64{}, int64(-7), MkInt(-7)},
		"float32": {&shape.Float32{}, float32(1.5), MkFloat(1.5)},
		"float64": {&shape.Float64{}, float64(1.5), MkFloat(1.5)},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got := FromGoReflect(mkNumShape(uc.kind), reflect.ValueOf(uc.value))
			assert.Equal(t, uc.want, got)
		})
	}

	t.Run("bool", func(t *testing.T) {
		assert.Equal(t, MkBool(true), FromGoReflect(boolShape, reflect.ValueOf(true)))
	})
	t.Run("string", func(t *testing.T) {
		assert.Equal(t, MkString("x"), FromGoReflect(strShape, reflect.ValueOf("x")))
	})
	t.Run("bool shape with non-bool value panics", func(t *testing.T) {
		assert.Panics(t, func() {
			FromGoReflect(boolShape, reflect.ValueOf("not-bool"))
		})
	})
	t.Run("string shape with non-string value panics", func(t *testing.T) {
		assert.Panics(t, func() {
			FromGoReflect(strShape, reflect.ValueOf(1))
		})
	})
}

func TestFromGoReflect_Any(t *testing.T) {
	anyShape := &shape.Any{}
	useCases := map[string]struct {
		value any
		want  Schema
	}{
		"bool":   {true, MkBool(true)},
		"string": {"x", MkString("x")},
		"int":    {-5, MkInt(-5)},
		"uint":   {uint(5), MkUint(5)},
		"float":  {1.5, MkFloat(1.5)},
		"bytes":  {[]byte{1, 2}, MkBinary([]byte{1, 2})},
		"slice":  {[]int{1, 2}, MkList(MkInt(1), MkInt(2))},
		"map":    {map[string]int{"a": 1}, MkMap(MkField("a", MkInt(1)))},
		"struct": {struct{ A int }{A: 1}, MkMap(MkField("A", MkInt(1)))},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got := FromGoReflect(anyShape, reflect.ValueOf(uc.value))
			assert.Equal(t, uc.want, got)
		})
	}

	t.Run("nil pointer becomes none", func(t *testing.T) {
		var p *int
		assert.Equal(t, MkNone(), FromGoReflect(anyShape, reflect.ValueOf(&p).Elem()))
	})
	t.Run("pointer dereferences", func(t *testing.T) {
		v := 5
		assert.Equal(t, MkInt(5), FromGoReflect(anyShape, reflect.ValueOf(&v)))
	})
	t.Run("nil interface becomes none", func(t *testing.T) {
		var i any
		assert.Equal(t, MkNone(), FromGoReflect(anyShape, reflect.ValueOf(&i).Elem()))
	})
	t.Run("unsupported kind panics", func(t *testing.T) {
		assert.Panics(t, func() {
			FromGoReflect(anyShape, reflect.ValueOf(make(chan int)))
		})
	})
}

func TestFromGoReflect_Composites(t *testing.T) {
	t.Run("list of strings", func(t *testing.T) {
		got := FromGoReflect(&shape.ListLike{Element: strShape}, reflect.ValueOf([]string{"a", "b"}))
		assert.Equal(t, MkList(MkString("a"), MkString("b")), got)
	})
	t.Run("binary list shape short-circuits to bytes", func(t *testing.T) {
		s := &shape.ListLike{Element: mkNumShape(&shape.UInt8{})}
		got := FromGoReflect(s, reflect.ValueOf([]byte{1, 2}))
		assert.Equal(t, MkBinary([]byte{1, 2}), got)
	})
	t.Run("list shape with non-slice panics", func(t *testing.T) {
		assert.Panics(t, func() {
			FromGoReflect(&shape.ListLike{Element: strShape}, reflect.ValueOf("nope"))
		})
	})

	t.Run("map of numbers", func(t *testing.T) {
		s := &shape.MapLike{Val: mkNumShape(&shape.Int{})}
		got := FromGoReflect(s, reflect.ValueOf(map[string]int{"a": 1}))
		assert.Equal(t, MkMap(MkField("a", MkInt(1))), got)
	})
	t.Run("map shape with non-map panics", func(t *testing.T) {
		assert.Panics(t, func() {
			FromGoReflect(&shape.MapLike{Val: strShape}, reflect.ValueOf(1))
		})
	})

	t.Run("struct", func(t *testing.T) {
		got := FromGoReflect(personShape(), reflect.ValueOf(personGo{Name: "Ala", Age: 42}))
		assert.Equal(t, MkMap(
			MkField("Name", MkString("Ala")),
			MkField("Age", MkInt(42)),
		), got)
	})
	t.Run("struct through a pointer", func(t *testing.T) {
		got := FromGoReflect(personShape(), reflect.ValueOf(&personGo{Name: "Ala", Age: 1}))
		assert.Equal(t, MkMap(
			MkField("Name", MkString("Ala")),
			MkField("Age", MkInt(1)),
		), got)
	})
	t.Run("shape field missing on the Go struct is skipped", func(t *testing.T) {
		s := mkStructShape("Person",
			mkField2("Name", strShape),
			mkField2("Ghost", strShape),
		)
		got := FromGoReflect(s, reflect.ValueOf(personGo{Name: "Ala"}))
		assert.Equal(t, MkMap(MkField("Name", MkString("Ala"))), got)
	})
	t.Run("struct shape with non-struct panics", func(t *testing.T) {
		assert.Panics(t, func() {
			FromGoReflect(personShape(), reflect.ValueOf(1))
		})
	})

	t.Run("alias delegates to the aliased shape", func(t *testing.T) {
		alias := &shape.AliasLike{Name: "S", PkgName: "schematest", Type: strShape}
		assert.Equal(t, MkString("x"), FromGoReflect(alias, reflect.ValueOf("x")))
	})

	t.Run("nil pointer shape becomes none", func(t *testing.T) {
		var p *string
		got := FromGoReflect(&shape.PointerLike{Type: strShape}, reflect.ValueOf(p))
		assert.Equal(t, MkNone(), got)
	})
	t.Run("pointer shape dereferences", func(t *testing.T) {
		v := "x"
		got := FromGoReflect(&shape.PointerLike{Type: strShape}, reflect.ValueOf(&v))
		assert.Equal(t, MkString("x"), got)
	})
}

func TestFromGoReflect_RefName(t *testing.T) {
	t.Run("registered ref resolves through the registry", func(t *testing.T) {
		// RefTarget{Value string} is registered in utils_shape_location_test.go
		got := FromGoReflect(refTargetRef(), reflect.ValueOf(struct{ Value string }{Value: "v"}))
		assert.Equal(t, MkMap(MkField("Value", MkString("v"))), got)
	})
	t.Run("unregistered ref falls back to MarshalJSON", func(t *testing.T) {
		ts := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
		ref := &shape.RefName{Name: "Time", PkgName: "time", PkgImportName: "time"}

		expected, err := json.Marshal(ts)
		require.NoError(t, err)

		got := FromGoReflect(ref, reflect.ValueOf(ts))
		assert.Equal(t, MkString(string(expected)), got)
	})
	t.Run("unregistered ref without MarshalJSON panics", func(t *testing.T) {
		ref := &shape.RefName{Name: "Nope", PkgName: "schematest", PkgImportName: "example.com/schematest"}
		assert.Panics(t, func() {
			FromGoReflect(ref, reflect.ValueOf(struct{ A int }{A: 1}))
		})
	})
}

func TestFromGoReflect_Union(t *testing.T) {
	// Location is this package's own union, registered by generated code.
	locationUnion, found := shape.LookupShape(&shape.RefName{
		Name:          "Location",
		PkgName:       "schema",
		PkgImportName: "github.com/widmogrod/mkunion/x/schema",
	})
	require.True(t, found)

	t.Run("active variant serialises with $type", func(t *testing.T) {
		var l Location = &LocationField{Name: "x"}
		got := FromGoReflect(locationUnion, reflect.ValueOf(&l).Elem())
		assert.Equal(t, MkMap(
			MkField("$type", MkString("schema.LocationField")),
			MkField("schema.LocationField", MkMap(MkField("Name", MkString("x")))),
		), got)
	})
	t.Run("nil union becomes none", func(t *testing.T) {
		var l Location
		got := FromGoReflect(locationUnion, reflect.ValueOf(&l).Elem())
		assert.Equal(t, MkNone(), got)
	})
	t.Run("value that is no variant panics", func(t *testing.T) {
		var l Location = &LocationField{Name: "x"}
		assert.Panics(t, func() {
			FromGoReflect(unionShape(), reflect.ValueOf(&l).Elem())
		})
	})
}

func TestToGoReflect_Primitives(t *testing.T) {
	useCases := map[string]struct {
		kind shape.NumberKind
		want any
	}{
		"uint":    {&shape.UInt{}, uint(7)},
		"uint8":   {&shape.UInt8{}, uint8(7)},
		"uint16":  {&shape.UInt16{}, uint16(7)},
		"uint32":  {&shape.UInt32{}, uint32(7)},
		"uint64":  {&shape.UInt64{}, uint64(7)},
		"int":     {&shape.Int{}, int(7)},
		"int8":    {&shape.Int8{}, int8(7)},
		"int16":   {&shape.Int16{}, int16(7)},
		"int32":   {&shape.Int32{}, int32(7)},
		"int64":   {&shape.Int64{}, int64(7)},
		"float32": {&shape.Float32{}, float32(7)},
		"float64": {&shape.Float64{}, float64(7)},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got, err := ToGoReflect(mkNumShape(uc.kind), MkInt(7), reflect.TypeOf(uc.want))
			require.NoError(t, err)
			assert.Equal(t, uc.want, got.Interface())
		})
	}

	t.Run("nil number kind converts to the target type", func(t *testing.T) {
		s := &shape.PrimitiveLike{Kind: &shape.NumberLike{}}
		got, err := ToGoReflect(s, MkInt(7), reflect.TypeOf(int32(0)))
		require.NoError(t, err)
		assert.Equal(t, int32(7), got.Interface())
	})
	t.Run("bool", func(t *testing.T) {
		got, err := ToGoReflect(boolShape, MkBool(true), reflect.TypeOf(false))
		require.NoError(t, err)
		assert.Equal(t, true, got.Interface())
	})
	t.Run("string", func(t *testing.T) {
		got, err := ToGoReflect(strShape, MkString("x"), reflect.TypeOf(""))
		require.NoError(t, err)
		assert.Equal(t, "x", got.Interface())
	})

	t.Run("mismatched data yields errors, not panics", func(t *testing.T) {
		_, err := ToGoReflect(boolShape, MkString("x"), reflect.TypeOf(false))
		assert.Error(t, err)

		_, err = ToGoReflect(strShape, MkInt(1), reflect.TypeOf(""))
		assert.Error(t, err)

		_, err = ToGoReflect(mkNumShape(&shape.Int{}), MkString("1"), reflect.TypeOf(0))
		assert.Error(t, err)
	})

	t.Run("none becomes the zero value", func(t *testing.T) {
		got, err := ToGoReflect(strShape, MkNone(), reflect.TypeOf(""))
		require.NoError(t, err)
		assert.Equal(t, "", got.Interface())
	})

	t.Run("any shape is not implemented", func(t *testing.T) {
		assert.Panics(t, func() {
			_, _ = ToGoReflect(&shape.Any{}, MkInt(1), reflect.TypeOf(0))
		})
	})
}

func TestToGoReflect_List(t *testing.T) {
	listOfInt := &shape.ListLike{Element: mkNumShape(&shape.Int{})}

	t.Run("list of ints", func(t *testing.T) {
		got, err := ToGoReflect(listOfInt, MkList(MkInt(1), MkInt(2)), reflect.TypeOf([]int{}))
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, got.Interface())
	})
	t.Run("empty list is the zero slice", func(t *testing.T) {
		got, err := ToGoReflect(listOfInt, MkList(), reflect.TypeOf([]int{}))
		require.NoError(t, err)
		assert.Nil(t, got.Interface())
	})
	t.Run("binary data with a byte-list shape", func(t *testing.T) {
		s := &shape.ListLike{Element: mkNumShape(&shape.UInt8{})}
		got, err := ToGoReflect(s, MkBinary([]byte{1, 2}), reflect.TypeOf([]byte{}))
		require.NoError(t, err)
		// the value keeps the named type schema.Binary; it stays assignable
		// to []byte, which is what struct-field Set relies on
		assert.Equal(t, Binary{1, 2}, got.Interface())
		assert.True(t, got.Type().AssignableTo(reflect.TypeOf([]byte{})))
	})
	t.Run("binary data with a non-byte list shape errors", func(t *testing.T) {
		_, err := ToGoReflect(listOfInt, MkBinary([]byte{1}), reflect.TypeOf([]int{}))
		assert.Error(t, err)
	})
	t.Run("non-list data errors", func(t *testing.T) {
		_, err := ToGoReflect(listOfInt, MkInt(1), reflect.TypeOf([]int{}))
		assert.Error(t, err)
	})
	t.Run("non-slice target errors", func(t *testing.T) {
		_, err := ToGoReflect(listOfInt, MkList(MkInt(1)), reflect.TypeOf(0))
		assert.Error(t, err)
	})
	t.Run("element conversion error propagates", func(t *testing.T) {
		_, err := ToGoReflect(listOfInt, MkList(MkString("nope")), reflect.TypeOf([]int{}))
		assert.Error(t, err)
	})
}

func TestToGoReflect_Map(t *testing.T) {
	mapOfInt := &shape.MapLike{Val: mkNumShape(&shape.Int{})}

	t.Run("map of ints", func(t *testing.T) {
		got, err := ToGoReflect(mapOfInt, MkMap(MkField("a", MkInt(1))), reflect.TypeOf(map[string]int{}))
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"a": 1}, got.Interface())
	})
	t.Run("empty map is the zero map", func(t *testing.T) {
		got, err := ToGoReflect(mapOfInt, MkMap(), reflect.TypeOf(map[string]int{}))
		require.NoError(t, err)
		assert.Nil(t, got.Interface())
	})
	t.Run("pointer target is dereferenced", func(t *testing.T) {
		got, err := ToGoReflect(mapOfInt, MkMap(MkField("a", MkInt(1))), reflect.TypeOf(&map[string]int{}))
		require.NoError(t, err)
		assert.Equal(t, map[string]int{"a": 1}, got.Interface())
	})
	t.Run("non-map data errors", func(t *testing.T) {
		_, err := ToGoReflect(mapOfInt, MkInt(1), reflect.TypeOf(map[string]int{}))
		assert.Error(t, err)
	})
	t.Run("non-map target errors", func(t *testing.T) {
		_, err := ToGoReflect(mapOfInt, MkMap(MkField("a", MkInt(1))), reflect.TypeOf(0))
		assert.Error(t, err)
	})
	t.Run("value conversion error propagates", func(t *testing.T) {
		_, err := ToGoReflect(mapOfInt, MkMap(MkField("a", MkString("nope"))), reflect.TypeOf(map[string]int{}))
		assert.Error(t, err)
	})
}

func TestToGoReflect_Struct(t *testing.T) {
	t.Run("struct from a map", func(t *testing.T) {
		got, err := ToGoReflect(personShape(), MkMap(
			MkField("Name", MkString("Ala")),
			MkField("Age", MkInt(42)),
		), reflect.TypeOf(personGo{}))
		require.NoError(t, err)
		assert.Equal(t, personGo{Name: "Ala", Age: 42}, got.Interface())
	})
	t.Run("pointer target returns a pointer", func(t *testing.T) {
		got, err := ToGoReflect(personShape(), MkMap(
			MkField("Name", MkString("Ala")),
		), reflect.TypeOf(&personGo{}))
		require.NoError(t, err)
		assert.Equal(t, &personGo{Name: "Ala"}, got.Interface())
	})
	t.Run("missing data fields stay zero", func(t *testing.T) {
		got, err := ToGoReflect(personShape(), MkMap(), reflect.TypeOf(personGo{}))
		require.NoError(t, err)
		assert.Equal(t, personGo{}, got.Interface())
	})
	t.Run("shape field absent on the Go struct errors", func(t *testing.T) {
		s := mkStructShape("Person", mkField2("Ghost", strShape))
		_, err := ToGoReflect(s, MkMap(MkField("Ghost", MkString("boo"))), reflect.TypeOf(personGo{}))
		assert.Error(t, err)
	})
	t.Run("non-map data errors", func(t *testing.T) {
		_, err := ToGoReflect(personShape(), MkInt(1), reflect.TypeOf(personGo{}))
		assert.Error(t, err)
	})
	t.Run("non-struct target errors", func(t *testing.T) {
		_, err := ToGoReflect(personShape(), MkMap(), reflect.TypeOf(0))
		assert.Error(t, err)
	})
	t.Run("field conversion error propagates", func(t *testing.T) {
		_, err := ToGoReflect(personShape(), MkMap(
			MkField("Age", MkString("old")),
		), reflect.TypeOf(personGo{}))
		assert.Error(t, err)
	})
}

func TestToGoReflect_AliasRefUnion(t *testing.T) {
	t.Run("alias converts to the named type", func(t *testing.T) {
		type MyString string
		alias := &shape.AliasLike{Name: "MyString", PkgName: "schematest", Type: strShape}
		got, err := ToGoReflect(alias, MkString("x"), reflect.TypeOf(MyString("")))
		require.NoError(t, err)
		assert.Equal(t, MyString("x"), got.Interface())
	})
	t.Run("alias to a pointer target", func(t *testing.T) {
		type MyString string
		alias := &shape.AliasLike{Name: "MyString", PkgName: "schematest", Type: strShape}
		got, err := ToGoReflect(alias, MkString("x"), reflect.TypeOf((*MyString)(nil)))
		require.NoError(t, err)
		want := MyString("x")
		assert.Equal(t, &want, got.Interface())
	})
	t.Run("alias propagates inner errors", func(t *testing.T) {
		alias := &shape.AliasLike{Name: "MyString", PkgName: "schematest", Type: strShape}
		_, err := ToGoReflect(alias, MkInt(1), reflect.TypeOf(""))
		assert.Error(t, err)
	})

	t.Run("unregistered ref errors", func(t *testing.T) {
		ref := &shape.RefName{Name: "Nope", PkgName: "schematest", PkgImportName: "example.com/schematest"}
		_, err := ToGoReflect(ref, MkInt(1), reflect.TypeOf(0))
		assert.Error(t, err)
	})

	locationType := reflect.TypeOf((*Location)(nil)).Elem()
	locationUnion, found := shape.LookupShape(&shape.RefName{
		Name:          "Location",
		PkgName:       "schema",
		PkgImportName: "github.com/widmogrod/mkunion/x/schema",
	})
	require.True(t, found)

	t.Run("union roundtrip through the type registry", func(t *testing.T) {
		data := MkMap(
			MkField("$type", MkString("schema.LocationField")),
			MkField("schema.LocationField", MkMap(MkField("Name", MkString("x")))),
		)
		got, err := ToGoReflect(locationUnion, data, locationType)
		require.NoError(t, err)
		assert.Equal(t, &LocationField{Name: "x"}, got.Interface())
	})
	t.Run("union with non-map data errors", func(t *testing.T) {
		_, err := ToGoReflect(locationUnion, MkInt(1), locationType)
		assert.Error(t, err)
	})
	t.Run("union with non-interface target errors", func(t *testing.T) {
		_, err := ToGoReflect(locationUnion, MkMap(), reflect.TypeOf(0))
		assert.Error(t, err)
	})
	t.Run("union with no known variant errors", func(t *testing.T) {
		_, err := ToGoReflect(locationUnion, MkMap(MkField("junk", MkInt(1))), locationType)
		assert.Error(t, err)
	})
	t.Run("union variant missing from the type registry errors", func(t *testing.T) {
		data := MkMap(MkField("schematest.Circle", MkMap()))
		_, err := ToGoReflect(unionShape(), data, locationType)
		assert.Error(t, err)
	})
}

// FromGoReflect and ToGoReflect must invert each other for a union value.
func TestReflectRoundtrip(t *testing.T) {
	locationUnion, found := shape.LookupShape(&shape.RefName{
		Name:          "Location",
		PkgName:       "schema",
		PkgImportName: "github.com/widmogrod/mkunion/x/schema",
	})
	require.True(t, found)

	var original Location = &LocationIndex{Index: 3}
	data := FromGoReflect(locationUnion, reflect.ValueOf(&original).Elem())

	back, err := ToGoReflect(locationUnion, data, reflect.TypeOf((*Location)(nil)).Elem())
	require.NoError(t, err)
	assert.Equal(t, original, back.Interface())
}
