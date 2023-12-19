package testutil

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
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
		TwoData: schema.MkInt(1),
		TwoNext: &ExampleOne{
			OneValue: "hello",
		},
	}

	result := schema.FromGoReflect(exampleTwo, reflect.ValueOf(subject))

	expected := schema.MkMap(
		schema.MkField(
			"TwoData",
			schema.MkMap(
				schema.MkField("$type", schema.MkString("schema.Number")),
				schema.MkField(
					"schema.Number", schema.MkInt(1),
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

	result, resultShape := schema.Get(subject, "TwoNext[*].OneValue")

	assert.Equal(t, schema.MkString("hello"), result)
	assert.Equal(t, &shape.StringLike{}, resultShape)
}
