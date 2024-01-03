package workflow

import (
	"errors"
	"fmt"
	"github.com/robfig/cron/v3"
	"github.com/widmogrod/mkunion/x/machine"
	"github.com/widmogrod/mkunion/x/schema"
	"time"
)

var (
	ErrAlreadyStarted         = errors.New("already started")
	ErrCallbackNotMatch       = errors.New("callback not match")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrExpressionHasResult    = errors.New("expression has result")
	ErrStateReachEnd          = errors.New("cannot apply commands, when workflow is completed")
	ErrMaxRetriesReached      = errors.New("max retries reached")
	ErrFlowNotFound           = errors.New("flow not found")
	ErrFlowNotSet             = errors.New("flow not set")
	ErrIntervalParse          = errors.New("failed to parse interval")
	ErrRunIDNotMatch          = errors.New("run id not match")
)

type Dependency interface {
	FindWorkflow(flowID string) (*Flow, error)
	FindFunction(funcID string) (Function, error)
	GenerateCallbackID() string
	GenerateRunID() string
	MaxRetries() int64
	TimeNow() time.Time
}

func NewMachine(di Dependency, state State) *machine.Machine[Command, State] {
	return machine.NewSimpleMachineWithState(func(cmd Command, state State) (State, error) {
		return Transition(cmd, state, di)
	}, state)
}

func Transition(cmd Command, state State, dep Dependency) (State, error) {
	switch state.(type) {
	case *Done:
		return nil, ErrStateReachEnd
	}

	return MustMatchCommandR2(
		cmd,
		func(x *Run) (State, error) {
			switch s := state.(type) {
			// resume scheduled or delayed execution
			case *Scheduled:
				flow, err := getFlow(s.BaseState.Flow, dep)
				if err != nil {
					return nil, err
				}

				newStatus := ExecuteAll(s.BaseState, flow, dep)
				return newStatus, nil

			// start new execution
			case nil:
				// resolve flow
				flow, err := getFlow(x.Flow, dep)
				if err != nil {
					return nil, err
				}

				context := BaseState{
					RunID: dep.GenerateRunID(),
					Flow:  x.Flow,
					Variables: map[string]schema.Schema{
						flow.Arg: x.Input,
					},
					ExprResult:        make(map[string]schema.Schema),
					DefaultMaxRetries: dep.MaxRetries(),
				}

				switch x.RunOption.(type) {
				case *ScheduleRun, *DelayRun:
					context.RunOption = x.RunOption
					context.RunOption = completeParentRunID(context)

					runTimestamp, err := calculateExpectedRunTimestamp(x.RunOption, dep)
					if err != nil {
						return nil, err
					}

					// schedule or delay execution
					return &Scheduled{
						ExpectedRunTimestamp: runTimestamp,
						BaseState:            context,
					}, nil
				}

				newStatus := ExecuteAll(context, flow, dep)
				return newStatus, nil
			}

			return nil, ErrInvalidStateTransition
		},
		func(x *Callback) (State, error) {
			switch s := state.(type) {
			case *Await:
				if s.CallbackID != x.CallbackID {
					return nil, ErrCallbackNotMatch
				}

				newContext := cloneBaseState(s.BaseState)
				if _, ok := newContext.ExprResult[s.BaseState.StepID]; ok {
					return nil, ErrExpressionHasResult
				}

				newContext.ExprResult[s.BaseState.StepID] = x.Result

				flow, err := getFlow(newContext.Flow, dep)
				if err != nil {
					return nil, err
				}

				newStatus := ExecuteAll(newContext, flow, dep)
				return newStatus, nil
			}

			return nil, ErrInvalidStateTransition
		},
		func(x *TryRecover) (State, error) {
			switch s := state.(type) {
			case *Error:
				if s.BaseState.RunID != x.RunID {
					return nil, ErrRunIDNotMatch
				}

				maxRetries := s.BaseState.DefaultMaxRetries
				//if s.MaxRetries > 0 {
				//	maxRetries = s.MaxRetries
				//}
				if s.Retried >= maxRetries {
					return nil, ErrMaxRetriesReached
				}

				newContext := cloneBaseState(s.BaseState)

				flow, err := getFlow(newContext.Flow, dep)
				if err != nil {
					return nil, err
				}

				newStatus := ExecuteAll(newContext, flow, dep)

				// when state is, the same error, then let's increment Retried counter
				errorState, isErrorState := newStatus.(*Error)
				if isErrorState && errorState.Code == s.Code && errorState.Reason == s.Reason {
					errorState.Retried = s.Retried + 1
				}

				return newStatus, nil
			}

			return nil, ErrInvalidStateTransition
		},
		func(x *StopSchedule) (State, error) {
			switch s := state.(type) {
			case *Scheduled:
				parentRunID := extractParentRunID(s.BaseState)
				if parentRunID != x.ParentRunID {
					return nil, ErrRunIDNotMatch
				}

				return &ScheduleStopped{
					BaseState: s.BaseState,
				}, nil
			}

			return nil, ErrInvalidStateTransition
		},
		func(x *ResumeSchedule) (State, error) {
			switch s := state.(type) {
			case *ScheduleStopped:
				parentRunID := extractParentRunID(s.BaseState)
				if parentRunID != x.ParentRunID {
					return nil, ErrRunIDNotMatch
				}

				runTimestamp, err := calculateExpectedRunTimestamp(s.BaseState.RunOption, dep)
				if err != nil {
					return nil, err
				}

				return &Scheduled{
					ExpectedRunTimestamp: runTimestamp,
					BaseState:            s.BaseState,
				}, nil
			}

			return nil, ErrInvalidStateTransition
		},
	)
}

