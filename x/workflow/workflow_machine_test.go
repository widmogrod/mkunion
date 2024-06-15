package workflow

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/machine"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
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
				ID:    "assign1",
				VarOk: "res",
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

	runId := "1"

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
		GenerateRunIDF: func() string {
			return runId
		},
	}

	store := schemaless.NewInMemoryRepository[schema.Schema]()
	repo := typedful.NewTypedRepository[State](store)
	state, err := repo.Get("1", "workflow")
	assert.ErrorIs(t, err, schemaless.ErrNotFound)

	work := NewMachine(di, state.Data)
	err = work.Handle(nil, &Run{
		Flow:  &FlowRef{FlowID: "hello_world_flow"},
		Input: schema.MkString("world"),
	})
	assert.NoError(t, err)

	newState := work.State()
	_, err = repo.UpdateRecords(schemaless.Save(schemaless.Record[State]{
		ID:   "1",
		Type: "workflow",
		Data: newState,
	}))
	assert.NoError(t, err)

	assert.Equal(t, &Done{
		Result: schema.MkString("hello world"),
		BaseState: BaseState{
			RunID:  runId,
			StepID: "end1",
			Flow:   &FlowRef{FlowID: "hello_world_flow"},
			Variables: map[string]schema.Schema{
				"input": schema.MkString("world"),
				"res":   schema.MkString("hello world"),
			},
			ExprResult:        make(map[string]schema.Schema),
			DefaultMaxRetries: 3,
		},
	}, newState)

	state, err = repo.Get("1", "workflow")
	assert.NoError(t, err)

	work = NewMachine(di, state.Data)
	err = work.Handle(nil, &Run{
		Flow:  &FlowRef{FlowID: "hello_world_flow"},
		Input: schema.MkString("world"),
	})
	assert.ErrorIs(t, err, ErrStateReachEnd)
}

