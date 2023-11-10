package shape

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type structA struct {
	Name  string
	Other Shape
}

func TestFromGoo(t *testing.T) {
	result := FromGo(structA{})
	assert.Equal(t, &StructLike{
		Name:          "structA",
		PkgName:       "shape",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape",
		Fields: []*FieldLike{
			{
				Name: "Name",
				Type: &StringLike{},
			},
			{
				Name: "Other",
				Type: &UnionLike{
					Name:          "Shape",
					PkgName:       "shape",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape",
					Variant: []Shape{
						&StructLike{
							Name:          "Any",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields:        []*FieldLike{},
						},
						&StructLike{
							Name:          "RefName",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
								{
									Name: "Name",
									Type: &StringLike{},
								},
								{
									Name: "PkgName",
									Type: &StringLike{},
								},
								{
									Name: "PkgImportName",
									Type: &StringLike{},
								},
							},
						},
						&StructLike{
							Name:          "BooleanLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields:        []*FieldLike{},
						},
						&StructLike{
							Name:          "StringLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields:        []*FieldLike{},
						},
						&StructLike{
							Name:          "NumberLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields:        []*FieldLike{},
						},
						&StructLike{
							Name:          "ListLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
								{
									Name: "Element",
									Type: &RefName{
										Name:          "Shape",
										PkgName:       "shape",
										PkgImportName: "github.com/widmogrod/mkunion/x/shape",
									},
								},
							},
						},
						&StructLike{
							Name:          "MapLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
								{
									Name: "Key",
									Type: &RefName{
										Name:          "Shape",
										PkgName:       "shape",
										PkgImportName: "github.com/widmogrod/mkunion/x/shape",
									},
								},
								{
									Name: "Val",
									Type: &RefName{
										Name:          "Shape",
										PkgName:       "shape",
										PkgImportName: "github.com/widmogrod/mkunion/x/shape",
									},
								},
							},
						},
						&StructLike{
							Name:          "StructLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
								{
									Name: "Name",
									Type: &StringLike{},
								},
								{
									Name: "PkgName",
									Type: &StringLike{},
								},
								{
									Name: "PkgImportName",
									Type: &StringLike{},
								},
								{
									Name: "Fields",
									Type: &ListLike{
										Element: &StructLike{
											Name:          "FieldLike",
											PkgName:       "shape",
											PkgImportName: "github.com/widmogrod/mkunion/x/shape",
											Fields: []*FieldLike{
												{
													Name: "Name",
													Type: &StringLike{},
												},
												{
													Name: "Type",
													Type: &RefName{
														Name:          "Shape",
														PkgName:       "shape",
														PkgImportName: "github.com/widmogrod/mkunion/x/shape",
													},
												},
											},
										},
									},
								},
							},
						},
						&StructLike{
							Name:          "UnionLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
								{
									Name: "Name",
									Type: &StringLike{},
								},
								{
									Name: "PkgName",
									Type: &StringLike{},
								},
								{
									Name: "PkgImportName",
									Type: &StringLike{},
								},
								{
									Name: "Variant",
									Type: &ListLike{
										Element: &RefName{
											Name:          "Shape",
											PkgName:       "shape",
											PkgImportName: "github.com/widmogrod/mkunion/x/shape",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, result)
}
