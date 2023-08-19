package workflow

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

type (
	Context struct {
		Functions map[string]Function
		Variables map[string]schema.Schema // variables are immutable, once set they can't be changed

		Result schema.Schema
		Name   string

		Root *Context
		Prev *Context

		ExecutionPath      []string                  // holds information how execution flow goes, what variables were set, and result of execution
		ExecutionVariables map[string]string         // holds what variables were set
		ExecutionInfo      map[string]*ExecutionInfo // holds information about execution
	}

	Function func(args []schema.Schema) (schema.Schema, error)

	ExecutionInfo struct {
		SetVariables []string
		DidFail      bool
		Retried      int
	}
)

func (c *Context) Errorf(format string, a ...any) error {
	return fmt.Errorf(fmt.Sprintf("%s: %s", c.Name, format), a...)
}

func (c *Context) GetForFlow(x Worflow) *Context {
	c.Root.ExecutionPath = append(c.Root.ExecutionPath, c.Name)
	//c.Name = c.Name + ".Flow"

	return &Context{
		Root: c,
		Prev: c,
		Name: c.Name + ".Flow",
	}
}

func (c *Context) GetForReshaper(x Reshaper) *Context {
	name := MustMatchReshaper(
		x,
		func(x *GetValue) string {
			return ".GetValue"
		},
		func(x *SetValue) string {
			return ".SetValue"
		},
	)

	c.Root.ExecutionPath = append(c.Root.ExecutionPath, c.Name)

	return &Context{
		Root: c.Root,
		Prev: c,
		Name: c.Name + name,
	}
}

func (c *Context) GetForExpr(x Expr) *Context {
	name := MustMatchExpr(
		x,
		func(x *End) string {
			return ".End"
		},
		func(x *Assign) string {
			return ".Assign"
		},
		func(x *Apply) string {
			return ".Apply"
		},
		func(x *Choose) string {
			return ".Choose"
		},
	)

	c.Root.ExecutionPath = append(c.Root.ExecutionPath, c.Name)

	return &Context{
		Root: c.Root,
		Prev: c,
		Name: c.Name + name,
	}
}

func (c *Context) GetVariable(name string) (schema.Schema, bool) {
	ctx := c
	for {
		value, ok := ctx.Variables[name]
		if ok {
			return value, ok
		}

		if ctx.Prev == nil {
			return nil, false
		}

		ctx = ctx.Prev
	}
}

func (c *Context) SetVariable(name string, val schema.Schema) (schema.Schema, error) {
	if _, ok := c.GetVariable(name); ok {
		return nil, c.Errorf("variable %s already set", name)
	}

	c.Root.Variables[name] = val
	c.Root.ExecutionVariables[name] = c.Name

	return val, nil
}

//func (c *Context) RecordExecutionInfo(info *ExecutionInfo) {
//	c.ExecutionInfo[c.Name] = info
//}

func TestExecution(t *testing.T) {
	program := &Flow{
		Name: "hello_world_flow",
		Arg:  "input",
		Body: []Expr{
			&Assign{
				Var: "res",
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

	context := &Context{
		Name:      "root",
		Variables: map[string]schema.Schema{
			//"input": schema.MkString("world"),
		},
		Functions: map[string]Function{
			"concat": func(args []schema.Schema) (schema.Schema, error) {
				return schema.MkString(
					schema.AsDefault(args[0], "") +
						schema.AsDefault(args[1], ""),
				), nil
			},
		},

		ExecutionVariables: make(map[string]string),
	}

	context.Root = context

	_, err := context.SetVariable("input", schema.MkString("world"))
	assert.NoError(t, err)

	result, err := ExecuteAll(context, program)
	assert.NoError(t, err)
	assert.Equal(t, schema.MkString("hello world"), result)

	expected := &Context{
		Variables: map[string]schema.Schema{
			"input": schema.MkString("world"),
			"res":   schema.MkString("hello world"),
		},
		Functions: context.Functions,

		Name:   "root",
		Result: schema.MkString("hello world"),

		ExecutionPath: []string{
			"root",
			"root.Flow",
			"root.Flow.Assign",
			"root.Flow.Assign.Apply",
			"root.Flow.Assign.Apply",
			//"root.Flow.Assign.Apply.SetValue",
			//"root.Flow.Assign.Apply.SetValue.GetValue",
			//"root.Flow.Assign.Apply.SetValue.GetValue.End",
			"root.Flow",
			"root.Flow.End",
		},
		ExecutionVariables: map[string]string{
			"input": "root",
			"res":   "root.Flow.Assign",
		},

		//ExecutionInfo: map[string]*Context{
		//	"root.Flow.Assign": {
		//		Variables: map[string]schema.Schema{
		//			"res": schema.MkString("hello world"),
		//		},
		//		Result: schema.MkString("hello world"),
		//		Err:    nil,
		//	},
		//},
	}
	expected.Root = expected
	assert.Equal(t, expected, context)
}

type ExecutionStack struct {
	Stack []ASTNode
	//Executed []Result{}
}

func (s *ExecutionStack) Push(node ASTNode) {
	s.Stack = append(s.Stack, node)
}

func (s *ExecutionStack) Pop() ASTNode {
	if len(s.Stack) == 0 {
		return nil
	}

	node := s.Stack[len(s.Stack)-1]
	s.Stack = s.Stack[:len(s.Stack)-1]
	return node
}

func (s *ExecutionStack) Peek() ASTNode {
	if len(s.Stack) == 0 {
		return nil
	}

	return s.Stack[len(s.Stack)-1]
}

func ExecuteAll(context *Context, program Worflow) (schema.Schema, error) {
	context = context.GetForFlow(program)
	return MustMatchWorflowR2(
		program,
		func(x *Flow) (schema.Schema, error) {
			for _, expr := range x.Body {
				_, err := ExecuteExpr(context, expr)
				if err != nil {
					return nil, context.Errorf("failed to execute expr: %w", err)
				}
			}

			return context.Root.Result, nil
		},
	)
}

func ExecuteReshaper(context *Context, reshaper Reshaper) (schema.Schema, error) {
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

func ExecuteExpr(context *Context, expr Expr) (schema.Schema, error) {
	context = context.GetForExpr(expr)
	return MustMatchExprR2(
		expr,
		func(x *End) (schema.Schema, error) {
			val, err := ExecuteReshaper(context, x.Result)
			if err != nil {
				return nil, context.Errorf("failed to execute result: %w", err)
			}

			//context.RecordExecutionInfo(&ExecutionInfo{
			//	EndOk: val,
			//})
			context.Root.Result = val
			return val, nil
		},
		func(x *Assign) (schema.Schema, error) {
			val, err := ExecuteExpr(context, x.Val)
			if err != nil {
				return nil, context.Errorf("failed to execute flow: %w", err)
			}

			return context.SetVariable(x.Var, val)
		},
		func(x *Apply) (schema.Schema, error) {
			args := make([]schema.Schema, len(x.Args))
			for i, arg := range x.Args {
				val, err := ExecuteReshaper(context, arg)
				if err != nil {
					return nil, context.Errorf("failed to execute arg: %w", err)
				}
				args[i] = val
			}

			if fn, ok := context.Root.Functions[x.Name]; ok {
				val, err := fn(args)
				if err != nil {
					return nil, context.Errorf("failed to execute function: %w", err)
				}
				return val, nil
			} else {
				return nil, context.Errorf("function %s not found", x.Name)
			}
		},
		func(x *Choose) (schema.Schema, error) {
			return nil, context.Errorf("not implemented")
		},
	)
}
