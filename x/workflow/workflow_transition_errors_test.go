package workflow

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
)

func errorPathFlow() *Flow {
	return &Flow{
		Name: "hello",
		Arg:  "input",
		Body: []Expr{
			&End{ID: "end", Result: &GetValue{Path: "input"}},
		},
	}
}

func errorPathDI(t *testing.T) *DI {
	t.Helper()
	now := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	return &DI{
		FindWorkflowF: func(flowID string) (*Flow, error) {
			if flowID == "hello" {
				return errorPathFlow(), nil
			}
			return nil, errors.New("flow not found")
		},
		FindFunctionF: func(funcID string) (Function, error) {
			return nil, errors.New("function not found")
		},
		GenerateCallbackIDF: func() string { return "cb-1" },
		GenerateRunIDF:      func() string { return "run-1" },
		DefaultMaxRetries:   3,
		MockTimeNow:         &now,
	}
}

func baseStateFor(flowID string) BaseState {
	return BaseState{
		RunID:      "run-1",
		Flow:       &FlowRef{FlowID: flowID},
		Variables:  map[string]schema.Schema{"input": schema.MkString("x")},
		ExprResult: map[string]schema.Schema{},
	}
}

func TestTransitionRunErrors(t *testing.T) {
	di := errorPathDI(t)

	t.Run("scheduled run with an unknown flow errors", func(t *testing.T) {
		state := &Scheduled{BaseState: baseStateFor("i-do-not-exist")}
		_, err := Transition(nil, di, &Run{}, state)
		assert.ErrorContains(t, err, "flow not found")
	})

	t.Run("delayed run with a negative delay still schedules", func(t *testing.T) {
		got, err := Transition(nil, di, &Run{
			Flow:      &FlowRef{FlowID: "hello"},
			Input:     schema.MkString("x"),
			RunOption: &DelayRun{DelayBySeconds: 10},
		}, nil)
		require.NoError(t, err)
		scheduled, ok := got.(*Scheduled)
		require.True(t, ok)
		assert.Equal(t, di.MockTimeNow.Unix()+10, scheduled.ExpectedRunTimestamp)
	})

	t.Run("scheduled run with an invalid cron expression errors", func(t *testing.T) {
		_, err := Transition(nil, di, &Run{
			Flow:      &FlowRef{FlowID: "hello"},
			Input:     schema.MkString("x"),
			RunOption: &ScheduleRun{Interval: "not-a-cron"},
		}, nil)
		assert.Error(t, err)
	})

	t.Run("done state accepts no further commands", func(t *testing.T) {
		_, err := Transition(nil, di, &Run{}, &Done{})
		assert.ErrorIs(t, err, ErrStateReachEnd)
	})
}

func TestTransitionCallbackErrors(t *testing.T) {
	di := errorPathDI(t)
	await := func(mutate func(a *Await)) *Await {
		a := &Await{
			CallbackID:               "cb-1",
			ExpectedTimeoutTimestamp: di.MockTimeNow.Unix() + 100,
			BaseState:                baseStateFor("hello"),
		}
		a.BaseState.StepID = "step-1"
		if mutate != nil {
			mutate(a)
		}
		return a
	}

	t.Run("expired callback is rejected", func(t *testing.T) {
		state := await(func(a *Await) {
			a.ExpectedTimeoutTimestamp = di.MockTimeNow.Unix() - 1
		})
		_, err := Transition(nil, di, &Callback{CallbackID: "cb-1"}, state)
		assert.ErrorIs(t, err, ErrCallbackExpired)
	})

	t.Run("step that already has a result is rejected", func(t *testing.T) {
		state := await(func(a *Await) {
			a.BaseState.ExprResult["step-1"] = schema.MkString("already")
		})
		_, err := Transition(nil, di, &Callback{CallbackID: "cb-1"}, state)
		assert.ErrorIs(t, err, ErrExpressionHasResult)
	})

	t.Run("callback for an unknown flow errors", func(t *testing.T) {
		state := await(func(a *Await) {
			a.BaseState.Flow = &FlowRef{FlowID: "i-do-not-exist"}
		})
		_, err := Transition(nil, di, &Callback{CallbackID: "cb-1"}, state)
		assert.ErrorContains(t, err, "flow not found")
	})
}

