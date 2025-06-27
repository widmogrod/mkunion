package workflow

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
)

func TestPlanStep_ReturnStep(t *testing.T) {
	// Arrange
	step := &ReturnStep{
		StepID:   "return-1",
		FromStep: "apply-1",
	}

	// Act & Assert
	assert.Equal(t, "return-1", step.StepID)
	assert.Equal(t, "apply-1", step.FromStep)
}

func TestPlanStep_MatchFunction(t *testing.T) {
	// Arrange
	var step PlanStep = &ReturnStep{
		StepID:   "return-1",
		FromStep: "apply-1",
	}

	// Act
	result := MatchPlanStepR1(
		step,
		func(x *ExecuteStep) string {
			return x.StepID
		},
		func(x *AssignStep) string {
			return x.StepID
		},
		func(x *ReturnStep) string {
			return x.StepID
		},
	)

	// Assert
	assert.Equal(t, "return-1", result)
}

func TestExecutionPlan_Creation(t *testing.T) {
	// Arrange & Act
	plan := NewExecutionPlan()

	// Assert
	assert.NotNil(t, plan)
	assert.Empty(t, plan.Steps)
	assert.Empty(t, plan.Queue)
	assert.NotNil(t, plan.Completed)
	assert.NotNil(t, plan.Results)
	assert.NotNil(t, plan.Variables)
	assert.Empty(t, plan.ExecutedSteps)
	assert.Empty(t, plan.SkippedSteps)
}

func TestExecutionPlan_WithVariables(t *testing.T) {
	// Arrange
	plan := NewExecutionPlan()
	
	// Act
	plan.Variables["input"] = schema.MkInt(42)
	
	// Assert
	assert.Equal(t, 1, len(plan.Variables))
	val, ok := plan.Variables["input"]
	assert.True(t, ok)
	assert.Equal(t, schema.MkInt(42), val)
}

func TestPlanGenerator_EmptyFlow(t *testing.T) {
	// Arrange
	flow := &Flow{
		Name: "empty",
		Arg:  "input",
		Body: []Expr{},
	}
	input := schema.MkInt(42)
	generator := NewPlanGenerator()

	// Act
	plan := generator.GeneratePlan(flow, input)

	// Assert
	assert.NotNil(t, plan)
	assert.Empty(t, plan.Steps)
	assert.Equal(t, schema.MkInt(42), plan.Variables["input"])
}

func TestPlanGenerator_EndExpression(t *testing.T) {
	// Arrange
	flow := &Flow{
		Name: "simple",
		Arg:  "input",
		Body: []Expr{
			&End{
				ID:     "end-1",
				Result: &GetValue{Path: "input"},
			},
		},
	}
	input := schema.MkInt(42)
	generator := NewPlanGenerator()

	// Act
	plan := generator.GeneratePlan(flow, input)

	// Assert
	assert.NotNil(t, plan)
	assert.Len(t, plan.Steps, 1)
	
	// Verify the generated step
	step := plan.Steps[0]
	executeStep, ok := step.(*ExecuteStep)
	assert.True(t, ok, "Expected ExecuteStep, got %T", step)
	assert.Equal(t, "end-1", executeStep.StepID)
	
	// Verify the expression in the step
	endExpr, ok := executeStep.Expr.(*End)
	assert.True(t, ok, "Expected End expression, got %T", executeStep.Expr)
	assert.Equal(t, "end-1", endExpr.ID)
}

func TestPlanExecutor_EmptyPlan(t *testing.T) {
	// Arrange
	plan := NewExecutionPlan()
	executor := NewPlanExecutor(&testDependency{})
	
	// Act
	state, err := executor.Execute(plan)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, state)
	done, ok := state.(*Done)
	assert.True(t, ok, "Expected Done state, got %T", state)
	assert.NotNil(t, done)
}

// testDependency is a test implementation of Dependency
type testDependency struct{}

func (d *testDependency) FindWorkflow(flowID string) (*Flow, error) {
	return nil, fmt.Errorf("not implemented")
}

