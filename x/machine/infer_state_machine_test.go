package machine

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInferStateMachine(t *testing.T) {
	infer := NewInferTransition[string, int]()
	infer.Record("inc", 10, 11, nil)
	infer.Record("dec", 11, 10, nil)
	infer.Record("unknown", 10, 10, fmt.Errorf("unknown cmd: unknown"))
	result := infer.ToMermaid()

	assert.Equal(t, `stateDiagram
	int: int

	int --> int: string
`, result)
}
