package testutil

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
	"reflect"
	"testing"
)

func TestToGo_ExampleOne(t *testing.T) {
	inferred, err := shape.InferFromFile("nested_recursive_model.go")
	if err != nil {
		t.Fatal(err)
	}

	exampleOne := inferred.RetrieveStruct("ExampleOne")
	assert.NotNil(t, exampleOne)

	subject := ExampleOne{
		OneValue: "hello",
	}

	result := schema.FromGoReflect(exampleOne, reflect.ValueOf(subject))
	assert.Equal(t,
		schema.MkMap(
			schema.MkField(
				"OneValue", schema.MkString("hello"),
			),
		),
		result,
	)

	output, err := schema.ToGoReflect(exampleOne, result, reflect.TypeOf(ExampleOne{}))
	assert.NoError(t, err)

	assert.Equal(t, subject, output.Interface())
}

func TestToGo_ExampleTwo(t *testing.T) {
	inferred, err := shape.InferFromFile("nested_recursive_model.go")
	if err != nil {
		t.Fatal(err)
	}

	exampleTwo := inferred.RetrieveStruct("ExampleTwo")
	assert.NotNil(t, exampleTwo)

	subject := ExampleTwo{
		TwoData: schema.MkBinary([]byte("world")),
		TwoNext: &ExampleOne{
			OneValue: "hello",
		},
	}

	result := schema.FromGoReflect(exampleTwo, reflect.ValueOf(subject))

	expected := schema.MkMap(
		schema.MkField(
			"TwoData",
			schema.MkMap(
				schema.MkField("$type", schema.MkString("schema.Binary")),
				schema.MkField(
					"schema.Binary", schema.MkBinary([]byte("world")),
				),
			),
		),
		schema.MkField(
			"TwoNext",
			schema.MkMap(
				schema.MkField("$type", schema.MkString("testutil.ExampleOne")),
				schema.MkField(
					"testutil.ExampleOne", schema.MkMap(
						schema.MkField(
							"OneValue", schema.MkString("hello"),
						),
					),
				),
			),
		),
	)

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Fatal(diff)
	}

	output, err := schema.ToGoReflect(exampleTwo, result, reflect.TypeOf(ExampleTwo{}))
	if assert.NoError(t, err) {
		assert.Equal(t, subject, output.Interface())
	}
}

func Test_GetShapeLocation(t *testing.T) {
	inferred, err := shape.InferFromFile("nested_recursive_model.go")
	if err != nil {
		t.Fatal(err)
	}

	exampleTwo := inferred.RetrieveStruct("ExampleTwo")
	assert.NotNil(t, exampleTwo)

	subject := ExampleTwo{
		TwoData: schema.MkInt(1),
		TwoNext: &ExampleOne{
			OneValue: "hello",
		},
	}

	result, resultShape, found := schema.Get(subject, "TwoNext[*].OneValue")

	assert.Equal(t, schema.MkString("hello"), result)
	assert.Equal(t, &shape.PrimitiveLike{Kind: &shape.StringLike{}}, resultShape)
	assert.True(t, found)
}

func Test_GetShapeLocation_Complex(t *testing.T) {
	subject := ExampleChange[Example]{
		After: ExampleRecord[Example]{
			Data: &ExampleTwo{
				TwoData: schema.MkInt(1),
				TwoNext: &ExampleOne{
					OneValue: "hello",
				},
			},
		},
	}

	result, resultShape, found := schema.Get[ExampleChange[Example]](subject, `After.Data[*].TwoNext[*].OneValue`)
	assert.Equal(t, schema.MkString("hello"), result)
	assert.Equal(t, &shape.PrimitiveLike{Kind: &shape.StringLike{}}, resultShape)
	assert.True(t, found)
}

