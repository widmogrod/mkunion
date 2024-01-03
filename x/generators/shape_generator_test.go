package generators

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestShapeGenerator(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	inferred, err := shape.InferFromFile("testutils/tree.go")
	assert.NoError(t, err)

	g := NewShapeGenerator(inferred.RetrieveUnion("Tree"))

	result, err := g.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `package testutils

import (
	"github.com/widmogrod/mkunion/x/shape"
)

func init() {
	shape.Register(TreeShape())
	shape.Register(BranchShape())
	shape.Register(LeafShape())
	shape.Register(KShape())
	shape.Register(PShape())
	shape.Register(MaShape())
	shape.Register(LaShape())
	shape.Register(KaShape())
}


func TreeShape() shape.Shape {
	return &shape.UnionLike{
		Name: "Tree",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		Variant: []shape.Shape{
			BranchShape(),
			LeafShape(),
			KShape(),
			PShape(),
			MaShape(),
			LaShape(),
			KaShape(),
		},
	}
}

func BranchShape() shape.Shape {
	return &shape.StructLike{
		Name: "Branch",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		Fields: []*shape.FieldLike{
			{
				Name: "Lit",
				Type: &shape.RefName{
					Name: "Tree",
					PkgName: "testutils",
					PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
				},
			},
			{
				Name: "List",
				Type: &shape.ListLike{
					Element: &shape.RefName{
						Name: "Tree",
						PkgName: "testutils",
						PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
					},
				},
			},
			{
				Name: "Map",
				Type: &shape.MapLike{
					Key: &shape.StringLike{},
					Val: &shape.RefName{
						Name: "Tree",
						PkgName: "testutils",
						PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
					},
				},
			},
			{
				Name: "Of",
				Type: &shape.RefName{
					Name: "ListOf",
					PkgName: "testutils",
					PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
					IsPointer: true,
					Indexed: []shape.Shape{
						&shape.RefName{
							Name: "Tree",
							PkgName: "testutils",
							PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
						},
					},
				},
				IsPointer: true,
				Tags: map[string]shape.Tag{
					"json": {
						Value: "just_of",
					},
				},
			},
			{
				Name: "L",
				Type: &shape.RefName{
					Name: "Leaf",
					PkgName: "testutils",
					PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
					IsPointer: true,
				},
				IsPointer: true,
			},
			{
				Name: "Kattr",
				Type: &shape.ListLike{
					Element: &shape.RefName{
						Name: "Leaf",
						PkgName: "testutils",
						PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
						IsPointer: true,
					},
					ElementIsPointer: true,
					ArrayLen: shape.Ptr(2),
				},
			},
		},
	}
}

func LeafShape() shape.Shape {
	return &shape.StructLike{
		Name: "Leaf",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		Fields: []*shape.FieldLike{
			{
				Name: "Value",
				Type: &shape.NumberLike{
					Kind: &shape.Int64{},
				},
			},
		},
	}
}

func KShape() shape.Shape {
	return &shape.AliasLike{
		Name: "K",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		Type: &shape.StringLike{},
	}
}

func PShape() shape.Shape {
	return &shape.AliasLike{
		Name: "P",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		Type: &shape.RefName{
				Name: "ListOf2",
				PkgName: "testutils",
				PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
				Indexed: []shape.Shape{
					&shape.RefName{
						Name: "ListOf",
						PkgName: "testutils",
						PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
						Indexed: []shape.Shape{
							&shape.Any{},
						},
					},
					&shape.RefName{
						Name: "ListOf2",
						PkgName: "testutils",
						PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
						IsPointer: true,
						Indexed: []shape.Shape{
							&shape.NumberLike{
								Kind: &shape.Int64{},
							},
							&shape.RefName{
								Name: "Duration",
								PkgName: "time",
								PkgImportName: "time",
								IsPointer: true,
							},
						},
					},
				},
			},
	}
}

func MaShape() shape.Shape {
	return &shape.AliasLike{
		Name: "Ma",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		Type: &shape.MapLike{
				Key: &shape.StringLike{},
				Val: &shape.RefName{
					Name: "Tree",
					PkgName: "testutils",
					PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
				},
			},
	}
}

func LaShape() shape.Shape {
	return &shape.AliasLike{
		Name: "La",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		Type: &shape.ListLike{
				Element: &shape.RefName{
					Name: "Tree",
					PkgName: "testutils",
					PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
				},
			},
	}
}

func KaShape() shape.Shape {
	return &shape.AliasLike{
		Name: "Ka",
		PkgName: "testutils",
		PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
		Type: &shape.ListLike{
				Element: &shape.MapLike{
					Key: &shape.StringLike{},
					Val: &shape.RefName{
						Name: "Tree",
						PkgName: "testutils",
						PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
					},
				},
			},
	}
}
`, string(result))
}
