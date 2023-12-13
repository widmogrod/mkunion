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
	assert.Equal(t, 9, len(union.Variant))

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
		},
	}

	if diff := cmp.Diff(expected, &union); diff != "" {
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
