package shape

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestToGoTypeName(t *testing.T) {
	inferred, err := InferFromFile("testasset/type_example.go")
	assert.NoError(t, err)

	useCases := []struct {
		typeName string
		expected string
		options  []ToGoTypeNameOptionFunc
	}{
		{
			typeName: "A",
			expected: "testasset.A",
		},
		{
			typeName: "ListOf2",
			expected: "testasset.ListOf2[T1,T2]",
		},
		{
			typeName: "Example",
			expected: "testasset.Example",
		},
		{
			typeName: "Option",
			expected: "testasset.Option[AZ]",
		},
		{
			typeName: "A",
			expected: "testasset.A",
			options: []ToGoTypeNameOptionFunc{
				WithInstantiation(),
			},
		},
		{
			typeName: "ListOf2",
			expected: "testasset.ListOf2[any,any]",
			options: []ToGoTypeNameOptionFunc{
				WithInstantiation(),
			},
		},
		{
			typeName: "Example",
			expected: "testasset.Example",
			options: []ToGoTypeNameOptionFunc{
				WithInstantiation(),
			},
		},
		{
			typeName: "Option",
			expected: "testasset.Option[ListOf2[*O,time.Location]]",
			options: []ToGoTypeNameOptionFunc{
				WithInstantiation(),
			},
		},
		{
			typeName: "A",
			expected: "A",
			options: []ToGoTypeNameOptionFunc{
				WithRootPkgName(inferred.PackageName()),
			},
		},
		{
			typeName: "ListOf2",
			expected: "ListOf2[T1,T2]",
			options: []ToGoTypeNameOptionFunc{
				WithRootPkgName(inferred.PackageName()),
			},
		},
		{
			typeName: "Example",
			expected: "Example",
			options: []ToGoTypeNameOptionFunc{
				WithRootPkgName(inferred.PackageName()),
			},
		},
		{
			typeName: "Option",
			expected: "Option[AZ]",
			options: []ToGoTypeNameOptionFunc{
				WithRootPkgName(inferred.PackageName()),
			},
		},
		{
			typeName: "A",
			expected: "github.com/widmogrod/mkunion/x/shape/testasset.A",
			options: []ToGoTypeNameOptionFunc{
				WithPkgImportName(),
			},
		},
		{
			typeName: "ListOf2",
			expected: "github.com/widmogrod/mkunion/x/shape/testasset.ListOf2[T1,T2]",
			options: []ToGoTypeNameOptionFunc{
				WithPkgImportName(),
			},
		},
		{
			typeName: "Example",
			expected: "github.com/widmogrod/mkunion/x/shape/testasset.Example",
			options: []ToGoTypeNameOptionFunc{
				WithPkgImportName(),
			},
		},
		{
			typeName: "Option",
			expected: "github.com/widmogrod/mkunion/x/shape/testasset.Option[AZ]",
			options: []ToGoTypeNameOptionFunc{
				WithPkgImportName(),
			},
		},
	}

	for _, useCase := range useCases {
		t.Run(useCase.typeName+"_"+useCase.expected, func(t *testing.T) {
			x := inferred.RetrieveShapeNamedAs(useCase.typeName)
			if actual := ToGoTypeName(x, useCase.options...); actual != useCase.expected {
				t.Errorf("Expected %q, got %q", useCase.expected, actual)
			}
		})
	}
}

func TestToGoTypeNameInst(t *testing.T) {
	subject := &RefName{
		Name:          "ListOf2",
		PkgName:       "testasset",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
		Indexed: []Shape{
			&RefName{
				Name:          "A",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			},
			&RefName{
				Name:          "B",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
			},
		},
	}

	result := ToGoTypeName(subject, WithPkgImportName(), WithInstantiation())
	assert.Equal(t, "github.com/widmogrod/mkunion/x/shape/testasset.ListOf2[github.com/widmogrod/mkunion/x/shape/testasset.A,github.com/widmogrod/mkunion/x/shape/testasset.B]", result)
}

func TestPkgName(t *testing.T) {
	useCases := []struct {
		input    Shape
		expected string
	}{
		{
			input: &RefName{
				Name:          "ListOf2",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Indexed: []Shape{
					&RefName{
						Name:          "A",
						PkgName:       "testasset",
						PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
					},
				},
			},
			expected: "testasset",
		},
		{
			input: &AliasLike{
				Name:          "FetchResult",
				PkgName:       "testutils",
				PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
				IsAlias:       true,
				Type: &RefName{
					Name:          "Result",
					PkgName:       "f",
					PkgImportName: "github.com/widmogrod/mkunion/f",
					Indexed: []Shape{
						&RefName{
							Name:          "Option",
							PkgName:       "f",
							PkgImportName: "github.com/widmogrod/mkunion/f",
							Indexed: []Shape{
								&RefName{
									Name:          "User",
									PkgName:       "testutils",
									PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
								},
							},
						},
						&RefName{
							Name:          "APIError",
							PkgName:       "testutils",
							PkgImportName: "github.com/widmogrod/mkunion/x/generators/testutils",
						},
					},
				},
			},
			expected: "testutils",
		},
	}
	for _, uc := range useCases {
		t.Run(ToStr(uc.input), func(t *testing.T) {
			actual := PkgName(uc.input)
			require.Equal(t, uc.expected, actual)
		})
	}
}
