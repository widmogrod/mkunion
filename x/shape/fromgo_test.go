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
				Type: &StringLike{},
				Desc: ptr("Name of the person"),
				Tags: map[string]FieldTag{
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
							Name:          "AliasLike",
							PkgName:       "shape",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape",
							Fields: []*FieldLike{
								{Name: "Name", Type: &StringLike{}},
								{Name: "PkgName", Type: &StringLike{}},
								{Name: "PkgImportName", Type: &StringLike{}},
								{Name: "IsAlias", Type: &BooleanLike{}},
								{Name: "Type", Type: &RefName{
									Name:          "Shape",
									PkgName:       "shape",
									PkgImportName: "github.com/widmogrod/mkunion/x/shape",
								}},
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
							Fields: []*FieldLike{
								{
									Name: "Kind",
									Type: &UnionLike{
										Name:          "NumberKind",
										PkgName:       "shape",
										PkgImportName: "github.com/widmogrod/mkunion/x/shape",
										Variant: []Shape{
											&StructLike{
												Name:          "UInt8",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
											&StructLike{
												Name:          "UInt16",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
											&StructLike{
												Name:          "UInt32",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
											&StructLike{
												Name:          "UInt64",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
											&StructLike{
												Name:          "Int8",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
											&StructLike{
												Name:          "Int16",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
											&StructLike{
												Name:          "Int32",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
											&StructLike{
												Name:          "Int64",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
											&StructLike{
												Name:          "Float32",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
											&StructLike{
												Name:          "Float64",
												PkgName:       "shape",
												PkgImportName: "github.com/widmogrod/mkunion/x/shape",
												Fields:        []*FieldLike{},
											},
										},
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
									Type: &BooleanLike{},
								},
								{
									Name:      "ArrayLen",
									Type:      &NumberLike{},
									IsPointer: true,
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
									Type: &BooleanLike{},
								},
								{
									Name: "ValIsPointer",
									Type: &BooleanLike{},
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
												{
													Name:      "Desc",
													Type:      &StringLike{},
													IsPointer: true,
												},
												{
													Name: "Guard",
													Type: &UnionLike{
														Name:          "Guard",
														PkgName:       "shape",
														PkgImportName: "github.com/widmogrod/mkunion/x/shape",
														Variant: []Shape{
															&StructLike{
																Name:          "Enum",
																PkgName:       "shape",
																PkgImportName: "github.com/widmogrod/mkunion/x/shape",
																Fields:        []*FieldLike{{Name: "Val", Type: &ListLike{Element: &StringLike{}}}},
															},
															&StructLike{
																Name:          "Required",
																PkgName:       "shape",
																PkgImportName: "github.com/widmogrod/mkunion/x/shape",
																Fields:        []*FieldLike{},
															},
															&StructLike{
																Name:          "AndGuard",
																PkgName:       "shape",
																PkgImportName: "github.com/widmogrod/mkunion/x/shape",
																Fields: []*FieldLike{
																	{
																		Name: "L",
																		Type: &ListLike{
																			Element: &RefName{
																				Name:          "Guard",
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
												{
													Name: "IsPointer",
													Type: &BooleanLike{},
												},
												{
													Name: "Tags",
													Type: &MapLike{
														Key: &StringLike{},
														Val: &StructLike{
															Name:          "FieldTag",
															PkgName:       "shape",
															PkgImportName: "github.com/widmogrod/mkunion/x/shape",
															Fields: []*FieldLike{
																{
																	Name: "Value",
																	Type: &StringLike{},
																},
																{
																	Name: "Options",
																	Type: &ListLike{Element: &StringLike{}},
																},
															},
														},
													},
												},
											},
										},
										ElementIsPointer: true,
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
				Tags: map[string]FieldTag{
					"desc": {Value: "Big bag of attributes"},
				},
			},
		},
	}

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("mismatch (-want +got):\n%s", diff)
	}
}
