package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInferFromFile_embeddedFields(t *testing.T) {
	inferred, err := InferFromFile("testasset/type_embedded.go")
	require.NoError(t, err)

	got := inferred.RetrieveStruct("Embedded")
	require.NotNil(t, got, "Embedded is a struct")

	names := make([]string, 0, len(got.Fields))
	for _, field := range got.Fields {
		names = append(names, field.Name)
	}
	assert.Equal(t, []string{"ListOf", "Some", "Time", "ListOf2", "Name"}, names,
		"embedded fields keep the name Go gives them, in declaration order")

	assert.Equal(t, &RefName{
		Name:          "ListOf",
		PkgName:       "testasset",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
		Indexed:       []Shape{&PrimitiveLike{Kind: &NumberLike{Kind: &Int{}}}},
	}, got.Fields[0].Type)

	assert.Equal(t, &RefName{
		Name:          "Some",
		PkgName:       "f",
		PkgImportName: "github.com/widmogrod/mkunion/f",
		Indexed: []Shape{&RefName{
			Name:          "Time",
			PkgName:       "time",
			PkgImportName: "time",
		}},
	}, got.Fields[1].Type, "embedded generic type from another package keeps its import path and type argument")

	assert.Equal(t, &PointerLike{Type: &RefName{
		Name:          "Time",
		PkgName:       "time",
		PkgImportName: "time",
	}}, got.Fields[2].Type)
}
