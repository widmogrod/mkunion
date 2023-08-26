package workflow

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/machine"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
	"time"
)

var functions = map[string]Function{
	"concat": func(body *FunctionInput) (*FunctionOutput, error) {
		args := body.Args
		a, ok := schema.As[string](args[0])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}
		b, ok := schema.As[string](args[1])
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}

		return &FunctionOutput{
			Result: schema.MkString(a + b),
		}, nil
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

	context := &BaseState{
		Flow: program,
		Variables: map[string]schema.Schema{
			"input": schema.MkString("world"),
		},
		ExprResult: make(map[string]schema.Schema),
	}

	result := ExecuteAll(context, program, di)
	assert.Equal(t, &Done{
		StepID: "end1",
		Result: schema.MkString("hello world"),
		BaseState: &BaseState{
			Flow: program,
			Variables: map[string]schema.Schema{
				"input": schema.MkString("world"),
				"res":   schema.MkString("hello world"),
			},
			ExprResult: make(map[string]schema.Schema),
		},
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

	program_await := &Flow{
		Name: "hello_world_flow_await",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				ID:  "assign1",
				Var: "res",
				Val: &Apply{
					ID:   "apply1",
					Name: "concat",
					Args: []Reshaper{
						&SetValue{Value: schema.MkString("hello ")},
						&GetValue{Path: "input"},
					},
					Await: &ApplyAwaitOptions{
						Timeout: time.Second * 10,
					},
				},
			},
			&End{
				ID:     "end1",
				Result: &GetValue{Path: "res"},
			},
		},
	}

	di := &DI{
		FindWorkflowF: func(flowID string) (*Flow, error) {
			switch flowID {
			case "hello_world_flow":
				return program, nil
			case "hello_world_flow_await":
				return program_await, nil
			}
			return nil, fmt.Errorf("flow %s not found", flowID)
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
				BaseState: &BaseState{
					Flow: program,
					Variables: map[string]schema.Schema{
						"input": schema.MkString("world"),
						"res":   schema.MkString("hello world"),
					},
					ExprResult: make(map[string]schema.Schema),
				},
			})
	})
	suite.Case("start execution that awaits for callback", func(c *machine.Case[Command, Status]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow_await"},
				Input: schema.MkString("world"),
			}).
			ThenState(&Await{
				StepID:     "apply1",
				Timeout:    10 * time.Second,
				CallbackID: "asdf",
				BaseState: &BaseState{
					Flow: program_await,
					Variables: map[string]schema.Schema{
						"input": schema.MkString("world"),
					},
					ExprResult: make(map[string]schema.Schema),
				},
			}).
			ForkCase("callback received", func(c *machine.Case[Command, Status]) {
				// Assuming that callback is received before timeout.
				c.
					GivenCommand(&Callback{
						CallbackID: "asdf",
						Result:     schema.MkString("hello + world"),
					}).
					ThenState(&Done{
						StepID: "end1",
						Result: schema.MkString("hello + world"),
						BaseState: &BaseState{
							Flow: program_await,
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
								"res":   schema.MkString("hello + world"),
							},
							ExprResult: map[string]schema.Schema{
								"apply1": schema.MkString("hello + world"),
							},
						},
					})
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
				BaseState: &BaseState{
					Flow: program,
					Variables: map[string]schema.Schema{
						"input": nil,
					},
					ExprResult: map[string]schema.Schema{},
				},
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
				BaseState: &BaseState{
					Flow: program,
					Variables: map[string]schema.Schema{
						"input": schema.MkString("world"),
					},
					ExprResult: map[string]schema.Schema{},
				},
			})
	})

	suite.Run(t)
	suite.Fuzzy(t)

	if true || suite.AssertSelfDocumentStateDiagram(t, "machine") {
		suite.SelfDocumentStateDiagram(t, "machine")
	}
}