func (d *testDependency) FindFunction(funcID string) (Function, error) {
	switch funcID {
	case "add":
		return func(input *FunctionInput) (*FunctionOutput, error) {
			// Simple add function for testing
			if len(input.Args) != 2 {
				return nil, fmt.Errorf("add expects 2 arguments")
			}
			a := float64(*input.Args[0].(*schema.Number))
			b := float64(*input.Args[1].(*schema.Number))
			return &FunctionOutput{
				Result: schema.MkFloat(a + b),
			}, nil
		}, nil
	case "multiply":
		return func(input *FunctionInput) (*FunctionOutput, error) {
			// Simple multiply function for testing
			if len(input.Args) != 2 {
				return nil, fmt.Errorf("multiply expects 2 arguments")
			}
			a := float64(*input.Args[0].(*schema.Number))
			b := float64(*input.Args[1].(*schema.Number))
			return &FunctionOutput{
				Result: schema.MkFloat(a * b),
			}, nil
		}, nil
	case "double":
		return func(input *FunctionInput) (*FunctionOutput, error) {
			// Simple double function for testing
			if len(input.Args) != 1 {
				return nil, fmt.Errorf("double expects 1 argument")
			}
			a := float64(*input.Args[0].(*schema.Number))
			return &FunctionOutput{
				Result: schema.MkFloat(a * 2),
			}, nil
		}, nil
	case "triple":
		return func(input *FunctionInput) (*FunctionOutput, error) {
			// Simple triple function for testing
			if len(input.Args) != 1 {
				return nil, fmt.Errorf("triple expects 1 argument")
			}
			a := float64(*input.Args[0].(*schema.Number))
			return &FunctionOutput{
				Result: schema.MkFloat(a * 3),
			}, nil
		}, nil
	default:
		return nil, fmt.Errorf("function %s not found", funcID)
	}
}

func (d *testDependency) GenerateCallbackID() string {
	return "test-callback-id"
}

func (d *testDependency) GenerateRunID() string {
	return "test-run-id"
}

func (d *testDependency) MaxRetries() int64 {
	return 3
}

func (d *testDependency) TimeNow() time.Time {
	return time.Unix(1000000, 0)
}

func TestPlanExecutor_ExecuteStep_End(t *testing.T) {
	// Arrange
	plan := NewExecutionPlan()
	plan.Variables["input"] = schema.MkInt(42)
	plan.Steps = []PlanStep{
		&ExecuteStep{
			StepID: "end-1",
			Expr: &End{
				ID:     "end-1",
				Result: &SetValue{Value: schema.MkInt(42)},
			},
			DependsOn: []string{},
		},
	}
	
	executor := NewPlanExecutor(&testDependency{})
	
	// Act
	state, err := executor.Execute(plan)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, state)
	
	done, ok := state.(*Done)
	assert.True(t, ok, "Expected Done state, got %T", state)
	assert.Equal(t, schema.MkInt(42), done.Result)
}

func TestPlanGenerator_AssignExpression(t *testing.T) {
	// Arrange
	flow := &Flow{
		Name: "assign",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				ID:    "assign-1",
				VarOk: "x",
				Val:   &Apply{
					ID:   "apply-1",
					Name: "add",
					Args: []Reshaper{
						&GetValue{Path: "input"},
						&SetValue{Value: schema.MkInt(10)},
					},
				},
			},
		},
	}
	input := schema.MkInt(32)
	generator := NewPlanGenerator()

	// Act
	plan := generator.GeneratePlan(flow, input)

	// Assert
	assert.NotNil(t, plan)
	// Should have 2 steps: one for Apply, one for Assign
	assert.Len(t, plan.Steps, 2)
	
	// First step should be the Apply
	applyStep, ok := plan.Steps[0].(*ExecuteStep)
	assert.True(t, ok, "Expected ExecuteStep for Apply, got %T", plan.Steps[0])
	assert.Equal(t, "apply-1", applyStep.StepID)
	
	// Second step should be Assign
	assignStep, ok := plan.Steps[1].(*AssignStep)
	assert.True(t, ok, "Expected AssignStep, got %T", plan.Steps[1])
	assert.Equal(t, "assign-1", assignStep.StepID)
	assert.Equal(t, "x", assignStep.VarName)
	assert.Equal(t, "apply-1", assignStep.FromStep)
}

