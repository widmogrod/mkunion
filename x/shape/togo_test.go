package shape

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestToGoTypeName(t *testing.T) {
	inferred, err := InferFromFile("testasset/type_example.go")
	assert.NoError(t, err)

	useCases := []struct {
		typeName string
		expected string
		options  []ToGoTypeNameOption
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
			expected: "testasset.Option[A]",
		},
		{
			typeName: "A",
			expected: "testasset.A",
			options: []ToGoTypeNameOption{
				WithInstantiation(),
			},
		},
		{
			typeName: "ListOf2",
			expected: "testasset.ListOf2[any,any]",
			options: []ToGoTypeNameOption{
				WithInstantiation(),
			},
		},
		{
			typeName: "Example",
			expected: "testasset.Example",
			options: []ToGoTypeNameOption{
				WithInstantiation(),
			},
		},
		{
			typeName: "Option",
			expected: "testasset.Option[testasset.ListOf2[*testasset.O,time.Location]]",
			options: []ToGoTypeNameOption{
				WithInstantiation(),
			},
		},
		{
			typeName: "A",
			expected: "A",
			options: []ToGoTypeNameOption{
				WithRootPackage(inferred.PackageName()),
			},
		},
		{
			typeName: "ListOf2",
			expected: "ListOf2[T1,T2]",
			options: []ToGoTypeNameOption{
				WithRootPackage(inferred.PackageName()),
			},
		},
		{
			typeName: "Example",
			expected: "Example",
			options: []ToGoTypeNameOption{
				WithRootPackage(inferred.PackageName()),
			},
		},
		{
			typeName: "Option",
			expected: "Option[A]",
			options: []ToGoTypeNameOption{
				WithRootPackage(inferred.PackageName()),
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
