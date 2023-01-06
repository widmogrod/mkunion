package schema

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSchemaConversions(t *testing.T) {
	useCases := map[string]struct {
		in    any
		rules []RuleMatcher
		out   Schema
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
				assert.Equal(t, uc.in, SchemaToGo(got, uc.rules, nil))
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
	Baz string
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
				Foo:   1,
				Bar:   "baz",
				Other: &TestStruct2{Baz: "baz2"},
			},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, uc.out, SchemaToGo(uc.in, uc.rules, nil))
		})
	}
}