func cloneBaseState(base BaseState) BaseState {
	result := BaseState{
		RunID:             base.RunID,
		StepID:            base.StepID,
		Flow:              base.Flow,
		Variables:         make(map[string]schema.Schema),
		ExprResult:        make(map[string]schema.Schema),
		DefaultMaxRetries: base.DefaultMaxRetries,
		RunOption:         base.RunOption,
	}
	for k, v := range base.Variables {
		result.Variables[k] = v
	}
	for k, v := range base.ExprResult {
		result.ExprResult[k] = v
	}
	return result
}

func getFlow(x Workflow, dep Dependency) (*Flow, error) {
	if x == nil {
		return nil, ErrFlowNotSet
	}

	return MustMatchWorkflowR2(
		x,
		func(x *Flow) (*Flow, error) {
			return initStepID(x), nil
		},
		func(x *FlowRef) (*Flow, error) {
			flow, err := dep.FindWorkflow(x.FlowID)
			if err != nil {
				return nil, fmt.Errorf("failed to find workflow %s: %w; %w", x.FlowID, err, ErrFlowNotFound)
			}

			return initStepID(flow), nil
		},
	)
}

var cronParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.DowOptional | cron.Descriptor)

func calculateExpectedRunTimestamp(x RunOption, dep Dependency) (int64, error) {
	return MustMatchRunOptionR2(
		x,
		func(x *ScheduleRun) (int64, error) {
			schedule, err := cronParser.Parse(x.Interval)
			if err != nil {
				return 0, fmt.Errorf("workflow.calculateExpectedRunTimestamp: failed to parse interval: %w; %w", err, ErrIntervalParse)
			}

			return schedule.Next(dep.TimeNow()).Unix(), nil
		},
		func(x *DelayRun) (int64, error) {
			// always calculate time in the future
			return dep.
				TimeNow().
				Add(time.Duration(x.DelayBySeconds) * time.Second).
				Unix(), nil
		},
	)
}

func extractParentRunID(context BaseState) string {
	return MustMatchRunOption(
		context.RunOption,
		func(y *ScheduleRun) string {
			if y.ParentRunID == "" {
				return context.RunID
			}

			return y.ParentRunID
		},
		func(y *DelayRun) string {
			// DelayRun is like scheduled one, so parent run is always the same as current run
			return context.RunID
		},
	)
}

func completeParentRunID(context BaseState) RunOption {
	return MustMatchRunOption(
		context.RunOption,
		func(y *ScheduleRun) RunOption {
			if y.ParentRunID == "" {
				return &ScheduleRun{
					Interval:    y.Interval,
					ParentRunID: context.RunID,
				}
			}

			return y
		},
		func(y *DelayRun) RunOption {
			// DelayRun is like scheduled one, so parent run is always the same as current run
			return y
		},
	)
}

func ExecuteAll(context BaseState, x *Flow, dep Dependency) State {
	for _, expr := range x.Body {
		status := ExecuteExpr(context, expr, dep)
		if shouldStopExecution(status) {
			return status
		}

		context = GetBaseState(status)
	}

	return &Done{
		BaseState: context,
	}
}

func GetBaseState(status State) BaseState {
	return MustMatchState(
		status,
		func(x *NextOperation) BaseState {
			return x.BaseState
		},
		func(x *Done) BaseState {
			return x.BaseState
		},
		func(x *Error) BaseState {
			return x.BaseState
		},
		func(x *Await) BaseState {
			return x.BaseState
		},
		func(x *Scheduled) BaseState {
			return x.BaseState
		},
		func(x *ScheduleStopped) BaseState {
			return x.BaseState
		},
	)
}

