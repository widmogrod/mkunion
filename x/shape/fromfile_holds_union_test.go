package shape

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func structNames(structs []*StructLike) []string {
	var names []string
	for _, x := range structs {
		names = append(names, x.Name)
	}
	return names
}

func TestInferredInfo_RetrieveStructsHoldingUnion(t *testing.T) {
	// Example is declared in type_example.go, a sibling of the fixture, so the
	// walk has to find the union on disk; the inferred file does not declare it.
	inferred, err := InferFromFile("testasset/type_holder.go")
	require.NoError(t, err)

	names := structNames(inferred.RetrieveStructsHoldingUnion())
	assert.Equal(t, []string{"Holder", "ViaAlias"}, names,
		"Plain holds nothing, OptedOut carries a serde tag")

	var aliases []string
	for _, x := range inferred.RetrieveAliasesHoldingUnion() {
		aliases = append(aliases, x.Name)
	}
	assert.Equal(t, []string{"Examples"}, aliases, "a named list of the union gets its own serde")
}

func TestInferredInfo_RetrieveStructsHoldingUnion_generatedSerde(t *testing.T) {
	// the serde mkunion generated for the fixture is the end-to-end check:
	// methods for the holders, nothing for Plain or the opted-out struct
	content, err := os.ReadFile("testasset/type_holder_serde_gen.go")
	require.NoError(t, err)

	serde := string(content)
	assert.Contains(t, serde, "func (r *Holder) MarshalJSON")
	assert.Contains(t, serde, "func (r *ViaAlias) MarshalJSON")
	assert.Contains(t, serde, "func (r *Examples) MarshalJSON")
	assert.NotContains(t, serde, "func (r *Plain) MarshalJSON")
	assert.NotContains(t, serde, "func (r *OptedOut) MarshalJSON")
}

func TestInferredInfo_RetrieveStructsHoldingUnion_sameFile(t *testing.T) {
	source := `package job

//go:tag mkunion:"State"
type (
	Queued  struct{}
	Done    struct{ Code int }
	Retries []State
)

// Job holds the union directly
type Job struct{ State State }

// Pipeline reaches the union through a named list
type Steps []State
type Pipeline struct{ Steps Steps }

// Batch reaches the union through an inline struct
type Batch struct {
	Inline struct{ Nested []State }
}

// Jobs refers to a named struct: Job has its own methods, the walk stops
type Jobs struct{ Items map[string]*Job }

// Tail refers to variants: they have their own methods, the walk stops
type Tail struct {
	Last Done
	Rest Retries
}

// Ring is recursive; the walk must end
type Ring []Ring
type Cycle struct{ R Ring }

// Keep is the user's call: it carries a serde tag
//
//go:tag serde:"json"
type Keep struct{ State State }
`
	inferred, err := InferFromFileWithContentBody(source, "example.com/job")
	require.NoError(t, err)

	names := structNames(inferred.RetrieveStructsHoldingUnion())
	assert.Equal(t, []string{"Batch", "Job", "Pipeline"}, names)
}
