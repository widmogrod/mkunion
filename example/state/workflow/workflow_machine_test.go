package workflow

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/machine"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

var functions = map[string]Function{
	"concat": func(args []schema.Schema) (schema.Schema, error) {
		return schema.MkString(
			schema.AsDefault(args[0], "") +
				schema.AsDefault(args[1], ""),
		), nil
	},
}

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
		FindFunctionF: func(funcID string) (Function, error) {
			if fn, ok := functions[funcID]; ok {
				return fn, nil
			}

			return nil, fmt.Errorf("function %s not found", funcID)
		},
	}

	context := di.NewContext()
	err := context.SetVariable("input", schema.MkString("world"))
	assert.NoError(t, err)

	result := ExecuteAll(context, program)
	assert.Equal(t, &Done{
		StepID: "end1",
		Result: schema.MkString("hello world"),
	}, result)
}

func TestMachine(t *testing.T) {
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
		FindFunctionF: func(funcID string) (Function, error) {
			if fn, ok := functions[funcID]; ok {
				return fn, nil
			}

			return nil, fmt.Errorf("function %s not found", funcID)
		},
	}

	suite := machine.NewTestSuite(func() *machine.Machine[Command, Status] {
		return machine.NewSimpleMachine(func(cmd Command, state Status) (Status, error) {
			return Transition(cmd, state, di)
		})
	})

	suite.Case("start execution", func(c *machine.Case[Command, Status]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow"},
				Input: schema.MkString("world"),
			}).
			ThenState(&Done{
				StepID: "end1",
				Result: schema.MkString("hello world"),
			})
	})
	suite.Case("start execution fails on function retrival", func(c *machine.Case[Command, Status]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow"},
				Input: schema.MkString("world"),
			}, machine.WithBefore(func() {
				di.FindFunctionF = func(funcID string) (Function, error) {
					return nil, fmt.Errorf("function funcID='%s' not found", funcID)
				}
			}), machine.WithAfter(func() {
				di.FindFunctionF = func(funcID string) (Function, error) {
					if fn, ok := functions[funcID]; ok {
						return fn, nil
					}

					return nil, fmt.Errorf("function %s not found", funcID)
				}
			})).
			ThenState(&Error{
				StepID: "apply1",
				Code:   "function-retrieve",
				Reason: "function funcID='concat' not found",
			})
	})

	suite.Run(t)
	suite.Fuzzy(t)

	if true || suite.AssertSelfDocumentStateDiagram(t, "machine") {
		suite.SelfDocumentStateDiagram(t, "machine")
	}
}
