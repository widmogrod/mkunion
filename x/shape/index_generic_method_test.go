package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexedTypeWalker_genericMethodTypeParamsAreNotTypes(t *testing.T) {
	walker := NewIndexedTypeWalkerWithContentBody(`package example

type Box[T any] struct{ Value T }

type Fx struct{}

// Do is a generic method: R belongs to the method, T to nothing here.
func (fx Fx) Do[R any](in Box[R]) R { return in.Value }

// Free is a generic function, for comparison.
func Free[R any](in Box[R]) R { return in.Value }

// Deep hides the type parameter inside a pointer, a slice and a map.
func Deep[Op any](sink *[]Op, index map[string]Box[Op]) *[]Box[Op] { return nil }

// Used is a real instantiation and must be indexed.
func Used(in Box[string]) Box[string] { return in }
`, func(x *IndexedTypeWalker) { x.SetPkgImportName("github.com/widmogrod/mkunion/example") })

	var names []string
	for name := range walker.IndexedShapes() {
		names = append(names, name)
	}
	assert.NotContains(t, names, "github.com/widmogrod/mkunion/example.Box[R]", "R is a type parameter of a method, not a type")
	assert.Contains(t, names, "github.com/widmogrod/mkunion/example.Box[string]", "a real instantiation is still indexed")
	for _, name := range names {
		assert.NotContains(t, name, "example.Op", "a type parameter inside a slice, map or pointer is not a type either: %s", name)
	}
}
