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
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"time"
)

func init() {
	shape.Register(ListOf2Shape())
}

func ListOf2Shape() shape.Shape {
	return &shape.StructLike{
		Name: "ListOf2",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		Fields: []*shape.FieldLike{
			{
				Name: "ID",
				Type: &shape.StringLike{},
			},
			{
				Name: "Data",
				Type: &shape.RefName{
					Name: "T1",
					PkgName: "",
					PkgImportName: "",
					IsPointer: false,
				},
			},
			{
				Name: "List",
				Type: &shape.ListLike{
					Element: &shape.RefName{
						Name: "T2",
						PkgName: "",
						PkgImportName: "",
						IsPointer: false,
					},
					ElementIsPointer: false,
				},
			},
			{
				Name: "Map",
				Type: &shape.MapLike{
					Key: &shape.RefName{
						Name: "T1",
						PkgName: "",
						PkgImportName: "",
						IsPointer: false,
					},
					KeyIsPointer: false,
					Val: &shape.RefName{
						Name: "T2",
						PkgName: "",
						PkgImportName: "",
						IsPointer: false,
					},
					ValIsPointer: false,
				},
			},
			{
				Name: "ListOf",
				Type: &shape.RefName{
					Name: "ListOf",
					PkgName: "testutils",
					PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
					IsPointer: false,
					Indexed: []shape.Shape{
						&shape.RefName{
							Name: "T1",
							PkgName: "",
							PkgImportName: "",
							IsPointer: false,
						},
					},
				},
			},
			{
				Name: "ListOfPtr",
				Type: &shape.RefName{
					Name: "ListOf",
					PkgName: "testutils",
					PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
					IsPointer: true,
					Indexed: []shape.Shape{
						&shape.RefName{
							Name: "T2",
							PkgName: "",
							PkgImportName: "",
							IsPointer: false,
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
					IsPointer: false,
				},
			},
			{
				Name: "Value",
				Type: &shape.RefName{
					Name: "Schema",
					PkgName: "schema",
					PkgImportName: "github.com/widmogrod/mkunion/x/schema",
					IsPointer: false,
				},
			},
		},
		TypeParams: []shape.Shape{
			&shape.TypeParam{
				Name: "T1",
				Type: &shape.Any{},
			},
			&shape.TypeParam{
				Name: "T2",
				Type: &shape.Any{},
			},
		},
	}
}
`, result)
}
