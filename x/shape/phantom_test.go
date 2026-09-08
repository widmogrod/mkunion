package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReturnsOf(t *testing.T) {
	inferred, err := InferFromFileWithContentBody(`package example

import "github.com/widmogrod/mkunion/f"

type Query struct {
	f.Returns[int]
	Name string
}
`, "github.com/widmogrod/mkunion/example")
	require.NoError(t, err)

	query := inferred.RetrieveStruct("Query")
	require.NotNil(t, query)
	require.Len(t, query.Fields, 2)

	answer, ok := ReturnsOf(query.Fields[0])
	assert.True(t, ok, "embedded f.Returns[int] is the marker")
	assert.Equal(t, &PrimitiveLike{Kind: &NumberLike{Kind: &Int{}}}, answer)
	assert.True(t, IsPhantomField(query.Fields[0]))

	_, ok = ReturnsOf(query.Fields[1])
	assert.False(t, ok, "a plain field is not the marker")
	assert.False(t, IsPhantomField(query.Fields[1]))

	assert.Equal(t, `export type Query = {
	Name?: string,
}
`, ToTypeScript(query, &TypeScriptOptions{currentPkgName: "example"}), "TypeScript export skips the phantom")

	assert.Equal(t, "example", ExtractPkgImportNames(query)["example"][len("github.com/widmogrod/mkunion/"):], "sanity: own package is listed")
	assert.NotContains(t, ExtractPkgImportNames(query), "f", "the phantom's package is not imported")
}