func ScheduleNext(x State, dep Dependency) *Run {
	base := GetBaseState(x)
	if base.RunOption == nil {
		return nil
	}

	switch base.RunOption.(type) {
	case *ScheduleRun:
		flow, err := getFlow(base.Flow, dep)
		if err != nil {
			return nil
		}

		return &Run{
			Flow:      base.Flow,
			Input:     base.Variables[flow.Arg],
			RunOption: base.RunOption,
		}
	}

	return nil
}

func GetRunID(state State) string {
	return GetBaseState(state).RunID
}

func ExecuteReshaper(context BaseState, reshaper Reshaper) (schema.Schema, error) {
	if reshaper == nil {
		return nil, nil
	}

	return MustMatchReshaperR2(
		reshaper,
		func(x *GetValue) (schema.Schema, error) {
			loc, err := schema.ParseLocation(x.Path)
			if err != nil {
				return nil, fmt.Errorf("workflow.ExecuteReshaper: failed to parse location: %w", err)
			}
			first, rest := loc[0], loc[1:]
			field, ok := first.(*schema.LocationField)
			if !ok {
				return nil, fmt.Errorf("workflow.ExecuteReshaper: expected location to start with field name, got %s", x.Path)
			}
			if val, ok := context.Variables[field.Name]; ok {
				return schema.GetSchemaLocation(val, rest), nil
			} else {
				return nil, fmt.Errorf("variable %s not found", x.Path)
			}
		},
		func(x *SetValue) (schema.Schema, error) {
			return x.Value, nil
		},
	)
}

func ExecutePredicate(context BaseState, predicate Predicate, dep Dependency) (bool, error) {
	return MustMatchPredicateR2(
		predicate,
		func(x *And) (bool, error) {
			for _, p := range x.L {
				result, err := ExecutePredicate(context, p, dep)
				if err != nil {
					return false, err
				}

				if !result {
					return false, nil
				}
			}

			return true, nil
		},
		func(x *Or) (bool, error) {
			for _, p := range x.L {
				result, err := ExecutePredicate(context, p, dep)
				if err != nil {
					return false, err
				}

				if result {
					return true, nil
				}
			}

			return false, nil
		},
		func(x *Not) (bool, error) {
			result, err := ExecutePredicate(context, x.P, dep)
			if err != nil {
				return false, err
			}

			return !result, nil
		},
		func(x *Compare) (bool, error) {
			left, err := ExecuteReshaper(context, x.Left)
			if err != nil {
				return false, fmt.Errorf("left comapre failed: %w", err)
			}

			right, err := ExecuteReshaper(context, x.Right)
			if err != nil {
				return false, fmt.Errorf("right comapre failed: %w", err)
			}

			cmp := schema.Compare(left, right)
			switch x.Operation {
			case "=":
				return cmp == 0, nil
			case "!=":
				return cmp != 0, nil
			case "<":
				return cmp < 0, nil
			case "<=":
				return cmp <= 0, nil
			case ">":
				return cmp > 0, nil
			case ">=":
				return cmp >= 0, nil
			default:
				return false, fmt.Errorf("invalid compare operator %s", x.Operation)
			}
		},
	)
}

