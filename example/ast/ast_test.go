package ast

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestAstJsonConversionOnSimpleType(t *testing.T) {
	in := &Lit{Value: float64(12)}

	s := schema.FromGo(in)

	out := &schema.Map{
		Field: []schema.Field{
			{
				Name: "Lit",
				Value: &schema.Map{
					Field: []schema.Field{
						{
							Name:  "Value",
							Value: schema.MkInt(12),
						},
					},
				},
			},
		},
	}
	assert.Equal(t, out, s)

	var data = schema.MustToGo(s)
	assert.Equal(t, in, data)

	jsonData, err := schema.ToJSON(s)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"Lit":{"Value":12}}`, string(jsonData))
}
func TestAstToJSONOnSumTypes(t *testing.T) {
	in := &Eq{
		L: &Accessor{[]string{"foo"}},
		R: &Lit{"baz"},
	}

	s := schema.FromGo(in, schema.WithOnlyTheseRules(
		CustomValueSchemaDef(),
		CustomOperationSchemaDef(),
	))

	out := &schema.Map{
		Field: []schema.Field{
			{
				Name: "Eq",
				Value: &schema.Map{
					Field: []schema.Field{
						{
							Name: "L",
							Value: &schema.Map{
								Field: []schema.Field{
									{
										Name: "Accessor",
										Value: &schema.Map{
											Field: []schema.Field{
												{
													Name: "Path",
													Value: &schema.List{
														Items: []schema.Schema{
															schema.MkString("foo"),
														},
													},
												},
											},
										},
									},
								},
							},
						},
						{
							Name: "R",
							Value: &schema.Map{
								Field: []schema.Field{
									{
										Name: "Lit",
										Value: &schema.Map{
											Field: []schema.Field{
												{
													Name:  "Value",
													Value: schema.MkString("baz"),
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

	assert.Equal(t, out, s, "schema transformation")

	var data = schema.MustToGo(s, schema.WithOnlyTheseRules(
		CustomValueSchemaDef(),
		CustomOperationSchemaDef(),
	))
	assert.Equal(t, in, data, "back to original golang structs")

	jsonData, err := schema.ToJSON(s)
	assert.NoError(t, err)
	t.Log(string(jsonData))
	assert.JSONEq(t, `{"Eq":{"L":{"Accessor":{"Path":["foo"]}},"R":{"Lit":{"Value":"baz"}}}}`, string(jsonData))
}

func TestASTSchema(t *testing.T) {
	v := &And{
		List: []Operator{
			&Eq{
				L: &Accessor{[]string{"foo"}},
				R: &Lit{"baz"},
			},
		},
	}
	s := schema.FromGo(v)
	result := schema.MustToGo(s)
	assert.Equal(t, v, result)
}
