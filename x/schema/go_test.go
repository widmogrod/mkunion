package schema

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGoToSchemaTypeDef(t *testing.T) {
	expected := AStruct{
		Foo: 123,
		Bar: 333,
	}

	schema := &Map{
		TypeDef: NewStructDef[AStruct](),
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
	result := MustToGo(schema)

	assert.Equal(
		t,
		expected,
		result,
	)
}

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
		Taz: map[string]string{
			"taz1": "taz2",
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
						}, {
							Name: "Taz",
							Value: &Map{
								Field: []Field{
									{
										Name:  "taz1",
										Value: MkString("taz2"),
									},
								},
							},
						}, {
							Name:  "BaseStruct",
							Value: &None{},
						},
						{
							Name:  "S",
							Value: &None{},
						},
						{
							Name:  "List",
							Value: &List{},
						},
						{
							Name:  "Ma",
							Value: &Map{},
						},
					},
				},
			},
		},
	}
	schema := FromGo(data, WrapStruct(BStruct{}, "BStruct"))
	assert.Equal(t, expected, schema)

	result := MustToGo(schema, UnwrapStruct(BStruct{}, "BStruct"))
	assert.Equal(t, data, result)
}

func TestGoToSchema4(t *testing.T) {
	someStr := "some string"

	data := BStruct{
		Foo: 123,
		Bars: []string{
			"bar",
			"baz",
		},
		Taz: map[string]string{
			"taz1": "taz2",
		},
		BaseStruct: &BaseStruct{
			Age: 123,
		},
		S: &someStr,
		List: []AStruct{
			{
				Foo: 444,
			},
		},
		Ma: map[string]AStruct{
			"key": {
				Foo: 666,
				Bar: 555,
			},
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
						}, {
							Name: "Taz",
							Value: &Map{
								Field: []Field{
									{
										Name:  "taz1",
										Value: MkString("taz2"),
									},
								},
							},
						}, {
							Name: "BaseStruct",
							Value: &Map{
								Field: []Field{
									{
										Name:  "Age",
										Value: MkInt(123),
									},
								},
							},
						}, {
							Name:  "S",
							Value: MkString("some string"),
						},
						{
							Name: "List",
							Value: &List{
								Items: []Schema{
									&Map{
										Field: []Field{
											{
												Name:  "Foo",
												Value: MkInt(444),
											},
											{
												Name:  "Bar",
												Value: MkInt(0),
											},
										},
									},
								},
							},
						},
						{
							Name: "Ma",
							Value: &Map{
								Field: []Field{
									{
										Name: "key",
										Value: &Map{
											Field: []Field{
												{
													Name:  "Foo",
													Value: MkInt(666),
												},
												{
													Name:  "Bar",
													Value: MkInt(555),
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	schema := FromGo(data,
		WrapStruct(BStruct{}, "BStruct"),
	)
	assert.Equal(t, expected, schema)

	result := MustToGo(schema,
		UnwrapStruct(BStruct{}, "BStruct"),
		WhenPath([]string{"*", "BStruct", "BaseStruct"}, UseStruct(&BaseStruct{})),
		WhenPath([]string{"*", "BStruct", "List", "[*]"}, UseStruct(AStruct{})),
		WhenPath([]string{"*", "BStruct", "Ma", "key"}, UseStruct(AStruct{})),
	)
	assert.Equal(t, data, result)
}
