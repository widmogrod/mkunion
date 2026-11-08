package shape

import (
	"strings"
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

	r := NewTypeScriptRenderer()
	r.AddShape(query)
	files, err := r.Render()
	require.NoError(t, err)
	assert.Equal(t, "export type Query = {\n  Name: string;\n};\n", files["example.ts"][strings.Index(files["example.ts"], "export type Query"):], "TypeScript export skips the phantom and keeps the field, untagged, under its Go name")
	assert.NotContains(t, files, "f.ts", "the phantom's package is not a file")

	assert.Equal(t, "example", ExtractPkgImportNames(query)["example"][len("github.com/widmogrod/mkunion/"):], "sanity: own package is listed")
	assert.NotContains(t, ExtractPkgImportNames(query), "f", "the phantom's package is not imported")
}

func TestUnionDeclaresReturns(t *testing.T) {
	inferred, err := InferFromFileWithContentBody(`package example

import "github.com/widmogrod/mkunion/f"

//go:tag mkunion:"Query"
type (
	GetUser struct{ f.Returns[int] }
	Touch   struct{ ID string }
)

//go:tag mkunion:"Plain"
type (
	Leaf   struct{ Value int }
	Branch struct{ Left, Right Plain }
)
`, "github.com/widmogrod/mkunion/example")
	require.NoError(t, err)

	assert.True(t, UnionDeclaresReturns(inferred.RetrieveUnion("Query")), "one variant with f.Returns is enough")
	assert.False(t, UnionDeclaresReturns(inferred.RetrieveUnion("Plain")), "no variant embeds f.Returns")
}
