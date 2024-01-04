package generators

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestShapeTagged_Struct(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/tree.go")
	if err != nil {
		t.Fatal(err)
	}

	generator := NewShapeTagged(
		inferred.RetrieveShapeNamedAs("ListOf2"),
	)

	result, err := generator.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `package testutils

import (
	"github.com/widmogrod/mkunion/x/shape"
)

func init() {
	shape.Register(ListOf2Shape())
}

func ListOf2Shape() shape.Shape {
	return &shape.StructLike{
		Name: "ListOf2",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		TypeParams: []shape.TypeParam{
			shape.TypeParam{
				Name: "T1",
				Type: &shape.Any{},
			},
			shape.TypeParam{
				Name: "T2",
				Type: &shape.Any{},
			},
		},
		Fields: []*shape.FieldLike{
			{
				Name: "ID",
				Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}},
			},
			{
				Name: "Data",
				Type: &shape.RefName{
					Name: "T1",
					PkgName: "",
					PkgImportName: "",
				},
			},
			{
				Name: "List",
				Type: &shape.ListLike{
					Element: &shape.RefName{
						Name: "T2",
						PkgName: "",
						PkgImportName: "",
					},
				},
			},
			{
				Name: "Map",
				Type: &shape.MapLike{
					Key: &shape.RefName{
						Name: "T1",
						PkgName: "",
						PkgImportName: "",
					},
					Val: &shape.RefName{
						Name: "T2",
						PkgName: "",
						PkgImportName: "",
					},
				},
				Tags: map[string]shape.Tag{
					"json": {
						Value: "map_of_tree",
					},
				},
			},
			{
				Name: "ListOf",
				Type: &shape.RefName{
					Name: "ListOf",
					PkgName: "testutils",
					PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
					Indexed: []shape.Shape{
						&shape.RefName{
							Name: "T1",
							PkgName: "",
							PkgImportName: "",
						},
					},
				},
			},
			{
				Name: "ListOfPtr",
				Type: &shape.PointerLike{
					Type: &shape.RefName{
						Name: "ListOf",
						PkgName: "testutils",
						PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
						Indexed: []shape.Shape{
							&shape.RefName{
								Name: "T2",
								PkgName: "",
								PkgImportName: "",
							},
						},
					},
				},
			},
			{
				Name: "Time",
				Type: &shape.RefName{
					Name: "Time",
					PkgName: "time",
					PkgImportName: "time",
				},
			},
			{
				Name: "Value",
				Type: &shape.RefName{
					Name: "Schema",
					PkgName: "schema",
					PkgImportName: "github.com/widmogrod/mkunion/x/schema",
				},
			},
		},
		Tags: map[string]shape.Tag{
			"serde": {
				Value: "json",
			},
		},
	}
}
`, result)
}
