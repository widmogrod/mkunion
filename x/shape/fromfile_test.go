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
						Name:      "Name",
						Type:      &StringLike{},
						Desc:      ptr("Name of the person"),
						Guard:     nil,
						IsPointer: false,
						Tags: map[string]FieldTag{
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
						Name:      "Age",
						Type:      &NumberLike{},
						Desc:      nil,
						Guard:     nil,
						IsPointer: false,
						Tags: map[string]FieldTag{
							"json": {
								Value: "age",
							},
						},
					},
					{
						Name: "A",
						Type: &RefName{
							Name:          "A",
							PkgName:       "testasset",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
							IsPointer:     true,
						},
						Desc:      nil,
						Guard:     nil,
						IsPointer: true,
						Tags:      nil,
					},
					{
						Name: "T",
						Type: &RefName{
							Name:          "Time",
							PkgName:       "time",
							PkgImportName: "time",
							IsPointer:     true,
						},
						Desc:      nil,
						Guard:     nil,
						IsPointer: true,
						Tags:      nil,
					},
				},
			},
			&AliasLike{
				Name:          "C",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type:          &StringLike{},
			},
			&AliasLike{
				Name:          "D",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type: &NumberLike{
					Kind: &Int64{},
				},
			},
			&AliasLike{
				Name:          "E",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type: &NumberLike{
					Kind: &Float64{},
				},
			},
			&AliasLike{
				Name:          "F",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type:          &BooleanLike{},
			},

			&AliasLike{
				Name:          "H",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &MapLike{
					Key: &StringLike{},
					Val: &RefName{
						Name:          "Example",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					},
					KeyIsPointer: false,
					ValIsPointer: false,
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
					ElementIsPointer: false,
				},
			},
			&AliasLike{
				Name:          "J",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				IsAlias:       false,
				Type: &ListLike{
					Element:          &StringLike{},
					ElementIsPointer: false,
					ArrayLen:         ptr(2),
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
					IsPointer:     false,
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
					IsPointer:     false,
					Indexed: []Shape{
						&RefName{
							Name:          "ListOf",
							PkgName:       "testasset",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
							IsPointer:     false,
							Indexed: []Shape{
								&Any{},
							},
						},
						&RefName{
							Name:          "ListOf2",
							PkgName:       "testasset",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
							IsPointer:     true,
							Indexed: []Shape{
								&NumberLike{
									Kind: &Int64{},
								},
								&RefName{
									Name:          "Duration",
									PkgName:       "time",
									PkgImportName: "time",
									IsPointer:     true,
								},
							},
						},
					},
				},
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
		Fields: nil,
	}
	if diff := cmp.Diff(expected2, strut); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}

//func TestEmbeds(t *testing.T) {
//	inferred, err := InferFromFile("testasset/type_embeds.go")
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	union := inferred.RetrieveUnion("EmbedsDSL")
//	assert.Equal(t, "EmbedsDSL", union.Name)
//	assert.Equal(t, "testasset", union.PkgName)
//	assert.Equal(t, "github.com/widmogrod/mkunion/x/shape/testasset", union.PkgImportName)
//	assert.Equal(t, 10, len(union.Variant))
//
//	expected := &UnionLike{
//		Name:          "EmbedsDSL",
//		PkgName:       "testasset",
//		PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
//	}
//
//	if diff := cmp.Diff(expected, &union); diff != "" {
//		t.Errorf("mismatch (-want +got):\n%s", diff)
//	}
//}
