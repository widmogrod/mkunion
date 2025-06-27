package workflow

import (
	"fmt"

	"github.com/widmogrod/mkunion/x/schema"
)

// PlanStep represents a single step in an execution plan
//
//go:tag mkunion:"PlanStep"
type (
	// ExecuteStep executes an expression and stores the result
	ExecuteStep struct {
		StepID    string
		Expr      Expr
		DependsOn []string // StepIDs that must complete first
	}

	// AssignStep assigns a value from another step to a variable
	AssignStep struct {
		StepID   string
		VarName  string
		FromStep string // StepID whose result to assign
	}

	// ReturnStep returns the result from another step
	ReturnStep struct {
		StepID   string
		FromStep string // StepID whose result to return
	}
)

// ExecutionPlan represents a plan of operations to be executed
//
//go:tag serde:"json"
type ExecutionPlan struct {
	// Steps contains all steps in the plan (initial and dynamically added)
	Steps []PlanStep
	// Queue contains StepIDs ready to execute
	Queue []string

	// Execution state
	Completed map[string]bool          // StepID -> completed
	Results   map[string]schema.Schema // StepID -> result
	Variables map[string]schema.Schema // Variable name -> value

	// For debugging/resume
	ExecutedSteps []string // Order of execution
	SkippedSteps  []string // Steps skipped due to branches not taken
}

// NewExecutionPlan creates a new ExecutionPlan with initialized fields
func NewExecutionPlan() *ExecutionPlan {
	return &ExecutionPlan{
		Steps:         []PlanStep{},
		Queue:         []string{},
		Completed:     make(map[string]bool),
		Results:       make(map[string]schema.Schema),
		Variables:     make(map[string]schema.Schema),
		ExecutedSteps: []string{},
		SkippedSteps:  []string{},
	}
}

// PlanGenerator generates execution plans from workflows
type PlanGenerator interface {
	GeneratePlan(flow *Flow, input schema.Schema) *ExecutionPlan
}

// planGenerator is the default implementation of PlanGenerator
type planGenerator struct {
	stepCounter int
	// Track which variables are provided by which steps
	variableProviders map[string]string // variable name -> step ID
}

// NewPlanGenerator creates a new PlanGenerator
func NewPlanGenerator() PlanGenerator {
	return &planGenerator{
		stepCounter:       0,
		variableProviders: make(map[string]string),
	}
}

// GeneratePlan generates an execution plan from a workflow
func (g *planGenerator) GeneratePlan(flow *Flow, input schema.Schema) *ExecutionPlan {
	plan := NewExecutionPlan()

	// Initialize with input variable
	if flow.Arg != "" && input != nil {
		plan.Variables[flow.Arg] = input
	}

	// Generate steps for each expression in flow.Body
	for _, expr := range flow.Body {
		steps := g.generateStepsForExpr(expr)
		plan.Steps = append(plan.Steps, steps...)
	}

	return plan
}

// findDependencies finds variable dependencies in a reshaper
func (g *planGenerator) findDependencies(reshaper Reshaper) []string {
	var deps []string
	switch r := reshaper.(type) {
	case *GetValue:
		// Check if this variable is provided by a step
		if stepID, ok := g.variableProviders[r.Path]; ok {
			deps = append(deps, stepID)
		}
	}
	return deps
}

// findExprDependencies finds all dependencies in an expression
func (g *planGenerator) findExprDependencies(expr Expr) []string {
	var allDeps []string
	depMap := make(map[string]bool)

	switch e := expr.(type) {
	case *Apply:
		// Check dependencies in all arguments
		for _, arg := range e.Args {
			deps := g.findDependencies(arg)
			for _, dep := range deps {
				if !depMap[dep] {
					depMap[dep] = true
					allDeps = append(allDeps, dep)
				}
			}
		}
	case *End:
		if e.Result != nil {
			deps := g.findDependencies(e.Result)
			for _, dep := range deps {
				if !depMap[dep] {
					depMap[dep] = true
					allDeps = append(allDeps, dep)
				}
			}
		}
	case *Choose:
		// Check dependencies in condition
		if compare, ok := e.If.(*Compare); ok {
			if compare.Left != nil {
				deps := g.findDependencies(compare.Left)
				for _, dep := range deps {
					if !depMap[dep] {
						depMap[dep] = true
						allDeps = append(allDeps, dep)
					}
				}
			}
			if compare.Right != nil {
				deps := g.findDependencies(compare.Right)
				for _, dep := range deps {
					if !depMap[dep] {
						depMap[dep] = true
						allDeps = append(allDeps, dep)
					}
				}
			}
		}
	}

	return allDeps
}

