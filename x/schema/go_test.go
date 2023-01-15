package schema

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGoToSchema(t *testing.T) {
	data := AStruct{
		Foo: 123,
		Bar: 333,
	}
	expected := &Map{
		Field: []Field{
			{
				Name:  "Foo",
				Value: MkInt(123),
			}, {
				Name:  "Bar",
				Value: MkInt(333),
			},
		},
	}
	schema := FromGo(data)

	assert.Equal(
		t,
		expected,
		schema,
	)
}

func TestGoToSchema2(t *testing.T) {
	data := AStruct{
		Foo: 123,
		Bar: 333,
	}
	expected := &Map{
		Field: []Field{
			{
				Name: "AStruct",
				Value: &Map{
					Field: []Field{
						{
							Name:  "Foo",
							Value: MkInt(123),
						}, {
							Name:  "Bar",
							Value: MkInt(333),
						},
					},
				},
			},
		},
	}
	schema := FromGo(data, WrapStruct(AStruct{}, "AStruct"))

	assert.Equal(
		t,
		expected,
		schema,
	)
}

func TestGoToSchema3(t *testing.T) {
	data := BStruct{
		Foo: 123,
		Bars: []string{
			"bar",
			"baz",
		},
	}
	expected := &Map{
		Field: []Field{
			{
				Name: "BStruct",
				Value: &Map{
					Field: []Field{
						{
							Name:  "Foo",
							Value: MkInt(123),
						}, {
							Name: "Bars",
							Value: &List{
								Items: []Schema{
									MkString("bar"),
									MkString("baz"),
								},
							},
						},
					},
				},
			},
		},
	}
	schema := FromGo(data, WrapStruct(BStruct{}, "BStruct"))
	assert.Equal(t, expected, schema)

	result := ToGo(schema, UnwrapStruct(BStruct{}, "BStruct"))
	assert.Equal(t, data, result)
}