func ExecuteExpr(context BaseState, expr Expr, dep Dependency) State {
	return MustMatchExpr(
		expr,
		func(x *End) State {
			newContext := cloneBaseState(context)
			newContext.StepID = x.ID

			val, err := ExecuteReshaper(context, x.Result)
			if err != nil {
				return &Error{
					Code:      "execute-reshaper",
					Reason:    "failed to execute reshaper in ok path",
					BaseState: newContext,
				}
			}

			return &Done{
				Result:    val,
				BaseState: newContext,
			}
		},
		func(x *Assign) State {
			newContext := cloneBaseState(context)
			newContext.StepID = x.ID

			status := ExecuteExpr(context, x.Val, dep)
			result, ok := status.(*NextOperation)
			if !ok {
				return status
			}

			if _, ok := newContext.Variables[x.VarOk]; ok {
				return &Error{
					Code:      "assign-variable",
					Reason:    fmt.Sprintf("variable %s already exists", x.VarOk),
					BaseState: newContext,
				}
			}

			newContext.Variables[x.VarOk] = result.Result

			// Since *Assign is expression, it means that it can return value
			// by it returns value that is assigned to variable.
			return &NextOperation{
				Result:    result.Result,
				BaseState: newContext,
			}
		},
		func(x *Apply) State {
			newContext := cloneBaseState(context)
			newContext.StepID = x.ID
			if val, ok := context.ExprResult[x.ID]; ok {
				return &NextOperation{
					Result:    val,
					BaseState: newContext,
				}
			}

			args := make([]schema.Schema, len(x.Args))
			for i, arg := range x.Args {
				val, err := ExecuteReshaper(context, arg)
				if err != nil {
					return &Error{
						Code:      "execute-reshaper",
						Reason:    fmt.Sprintf("failed to execute reshaper while preparing args for function %s(), reason: %s", x.Name, err.Error()),
						BaseState: newContext,
					}
				}
				args[i] = val
			}

			fn, err := dep.FindFunction(x.Name)
			if err != nil {
				return &Error{
					Code:      "function-missing",
					Reason:    fmt.Sprintf("function %s() not found, details: %s", x.Name, err.Error()),
					BaseState: newContext,
				}
			}

			input := &FunctionInput{
				Name: x.Name,
				Args: args,
			}
			// IF function is async, we need to generate a callback ID
			// so that, when the function is done, it can call us back with the result
			if x.Await != nil {
				input.CallbackID = dep.GenerateCallbackID()
			}

			val, err := fn(input)
			if err != nil {
				return &Error{
					Code:      "function-execution",
					Reason:    fmt.Sprintf("function %s() returned error: %s", x.Name, err.Error()),
					BaseState: newContext,
				}
			}

			if x.Await != nil {
				return &Await{
					Timeout:    x.Await.Timeout,
					CallbackID: input.CallbackID,
					BaseState:  newContext,
				}
			}

			return &NextOperation{
				Result:    val.Result,
				BaseState: newContext,
			}
		},
		func(x *Choose) State {
			newContext := cloneBaseState(context)
			newContext.StepID = x.ID

			isTrue, err := ExecutePredicate(newContext, x.If, dep)
			if err != nil {
				return &Error{
					Code:      "choose-evaluate-predicate",
					Reason:    "failed to evaluate predicate, reason: " + err.Error(),
					BaseState: newContext,
				}
			}

			// THEN branch cannot be empty, ELSE can, since it is optional
			if len(x.Then) == 0 {
				return &Error{
					Code:      "choose-then-empty",
					Reason:    "then branch cannot be empty",
					BaseState: newContext,
				}
			}

			// select which branch to evaluate
			evaluate := x.Else
			if isTrue {
				evaluate = x.Then
			}

			// Since *Choose is expression, it means that it can return value
			// by default it returns result of predicate evaluation,
			// but if there are any expressions in THEN or ELSE branches
			// then result of last expression is returned
			var status State = &NextOperation{
				Result:    schema.MkBool(isTrue),
				BaseState: newContext,
			}

			for _, expr2 := range evaluate {
				status = ExecuteExpr(newContext, expr2, dep)
				if shouldStopExecution(status) {
					return status
				}

				newContext = GetBaseState(status)
			}

			return status
		},
	)
}

func shouldStopExecution(state State) bool {
	switch state.(type) {
	case *Error, *Done, *Await:
		return true
	}

	return false
}

func initStepID(x *Flow) *Flow {
	if x == nil {
		return nil
	}

	steps := map[string]int{}
	for i, expr := range x.Body {
		x.Body[i] = initExprStepID(expr, steps)
	}

	return x
}

func initExprStepID(x Expr, steps map[string]int) Expr {
	return MustMatchExpr(
		x,
		func(x *End) Expr {
			x.ID = stepId(x.ID, "end", steps)
			return x

		},
		func(x *Assign) Expr {
			x.ID = stepId(x.ID, "assign", steps)
			x.Val = initExprStepID(x.Val, steps)
			return x
		},
		func(x *Apply) Expr {
			x.ID = stepId(x.ID, "apply-"+x.Name, steps)
			return x
		},
		func(x *Choose) Expr {
			x.ID = stepId(x.ID, "choose", steps)
			for i, expr := range x.Then {
				x.Then[i] = initExprStepID(expr, steps)
			}
			for i, expr := range x.Else {
				x.Else[i] = initExprStepID(expr, steps)
			}
			return x
		},
	)
}

func stepId(stepID, orName string, steps map[string]int) string {
	if stepID == "" {
		stepID = orName
	}

	if _, ok := steps[stepID]; !ok {
		steps[stepID] = 0
		return stepID
	}

	steps[stepID]++
	return fmt.Sprintf("%s-%d", stepID, steps[stepID])
}