func TestPlanGenerator_ApplyExpression(t *testing.T) {
	// Arrange
	flow := &Flow{
		Name: "apply",
		Arg:  "input",
		Body: []Expr{
			&Apply{
				ID:   "apply-1",
				Name: "multiply",
				Args: []Reshaper{
					&GetValue{Path: "input"},
					&SetValue{Value: schema.MkInt(2)},
				},
			},
		},
	}
	input := schema.MkInt(21)
	generator := NewPlanGenerator()

	// Act
	plan := generator.GeneratePlan(flow, input)

	// Assert
	assert.NotNil(t, plan)
	assert.Len(t, plan.Steps, 1)
	
	// Verify the generated step
	step := plan.Steps[0]
	executeStep, ok := step.(*ExecuteStep)
	assert.True(t, ok, "Expected ExecuteStep, got %T", step)
	assert.Equal(t, "apply-1", executeStep.StepID)
	
	// Verify the expression in the step
	applyExpr, ok := executeStep.Expr.(*Apply)
	assert.True(t, ok, "Expected Apply expression, got %T", executeStep.Expr)
	assert.Equal(t, "apply-1", applyExpr.ID)
	assert.Equal(t, "multiply", applyExpr.Name)
	assert.Len(t, applyExpr.Args, 2)
}

func TestPlanExecutor_ExecuteStep_Apply(t *testing.T) {
	// Arrange
	plan := NewExecutionPlan()
	plan.Variables["input"] = schema.MkInt(10)
	plan.Steps = []PlanStep{
		&ExecuteStep{
			StepID: "apply-1",
			Expr: &Apply{
				ID:   "apply-1",
				Name: "add",
				Args: []Reshaper{
					&GetValue{Path: "input"},
					&SetValue{Value: schema.MkInt(5)},
				},
			},
			DependsOn: []string{},
		},
	}
	
	// Create test dependency with a function
	dep := &testDependency{}
	executor := NewPlanExecutor(dep)
	
	// Act
	state, err := executor.Execute(plan)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, state)
	
	// Should be Done state with no result (Apply doesn't end the flow)
	done, ok := state.(*Done)
	assert.True(t, ok, "Expected Done state, got %T", state)
	assert.Nil(t, done.Result)
	
	// Check that the Apply result was stored
	assert.Contains(t, plan.Results, "apply-1")
	assert.Contains(t, plan.Completed, "apply-1")
	assert.Contains(t, plan.ExecutedSteps, "apply-1")
}

func TestPlanGenerator_ChooseExpression(t *testing.T) {
	// Arrange
	flow := &Flow{
		Name: "choose",
		Arg:  "input",
		Body: []Expr{
			&Choose{
				ID: "choose-1",
				If: &Compare{
					Operation: ">",
					Left:  &GetValue{Path: "input"},
					Right: &SetValue{Value: schema.MkInt(10)},
				},
				Then: []Expr{
					&Apply{
						ID:   "apply-then",
						Name: "multiply",
						Args: []Reshaper{
							&GetValue{Path: "input"},
							&SetValue{Value: schema.MkInt(2)},
						},
					},
				},
				Else: []Expr{
					&Apply{
						ID:   "apply-else",
						Name: "add",
						Args: []Reshaper{
							&GetValue{Path: "input"},
							&SetValue{Value: schema.MkInt(5)},
						},
					},
				},
			},
		},
	}
	input := schema.MkInt(15)
	generator := NewPlanGenerator()

	// Act
	plan := generator.GeneratePlan(flow, input)

	// Assert
	assert.NotNil(t, plan)
	// Should have at least one step for Choose
	assert.Greater(t, len(plan.Steps), 0)
	
	// First step should be the Choose condition
	firstStep, ok := plan.Steps[0].(*ExecuteStep)
	assert.True(t, ok, "Expected ExecuteStep for Choose, got %T", plan.Steps[0])
	assert.Equal(t, "choose-1", firstStep.StepID)
	
	// Verify the expression in the step
	chooseExpr, ok := firstStep.Expr.(*Choose)
	assert.True(t, ok, "Expected Choose expression, got %T", firstStep.Expr)
	assert.Equal(t, "choose-1", chooseExpr.ID)
}

