package generators

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestGenerateTypeRegistry(t *testing.T) {
	log.SetLevel(log.ErrorLevel)
	contents := `package testutils

import (
	. "github.com/widmogrod/mkunion/f"
)

type User struct{ Name string }

type APIError struct {
	Code    int
	Message string
}

// --8<-- [start:fetch-type]

// FetchResult combine unions for rich error handling
type FetchResult = Result[Option[User], APIError]

// handleFetch uses nested pattern matching to handle result
func handleFetch(result FetchResult) string {
	return MatchResultR1(result,
		func(ok *Ok[Option[User], APIError]) string {
			return MatchOptionR1(ok.Value,
				func(*None[User]) string { return "User not found" },
				func(some *Some[User]) string {
					return fmt.Sprintf("Found user: %s", some.Value.Name)
				},
			)
		},
		func(err *Err[Option[User], APIError]) string {
			return fmt.Sprintf("API error: %v", err.Error)
		},
	)
}
`

	pkgName := "github.com/widmogrod/mkunion/x/generators/testutils"

	inferred, err := shape.InferFromFileWithContentBody(contents, pkgName)
	require.NoError(t, err)

	// Used to generate type registry
	walker := shape.NewIndexedTypeWalkerWithContentBody(contents,
		func(x *shape.IndexedTypeWalker) {
			x.SetPkgImportName(pkgName)
		},
	)

	lookupShapes := func(ref *shape.RefName) (shape.Shape, bool) {
		for _, sh := range inferred.RetrieveShapes() {
			if shape.Name(sh) == ref.Name &&
				shape.PkgName(sh) == ref.PkgName {
				return sh, true
			}
		}
		return shape.LookupShapeOnDisk(ref)
	}

	t.Log("Print ExpandedShapes():")
	expanded := walker.ExpandedShapes()
	for _, s := range expanded {
		if ref, ok := s.(*shape.RefName); ok {
			s, found := lookupShapes(ref)
			if found {
				s = shape.IndexWith(s, ref)
			}
		}

		tagged := NewShapeTagged(s)
		str, err := tagged.generateShapeFunc(s)
		t.Log(str)
		t.Log(err)
	}

	t.Log("Print TypeRegistry:")
	buff, err3 := GenerateTypeRegistry(walker, lookupShapes)
	require.NoError(t, err3)

	typeRegistryContentns := string(buff.Bytes())
	t.Log(typeRegistryContentns)

	require.Contains(t, typeRegistryContentns, `shared.TypeRegistryStore[f.None[User]]("github.com/widmogrod/mkunion/f.None[github.com/widmogrod/mkunion/x/generators/testutils.User]"`)
	require.NotContains(t, typeRegistryContentns, `[None[User]]`)

}
