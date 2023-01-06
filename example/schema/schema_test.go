package schema

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type AStruct struct {
	Foo float64 `json:"foo"`
	Bar float64 `json:"bar"`
}

func TestJsonToSchema(t *testing.T) {
	json := []byte(`{"Foo": 1, "Bar": 2}`)
	schema, err := JsonToSchema(json)
	assert.NoError(t, err)

	gonative := SchemaToGo(schema)
	assert.Equal(t, map[string]interface{}{
		"Foo": float64(1),
		"Bar": float64(2),
	}, gonative)

	gostruct := SchemaToGo(
		schema,
		WhenPath([]string{}, UseStruct(AStruct{})),
	)
	assert.Equal(t, AStruct{
		Foo: 1,
		Bar: 2,
	}, gostruct)
}

func TestSchemaConversions(t *testing.T) {
	useCases := map[string]struct {
		in  any
		out Schema
	}{
		"go list to schema and back": {
			in: []interface{}{1, 2, 3},
			out: &List{
				Items: []Schema{
					&Value{V: 1},
					&Value{V: 2},
					&Value{V: 3},
				},
			},
		},
		"go map to schema and back": {
			in: map[string]interface{}{
				"foo": 1,
				"bar": 2,
			},
			out: &Map{
				Field: []Field{
					{
						Name:  "foo",
						Value: &Value{V: 1},
					},
					{
						Name:  "bar",
						Value: &Value{V: 2},
					},
				},
			},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got := GoToSchema(uc.in)
			if assert.Equal(t, uc.out, got) {
				assert.Equal(t, uc.in, SchemaToGo(got))
			}
		})
	}
}

type TestStruct1 struct {
	Foo   int
	Bar   string
	Other SharedStruct
}

type TestStruct2 struct {
	Baz   string
	Count int
}

type SharedStruct interface {
	shared()
}

var (
	_ SharedStruct = (*TestStruct1)(nil)
	_ SharedStruct = (*TestStruct2)(nil)
)

func (t *TestStruct1) shared() {}
func (t *TestStruct2) shared() {}

func TestSchemaToGoStructs(t *testing.T) {
	useCases := map[string]struct {
		in    Schema
		rules []RuleMatcher
		out   interface{}
	}{
		"schema struct to go struct": {
			in: &Map{
				Field: []Field{
					{
						Name:  "Foo",
						Value: &Value{V: 1},
					},
					{
						Name:  "Bar",
						Value: &Value{V: "baz"},
					},
				},
			},
			rules: []RuleMatcher{
				WhenPath([]string{}, UseStruct(TestStruct1{})),
			},
			out: TestStruct1{
				Foo: 1,
				Bar: "baz",
			},
		},
		"schema with list of structs": {
			in: &List{
				Items: []Schema{
					&Map{
						Field: []Field{
							{
								Name:  "Foo",
								Value: &Value{V: 1},
							},
							{
								Name:  "Bar",
								Value: &Value{V: "baz"},
							},
						},
					},
					&Map{
						Field: []Field{
							{
								Name:  "Foo",
								Value: &Value{V: 13},
							},
							{
								Name:  "Bar",
								Value: &Value{V: "baz3"},
							},
						},
					},
				},
			},
			rules: []RuleMatcher{
				WhenPath([]string{"[*]"}, UseStruct(TestStruct1{})),
			},
			out: []any{
				TestStruct1{Foo: 1, Bar: "baz"},
				TestStruct1{Foo: 13, Bar: "baz3"},
			},
		},
		"struct with nested struct ": {
			in: &Map{
				Field: []Field{
					{
						Name:  "Foo",
						Value: &Value{V: 1},
					},
					{
						Name:  "Bar",
						Value: &Value{V: "baz"},
					}, {
						Name: "Other",
						Value: &Map{
							Field: []Field{
								{
									Name:  "Count",
									Value: &Value{V: 41},
								},
								{
									Name:  "Baz",
									Value: &Value{V: "baz2"},
								},
							},
						},
					},
				},
			},
			rules: []RuleMatcher{
				WhenPath([]string{}, UseStruct(TestStruct1{})),
				WhenPath([]string{"Other"}, UseStruct(&TestStruct2{})),
			},
			out: TestStruct1{
				Foo: 1,
				Bar: "baz",
				Other: &TestStruct2{
					Baz:   "baz2",
					Count: 41,
				},
			},
		},
		"schema with list of structs with nested struct": {
			in: &List{
				Items: []Schema{
					&Map{
						Field: []Field{
							{
								Name:  "Foo",
								Value: &Value{V: 1},
							},
							{
								Name:  "Bar",
								Value: &Value{V: "baz"},
							},
							{
								Name: "Other",
								Value: &Map{
									Field: []Field{
										{
											Name:  "Baz",
											Value: &Value{V: "baz2"},
										},
									},
								},
							},
						},
					},
					&Map{
						Field: []Field{
							{
								Name:  "Foo",
								Value: &Value{V: 55},
							},
							{
								Name:  "Bar",
								Value: &Value{V: "baz55"},
							},
							{
								Name: "Other",
								Value: &Map{
									Field: []Field{
										{
											Name:  "Foo",
											Value: &Value{V: 66},
										},
										{
											Name:  "Bar",
											Value: &Value{V: "baz66"},
										},
									},
								},
							},
						},
					},
				},
			},
			rules: []RuleMatcher{
				WhenPath([]string{"[*]"}, UseStruct(TestStruct1{})),
				WhenPath([]string{"[*]", "Other?.Foo"}, UseStruct(&TestStruct1{})),
				WhenPath([]string{"[*]", "Other?.Baz"}, UseStruct(&TestStruct2{})),
			},
			out: []any{
				TestStruct1{
					Foo: 1,
					Bar: "baz",
					Other: &TestStruct2{
						Baz: "baz2",
					},
				},
				TestStruct1{
					Foo: 55,
					Bar: "baz55",
					Other: &TestStruct1{
						Foo: 66,
						Bar: "baz66",
					},
				},
			},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.out, SchemaToGo(uc.in, uc.rules...))
			//gonative := SchemaToGo(uc.in)
			//data, err := json.Marshal(gonative)
			//assert.NoError(t, err)

			//fromJSON, err := JsonToSchema(data)
			//assert.NoError(t, err)
			//assert.Equal(t, uc.out, SchemaToGo(fromJSON, uc.rules...))
		})
	}
}