func Test_NewTypedLocation(t *testing.T) {
	val := &ExampleTwo{
		TwoData: schema.MkInt(1),
		TwoNext: &ExampleOne{
			OneValue: "hello",
		},
	}

	subject := ExampleChange[Example]{
		After: ExampleRecord[Example]{
			Data: &ExampleTree{
				Items:   []Example{val},
				Schemas: nil,
				Map: map[string]Example{
					"mykey": val,
				},
				Any:    &val,
				Alias1: false,
				Alias2: 100,
			},
		},
	}

	loc, err := schema.NewTypedLocation[ExampleChange[Example]]()
	assert.NoError(t, err)

	sc := loc.ShapeDef()
	_ = sc

	data := schema.FromGo[ExampleChange[Example]](subject)

	useCases := []struct {
		found    bool
		location string
	}{
		{found: true, location: "After"},
		{found: true, location: "After.Data"},
		{found: true, location: `After.Data["testutil.ExampleTree"]`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Items`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Items[0]`},
		{found: false, location: `After.Data["testutil.ExampleTree"].Items[10000]`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Items[0]["testutil.ExampleTwo"]`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Items[0]["testutil.ExampleTwo"].TwoNext`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Items[0]["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"]`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Items[0]["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"].OneValue`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Map`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Map["mykey"]["testutil.ExampleTwo"]`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Map["mykey"]["testutil.ExampleTwo"].TwoNext`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Map["mykey"]["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"]`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Map["mykey"]["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"].OneValue`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Any`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Alias1`},
		{found: true, location: `After.Data["testutil.ExampleTree"].Alias2`},
	}
	for _, uc := range useCases {
		t.Run(uc.location, func(t *testing.T) {
			loc2, err := schema.ParseLocation(uc.location)
			assert.NoError(t, err)

			t.Log(schema.LocationToStr(loc2))

			locWrap, err := loc.WrapLocation(loc2)
			assert.NoError(t, err)

			finnalLoc := schema.LocationToStr(locWrap)
			t.Log(finnalLoc)

			sch, sh, found := schema.GetShapeSchemaLocation(loc.ShapeDef(), data, locWrap, false)

			result, _, _ := schema.Get[ExampleChange[Example]](subject, finnalLoc)
			t.Logf("%+#v", result)

			_ = sch
			_ = sh
			if assert.Equal(t, uc.found, found) {
				d, _ := shared.JSONMarshal(result)
				t.Log(string(d))
			}
		})
	}
}

func TestShapeLocationConversion(t *testing.T) {
	val := &ExampleTwo{
		TwoData: schema.MkInt(1),
		TwoNext: &ExampleOne{
			OneValue: "hello",
		},
	}

	intT := 1

	data := &ExampleTree{
		Items:   []Example{val},
		Schemas: nil,
		Map: map[string]Example{
			"mykey": val,
		},
		Any:    &val,
		Alias1: false,
		Alias2: 100,
		Ptr:    &intT,
	}

	recordExample := ExampleRecord[Example]{
		Data: data,
	}

	record2Example := ExampleRecord[*ExampleTree]{
		Data: data,
	}

	recordSchema := ExampleRecord[schema.Schema]{
		Data: schema.FromGo[Example](data),
	}

	recordSchema2 := ExampleRecord[schema.Schema]{
		Data: schema.FromGo[*ExampleTree](data),
	}

	locRec, err := schema.NewTypedLocation[ExampleRecord[Example]]()
	assert.NoError(t, err)

	locRec2, err := schema.NewTypedLocation[ExampleRecord[*ExampleTree]]()
	assert.NoError(t, err)

	locEncodedAsSch, err := schema.NewTypedLocation[schema.Schema]()
	assert.NoError(t, err)

	locEncodedAsRSS, err := schema.NewTypedLocation[ExampleRecord[schema.Schema]]()
	assert.NoError(t, err)

	useCases := []struct {
		found          bool
		data           schema.Schema
		encodedAs      shape.Shape
		typedLocation  *schema.TypedLocation
		givenLocation  string
		expectLocation string
	}{
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"]`,
			expectLocation: `Data["testutil.ExampleTree"]`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"]`,
			expectLocation: `Data["schema.Map"]["testutil.ExampleTree"]["schema.Map"]`,
		},
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Items[0]`,
			expectLocation: `Data["testutil.ExampleTree"].Items[0]`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Items[0]`,
			expectLocation: `Data["schema.Map"]["testutil.ExampleTree"]["schema.Map"].Items["schema.List"][0]["schema.Map"]`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Items[0]["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"].OneValue`,
			expectLocation: `Data["testutil.ExampleTree"].Items[0]["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"].OneValue`,
		},
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Items[0]["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"].OneValue`,
			expectLocation: `Data["schema.Map"]["testutil.ExampleTree"]["schema.Map"].Items["schema.List"][0]["schema.Map"]["testutil.ExampleTwo"]["schema.Map"].TwoNext["schema.Map"]["testutil.ExampleOne"]["schema.Map"].OneValue["schema.String"]`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Map.mykey["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"].OneValue`,
			expectLocation: `Data["testutil.ExampleTree"].Map.mykey["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"].OneValue`,
		},
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Map.mykey["testutil.ExampleTwo"].TwoNext["testutil.ExampleOne"].OneValue`,
			expectLocation: `Data["schema.Map"]["testutil.ExampleTree"]["schema.Map"].Map["schema.Map"].mykey["schema.Map"]["testutil.ExampleTwo"]["schema.Map"].TwoNext["schema.Map"]["testutil.ExampleOne"]["schema.Map"].OneValue["schema.String"]`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Any`,
			expectLocation: `Data["testutil.ExampleTree"].Any`,
		},
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Any`,
			expectLocation: `Data["schema.Map"]["testutil.ExampleTree"]["schema.Map"].Any`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Alias1`,
			expectLocation: `Data["testutil.ExampleTree"].Alias1`,
		},
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Alias1`,
			expectLocation: `Data["schema.Map"]["testutil.ExampleTree"]["schema.Map"].Alias1["schema.Bool"]`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Alias2`,
			expectLocation: `Data["testutil.ExampleTree"].Alias2`,
		},
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Alias2`,
			expectLocation: `Data["schema.Map"]["testutil.ExampleTree"]["schema.Map"].Alias2["schema.Number"]`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Ptr`,
			expectLocation: `Data["testutil.ExampleTree"].Ptr`,
		},
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Ptr`,
			expectLocation: `Data["schema.Map"]["testutil.ExampleTree"]["schema.Map"].Ptr["schema.Number"]`,
		},
		// separate case
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[*ExampleTree]](record2Example),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec2,
			givenLocation:  `Data.Ptr`,
			expectLocation: `Data.Ptr`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema2),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec2,
			givenLocation:  `Data.Ptr`,
			expectLocation: `Data["schema.Map"].Ptr["schema.Number"]`,
		},
		// $type first level
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["$type"]`,
			expectLocation: `Data["$type"]`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["$type"]`,
			expectLocation: `Data["schema.Map"]["$type"]["schema.String"]`,
		},
		// $type second level
		{
			found:          true,
			data:           schema.FromGo[ExampleRecord[Example]](recordExample),
			encodedAs:      locEncodedAsSch.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Items[0]["$type"]`,
			expectLocation: `Data["testutil.ExampleTree"].Items[0]["$type"]`,
		}, {
			found:          true,
			data:           schema.FromGo[ExampleRecord[schema.Schema]](recordSchema),
			encodedAs:      locEncodedAsRSS.ShapeDef(),
			typedLocation:  locRec,
			givenLocation:  `Data["testutil.ExampleTree"].Items[0]["$type"]`,
			expectLocation: `Data["schema.Map"]["testutil.ExampleTree"]["schema.Map"].Items["schema.List"][0]["schema.Map"]["$type"]["schema.String"]`,
		},
	}
	for _, uc := range useCases {
		t.Run(uc.givenLocation, func(t *testing.T) {
			loc, err := schema.ParseLocation(uc.givenLocation)
			assert.NoError(t, err)

			typedLocation, err := uc.typedLocation.WithEncodedAs(uc.encodedAs).WrapLocation(loc)
			//typedLocation, err := uc.typedLocation.WrapLocationEncodedAs(loc, uc.typedLocation.ShapeDef(), uc.encodedAs)
			if !assert.NoError(t, err) {
				return
			}

			typedLocationStr := schema.LocationToStr(typedLocation)
			t.Logf("typedLocation=%s", typedLocationStr)

			d1, found1 := schema.GetSchemaLocation(uc.data, typedLocation, false)
			d2, _ := schema.GetSchemaLocation(uc.data, schema.MustParseLocation(uc.expectLocation), false)

			t.Logf("d1=%v", d1)
			t.Logf("d2=%v", d2)
			t.Logf("found1=%v", found1)

			if !assert.Equal(t, uc.expectLocation, typedLocationStr) {
				return
			}

			assert.Equal(t, uc.found, found1)

			if diff := cmp.Diff(d1, d2); diff != "" {
				t.Fatalf("unexpected diff: %s", diff)
			}
		})
	}
}