func TestPlanExecutor_ExecuteStep_Choose_TrueBranch(t *testing.T) {
	// Arrange
	plan := NewExecutionPlan()
	plan.Variables["input"] = schema.MkInt(15) // Greater than 10, so should take Then branch
	plan.Steps = []PlanStep{
		&ExecuteStep{
			StepID: "choose-1",
			Expr: &Choose{
				ID: "choose-1",
				If: &Compare{
					Operation: ">",
					Left:  &GetValue{Path: "input"},
					Right: &SetValue{Value: schema.MkInt(10)},
				},
				Then: []Expr{
					&End{
						ID:     "end-then",
						Result: &SetValue{Value: schema.MkString("then branch")},
					},
				},
				Else: []Expr{
					&End{
						ID:     "end-else",
						Result: &SetValue{Value: schema.MkString("else branch")},
					},
				},
			},
			DependsOn: []string{},
		},
	}
	
	executor := NewPlanExecutor(&testDependency{})
	
	// Act
	state, err := executor.Execute(plan)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, state)
	
	// Should be Done state with the Then result
	done, ok := state.(*Done)
	assert.True(t, ok, "Expected Done state, got %T", state)
	if done != nil {
		assert.Equal(t, schema.MkString("then branch"), done.Result)
	}
	
	// Check that the Choose result was stored
	assert.Contains(t, plan.Results, "choose-1")
	assert.Contains(t, plan.Completed, "choose-1")
}

func TestPlanExecutor_ExecuteStep_Choose_FalseBranch(t *testing.T) {
	// Arrange
	plan := NewExecutionPlan()
	plan.Variables["input"] = schema.MkInt(5) // Less than 10, so should take Else branch
	plan.Steps = []PlanStep{
		&ExecuteStep{
			StepID: "choose-1",
			Expr: &Choose{
				ID: "choose-1",
				If: &Compare{
					Operation: ">",
					Left:  &GetValue{Path: "input"},
					Right: &SetValue{Value: schema.MkInt(10)},
				},
				Then: []Expr{
					&End{
						ID:     "end-then",
						Result: &SetValue{Value: schema.MkString("then branch")},
					},
				},
				Else: []Expr{
					&End{
						ID:     "end-else",
						Result: &SetValue{Value: schema.MkString("else branch")},
					},
				},
			},
			DependsOn: []string{},
		},
	}
	
	executor := NewPlanExecutor(&testDependency{})
	
	// Act
	state, err := executor.Execute(plan)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, state)
	
	// Should be Done state with the Else result
	done, ok := state.(*Done)
	assert.True(t, ok, "Expected Done state, got %T", state)
	assert.Equal(t, schema.MkString("else branch"), done.Result)
}

func TestExecutionPlan_Serialization(t *testing.T) {
	// Arrange
	plan := NewExecutionPlan()
	plan.Variables["input"] = schema.MkInt(42)
	plan.Variables["name"] = schema.MkString("test")
	
	plan.Steps = []PlanStep{
		&ExecuteStep{
			StepID: "apply-1",
			Expr: &Apply{
				ID:   "apply-1",
				Name: "process",
				Args: []Reshaper{
					&GetValue{Path: "input"},
				},
			},
			DependsOn: []string{},
		},
		&AssignStep{
			StepID:   "assign-1",
			VarName:  "result",
			FromStep: "apply-1",
		},
	}
	
	plan.Results["apply-1"] = schema.MkFloat(84)
	plan.Completed["apply-1"] = true
	plan.ExecutedSteps = []string{"apply-1"}
	
	// Act - Serialize
	serialized, err := json.Marshal(plan)
	assert.NoError(t, err)
	assert.NotEmpty(t, serialized)
	
	// Act - Deserialize
	var deserialized ExecutionPlan
	err = json.Unmarshal(serialized, &deserialized)
	assert.NoError(t, err)
	
	// Assert
	assert.Equal(t, len(plan.Steps), len(deserialized.Steps))
	assert.Equal(t, len(plan.Variables), len(deserialized.Variables))
	assert.Equal(t, len(plan.Results), len(deserialized.Results))
	assert.Equal(t, len(plan.Completed), len(deserialized.Completed))
	assert.Equal(t, plan.ExecutedSteps, deserialized.ExecutedSteps)
	
	// Check variables
	assert.Equal(t, plan.Variables["input"], deserialized.Variables["input"])
	assert.Equal(t, plan.Variables["name"], deserialized.Variables["name"])
	
	// Check results
	assert.Equal(t, plan.Results["apply-1"], deserialized.Results["apply-1"])
}