func TestMachine(t *testing.T) {
	program := &Flow{
		Name: "hello_world_flow",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				VarOk: "res",
				Val: &Apply{Name: "concat", Args: []Reshaper{
					&SetValue{Value: schema.MkString("hello ")},
					&GetValue{Path: "input"},
				}},
			},
			&End{
				Result: &GetValue{Path: "res"},
			},
		},
	}

	callbackID := "callback1"
	runID := "123"

	program_await := &Flow{
		Name: "hello_world_flow_await",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				VarOk: "res",
				Val: &Apply{
					Name: "concat",
					Args: []Reshaper{
						&SetValue{Value: schema.MkString("hello ")},
						&GetValue{Path: "input"},
					},
					Await: &ApplyAwaitOptions{
						TimeoutSeconds: 10,
					},
				},
			},
			&End{
				Result: &GetValue{Path: "res"},
			},
		},
	}

	program_if := &Flow{
		Name: "hello_world_flow_if",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				VarOk: "res",
				Val: &Apply{ID: "apply1", Name: "concat", Args: []Reshaper{
					&SetValue{Value: schema.MkString("hello ")},
					&GetValue{Path: "input"},
				}},
			},
			&Choose{
				If: &Compare{
					Operation: "=",
					Left:      &GetValue{Path: "res"},
					Right:     &SetValue{Value: schema.MkString("hello world")},
				},
				Then: []Expr{
					&End{
						Result: &GetValue{Path: "res"},
					},
				},
				Else: []Expr{
					&End{
						Result: &SetValue{Value: schema.MkString("only Spanish will work!")},
					},
				},
			},
		},
	}

	timeNow := time.Now()
	di := &DI{
		FindWorkflowF: func(flowID string) (*Flow, error) {
			switch flowID {
			case "hello_world_flow":
				return program, nil
			case "hello_world_flow_await":
				return program_await, nil
			case "hello_world_flow_if":
				return program_if, nil
			}
			return nil, fmt.Errorf("flow %s not found", flowID)
		},
		FindFunctionF: func(funcID string) (Function, error) {
			if fn, ok := functions[funcID]; ok {
				return fn, nil
			}

			return nil, fmt.Errorf("function %s not found", funcID)
		},

		GenerateCallbackIDF: func() string {
			return callbackID
		},

		GenerateRunIDF: func() string {
			return runID
		},
		MockTimeNow: &timeNow,
	}

	suite := machine.NewTestSuite[Dependency](di, NewMachine)

	suite.Case(t, "start execution", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow"},
				Input: schema.MkString("world"),
			}).
			ThenState(t, &Done{
				Result: schema.MkString("hello world"),
				BaseState: BaseState{
					RunID:  runID,
					StepID: "end",
					Flow:   &FlowRef{FlowID: "hello_world_flow"},
					Variables: map[string]schema.Schema{
						"input": schema.MkString("world"),
						"res":   schema.MkString("hello world"),
					},
					ExprResult:        make(map[string]schema.Schema),
					DefaultMaxRetries: 3,
				},
			})
	})
	suite.Case(t, "start scheduled execution delay 10s", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow"},
				Input: schema.MkString("world"),
				RunOption: &DelayRun{
					DelayBySeconds: 10,
				},
			}).
			ThenState(t, &Scheduled{
				ExpectedRunTimestamp: di.TimeNow().Add(time.Duration(10) * time.Second).Unix(),
				BaseState: BaseState{
					RunID:  runID,
					StepID: "",
					Flow:   &FlowRef{FlowID: "hello_world_flow"},
					Variables: map[string]schema.Schema{
						"input": schema.MkString("world"),
					},
					ExprResult:        make(map[string]schema.Schema),
					DefaultMaxRetries: 3,
					RunOption: &DelayRun{
						DelayBySeconds: 10,
					},
				},
			}).
			ForkCase(t, "resume execution", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					GivenCommand(&Run{}).
					ThenState(t, &Done{
						Result: schema.MkString("hello world"),
						BaseState: BaseState{
							RunID:  runID,
							StepID: "end",
							Flow:   &FlowRef{FlowID: "hello_world_flow"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
								"res":   schema.MkString("hello world"),
							},
							ExprResult:        make(map[string]schema.Schema),
							DefaultMaxRetries: 3,
							RunOption: &DelayRun{
								DelayBySeconds: 10,
							},
						},
					})
			}).
			ForkCase(t, "stop execution", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					GivenCommand(&StopSchedule{
						ParentRunID: runID,
					}).
					ThenState(t, &ScheduleStopped{
						BaseState: BaseState{
							RunID:  runID,
							StepID: "",
							Flow:   &FlowRef{FlowID: "hello_world_flow"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
							},
							ExprResult:        make(map[string]schema.Schema),
							DefaultMaxRetries: 3,
							RunOption: &DelayRun{
								DelayBySeconds: 10,
							},
						},
					}).
					GivenCommand(&ResumeSchedule{
						ParentRunID: runID,
					}).
					ThenState(t, &Scheduled{
						ExpectedRunTimestamp: di.TimeNow().Add(time.Duration(10) * time.Second).Unix(),
						BaseState: BaseState{
							RunID:  runID,
							StepID: "",
							Flow:   &FlowRef{FlowID: "hello_world_flow"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
							},
							ExprResult:        make(map[string]schema.Schema),
							DefaultMaxRetries: 3,
							RunOption: &DelayRun{
								DelayBySeconds: 10,
							},
						},
					})
			})
	})
	suite.Case(t, "start execution that awaits for callback", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow_await"},
				Input: schema.MkString("world"),
			}).
			ThenState(t, &Await{
				ExpectedTimeoutTimestamp: time.Now().Add(10 * time.Second).Unix(),
				CallbackID:               callbackID,
				BaseState: BaseState{
					RunID:  runID,
					StepID: "apply-concat",
					Flow:   &FlowRef{FlowID: "hello_world_flow_await"},
					Variables: map[string]schema.Schema{
						"input": schema.MkString("world"),
					},
					ExprResult:        make(map[string]schema.Schema),
					DefaultMaxRetries: 3,
				},
			}).
			ForkCase(t, "cannot expire await callback, when timeout valid", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					GivenCommand(&ExpireAsync{
						RunID: runID,
					}).
					ThenStateAndError(t, &Await{
						ExpectedTimeoutTimestamp: time.Now().Add(10 * time.Second).Unix(),
						CallbackID:               callbackID,
						BaseState: BaseState{
							RunID:  runID,
							StepID: "apply-concat",
							Flow:   &FlowRef{FlowID: "hello_world_flow_await"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
							},
							ExprResult:        make(map[string]schema.Schema),
							DefaultMaxRetries: 3,
						},
					}, ErrCannotExpireAsync)
			}).
			ForkCase(t, "callback not received within timeout, expire is allowed", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					BeforeCommand(func(t testing.TB, di Dependency) {
						future := time.Now().Add(11 * time.Second)
						di.(*DI).MockTimeNow = &future
					}).
					GivenCommand(&ExpireAsync{
						RunID: runID,
					}).
					AfterCommand(func(t testing.TB, di Dependency) {
						di.(*DI).MockTimeNow = &timeNow
					}).
					ThenState(t, &Error{
						Code:    "async-timeout",
						Reason:  "callback not received in time window",
						Retried: 3,
						BaseState: BaseState{
							RunID:  runID,
							StepID: "apply-concat",
							Flow:   &FlowRef{FlowID: "hello_world_flow_await"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
							},
							ExprResult:        make(map[string]schema.Schema),
							DefaultMaxRetries: 3,
						},
					})
			}).
			ForkCase(t, "callback received, within timeout", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					GivenCommand(&Callback{
						CallbackID: callbackID,
						Result:     schema.MkString("hello + world"),
					}).
					ThenState(t, &Done{
						Result: schema.MkString("hello + world"),
						BaseState: BaseState{
							RunID:  runID,
							StepID: "end",
							Flow:   &FlowRef{FlowID: "hello_world_flow_await"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
								"res":   schema.MkString("hello + world"),
							},
							ExprResult: map[string]schema.Schema{
								"apply-concat": schema.MkString("hello + world"),
							},
							DefaultMaxRetries: 3,
						},
					})
			}).
			ForkCase(t, "callback received, after timeout", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					BeforeCommand(func(t testing.TB, di Dependency) {
						future := time.Now().Add(11 * time.Second)
						di.(*DI).MockTimeNow = &future
					}).
					GivenCommand(&Callback{
						CallbackID: callbackID,
						Result:     schema.MkString("hello + world"),
					}).
					AfterCommand(func(t testing.TB, di Dependency) {
						di.(*DI).MockTimeNow = &timeNow
					}).
					ThenStateAndError(t, &Await{
						ExpectedTimeoutTimestamp: time.Now().Add(10 * time.Second).Unix(),
						CallbackID:               callbackID,
						BaseState: BaseState{
							RunID:  runID,
							StepID: "apply-concat",
							Flow:   &FlowRef{FlowID: "hello_world_flow_await"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
							},
							ExprResult:        make(map[string]schema.Schema),
							DefaultMaxRetries: 3,
						},
					}, ErrCallbackExpired)
			}).
			ForkCase(t, "received invalid callbackID", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					GivenCommand(&Callback{
						CallbackID: "invalid_callback_id",
						Result:     schema.MkString("hello + world"),
					}).
					ThenStateAndError(t, &Await{
						ExpectedTimeoutTimestamp: timeNow.Add(10 * time.Second).Unix(),
						CallbackID:               callbackID,
						BaseState: BaseState{
							RunID:  runID,
							StepID: "apply-concat",
							Flow:   &FlowRef{FlowID: "hello_world_flow_await"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
							},
							ExprResult:        make(map[string]schema.Schema),
							DefaultMaxRetries: 3,
						},
					}, ErrCallbackNotMatch)
			})
	})
	suite.Case(t, "start execution no input variable", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
		c.
			GivenCommand(&Run{
				Flow: &FlowRef{FlowID: "hello_world_flow"},
			}).
			ThenState(t, &Error{
				Code:   "function-execution",
				Reason: "function concat() returned error: expected string, got <nil>",
				BaseState: BaseState{
					RunID:  runID,
					StepID: "apply-concat",
					Flow:   &FlowRef{FlowID: "hello_world_flow"},
					Variables: map[string]schema.Schema{
						"input": nil,
					},
					ExprResult:        map[string]schema.Schema{},
					DefaultMaxRetries: 3,
				},
			})
	})
	suite.Case(t, "start execution fails on non existing flowID", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow_non_existing"},
				Input: schema.MkString("world"),
			}).
			ThenStateAndError(t, nil, ErrFlowNotFound)
	})
	suite.Case(t, "start execution fails on function retrival", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow"},
				Input: schema.MkString("world"),
			}).
			BeforeCommand(func(t testing.TB, di Dependency) {
				di.(*DI).FindFunctionF = func(funcID string) (Function, error) {
					return nil, fmt.Errorf("function funcID='%s' not found", funcID)
				}
			}).
			AfterCommand(func(t testing.TB, di Dependency) {
				di.(*DI).FindFunctionF = func(funcID string) (Function, error) {
					if fn, ok := functions[funcID]; ok {
						return fn, nil
					}

					return nil, fmt.Errorf("function %s not found", funcID)
				}
			}).
			ThenState(t, &Error{
				Code:   "function-missing",
				Reason: "function concat() not found, details: function funcID='concat' not found",
				BaseState: BaseState{
					RunID:  runID,
					StepID: "apply-concat",
					Flow:   &FlowRef{FlowID: "hello_world_flow"},
					Variables: map[string]schema.Schema{
						"input": schema.MkString("world"),
					},
					ExprResult:        map[string]schema.Schema{},
					DefaultMaxRetries: 3,
				},
			}).
			ForkCase(t, "retry execution", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					GivenCommand(&TryRecover{
						RunID: runID,
					}).
					BeforeCommand(func(t testing.TB, di Dependency) {
						di.(*DI).FindFunctionF = func(funcID string) (Function, error) {
							return nil, fmt.Errorf("function funcID='%s' not found", funcID)
						}
					}).
					AfterCommand(func(t testing.TB, di Dependency) {
						di.(*DI).FindFunctionF = func(funcID string) (Function, error) {
							if fn, ok := functions[funcID]; ok {
								return fn, nil
							}

							return nil, fmt.Errorf("function %s not found", funcID)
						}
					}).
					ThenState(t, &Error{
						Code:    "function-missing",
						Reason:  "function concat() not found, details: function funcID='concat' not found",
						Retried: 1,
						BaseState: BaseState{
							RunID:  runID,
							StepID: "apply-concat",
							Flow:   &FlowRef{FlowID: "hello_world_flow"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("world"),
							},
							ExprResult:        map[string]schema.Schema{},
							DefaultMaxRetries: 3,
						},
					})
			})
	})
	suite.Case(t, "execute function with if statement", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow_if"},
				Input: schema.MkString("El Mundo"),
			}).
			ThenState(t, &Done{
				Result: schema.MkString("only Spanish will work!"),
				BaseState: BaseState{
					RunID:  runID,
					StepID: "end-1",
					Flow:   &FlowRef{FlowID: "hello_world_flow_if"},
					Variables: map[string]schema.Schema{
						"input": schema.MkString("El Mundo"),
						"res":   schema.MkString("hello El Mundo"),
					},
					ExprResult:        map[string]schema.Schema{},
					DefaultMaxRetries: 3,
				},
			})
	})
	suite.Case(t, "scheduled run", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
		c.
			GivenCommand(&Run{
				Flow:  &FlowRef{FlowID: "hello_world_flow_if"},
				Input: schema.MkString("El Mundo"),
				RunOption: &ScheduleRun{
					Interval: "@every 1s",
				},
			}).
			ThenState(t, &Scheduled{
				ExpectedRunTimestamp: di.TimeNow().Add(time.Duration(1) * time.Second).Unix(),
				BaseState: BaseState{
					RunID:  runID,
					StepID: "",
					Flow:   &FlowRef{FlowID: "hello_world_flow_if"},
					Variables: map[string]schema.Schema{
						"input": schema.MkString("El Mundo"),
					},
					ExprResult:        map[string]schema.Schema{},
					DefaultMaxRetries: 3,
					RunOption: &ScheduleRun{
						Interval:    "@every 1s",
						ParentRunID: runID,
					},
				},
			}).
			ForkCase(t, "run scheduled run", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					GivenCommand(&Run{}).
					ThenState(t, &Done{
						Result: schema.MkString("only Spanish will work!"),
						BaseState: BaseState{
							RunID:  runID,
							StepID: "end-1",
							Flow:   &FlowRef{FlowID: "hello_world_flow_if"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("El Mundo"),
								"res":   schema.MkString("hello El Mundo"),
							},
							ExprResult:        map[string]schema.Schema{},
							DefaultMaxRetries: 3,
							RunOption: &ScheduleRun{
								Interval:    "@every 1s",
								ParentRunID: runID,
							},
						},
					})
			}).
			ForkCase(t, "stop scheduled run", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
				c.
					GivenCommand(&StopSchedule{
						ParentRunID: runID,
					}).
					ThenState(t, &ScheduleStopped{
						BaseState: BaseState{
							RunID:  runID,
							StepID: "",
							Flow:   &FlowRef{FlowID: "hello_world_flow_if"},
							Variables: map[string]schema.Schema{
								"input": schema.MkString("El Mundo"),
							},
							ExprResult:        map[string]schema.Schema{},
							DefaultMaxRetries: 3,
							RunOption: &ScheduleRun{
								Interval:    "@every 1s",
								ParentRunID: runID,
							},
						},
					}).
					ForkCase(t, "run stopped", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
						c.
							GivenCommand(&Run{}).
							ThenStateAndError(t, &ScheduleStopped{
								BaseState: BaseState{
									RunID:  runID,
									StepID: "",
									Flow:   &FlowRef{FlowID: "hello_world_flow_if"},
									Variables: map[string]schema.Schema{
										"input": schema.MkString("El Mundo"),
									},
									ExprResult:        map[string]schema.Schema{},
									DefaultMaxRetries: 3,
									RunOption: &ScheduleRun{
										Interval:    "@every 1s",
										ParentRunID: runID,
									},
								},
							}, ErrInvalidStateTransition)
					}).
					ForkCase(t, "resume scheduled run", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
						c.
							GivenCommand(&ResumeSchedule{
								ParentRunID: runID,
							}).
							ThenState(t, &Scheduled{
								ExpectedRunTimestamp: di.TimeNow().Add(time.Duration(1) * time.Second).Unix(),
								BaseState: BaseState{
									RunID:  runID,
									StepID: "",
									Flow:   &FlowRef{FlowID: "hello_world_flow_if"},
									Variables: map[string]schema.Schema{
										"input": schema.MkString("El Mundo"),
									},
									ExprResult:        map[string]schema.Schema{},
									DefaultMaxRetries: 3,
									RunOption: &ScheduleRun{
										Interval:    "@every 1s",
										ParentRunID: runID,
									},
								},
							})
					})
			})
	})

	if suite.AssertSelfDocumentStateDiagram(t, "machine") {
		suite.SelfDocumentStateDiagram(t, "machine")
	}
}