// generateStepsForExpr generates plan steps for a single expression
func (g *planGenerator) generateStepsForExpr(expr Expr) []PlanStep {
	switch e := expr.(type) {
	case *End:
		deps := g.findExprDependencies(e)
		return []PlanStep{
			&ExecuteStep{
				StepID:    e.ID,
				Expr:      e,
				DependsOn: deps,
			},
		}
	case *Assign:
		var steps []PlanStep

		// First, generate steps for the value expression
		valSteps := g.generateStepsForExpr(e.Val)
		steps = append(steps, valSteps...)

		// Get the step ID from the value expression
		var fromStepID string
		if len(valSteps) > 0 {
			// Get the last step's ID using exhaustive pattern matching
			fromStepID = MatchPlanStepR1(
				valSteps[len(valSteps)-1],
				func(s *ExecuteStep) string { return s.StepID },
				func(s *AssignStep) string { return s.StepID },
				func(s *ReturnStep) string { return s.StepID },
			)
		}

		// Then add the assign step
		assignStep := &AssignStep{
			StepID:   e.ID,
			VarName:  e.VarOk,
			FromStep: fromStepID,
		}
		steps = append(steps, assignStep)

		// Track that this variable is provided by this assign step
		g.variableProviders[e.VarOk] = e.ID

		return steps
	case *Apply:
		deps := g.findExprDependencies(e)
		return []PlanStep{
			&ExecuteStep{
				StepID:    e.ID,
				Expr:      e,
				DependsOn: deps,
			},
		}
	case *Choose:
		// For Choose, we execute it as a single step
		// The executor will handle the dynamic branching
		deps := g.findExprDependencies(e)
		return []PlanStep{
			&ExecuteStep{
				StepID:    e.ID,
				Expr:      e,
				DependsOn: deps,
			},
		}
	default:
		// TODO: Handle other expression types
		return []PlanStep{}
	}
}

// PlanExecutor executes plans and produces workflow states
type PlanExecutor interface {
	Execute(plan *ExecutionPlan) (State, error)
}

// planExecutor is the default implementation of PlanExecutor
type planExecutor struct {
	dep           Dependency
	planGenerator PlanGenerator
}

// NewPlanExecutor creates a new PlanExecutor
func NewPlanExecutor(dep Dependency) PlanExecutor {
	return &planExecutor{
		dep:           dep,
		planGenerator: NewPlanGenerator(),
	}
}

// Execute executes a plan and returns the final state
func (e *planExecutor) Execute(plan *ExecutionPlan) (State, error) {
	// Initialize base state from plan
	baseState := BaseState{
		RunID:             e.dep.GenerateRunID(),
		Variables:         plan.Variables,
		ExprResult:        make(map[string]schema.Schema),
		DefaultMaxRetries: e.dep.MaxRetries(),
	}

	// Empty plan results in Done state
	if len(plan.Steps) == 0 {
		return &Done{
			BaseState: baseState,
		}, nil
	}

	// Execute steps in order (respecting dependencies)
	for i := 0; i < len(plan.Steps); i++ {
		step := plan.Steps[i]

		// Check if all dependencies are satisfied using exhaustive pattern matching
		canExecute := MatchPlanStepR1(
			step,
			func(execStep *ExecuteStep) bool {
				// Check all dependencies for ExecuteStep
				for _, dep := range execStep.DependsOn {
					if !plan.Completed[dep] {
						return false
					}
				}
				return true
			},
			func(assignStep *AssignStep) bool {
				// AssignStep depends on its FromStep
				return plan.Completed[assignStep.FromStep]
			},
			func(returnStep *ReturnStep) bool {
				// ReturnStep depends on its FromStep
				return plan.Completed[returnStep.FromStep]
			},
		)

		if !canExecute {
			// Skip this step for now, we'll come back to it
			continue
		}

		state, result, newSteps, err := e.executeStepWithState(step, plan, baseState)
		if err != nil {
			return nil, err
		}

		// Update base state from execution
		baseState = GetBaseState(state)

		// Handle step completion using exhaustive pattern matching
		doneState := MatchPlanStepR1(
			step,
			func(s *ExecuteStep) State {
				if result != nil {
					plan.Results[s.StepID] = result
				}
				plan.Completed[s.StepID] = true
				plan.ExecutedSteps = append(plan.ExecutedSteps, s.StepID)

				// If we got a Done state, return it
				if done, isDone := state.(*Done); isDone {
					return done
				}
				return nil
			},
			func(s *AssignStep) State {
				// AssignStep doesn't produce a result in plan.Results
				// but updates variables which is handled in executeStepInternal
				plan.Completed[s.StepID] = true
				plan.ExecutedSteps = append(plan.ExecutedSteps, s.StepID)
				return nil
			},
			func(s *ReturnStep) State {
				plan.Completed[s.StepID] = true
				plan.ExecutedSteps = append(plan.ExecutedSteps, s.StepID)
				// ReturnStep always produces Done state
				if done, isDone := state.(*Done); isDone {
					return done
				}
				return nil
			},
		)

		if doneState != nil {
			return doneState, nil
		}

		// Add any new steps (for dynamic execution)
		if len(newSteps) > 0 {
			plan.Steps = append(plan.Steps, newSteps...)
		}

		// If we've reached the end but have uncompleted steps, restart from beginning
		if i == len(plan.Steps)-1 {
			hasUncompleted := false
			for j := 0; j < len(plan.Steps); j++ {
				// Get step ID using exhaustive pattern matching
				stepID := MatchPlanStepR1(
					plan.Steps[j],
					func(s *ExecuteStep) string { return s.StepID },
					func(s *AssignStep) string { return s.StepID },
					func(s *ReturnStep) string { return s.StepID },
				)
				if stepID != "" && !plan.Completed[stepID] {
					hasUncompleted = true
					break
				}
			}
			if hasUncompleted {
				i = -1 // Will be incremented to 0 in the next iteration
			}
		}
	}

	// Default to Done state
	return &Done{
		BaseState: baseState,
	}, nil
}