func TestPlanGenerator_DependencyTracking(t *testing.T) {
	// Arrange
	flow := &Flow{
		Name: "dependency-test",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				ID:    "assign-1",
				VarOk: "x",
				Val: &Apply{
					ID:   "apply-1",
					Name: "double",
					Args: []Reshaper{
						&GetValue{Path: "input"},
					},
				},
			},
			&Assign{
				ID:    "assign-2",
				VarOk: "y",
				Val: &Apply{
					ID:   "apply-2",
					Name: "triple",
					Args: []Reshaper{
						&GetValue{Path: "input"},
					},
				},
			},
			&Apply{
				ID:   "apply-3",
				Name: "combine",
				Args: []Reshaper{
					&GetValue{Path: "x"},
					&GetValue{Path: "y"},
				},
			},
		},
	}
	input := schema.MkInt(10)
	generator := NewPlanGenerator()
	
	// Act
	plan := generator.GeneratePlan(flow, input)
	
	// Assert
	assert.NotNil(t, plan)
	// Should have 5 steps: apply-1, assign-1, apply-2, assign-2, apply-3
	assert.Len(t, plan.Steps, 5)
	
	// Find apply-3 step
	var apply3Step *ExecuteStep
	for _, step := range plan.Steps {
		if execStep, ok := step.(*ExecuteStep); ok {
			if applyExpr, ok := execStep.Expr.(*Apply); ok && applyExpr.ID == "apply-3" {
				apply3Step = execStep
				break
			}
		}
	}
	
	assert.NotNil(t, apply3Step, "apply-3 step not found")
	// apply-3 should depend on assign-1 and assign-2
	assert.Contains(t, apply3Step.DependsOn, "assign-1")
	assert.Contains(t, apply3Step.DependsOn, "assign-2")
}

func TestPlanExecutor_ParallelExecution(t *testing.T) {
	// Arrange
	plan := NewExecutionPlan()
	plan.Variables["input"] = schema.MkInt(10)
	
	// Two independent Apply steps that can run in parallel
	plan.Steps = []PlanStep{
		&ExecuteStep{
			StepID: "apply-1",
			Expr: &Apply{
				ID:   "apply-1",
				Name: "double",
				Args: []Reshaper{
					&GetValue{Path: "input"},
				},
			},
			DependsOn: []string{}, // No dependencies
		},
		&ExecuteStep{
			StepID: "apply-2",
			Expr: &Apply{
				ID:   "apply-2",
				Name: "triple",
				Args: []Reshaper{
					&GetValue{Path: "input"},
				},
			},
			DependsOn: []string{}, // No dependencies
		},
		&AssignStep{
			StepID:   "assign-1",
			VarName:  "x",
			FromStep: "apply-1",
		},
		&AssignStep{
			StepID:   "assign-2",
			VarName:  "y",
			FromStep: "apply-2",
		},
		&ExecuteStep{
			StepID: "apply-3",
			Expr: &Apply{
				ID:   "apply-3",
				Name: "add",
				Args: []Reshaper{
					&GetValue{Path: "x"},
					&GetValue{Path: "y"},
				},
			},
			DependsOn: []string{"assign-1", "assign-2"}, // Depends on both assigns
		},
	}
	
	// Create test dependency with functions
	dep := &testDependency{}
	executor := NewPlanExecutor(dep)
	
	// Act
	state, err := executor.Execute(plan)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, state)
	
	// Check that all steps were executed
	assert.Contains(t, plan.Completed, "apply-1")
	assert.Contains(t, plan.Completed, "apply-2")
	assert.Contains(t, plan.Completed, "assign-1")
	assert.Contains(t, plan.Completed, "assign-2")
	assert.Contains(t, plan.Completed, "apply-3")
	
	// Check results
	assert.Equal(t, schema.MkFloat(20), plan.Results["apply-1"]) // 10 * 2
	assert.Equal(t, schema.MkFloat(30), plan.Results["apply-2"]) // 10 * 3
	assert.Equal(t, schema.MkFloat(50), plan.Results["apply-3"]) // 20 + 30
}