func TestBaseState_Serde(t *testing.T) {
	subject := BaseState{
		RunID:  "qrwqer",
		StepID: "end-1",
		Flow:   &FlowRef{FlowID: "hello_world_flow_if"},
		Variables: map[string]schema.Schema{
			"input": schema.MkString("El Mundo"),
			"res":   schema.MkString("hello El Mundo"),
		},
		ExprResult:        map[string]schema.Schema{},
		DefaultMaxRetries: 3,
	}

	result, err := shared.JSONMarshal[BaseState](subject)
	assert.NoError(t, err)

	output, err := shared.JSONUnmarshal[BaseState](result)
	assert.NoError(t, err)

	if diff := cmp.Diff(subject, output); diff != "" {
		t.Errorf("BaseState mismatch (-want +got):\n%s", diff)
	}

	expectedJSON := `{
  "DefaultMaxRetries": 3,
  "ExprResult": {},
  "Flow": {
    "$type": "workflow.FlowRef",
    "workflow.FlowRef": {
      "FlowID": "hello_world_flow_if"
    }
  },
  "RunID": "qrwqer",
  "RunOption": null,
  "StepID": "end-1",
  "Variables": {
    "input": {
      "$type": "schema.String",
      "schema.String": "El Mundo"
    },
    "res": {
      "$type": "schema.String",
      "schema.String": "hello El Mundo"
    }
  }
}
`
	if !assert.JSONEq(t, expectedJSON, string(result)) {
		t.Log(string(result))
	}

}

