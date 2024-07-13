package workflow

import (
	"github.com/google/go-cmp/cmp"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
	"time"
)

func TestToStrWorkflow(t *testing.T) {
	program := &Flow{
		Name: "hello_world_flow",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				ID:    "assign-res",
				VarOk: "res",
				Val: &Apply{
					ID:   "apply-concat",
					Name: "concat",
					Args: []Reshaper{
						&SetValue{Value: schema.MkString("hello ")},
						&GetValue{Path: "input"},
					},
					Await: &ApplyAwaitOptions{
						TimeoutSeconds: int64(time.Second * 10),
					},
				},
			},
			&Choose{
				ID: "choose-res",
				If: &Compare{
					Operation: "=",
					Left:      &GetValue{Path: "res"},
					Right:     &SetValue{Value: schema.MkString("hello world")},
				},
				Then: []Expr{
					&End{
						ID:     "end-then-res",
						Result: &GetValue{Path: "res"},
					},
				},
				Else: []Expr{
					&End{
						ID:     "end-else-res",
						Result: &SetValue{Value: schema.MkString("only Spanish will work!")},
					},
				},
			},
		},
	}

	t.Run("Workflow without context", func(t *testing.T) {
		result := ToStrWorkflow(program, nil)
		expected := `flow hello_world_flow(input)  {
	var res = await concat("hello ", input) @timeout(10000000000)
	if res="hello world" {
		return res
	} else {
		return "only Spanish will work!"
	}
}
`
		if diff := cmp.Diff(result, expected); diff != "" {
			t.Errorf("unexpected result (-want +got):\n%s", diff)
		}
	})

	t.Run("Workflow with error context", func(t *testing.T) {
		state := &Error{
			Code:   "function-execution",
			Reason: "function concat() returned error: expected string, got <nil>",
			BaseState: BaseState{
				RunID:  "run-id-xxxxxxxxxx",
				StepID: "apply-concat",
				Flow:   &FlowRef{FlowID: "hello_world_flow"},
				Variables: map[string]schema.Schema{
					"input": nil,
				},
				ExprResult:        map[string]schema.Schema{},
				DefaultMaxRetries: 3,
			},
		}

		result := ToStrWorkflow(program, ToStrContextFromState(state))
		expected := `flow hello_world_flow(input)  {
	var res = await concat("hello ", input) @timeout(10000000000)
	
		🔺error: function-execution:
			function concat() returned error: expected string, got <nil>
	
	if res="hello world" {
		return res
	} else {
		return "only Spanish will work!"
	}
}
`

		if diff := cmp.Diff(result, expected); diff != "" {
			t.Errorf("unexpected result (-want +got):\n%s", diff)
		}
	})
}
