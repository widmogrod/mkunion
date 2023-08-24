package workflow

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
)

var (
	ErrAlreadyStarted   = errors.New("already started")
	ErrFailFindWorkflow = errors.New("failed to find workflow")
)

type Dependency interface {
	FindWorkflow(flowID string) (*Flow, error)
	FindFunction(funcID string) (Function, error)
	NewContext() *Context
}

func Transition(cmd Command, state Status, dep Dependency) (Status, error) {
	return MustMatchCommandR2(
		cmd,
		func(x *Run) (Status, error) {
			if state != nil {
				return nil, ErrAlreadyStarted
			}

			// resolve flow
			flow, err := MustMatchWorflowR2(
				x.Flow,
				func(x *Flow) (*Flow, error) {
					return x, nil
				},
				func(x *FlowRef) (*Flow, error) {
					flow, err := dep.FindWorkflow(x.FlowID)
					if err != nil {
						return nil, fmt.Errorf("fauked to find workflow %s: %w", x.FlowID, err)
					}

					return flow, nil
				},
			)
			if err != nil {
				return nil, err
			}

			context := dep.NewContext()
			err = context.SetVariable(flow.Arg, x.Input)
			if err != nil {
				return nil, fmt.Errorf("failed to initiate context with starting variable %s: %w", flow.Arg, err)
			}

			newStatus := ExecuteAll(context, flow)
			return newStatus, nil
		},
		func(x *Schedule) (Status, error) {
			panic("implement me")
		},
		func(x *Callback) (Status, error) {
			panic("implement me")
		},
		func(x *Retry) (Status, error) {
			panic("implement me")
		},
	)
}

func ExecuteAll(context *Context, x *Flow) Status {
	context = context.GetForFlow(x)
	for _, expr := range x.Body {
		status := ExecuteExpr(context, expr)
		switch status.(type) {
		case *Done, *Fail, *Error, *Await:
			return status
		}
	}

	return &Done{
		StepID: x.Name,
	}
}

func ExecuteReshaper(context *Context, reshaper Reshaper) (schema.Schema, error) {
	if reshaper == nil {
		return nil, nil
	}

	context = context.GetForReshaper(reshaper)
	return MustMatchReshaperR2(
		reshaper,
		func(x *GetValue) (schema.Schema, error) {
			if val, ok := context.GetVariable(x.Path); ok {
				return val, nil
			} else {
				return nil, context.Errorf("variable %s not found", x.Path)
			}
		},
		func(x *SetValue) (schema.Schema, error) {
			return x.Value, nil
		},
	)
}

func ExecuteExpr(context *Context, expr Expr) Status {
	context = context.GetForExpr(expr)
	return MustMatchExpr(
		expr,
		func(x *End) Status {
			if x.Fail != nil {
				val, err := ExecuteReshaper(context, x.Result)
				if err != nil {
					return &Error{
						StepID:  x.ID,
						Code:    "execute-reshaper",
						Reason:  "failed to execute reshaper in fail path",
						Retried: 0,
					}
				}

				return &Fail{
					StepID: x.ID,
					Result: val,
				}
			}

			val, err := ExecuteReshaper(context, x.Result)
			if err != nil {
				return &Error{
					StepID:  x.ID,
					Code:    "execute-reshaper",
					Reason:  "failed to execute reshaper in ok path",
					Retried: 0,
				}
			}

			return &Done{
				StepID: x.ID,
				Result: val,
			}
		},
		func(x *Assign) Status {
			status := ExecuteExpr(context, x.Val)
			result, ok := status.(*Result)
			if !ok {
				return status
			}

			err := context.SetVariable(x.Var, result.Result)
			if err != nil {
				return &Error{
					StepID:  x.ID,
					Code:    "assign-variable",
					Reason:  err.Error(),
					Retried: 0,
				}
			}

			return result
		},
		func(x *Apply) Status {
			args := make([]schema.Schema, len(x.Args))
			for i, arg := range x.Args {
				val, err := ExecuteReshaper(context, arg)
				if err != nil {
					return &Error{
						StepID:  x.ID,
						Code:    "execute-reshaper",
						Reason:  "failed to execute reshaper while preparing func args",
						Retried: 0,
					}
				}
				args[i] = val
			}

			fn, err := context.FindFunction(x.Name)
			if err != nil {
				return &Error{
					StepID:  x.ID,
					Code:    "function-missing",
					Reason:  fmt.Sprintf("function %s() not found, details: %s", x.Name, err.Error()),
					Retried: 0,
				}
			}

			val, err := fn(args)
			if err != nil {
				return &Error{
					StepID:  x.ID,
					Code:    "function-execution",
					Reason:  fmt.Sprintf("function %s() returned error: %s", x.Name, err.Error()),
					Retried: 0,
				}
			}

			return &Result{
				StepID: x.ID,
				Result: val,
			}
		},
		func(x *Choose) Status {
			panic("implement me")
		},
	)
}