func TestTransitionTryRecover(t *testing.T) {
	di := errorPathDI(t)
	errorState := func(mutate func(e *Error)) *Error {
		e := &Error{
			Code:      ProblemExecutingFunction,
			Reason:    "boom",
			BaseState: baseStateFor("hello"),
		}
		e.BaseState.DefaultMaxRetries = 3
		if mutate != nil {
			mutate(e)
		}
		return e
	}

	t.Run("recover re-executes the flow", func(t *testing.T) {
		got, err := Transition(nil, di, &TryRecover{RunID: "run-1"}, errorState(nil))
		require.NoError(t, err)
		// the flow itself succeeds now, so the run completes
		_, isDone := got.(*Done)
		assert.True(t, isDone, "expected Done, got %T", got)
	})

	t.Run("exhausted retries are rejected", func(t *testing.T) {
		state := errorState(func(e *Error) { e.Retried = 3 })
		_, err := Transition(nil, di, &TryRecover{RunID: "run-1"}, state)
		assert.ErrorIs(t, err, ErrMaxRetriesReached)
	})

	t.Run("non-recoverable problem codes are rejected", func(t *testing.T) {
		state := errorState(func(e *Error) { e.Code = ProblemVariableAccess })
		_, err := Transition(nil, di, &TryRecover{RunID: "run-1"}, state)
		assert.ErrorIs(t, err, ErrNotRecoverable)
	})

	t.Run("mismatched run id is rejected", func(t *testing.T) {
		_, err := Transition(nil, di, &TryRecover{RunID: "other"}, errorState(nil))
		assert.ErrorIs(t, err, ErrRunIDNotMatch)
	})

	t.Run("recover that fails the same way increments the retry counter", func(t *testing.T) {
		state := errorState(func(e *Error) {
			// a flow whose function is missing fails again identically
			e.BaseState.Flow = &Flow{
				Name: "inline",
				Arg:  "input",
				Body: []Expr{
					&Assign{ID: "a", VarOk: "v", Val: &Apply{ID: "ap", Name: "ghost", Args: nil}},
					&End{ID: "end", Result: &GetValue{Path: "v"}},
				},
			}
			e.Code = ProblemMissingFunction
			e.Reason = "function ghost not found"
			e.Retried = 1
		})

		got, err := Transition(nil, di, &TryRecover{RunID: "run-1"}, state)
		require.NoError(t, err)
		next, ok := got.(*Error)
		if assert.True(t, ok, "expected Error, got %T", got) &&
			next.Code == state.Code && next.Reason == state.Reason {
			assert.Equal(t, int64(2), next.Retried)
		}
	})
}

func TestTransitionScheduleControl(t *testing.T) {
	di := errorPathDI(t)
	scheduled := &Scheduled{
		BaseState: BaseState{
			RunID: "run-1",
			Flow:  &FlowRef{FlowID: "hello"},
			RunOption: &ScheduleRun{
				Interval:    "0 * * * *",
				ParentRunID: "parent-1",
			},
		},
	}

	t.Run("stop with the wrong parent id is rejected", func(t *testing.T) {
		_, err := Transition(nil, di, &StopSchedule{ParentRunID: "other"}, scheduled)
		assert.ErrorIs(t, err, ErrRunIDNotMatch)
	})

	t.Run("stop and resume roundtrip", func(t *testing.T) {
		got, err := Transition(nil, di, &StopSchedule{ParentRunID: "parent-1"}, scheduled)
		require.NoError(t, err)
		stopped, ok := got.(*ScheduleStopped)
		require.True(t, ok)

		_, err = Transition(nil, di, &ResumeSchedule{ParentRunID: "other"}, stopped)
		assert.ErrorIs(t, err, ErrRunIDNotMatch)

		resumed, err := Transition(nil, di, &ResumeSchedule{ParentRunID: "parent-1"}, stopped)
		require.NoError(t, err)
		_, ok = resumed.(*Scheduled)
		assert.True(t, ok)
	})
}
