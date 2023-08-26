package workflow

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
)

var (
	ErrAlreadyStarted         = errors.New("already started")
	ErrCallbackNotMatch       = errors.New("callback not match")
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrExpressionHasResult    = errors.New("expression has result")
)

type Dependency interface {
	FindWorkflow(flowID string) (*Flow, error)
	FindFunction(funcID string) (Function, error)
	GenerateCallbackID() string
}

func Transition(cmd Command, state State, dep Dependency) (State, error) {
	return MustMatchCommandR2(
		cmd,
		func(x *Run) (State, error) {
			if state != nil {
				return nil, ErrAlreadyStarted
			}

			// resolve flow
			flow, err := getFlow(x.Flow, dep)
			if err != nil {
				return nil, err
			}

			context := BaseState{
				Flow: x.Flow,
				Variables: map[string]schema.Schema{
					flow.Arg: x.Input,
				},
				ExprResult: make(map[string]schema.Schema),
			}

			newStatus := ExecuteAll(context, flow, dep)
			return newStatus, nil
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
		//func(x *Retry) (State, error) {
		//	panic("implement me")
		//},
	)
}

func cloneBaseState(base BaseState) BaseState {
	result := BaseState{
		StepID:     base.StepID,
		Flow:       base.Flow,
		Variables:  make(map[string]schema.Schema),
		ExprResult: make(map[string]schema.Schema),
	}
	for k, v := range base.Variables {
		result.Variables[k] = v
	}
	for k, v := range base.ExprResult {
		result.ExprResult[k] = v
	}
	return result
}

func getFlow(x Worflow, dep Dependency) (*Flow, error) {
	return MustMatchWorflowR2(
		x,
		func(x *Flow) (*Flow, error) {
			return x, nil
		},
		func(x *FlowRef) (*Flow, error) {
			flow, err := dep.FindWorkflow(x.FlowID)
			if err != nil {
				return nil, fmt.Errorf("failed to find workflow %s: %w", x.FlowID, err)
			}

			return flow, nil
		},
	)
}

func ExecuteAll(context BaseState, x *Flow, dep Dependency) State {
	for _, expr := range x.Body {
		status := ExecuteExpr(context, expr, dep)
		switch status.(type) {
		case *Done, *Fail, *Error, *Await:
			return status
		}

		context = getBaseState(status)
	}

	return &Done{
		BaseState: context,
	}
}

func getBaseState(status State) BaseState {
	return MustMatchState(
		status,
		func(x *NextOperation) BaseState {
			return x.BaseState
		},
		func(x *Done) BaseState {
			return x.BaseState
		},
		func(x *Fail) BaseState {
			return x.BaseState
		},
		func(x *Error) BaseState {
			return x.BaseState
		},
		func(x *Await) BaseState {
			return x.BaseState
		},
	)
}

func ExecuteReshaper(context BaseState, reshaper Reshaper) (schema.Schema, error) {
	if reshaper == nil {
		return nil, nil
	}

	return MustMatchReshaperR2(
		reshaper,
		func(x *GetValue) (schema.Schema, error) {
			if val, ok := context.Variables[x.Path]; ok {
				return val, nil
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

			if x.Fail != nil {
				val, err := ExecuteReshaper(context, x.Fail)
				if err != nil {
					return &Error{
						Code:      "execute-reshaper",
						Reason:    "failed to execute reshaper in fail path",
						Retried:   0,
						BaseState: newContext,
					}
				}

				return &Fail{
					Result:    val,
					BaseState: newContext,
				}
			}

			val, err := ExecuteReshaper(context, x.Result)
			if err != nil {
				return &Error{
					Code:      "execute-reshaper",
					Reason:    "failed to execute reshaper in ok path",
					Retried:   0,
					BaseState: newContext,
				}
			}

			return &Done{
				Result:    val,
				BaseState: newContext,
			}
		},
		func(x *Assign) State {
			status := ExecuteExpr(context, x.Val, dep)
			result, ok := status.(*NextOperation)
			if !ok {
				return status
			}

			if _, ok := context.Variables[x.Var]; ok {
				newContext := cloneBaseState(context)
				newContext.StepID = x.ID
				return &Error{
					Code:      "assign-variable",
					Reason:    fmt.Sprintf("variable %s already exists", x.Var),
					Retried:   0,
					BaseState: newContext,
				}
			}

			newContext := cloneBaseState(context)
			newContext.Variables[x.Var] = result.Result
			newContext.StepID = x.ID

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
						Reason:    "failed to execute reshaper while preparing func args, reason: " + err.Error(),
						Retried:   0,
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
					Retried:   0,
					BaseState: newContext,
				}
			}

			input := &FunctionInput{
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
					Retried:   0,
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
					Retried:   0,
					BaseState: newContext,
				}
			}

			// THEN branch cannot be empty, ELSE can, since it is optional
			if len(x.Then) == 0 {
				return &Error{
					Code:      "choose-then-empty",
					Reason:    "then branch cannot be empty",
					Retried:   0,
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
				switch status.(type) {
				case *Done, *Fail, *Error, *Await:
					return status
				}

				newContext = getBaseState(status)
			}

			return status
		},
	)
}