func TestState_Serde(t *testing.T) {
	subject := &Done{
		Result: schema.MkString("only Spanish will work!"),
		BaseState: BaseState{
			RunID:  "qrwqer",
			StepID: "end-1",
			Flow:   &FlowRef{FlowID: "hello_world_flow_if"},
			Variables: map[string]schema.Schema{
				"input": schema.MkString("El Mundo"),
				"res":   schema.MkString("hello El Mundo"),
			},
			ExprResult:        map[string]schema.Schema{},
			DefaultMaxRetries: 3,
		},
	}

	result, err := shared.JSONMarshal[State](subject)
	assert.NoError(t, err)

	output, err := shared.JSONUnmarshal[State](result)
	assert.NoError(t, err)

	if diff := cmp.Diff(subject, output); diff != "" {
		t.Errorf("State mismatch (-want +got):\n%s", diff)
	}

	expectedJSON := `{
  "$type": "workflow.Done",
  "workflow.Done": {
    "BaseState": {
      "DefaultMaxRetries": 3,
      "ExprResult": {},
      "Flow": {
        "$type": "workflow.FlowRef",
        "workflow.FlowRef": {
          "FlowID": "hello_world_flow_if"
        }
      },
      "RunID": "qrwqer",
      "RunOption": null,
      "StepID": "end-1",
      "Variables": {
        "input": {
          "$type": "schema.String",
          "schema.String": "El Mundo"
        },
        "res": {
          "$type": "schema.String",
          "schema.String": "hello El Mundo"
        }
      }
    },
    "Result": {
      "$type": "schema.String",
      "schema.String": "only Spanish will work!"
    }
  }
}
`
	if !assert.JSONEq(t, expectedJSON, string(result)) {
		t.Log(string(result))
	}
}
