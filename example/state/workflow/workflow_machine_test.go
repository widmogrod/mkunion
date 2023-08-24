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
		a, ok := schema.As[string](args[0])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}
		b, ok := schema.As[string](args[1])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}

		return schema.MkString(a + b), nil
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
			if flowID != "hello_world_flow" {
				return nil, fmt.Errorf("flow %s not found", flowID)
			}
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
	suite.Case("start execution no input variable", func(c *machine.Case[Command, Status]) {
		c.
			GivenCommand(&Run{
				Flow: &FlowRef{FlowID: "hello_world_flow"},
			}).
			ThenState(&Error{
				StepID: "apply1",
				Code:   "function-execution",
				Reason: "function concat() returned error: expected string, got <nil>",
			})
	})
	suite.Case("start execution fails on non existing flowID", func(c *machine.Case[Command, Status]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow_non_existing"},
				Input: schema.MkString("world"),
			}).
			ThenStateAndError(nil, fmt.Errorf("flow hello_world_flow_non_existing not found"))
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
				Code:   "function-missing",
				Reason: "function concat() not found, details: function funcID='concat' not found",
			})
	})

	suite.Run(t)
	suite.Fuzzy(t)

	if true || suite.AssertSelfDocumentStateDiagram(t, "machine") {
		suite.SelfDocumentStateDiagram(t, "machine")
	}
}
