package workflow

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"strings"
)

// ToStrWorkflow returns string representation of workflow AST,
// a string is a meta program code, similar to go code, just declarative.
// Example:
//
//	func FlowHelloWorld(input string) string {
//		var res string
//		res = concat("hello ", input)
//		return res
//	}
func ToStrWorkflow(workflow Worflow, depth int) string {
	result := strings.Builder{}

	return MustMatchWorflow(
		workflow,
		func(x *Flow) string {
			result.WriteString("flow ")
			result.WriteString(x.Name)
			result.WriteString("(")
			result.WriteString(x.Arg)
			result.WriteString(")  {\n")
			for _, expr := range x.Body {
				result.WriteString(ToStrExpr(expr, depth+1))
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

func ToStrExpr(expr Expr, depth int) string {
	result := strings.Builder{}

	return MustMatchExpr(
		expr,
		func(x *End) string {
			result.WriteString(strings.Repeat("\t", depth))
			result.WriteString("return ")
			result.WriteString(ToStrReshaper(x.Result, depth+1))
			return result.String()
		},
		func(x *Assign) string {
			result.WriteString(strings.Repeat("\t", depth))
			result.WriteString("var ")
			result.WriteString(x.VarOk)
			result.WriteString(" = ")
			result.WriteString(ToStrExpr(x.Val, depth+1))
			return result.String()
		},
		func(x *Apply) string {
			if x.Await != nil {
				result.WriteString("await ")
			}
			result.WriteString(x.Name)
			result.WriteString("(")
			for i, arg := range x.Args {
				result.WriteString(ToStrReshaper(arg, depth+1))
				if i < len(x.Args)-1 {
					result.WriteString(", ")
				}
			}
			result.WriteString(")")

			if x.Await != nil {
				result.WriteString(fmt.Sprintf(" @timeout(%d)", x.Await.Timeout))
			}

			return result.String()
		},
		func(x *Choose) string {
			result.WriteString(strings.Repeat("\t", depth))
			result.WriteString("if ")
			result.WriteString(ToStrPredicate(x.If, depth+1))
			result.WriteString(" {\n")
			for _, expr := range x.Then {
				result.WriteString(ToStrExpr(expr, depth+1))
				result.WriteString("\n")
			}

			if len(x.Else) > 0 {
				result.WriteString("} else {\n")
				for _, expr := range x.Else {
					result.WriteString(ToStrExpr(expr, depth+1))
					result.WriteString("\n")
				}
			}

			result.WriteString(strings.Repeat("\t", depth))
			result.WriteString("}")
			return result.String()
		},
	)
}

func ToStrReshaper(reshaper Reshaper, depth int) string {
	return MustMatchReshaper(
		reshaper,
		func(x *GetValue) string {
			return x.Path
		},
		func(x *SetValue) string {
			return ToStrSchema(x.Value, depth+1)
		},
	)
}

func ToStrSchema(x schema.Schema, depth int) string {
	return schema.MustMatchSchema(
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
				result.WriteString(ToStrSchema(v, depth+1))
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
				result.WriteString(ToStrSchema(v, depth+1))
			}
			result.WriteString("}")
			return result.String()
		},
	)
}

func ToStrPredicate(predicate Predicate, depth int) string {
	return MustMatchPredicate(
		predicate,
		func(x *And) string {
			result := strings.Builder{}
			result.WriteString("(")
			for i, p := range x.L {
				result.WriteString(ToStrPredicate(p, depth+1))
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
				result.WriteString(ToStrPredicate(p, depth+1))
				if i < len(x.L)-1 {
					result.WriteString(" || ")
				}
			}
			result.WriteString(")")
			return result.String()
		},
		func(x *Not) string {
			return "!" + ToStrPredicate(x.P, depth+1)
		},
		func(x *Compare) string {
			return ToStrReshaper(x.Left, depth+1) + x.Operation + ToStrReshaper(x.Right, depth+1)
		},
	)
}
