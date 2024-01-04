package shape

import (
	"github.com/google/go-cmp/cmp"
	"testing"
)

type structA struct {
	Name  string `desc:"Name of the person"`
	Other Shape  `desc:"Big bag of attributes"`
}

func ptr[A any](a A) *A {
	return &a
}

func TestFromGoo(t *testing.T) {
	result := FromGo(structA{})
	expected := &StructLike{
		Name:          "structA",
		PkgName:       "shape",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape",
		Fields: []*FieldLike{
			{
				Name: "Name",
				Type: &PrimitiveLike{Kind: &StringLike{}},
				Desc: ptr("Name of the person"),
				Tags: map[string]Tag{
					"desc": {Value: "Name of the person"},
				},
			},
			{
				Name: "Other",
				Desc: ptr("Big bag of attributes"),
				Type: &UnionLike{
					Name:          "Shape",
					PkgName:       "shape",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape",
					Variant: []Shape{
						&StructLike{
							Name:          "Any",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields:        nil,
						},
						&StructLike{
							Name:          "RefName",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
								{
									Name: "Name",
									Type: &PrimitiveLike{Kind: &StringLike{}},
								},
								{
									Name: "PkgName",
									Type: &PrimitiveLike{Kind: &StringLike{}},
								},
								{
									Name: "PkgImportName",
									Type: &PrimitiveLike{Kind: &StringLike{}},
								},
								{
									Name: "IsPointer",
									Type: &PrimitiveLike{Kind: &BooleanLike{}},
								},
								{
									Name: "Indexed",
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
						&StructLike{
							Name:          "PointerLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
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
						&StructLike{
							Name:          "AliasLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
								{Name: "Name", Type: &PrimitiveLike{Kind: &StringLike{}}},
								{Name: "PkgName", Type: &PrimitiveLike{Kind: &StringLike{}}},
								{Name: "PkgImportName", Type: &PrimitiveLike{Kind: &StringLike{}}},
								{Name: "IsAlias", Type: &PrimitiveLike{Kind: &BooleanLike{}}},
								{Name: "Type", Type: &RefName{
									Name:          "Shape",
									PkgName:       "shape",
									PkgImportName: "github.com/widmogrod/mkunion/x/shape",
								}},
								{Name: "Tags", Type: &MapLike{
									Key: &PrimitiveLike{Kind: &StringLike{}},
									Val: &RefName{
										Name:          "Tag",
										PkgName:       "shape",
										PkgImportName: "github.com/widmogrod/mkunion/x/shape",
									},
								}},
							},
						},
						&StructLike{
							Name:          "PrimitiveLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
								{
									Name: "Kind",
									Type: &RefName{
										Name:          "PrimitiveKind",
										PkgName:       "shape",
										PkgImportName: "github.com/widmogrod/mkunion/x/shape",
									},
								},
							},
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
								{
									Name: "ElementIsPointer",
									Type: &PrimitiveLike{Kind: &BooleanLike{}},
								},
								{
									Name: "ArrayLen",
									Type: &PrimitiveLike{&NumberLike{}},
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
								{
									Name: "KeyIsPointer",
									Type: &PrimitiveLike{Kind: &BooleanLike{}},
								},
								{
									Name: "ValIsPointer",
									Type: &PrimitiveLike{Kind: &BooleanLike{}},
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
									Type: &PrimitiveLike{Kind: &StringLike{}},
								},
								{
									Name: "PkgName",
									Type: &PrimitiveLike{Kind: &StringLike{}},
								},
								{
									Name: "PkgImportName",
									Type: &PrimitiveLike{Kind: &StringLike{}},
								},
								{
									Name: "TypeParams",
									Type: &ListLike{
										Element: &RefName{
											Name:          "TypeParam",
											PkgName:       "shape",
											PkgImportName: "github.com/widmogrod/mkunion/x/shape",
										},
									},
								},
								{
									Name: "Fields",
									Type: &ListLike{
										Element: &RefName{
											Name:          "FieldLike",
											PkgName:       "shape",
											PkgImportName: "github.com/widmogrod/mkunion/x/shape",
										},
									},
								},
								{
									Name: "Tags",
									Type: &MapLike{
										Key: &PrimitiveLike{Kind: &StringLike{}},
										Val: &RefName{
											Name:          "Tag",
											PkgName:       "shape",
											PkgImportName: "github.com/widmogrod/mkunion/x/shape",
										},
									},
								},
								{
									Name: "IsPointer",
									Type: &PrimitiveLike{Kind: &BooleanLike{}},
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
									Type: &PrimitiveLike{Kind: &StringLike{}},
								},
								{
									Name: "PkgName",
									Type: &PrimitiveLike{Kind: &StringLike{}},
								},
								{
									Name: "PkgImportName",
									Type: &PrimitiveLike{Kind: &StringLike{}},
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
								{
									Name: "Tags",
									Type: &MapLike{
										Key: &PrimitiveLike{Kind: &StringLike{}},
										Val: &RefName{
											Name:          "Tag",
											PkgName:       "shape",
											PkgImportName: "github.com/widmogrod/mkunion/x/shape",
										},
									},
								},
							},
						},
					},
				},
				Tags: map[string]Tag{
					"desc": {Value: "Big bag of attributes"},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