// stepResult contains the result of executing a step
//
//go:tag serde:"json"
type stepResult struct {
	state    State
	result   schema.Schema
	newSteps []PlanStep
}

// executeStepWithState executes a single plan step and returns the state
func (e *planExecutor) executeStepWithState(step PlanStep, plan *ExecutionPlan, baseState BaseState) (State, schema.Schema, []PlanStep, error) {
	res, err := e.executeStepInternal(step, plan, baseState)
	if err != nil {
		return nil, nil, nil, err
	}
	return res.state, res.result, res.newSteps, nil
}

// executeStepInternal executes a single plan step
func (e *planExecutor) executeStepInternal(step PlanStep, plan *ExecutionPlan, baseState BaseState) (*stepResult, error) {
	return MatchPlanStepR2(
		step,
		func(s *ExecuteStep) (*stepResult, error) {
			// Execute the expression
			state := ExecuteExpr(baseState, s.Expr, e.dep)

			// Extract result from state
			switch st := state.(type) {
			case *Done:
				return &stepResult{state: state, result: st.Result}, nil
			case *NextOperation:
				return &stepResult{state: state, result: st.Result}, nil
			case *Error:
				return nil, fmt.Errorf("error executing step %s: %s", s.StepID, st.Reason)
			default:
				return nil, fmt.Errorf("unexpected state type: %T", state)
			}
		},
		func(s *AssignStep) (*stepResult, error) {
			// Get result from the source step
			if result, ok := plan.Results[s.FromStep]; ok {
				// Update variables in both base state and plan
				baseState.Variables[s.VarName] = result
				plan.Variables[s.VarName] = result
				state := &NextOperation{
					Result:    result,
					BaseState: baseState,
				}
				return &stepResult{state: state, result: result}, nil
			}
			return nil, fmt.Errorf("step %s not found in results", s.FromStep)
		},
		func(s *ReturnStep) (*stepResult, error) {
			// Return result from another step
			if result, ok := plan.Results[s.FromStep]; ok {
				state := &Done{
					Result:    result,
					BaseState: baseState,
				}
				return &stepResult{state: state, result: result}, nil
			}
			return nil, fmt.Errorf("step %s not found in results", s.FromStep)
		},
	)
}

// executeStep executes a single plan step
func (e *planExecutor) executeStep(step PlanStep, plan *ExecutionPlan, baseState BaseState) (schema.Schema, []PlanStep, error) {
	return MatchPlanStepR3(
		step,
		func(s *ExecuteStep) (schema.Schema, []PlanStep, error) {
			// Execute the expression
			state := ExecuteExpr(baseState, s.Expr, e.dep)

			// Update the base state from the result
			baseState = GetBaseState(state)

			// Extract result from state
			switch st := state.(type) {
			case *Done:
				return st.Result, nil, nil
			case *NextOperation:
				return st.Result, nil, nil
			case *Error:
				return nil, nil, fmt.Errorf("error executing step %s: %s", s.StepID, st.Reason)
			default:
				return nil, nil, fmt.Errorf("unexpected state type: %T", state)
			}
		},
		func(s *AssignStep) (schema.Schema, []PlanStep, error) {
			// Get result from the source step
			if result, ok := plan.Results[s.FromStep]; ok {
				// Update variables in base state
				baseState.Variables[s.VarName] = result
				return result, nil, nil
			}
			return nil, nil, fmt.Errorf("step %s not found in results", s.FromStep)
		},
		func(s *ReturnStep) (schema.Schema, []PlanStep, error) {
			// Return result from another step
			if result, ok := plan.Results[s.FromStep]; ok {
				return result, nil, nil
			}
			return nil, nil, fmt.Errorf("step %s not found in results", s.FromStep)
		},
	)
}
