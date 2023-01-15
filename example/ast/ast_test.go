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

	var data = schema.ToGo(s)
	assert.Equal(t, in, data)
}
func TestAstToJSONOnSumTypes(t *testing.T) {
	in := &Eq{
		L: &Accessor{[]string{"foo"}},
		R: &Lit{"baz"},
	}

	s := schema.FromGo(in)

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

	var data = schema.ToGo(s)
	assert.Equal(t, in, data, "back to original golang structs")
}