func TestIntegration_PlanVsDirectExecution(t *testing.T) {
	// Arrange - Create a complex workflow
	flow := &Flow{
		Name: "integration-test",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				ID:    "assign-1",
				VarOk: "doubled",
				Val: &Apply{
					ID:   "apply-1",
					Name: "double",
					Args: []Reshaper{
						&GetValue{Path: "input"},
					},
				},
			},
			&Choose{
				ID: "choose-1",
				If: &Compare{
					Operation: ">",
					Left:  &GetValue{Path: "doubled"},
					Right: &SetValue{Value: schema.MkInt(50)},
				},
				Then: []Expr{
					&Apply{
						ID:   "apply-then",
						Name: "multiply",
						Args: []Reshaper{
							&GetValue{Path: "doubled"},
							&SetValue{Value: schema.MkInt(2)},
						},
					},
				},
				Else: []Expr{
					&Apply{
						ID:   "apply-else",
						Name: "add",
						Args: []Reshaper{
							&GetValue{Path: "doubled"},
							&SetValue{Value: schema.MkInt(10)},
						},
					},
				},
			},
			&End{
				ID:     "end-1",
				Result: &GetValue{Path: "doubled"},
			},
		},
	}
	
	input := schema.MkInt(30)
	dep := &testDependency{}
	
	// Act 1 - Direct execution
	directResult := runFlowDirectly(t, flow, input, dep)
	
	// Act 2 - Plan execution
	generator := NewPlanGenerator()
	plan := generator.GeneratePlan(flow, input)
	executor := NewPlanExecutor(dep)
	planState, err := executor.Execute(plan)
	assert.NoError(t, err)
	
	// Assert - Both executions should produce the same result
	assert.NotNil(t, directResult)
	assert.NotNil(t, planState)
	
	done, ok := planState.(*Done)
	assert.True(t, ok, "Expected Done state from plan execution")
	
	assert.Equal(t, directResult, done.Result, "Plan execution should produce same result as direct execution")
	assert.Equal(t, schema.MkFloat(60), done.Result) // input=30, doubled=60, 60>50 so no branch taken, result=60
}

// Helper function to run flow directly without plan
func runFlowDirectly(t *testing.T, flow *Flow, input schema.Schema, dep Dependency) schema.Schema {
	baseState := BaseState{
		RunID:             dep.GenerateRunID(),
		Variables:         make(map[string]schema.Schema),
		ExprResult:        make(map[string]schema.Schema),
		DefaultMaxRetries: dep.MaxRetries(),
	}
	
	if flow.Arg != "" && input != nil {
		baseState.Variables[flow.Arg] = input
	}
	
	var lastState State
	for _, expr := range flow.Body {
		lastState = ExecuteExpr(baseState, expr, dep)
		if done, ok := lastState.(*Done); ok {
			return done.Result
		}
		if _, ok := lastState.(*Error); ok {
			t.Fatalf("Error during direct execution: %v", lastState)
		}
		baseState = GetBaseState(lastState)
	}
	
	if done, ok := lastState.(*Done); ok {
		return done.Result
	}
	
	return nil
}

func TestIntegration_PlanVsDirectExecution_ElseBranch(t *testing.T) {
	// Same workflow as before
	flow := &Flow{
		Name: "integration-test",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				ID:    "assign-1",
				VarOk: "doubled",
				Val: &Apply{
					ID:   "apply-1",
					Name: "double",
					Args: []Reshaper{
						&GetValue{Path: "input"},
					},
				},
			},
			&Choose{
				ID: "choose-1",
				If: &Compare{
					Operation: ">",
					Left:  &GetValue{Path: "doubled"},
					Right: &SetValue{Value: schema.MkInt(50)},
				},
				Then: []Expr{
					&End{
						ID:     "end-then",
						Result: &SetValue{Value: schema.MkString("then branch")},
					},
				},
				Else: []Expr{
					&End{
						ID:     "end-else",
						Result: &SetValue{Value: schema.MkString("else branch")},
					},
				},
			},
		},
	}
	
	input := schema.MkInt(20) // Will result in doubled=40, which is < 50
	dep := &testDependency{}
	
	// Act 1 - Direct execution
	directResult := runFlowDirectly(t, flow, input, dep)
	
	// Act 2 - Plan execution
	generator := NewPlanGenerator()
	plan := generator.GeneratePlan(flow, input)
	executor := NewPlanExecutor(dep)
	planState, err := executor.Execute(plan)
	assert.NoError(t, err)
	
	// Assert - Both executions should produce the same result
	assert.NotNil(t, directResult)
	assert.NotNil(t, planState)
	
	done, ok := planState.(*Done)
	assert.True(t, ok, "Expected Done state from plan execution")
	
	assert.Equal(t, directResult, done.Result, "Plan execution should produce same result as direct execution")
	assert.Equal(t, schema.MkString("else branch"), done.Result) // input=20, doubled=40, 40<50 so else branch taken
}