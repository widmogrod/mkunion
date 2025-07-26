package shape

import (
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
			},
			&AliasLike{
				Name:          "C",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type:          &PrimitiveLike{Kind: &StringLike{}},
				Tags:          map[string]Tag{"mkunion": {Value: "Example"}},
			},
			&AliasLike{
				Name:          "D",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type: &PrimitiveLike{Kind: &NumberLike{
					Kind: &Int64{},
				}},
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
			},
			&AliasLike{
				Name:          "E",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type: &PrimitiveLike{Kind: &NumberLike{
					Kind: &Float64{},
				}},
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
			},
			&AliasLike{
				Name:          "F",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Type:          &PrimitiveLike{Kind: &BooleanLike{}},
				Tags:          map[string]Tag{"mkunion": {Value: "Example"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
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
					"mkunion": {
						Value: "Example",
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "Example"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "AliasExample"}},
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
				Tags: map[string]Tag{"mkunion": {Value: "AliasExample"}},
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

	t.Run("variant types should be marked as union", func(t *testing.T) {
		for _, v := range union.Variant {
			unionRef := RetrieveVariantTypeRef(v)
			assert.Equal(t, "Example", unionRef.Name)
		}
	})

	t.Run("should found tagged ast.Node", func(t *testing.T) {
		if assert.Len(t, inferred.taggedNodes, 4) {
			assert.Equal(t, "Example", inferred.taggedNodes["mkunion"][0].Tag.Value)
			assert.Equal(t, "AliasExample", inferred.taggedNodes["mkunion"][1].Tag.Value)
			assert.Equal(t, "Option[AZ]", inferred.taggedNodes["mkunion"][2].Tag.Value)
			assert.Equal(t, "list", inferred.taggedNodes["json"][0].Tag.Value)
			assert.Equal(t, "MyMatchWithCustomName", inferred.taggedNodes["mkmatch"][0].Tag.Value)
		}
	})
}

func TestIndexedTypeWalker_ExpandedShapes(t *testing.T) {
	inferred, err := NewIndexTypeInDir("testasset")
	assert.NoError(t, err)
	assert.NotNil(t, inferred)

	indexed := inferred.IndexedShapes()
	expanded := inferred.ExpandedShapes()

	t.Run("should detect union type", func(t *testing.T) {
		require.Contains(t, indexed, "testasset.Option[ListOf2[*O,time.Location]]")
		require.Contains(t, expanded, "github.com/widmogrod/mkunion/x/shape/testasset.Option[ListOf2[*O,time.Location]]")
	})
	t.Run("expanded should have variant of a union", func(t *testing.T) {
		require.NotContains(t, indexed, "testasset.Some[ListOf2[*O,time.Location]]")
		require.Contains(t, expanded, "github.com/widmogrod/mkunion/x/shape/testasset.Some[ListOf2[*O,time.Location]]")
		require.NotContains(t, indexed, "testasset.None[ListOf2[*O,time.Location]]")
		require.Contains(t, expanded, "github.com/widmogrod/mkunion/x/shape/testasset.None[ListOf2[*O,time.Location]]")
	})
}

func TestIndexedTypeWalker_Visit(t *testing.T) {
	useCases := map[string]struct {
		body            string
		expected        map[string]Shape
		expectedPkgTags map[string]Tag
	}{
		"from interface declaration": {
			body: `package test_package

import "time"

type OptionVisitor[T1 ListOf2[*O,time.Location]] interface {
	VisitSome(v *Some[T1]) any 		// should be ignored
	VisitNone(v *None[T1]) any 		// should be ignored
}`,
			expected: map[string]Shape{
				"github.com/test_package.ListOf2[*O,time.Location]": &RefName{
					Name:          "ListOf2",
					PkgName:       "test_package",
					PkgImportName: "github.com/test_package",
					Indexed: []Shape{
						&PointerLike{
							Type: &RefName{
								Name:          "O",
								PkgName:       "test_package",
								PkgImportName: "github.com/test_package",
							},
						},
						&RefName{
							Name:          "Location",
							PkgName:       "time",
							PkgImportName: "time",
						},
					},
				},
			},
		},
		"from variables top level": {
			body: `package test_package
import "time"

var (
	_ Option[ListOf2[*O,time.Location]] = (*Some[ListOf2[*O,time.Location]])(nil)
	_ Option[ListOf2[*O,time.Location]] = (*None[ListOf2[*O,time.Location]])(nil)
)
`,
			expected: map[string]Shape{
				"github.com/test_package.Option[ListOf2[*O,time.Location]]": &RefName{
					Name:          "Option",
					PkgName:       "test_package",
					PkgImportName: "github.com/test_package",
					Indexed: []Shape{
						&RefName{
							Name:          "ListOf2",
							PkgName:       "test_package",
							PkgImportName: "github.com/test_package",
							Indexed: []Shape{
								&PointerLike{
									Type: &RefName{
										Name:          "O",
										PkgName:       "test_package",
										PkgImportName: "github.com/test_package",
									},
								},
								&RefName{
									Name:          "Location",
									PkgName:       "time",
									PkgImportName: "time",
								},
							},
						},
					},
				},
			},
		},
		"from function declaration": {
			body: `package test_package

import "time"

func OptionToJSON[T1 ListOf2[*O,time.Location]](x Option[T1]) ([]byte, error) {
	return MatchOptionR2(
		x,
		func (y *Some[T1]) ([]byte, error) {
			body, err := SomeToJSON[T1](y)
			if err != nil {
				return nil, fmt.Errorf("testasset.OptionToJSON[T1]: %w", err)
			}
			return json.Marshal(OptionUnionJSON[T1]{
				Type: "testasset.Some",
				Some: body,
			})
		},
		func (y *None[int]) ([]byte, error) {
			body, err := NoneToJSON[int](y)
			if err != nil {
				return nil, fmt.Errorf("testasset.OptionToJSON[T1]: %w", err)
			}
			return json.Marshal(OptionUnionJSON[T1]{
				Type: "testasset.None",
				None: body,
			})
		},
	)
}
`,
			expected: map[string]Shape{
				"github.com/test_package.ListOf2[*O,time.Location]": &RefName{
					Name:          "ListOf2",
					PkgName:       "test_package",
					PkgImportName: "github.com/test_package",
					Indexed: []Shape{
						&PointerLike{
							Type: &RefName{
								Name:          "O",
								PkgName:       "test_package",
								PkgImportName: "github.com/test_package",
							},
						},
						&RefName{
							Name:          "Location",
							PkgName:       "time",
							PkgImportName: "time",
						},
					},
				},
				"github.com/test_package.None[int]": &RefName{
					Name:          "None",
					PkgName:       "test_package",
					PkgImportName: "github.com/test_package",
					Indexed: []Shape{
						&PrimitiveLike{Kind: &NumberLike{Kind: &Int{}}},
					},
				},
			},
		},
		"from function receiver": {
			body: `package test_package

func (r *Some[T1]) _unmarshalJSONSomeLb_T1_bL(data []byte) (Some[T1], error) {
	result := Some[int]{}
	var partial map[string]json.RawMessage
	err := json.Unmarshal(data, &partial)
	if err != nil {
		return result, fmt.Errorf("testasset: Some[T1]._unmarshalJSONSomeLb_T1_bL: native struct unwrap; %w", err)
	}
	if fieldData, ok := partial["Data"]; ok {
		result.Data, err = r._unmarshalJSONT1(fieldData)
		if err != nil {
			return result, fmt.Errorf("testasset: Some[T1]._unmarshalJSONSomeLb_T1_bL: field Data; %w", err)
		}
	}
	return result, nil
}
`,
			expected: map[string]Shape{
				"github.com/test_package.Some[int]": &RefName{
					Name:          "Some",
					PkgName:       "test_package",
					PkgImportName: "github.com/test_package",
					Indexed: []Shape{
						&PrimitiveLike{Kind: &NumberLike{Kind: &Int{}}},
					},
				},
			},
		},
		"from init function with embeded function": {
			body: `package test_package

import "time"

func init() {
	shared.JSONMarshallerRegister[T1]("github.com/widmogrod/mkunion/x/shape/testasset.None[ListOf2[*.O,time.Location]]", ListOf2[*O,time.Location]{}, &ListOf2[*O,time.Location]{})
	shared.JSONMarshallerRegister[T1]("github.com/widmogrod/mkunion/x/shape/testasset.None[ListOf2[*.O,time.Location]]", NoneFromJSON[ListOf2[*O,time.Location]], NoneToJSON[ListOf2[*O,time.Location]])
}`,
			expected: map[string]Shape{
				"github.com/test_package.ListOf2[*O,time.Location]": &RefName{
					Name:          "ListOf2",
					PkgName:       "test_package",
					PkgImportName: "github.com/test_package",
					Indexed: []Shape{
						&PointerLike{
							Type: &RefName{
								Name:          "O",
								PkgName:       "test_package",
								PkgImportName: "github.com/test_package",
							},
						},
						&RefName{
							Name:          "Location",
							PkgName:       "time",
							PkgImportName: "time",
						},
					},
				},
			},
		},
		"contrastive example, commented code is not initialised": {
			body: `package projection

func init () {
	delay := 1 * time.Second
	timer := time.NewTimer(delay)
	defer timer.Stop()
}

//go:tag mkunion:"WindowFlushMode"
type (
	//Accumulate struct {
	//	AllowLateArrival time.Duration
	//}
	Discard struct{}
	//AccumulatingAndRetracting struct {
	//	AllowLateArrival time.Duration
	//}
)
`,
			expected: make(map[string]Shape),
		},
		"contrastive example, pointer to type parameter is not initialisation": {
			body: `// projection package is responsible for storing and retrieving data from the storage.
//go:tag mytag:"-"
package projection

type Storage[T any] interface {
	GetAs(id string, x *T) error
}
`,
			expected: make(map[string]Shape),
			expectedPkgTags: map[string]Tag{
				"mytag": {Value: "-"},
			},
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			w := newIndexedTypeWalkerWithContentBody(
				uc.body,
				func(x *IndexedTypeWalker) {
					x.SetPkgImportName("github.com/test_package")
				},
			)
			assert.NotNil(t, w)
			for _, i := range w.ExpandedShapes() {
				t.Log(ToGoTypeName(i, WithInstantiation()))
			}

			if diff := cmp.Diff(uc.expected, w.ExpandedShapes()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}

			if diff := cmp.Diff(uc.expectedPkgTags, w.PackageTags()); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExpandedShapes_UsesPkgImportNameForKeys(t *testing.T) {
	// Test that ExpandedShapes uses full package import names for keys
	// to prevent duplicate registrations with dot imports
	indexed, err := NewIndexTypeInDir("testasset")
	assert.NoError(t, err)
	assert.NotNil(t, indexed)

	expanded := indexed.ExpandedShapes()

	// All keys should use the full package import name format
	// Check that types from known packages have proper import names
	hasExpectedKeys := false
	for key := range expanded {
		t.Logf("Key: %s", key)
		// Keys should contain the full package import path for types from testasset
		if strings.Contains(key, "testasset.") && strings.Contains(key, ".") {
			assert.True(t, 
				strings.Contains(key, "github.com/widmogrod/mkunion/x/shape/testasset."),
				"Key %s should use full package import name", key)
			hasExpectedKeys = true
		}
	}
	
	// Ensure we actually tested some keys
	assert.True(t, hasExpectedKeys, "Should have found keys with testasset package")
}
