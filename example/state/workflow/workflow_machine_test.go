package workflow

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func TestExecution(t *testing.T) {
	program := &Flow{
		Name: "hello_world_flow",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				ID:  "assign1",
				Var: "res",
				Val: &Apply{ID: "apply1", Name: "concat", Args: []Reshaper{
					&SetValue{Value: schema.MkString("hello ")},
					&GetValue{Path: "input"},
				}},
			},
			&End{
				ID:     "end1",
				Result: &GetValue{Path: "res"},
			},
		},
	}

	di := &DI{
		FindWorkflowF: func(flowID string) (*Flow, error) {
			return program, nil
		},
	}

	result := ExecuteAll(di.NewContext(), program)
	assert.Equal(t, &Done{
		Result: schema.MkString("hello world"),
	}, result)
}
