package workflow

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"strings"
)

type (
	ToStrContext struct {
		Errors map[StepID]ToStrErrInfo
	}

	ToStrErrInfo struct {
		StepID  string
		Code    string
		Message string
	}
)

func ToStrContextFromState(x State) *ToStrContext {
	result := &ToStrContext{}

	MatchStateR0(
		x,
		func(x *NextOperation) {},
		func(x *Done) {},
		func(x *Error) {
			result.Errors = map[StepID]ToStrErrInfo{
				x.BaseState.StepID: {
					StepID:  x.BaseState.StepID,
					Code:    x.Code,
					Message: x.Reason,
				},
			}
		},
		func(x *Await) {},
		func(x *Scheduled) {},
		func(x *ScheduleStopped) {})

	return result
}

func injectError(result *strings.Builder, c *ToStrContext, id StepID) *strings.Builder {
	if c == nil {
		return result
	}

	if info, ok := c.Errors[id]; ok {
		result.WriteString(fmt.Sprintf("\n\n\t🔺error: %s:\n\t\t%s\n", info.Code, info.Message))
	}

	return result
}

// ToStrWorkflow returns string representation of workflow AST,
// a string is a meta program code, similar to go code, just declarative.
// Example:
//
//	func FlowHelloWorld(input string) string {
//		var res string
//		res = concat("hello ", input)
//		return res
//	}
func ToStrWorkflow(workflow Workflow, c *ToStrContext) string {
	result := strings.Builder{}

	return MatchWorkflowR1(
		workflow,
		func(x *Flow) string {
			result.WriteString("flow ")
			result.WriteString(x.Name)
			result.WriteString("(")
			result.WriteString(x.Arg)
			result.WriteString(")  {\n")
			for _, expr := range x.Body {
				result.WriteString(PadLeft(1, ToStrExpr(expr, c)))
				result.WriteString("\n")
			}
			result.WriteString("}\n")
			return result.String()

		},
		func(x *FlowRef) string {
			return x.FlowID
		},
	)
}

func ToStrExpr(expr Expr, c *ToStrContext) string {
	result := &strings.Builder{}
	result = MatchExprR1(
		expr,
		func(x *End) *strings.Builder {
			result.WriteString("return ")
			result.WriteString(ToStrReshaper(x.Result))
			return result
		},
		func(x *Assign) *strings.Builder {
			result.WriteString("var ")
			result.WriteString(x.VarOk)
			result.WriteString(" = ")
			result.WriteString(ToStrExpr(x.Val, c))
			return result
		},
		func(x *Apply) *strings.Builder {
			if x.Await != nil {
				result.WriteString("await ")
			}
			result.WriteString(x.Name)
			result.WriteString("(")
			for i, arg := range x.Args {
				result.WriteString(ToStrReshaper(arg))
				if i < len(x.Args)-1 {
					result.WriteString(", ")
				}
			}
			result.WriteString(")")

			if x.Await != nil {
				result.WriteString(fmt.Sprintf(" @timeout(%d)", x.Await.TimeoutSeconds))
			}

			return result
		},
		func(x *Choose) *strings.Builder {
			result.WriteString("if ")
			result.WriteString(ToStrPredicate(x.If))
			result.WriteString(" {\n")
			for _, expr := range x.Then {
				result.WriteString(PadLeft(1, ToStrExpr(expr, c)))
				result.WriteString("\n")
			}

			if len(x.Else) > 0 {
				result.WriteString("} else {\n")
				for _, expr := range x.Else {
					result.WriteString(PadLeft(1, ToStrExpr(expr, c)))
					result.WriteString("\n")
				}
			}

			result.WriteString("}")
			return result
		},
	)

	result = injectError(result, c, GetStepIDFromExpr(expr))
	return result.String()
}

func ToStrReshaper(reshaper Reshaper) string {
	return MatchReshaperR1(
		reshaper,
		func(x *GetValue) string {
			return x.Path
		},
		func(x *SetValue) string {
			return ToStrSchema(x.Value)
		},
	)
}

func ToStrSchema(x schema.Schema) string {
	return schema.MatchSchemaR1(
		x,
		func(x *schema.None) string {
			return "none"
		},
		func(x *schema.Bool) string {
			return fmt.Sprintf("%t", *x)
		},
		func(x *schema.Number) string {
			return fmt.Sprintf("%f", *x)
		},
		func(x *schema.String) string {
			return fmt.Sprintf("%q", *x)
		},
		func(x *schema.Binary) string {
			return fmt.Sprintf("%q", *x)
		},
		func(x *schema.List) string {
			result := strings.Builder{}
			result.WriteString("[")
			for i, v := range *x {
				result.WriteString(ToStrSchema(v))
				if i < len(*x)-1 {
					result.WriteString(", ")
				}
			}
			result.WriteString("]")
			return result.String()
		},
		func(x *schema.Map) string {
			result := strings.Builder{}
			result.WriteString("{")
			i := 0
			for key, v := range *x {
				if i > 0 {
					result.WriteString(", ")
				}
				i++
				result.WriteString(fmt.Sprintf("%q: ", key))
				result.WriteString(PadLeft(1, ToStrSchema(v)))
			}
			result.WriteString("}")
			return result.String()
		},
	)
}

func ToStrPredicate(predicate Predicate) string {
	return MatchPredicateR1(
		predicate,
		func(x *And) string {
			result := strings.Builder{}
			result.WriteString("(")
			for i, p := range x.L {
				result.WriteString(ToStrPredicate(p))
				if i < len(x.L)-1 {
					result.WriteString(" && ")
				}
			}
			result.WriteString(")")
			return result.String()
		},
		func(x *Or) string {
			result := strings.Builder{}
			result.WriteString("(")
			for i, p := range x.L {
				result.WriteString(ToStrPredicate(p))
				if i < len(x.L)-1 {
					result.WriteString(" || ")
				}
			}
			result.WriteString(")")
			return result.String()
		},
		func(x *Not) string {
			return "!" + ToStrPredicate(x.P)
		},
		func(x *Compare) string {
			return ToStrReshaper(x.Left) + x.Operation + ToStrReshaper(x.Right)
		},
	)
}

func PadLeft(depth int, s string) string {
	// split new lines
	// add to each line \t repeated depth times
	// join lines
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat("\t", depth) + line
	}
	return strings.Join(lines, "\n")
}
