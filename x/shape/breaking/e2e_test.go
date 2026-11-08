package breaking

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mangaFile(v string) string { return filepath.Join("testdata", "manga", v, "api.go") }

func compareManga(t *testing.T, from, to string) []Finding {
	t.Helper()
	old, err := Take([]string{mangaFile(from)})
	require.NoError(t, err)
	current, err := Take([]string{mangaFile(to)})
	require.NoError(t, err)
	loc, err := Locate([]string{mangaFile(to)})
	require.NoError(t, err)
	return Compare(old, current, loc)
}

func lines(findings []Finding) []string {
	var out []string
	for _, f := range findings {
		out = append(out, f.String())
	}
	return out
}

func TestMangaSnapshotRoundTrip(t *testing.T) {
	v1, err := Take([]string{mangaFile("v1")})
	require.NoError(t, err)
	assert.Len(t, v1.Shapes, 7, "Request, Response, JobState, Chapter, Job, SearchHit, SourceError")

	path := filepath.Join(t.TempDir(), "shapes.json")
	require.NoError(t, WriteFile(path, v1))
	loaded, err := ReadFile(path)
	require.NoError(t, err)

	assert.Empty(t, Compare(v1, loaded, nil), "a snapshot read back compares equal")
	assert.Empty(t, Compare(loaded, v1, nil))

	again := filepath.Join(t.TempDir(), "again.json")
	require.NoError(t, WriteFile(again, loaded))
	a, _ := readAll(t, path)
	b, _ := readAll(t, again)
	assert.Equal(t, a, b, "the snapshot is byte-for-byte stable")

	_, err = ReadFile(filepath.Join(t.TempDir(), "missing.json"))
	assert.Error(t, err)
	_, err = Take([]string{filepath.Join(t.TempDir(), "missing.go")})
	assert.Error(t, err)
}

func TestMangaV1ToV2IsCompatible(t *testing.T) {
	findings := compareManga(t, "v1", "v2")
	assert.Equal(t, 0, CountBreaking(findings), lines(findings))
	assert.Equal(t, []string{
		mangaFile("v2") + ":65: compatible: Request: variant CancelJob added: a new operation old clients never send",
		mangaFile("v2") + `:170: compatible: Job.Cancelled: field "cancelled" added; old clients ignore unknown keys`,
	}, lines(findings))
}

func TestMangaV1ToV3HasFiveBreakingChanges(t *testing.T) {
	findings := compareManga(t, "v1", "v3")
	assert.Equal(t, 5, CountBreaking(findings))
	f := mangaFile("v3")
	assert.Equal(t, []string{
		f + ":31: breaking: Request: variant ListJobs removed; clients sending it get ErrUnknownType from the server (unknown_request)",
		f + ":38: breaking: Search: answer type changed f.Returns[*SearchResults] → f.Returns[*Series]; every client's send(Search) decodes the old answer and fails",
		f + `:39: breaking: Search.Query: json name changed "query" → "q"; old messages carry "query" and the reader sees the zero value`,
		f + ":43: breaking: Search.Limit: type changed int → string",
		f + ":89: breaking: Series.Chapters: became a pointer; Optional in Swift, nullable in TypeScript, and the server may now send null",
	}, lines(findings))
}

func TestMangaV1ToV4RenamesAVariant(t *testing.T) {
	findings := compareManga(t, "v1", "v4")
	assert.Equal(t, 1, CountBreaking(findings))
	require.Len(t, findings, 1, lines(findings))
	assert.Equal(t, mangaFile("v4")+":72: breaking: Response: variant Info renamed to ServerInfo: $type api.Info becomes api.ServerInfo; old clients decoding it throw MkunionJSON.Error.unknownType (Swift) or see an object without a known key (TypeScript)", findings[0].String())
}

func TestMangaDirections(t *testing.T) {
	v1, err := Take([]string{mangaFile("v1")})
	require.NoError(t, err)
	ix := newIndex(v1)
	assert.Equal(t, dirRequest, ix.directionOf("Request"))
	assert.Equal(t, dirRequest, ix.directionOf("Search"), "a variant has its union's direction")
	assert.Equal(t, dirResponse, ix.directionOf("Response"))
	assert.Equal(t, dirResponse, ix.directionOf("Job"), "reached through Jobs.Items")
	assert.Equal(t, dirResponse, ix.directionOf("JobState"), "reached through Job.State")
	assert.Equal(t, dirResponse, ix.directionOf("Chapter"))
	assert.Equal(t, Direction(0), ix.directionOf("Nope"))
	assert.Equal(t, "api", ix.pkg)
}

func TestLocate(t *testing.T) {
	loc, err := Locate([]string{mangaFile("v1")})
	require.NoError(t, err)
	file, line := loc.Type("Request")
	assert.Equal(t, mangaFile("v1"), file)
	assert.Equal(t, 31, line, "a union is at its type group, under the //go:tag")
	_, line = loc.Type("Search")
	assert.Equal(t, 37, line)
	_, line = loc.Field("Search", "Query")
	assert.Equal(t, 39, line)
	_, line = loc.Field("GetInfo", "Returns")
	assert.Equal(t, 33, line, "an embedded field is found by its type name")
	_, line = loc.Field("Search", "Gone")
	assert.Equal(t, 37, line, "a missing field falls back to its type")
	file, line = loc.Type("Gone")
	assert.Equal(t, mangaFile("v1"), file)
	assert.Equal(t, 0, line, "a missing type falls back to the first file")

	var none *Locator
	file, line = none.Field("Search", "Query")
	assert.Equal(t, "", file)
	assert.Equal(t, 0, line)

	_, err = Locate([]string{filepath.Join(t.TempDir(), "missing.go")})
	assert.Error(t, err)
}

func TestLocateEmbeddedFields(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "x.go")
	src := `package x

import "github.com/widmogrod/mkunion/f"

//go:tag mkunion:"Op"
type A struct {
	*f.Returns[*B]
	Box[int, string]
	f.Returns[*B]
	Name string
}

type B struct{}
`
	require.NoError(t, writeFile(file, src))
	loc, err := Locate([]string{file})
	require.NoError(t, err)
	_, line := loc.Field("A", "Returns")
	assert.Equal(t, 9, line, "the last embedded Returns wins; both are found by type name")
	_, line = loc.Field("A", "Box")
	assert.Equal(t, 8, line)
	_, line = loc.Field("A", "Name")
	assert.Equal(t, 10, line)
	_, line = loc.Type("Op")
	assert.Equal(t, 6, line, "a tag on a single type spec names the union at the spec")
}
