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
	schema := GoToSchema(data)

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
	schema := GoToSchema(data, WhenStruct[AStruct]("AStruct"))

	assert.Equal(
		t,
		expected,
		schema,
	)
}
