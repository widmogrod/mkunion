package shape

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInferFromFile(t *testing.T) {
	inferred, err := InferFromFile("testasset/type_example.go")
	if err != nil {
		t.Fatal(err)
	}

	union := inferred.RetrieveUnion("Example")
	assert.Equal(t, "Example", union.Name)
	assert.Equal(t, "testasset", union.PkgName)
	assert.Equal(t, "github.com/widmogrod/mkunion/x/shape/testasset", union.PkgImportName)

	expected := &UnionLike{
		Name:          "Example",
		PkgName:       "testasset",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
		Variant: []Shape{
			&StructLike{
				Name:          "A",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Fields: []*FieldLike{
					{
						Name:  "Name",
						Type:  &PrimitiveLike{Kind: &StringLike{}},
						Desc:  ptr("Name of the person"),
						Guard: nil,
						Tags: map[string]Tag{
							"json": {
								Value: "name",
							},
							"desc": {
								Value: "Name of the person",
							},
						},
					},
				},
			},
			&StructLike{
				Name:          "B",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Fields: []*FieldLike{
					{
						Name: "Age",
						Type: &PrimitiveLike{Kind: &NumberLike{
							Kind: &Int{},
						}},
						Desc:  nil,
						Guard: nil,
						Tags: map[string]Tag{
							"json": {
								Value: "age",
							},
						},
					},
					{
						Name: "A",
						Type: &PointerLike{
							Type: &RefName{
								Name:          "A",
								PkgName:       "testasset",
								PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
							},
						},
						Desc:  nil,
						Guard: nil,
						Tags:  nil,
					},
					{
						Name: "T",
						Type: &PointerLike{
							Type: &RefName{
								Name:          "Time",
								PkgName:       "time",
								PkgImportName: "time",
							},
						},
						Desc:  nil,
						Guard: nil,
						Tags:  nil,
					},
				},
			},
			&AliasLike{
				Name:          "C",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type:          &PrimitiveLike{Kind: &StringLike{}},
			},
			&AliasLike{
				Name:          "D",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type: &PrimitiveLike{Kind: &NumberLike{
					Kind: &Int64{},
				}},
			},
			&AliasLike{
				Name:          "E",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type: &PrimitiveLike{Kind: &NumberLike{
					Kind: &Float64{},
				}},
			},
			&AliasLike{
				Name:          "F",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type:          &PrimitiveLike{Kind: &BooleanLike{}},
			},

			&AliasLike{
				Name:          "H",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &MapLike{
					Key: &PrimitiveLike{Kind: &StringLike{}},
					Val: &RefName{
						Name:          "Example",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					},
				},
			},

			&AliasLike{
				Name:          "I",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &ListLike{
					Element: &RefName{
						Name:          "Example",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					},
				},
			},
			&AliasLike{
				Name:          "J",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &ListLike{
					Element:  &PrimitiveLike{Kind: &StringLike{}},
					ArrayLen: ptr(2),
				},
			},
			&AliasLike{
				Name:          "K",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &RefName{
					Name:          "A",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&AliasLike{
				Name:          "L",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       true,
				Type: &RefName{
					Name:          "List",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&AliasLike{
				Name:          "M",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &RefName{
					Name:          "List",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},

				Tags: map[string]Tag{
					"json": {
						Value: "m_list",
						Options: []string{
							"omitempty",
						},
					},
				},
			},
			&AliasLike{
				Name:          "N",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &RefName{
					Name:          "Duration",
					PkgName:       "time",
					PkgImportName: "time",
				},
			},
			&AliasLike{
				Name:          "O",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &RefName{
					Name:          "ListOf",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					Indexed: []Shape{
						&RefName{
							Name:          "Duration",
							PkgName:       "time",
							PkgImportName: "time",
						},
					},
				},
			},
			&AliasLike{
				Name:          "P",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &RefName{
					Name:          "ListOf2",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					Indexed: []Shape{
						&RefName{
							Name:          "ListOf",
							PkgName:       "testasset",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
							Indexed: []Shape{
								&Any{},
							},
						},
						&PointerLike{
							&RefName{
								Name:          "ListOf2",
								PkgName:       "testasset",
								PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
								Indexed: []Shape{
									&PrimitiveLike{
										Kind: &NumberLike{
											Kind: &Int64{},
										},
									},
									&PointerLike{
										&RefName{
											Name:          "Duration",
											PkgName:       "time",
											PkgImportName: "time",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Tags: map[string]Tag{
			"mkunion": {
				Value: "Example",
			},
		},
	}

	if diff := cmp.Diff(expected, union); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	strut := inferred.RetrieveStruct("ListOf2")
	expected2 := &StructLike{
		Name:          "ListOf2",
		PkgName:       "testasset",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
		TypeParams: []TypeParam{
			{
				Name: "T1",
				Type: &Any{},
			},
			{
				Name: "T2",
				Type: &Any{},
			},
		},
		Fields: []*FieldLike{
			{
				Name: "Data",
				Type: &RefName{
					Name:          "T1",
					PkgName:       "",
					PkgImportName: "",
				},
			},
			{
				Name: "ListOf",
				Type: &RefName{
					Name:          "ListOf",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					Indexed: []Shape{
						&RefName{
							Name:          "T1",
							PkgName:       "",
							PkgImportName: "",
						},
					},
				},
				Tags: map[string]Tag{
					"json": {
						Value: "list_of",
					},
				},
			},
		},
		Tags: map[string]Tag{
			"serde": {
				Value: "json",
			},
			"json": {
				Value: "list_of_2",
				Options: []string{
					"omitempty",
				},
			},
		},
	}
	if diff := cmp.Diff(expected2, strut); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	alias := inferred.RetrieveUnion("AliasExample")
	expected3 := &UnionLike{
		Name:          "AliasExample",
		PkgName:       "testasset",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
		Variant: []Shape{
			&AliasLike{
				Name:          "A2",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       true,
				Type: &RefName{
					Name:          "A",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&AliasLike{
				Name:          "B2",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       true,
				Type: &RefName{
					Name:          "B",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
		},
		Tags: map[string]Tag{
			"mkunion": {
				Value: "AliasExample",
			},
		},
	}

	if diff := cmp.Diff(expected3, alias); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}

	instantiations := inferred.FindInstantiationsOf(&RefName{
		Name:          "ListOf2",
		PkgName:       "testasset",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
	})
	expected4 := []*RefName{
		// ListOf2[*float64, *ListOf2[*A2, *time.Ticker]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&PointerLike{
					Type: &RefName{
						Name:          "A2",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					},
				},
				&PointerLike{
					Type: &RefName{
						Name:          "Ticker",
						PkgName:       "time",
						PkgImportName: "time",
					},
				},
			},
		},
		// ListOf2[*B2, time.Month]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&PointerLike{
					Type: &RefName{
						Name:          "B2",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					},
				},
				&RefName{
					Name:          "Month",
					PkgName:       "time",
					PkgImportName: "time",
				},
			},
		},
		// ListOf2[*F, float64]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&PointerLike{
					Type: &RefName{
						Name:          "F",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					},
				},
				&PrimitiveLike{
					Kind: &NumberLike{
						Kind: &Float64{},
					},
				},
			},
		},
		// ListOf2[*K,time.Weekday]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&PointerLike{
					Type: &RefName{
						Name:          "K",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					},
				},
				&RefName{
					Name:          "Weekday",
					PkgName:       "time",
					PkgImportName: "time",
				},
			},
		},
		// ListOf2[*L, time.Location]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&PointerLike{Type: &RefName{
					Name:          "L",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				}},
				&RefName{
					Name:          "Location",
					PkgName:       "time",
					PkgImportName: "time",
				},
			},
		},
		// ListOf2[*O,time.Location]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&PointerLike{
					Type: &RefName{
						Name:          "O",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					},
				},
				&RefName{
					Name:          "Location",
					PkgName:       "time",
					PkgImportName: "time",
				},
			},
		},
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&PointerLike{
					Type: &PrimitiveLike{
						Kind: &NumberLike{
							Kind: &Float64{},
						},
					},
				},
				&PointerLike{
					Type: &RefName{
						Name:          "ListOf2",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
						Indexed: []Shape{
							&PointerLike{
								&RefName{
									Name:          "A2",
									PkgName:       "testasset",
									PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
								},
							},
							&PointerLike{
								&RefName{
									Name:          "Ticker",
									PkgName:       "time",
									PkgImportName: "time",
								},
							},
						},
					},
				},
			},
		},
		// ListOf2[Example,*time.Time]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&RefName{
					Name:          "Example",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
				&PointerLike{
					Type: &RefName{
						Name:          "Time",
						PkgName:       "time",
						PkgImportName: "time",
					},
				},
			},
		},
		// ListOf2[ListOf[*bool],*ListOf2[Example,*time.Time]]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&RefName{
					Name:          "ListOf",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					Indexed: []Shape{
						&PointerLike{
							Type: &PrimitiveLike{Kind: &BooleanLike{}},
						},
					},
				},
				&PointerLike{
					Type: &RefName{
						Name:          "ListOf2",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
						Indexed: []Shape{
							&RefName{
								Name:          "Example",
								PkgName:       "testasset",
								PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
							},
							&PointerLike{
								Type: &RefName{
									Name:          "Time",
									PkgName:       "time",
									PkgImportName: "time",
								},
							},
						},
					},
				},
			},
		},
		// ListOf2[ListOf[any],*ListOf2[int64,*time.Duration]]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&RefName{
					Name:          "ListOf",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					Indexed: []Shape{
						&Any{},
					},
				},
				&PointerLike{
					Type: &RefName{
						Name:          "ListOf2",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
						Indexed: []Shape{
							&PrimitiveLike{Kind: &NumberLike{Kind: &Int64{}}},
							&PointerLike{
								Type: &RefName{
									Name:          "Duration",
									PkgName:       "time",
									PkgImportName: "time",
								},
							},
						},
					},
				},
			},
		},
		// ListOf2[int64,*time.Duration]
		{
			Name:          "ListOf2",
			PkgName:       "testasset",
			PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			Indexed: []Shape{
				&PrimitiveLike{
					Kind: &NumberLike{
						Kind: &Int64{},
					},
				},
				&PointerLike{
					Type: &RefName{
						Name:          "Duration",
						PkgName:       "time",
						PkgImportName: "time",
					},
				},
			},
		},
	}

	if diff := cmp.Diff(expected4, instantiations); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
