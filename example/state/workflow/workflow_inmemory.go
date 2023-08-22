package workflow

import (
	"github.com/widmogrod/mkunion/x/schema"
)

var _ Dependency = (*DI)(nil)

type DI struct {
	FindWorkflowF func(flowID string) (*Flow, error)
}

func (di DI) FindWorkflow(flowID string) (*Flow, error) {
	return di.FindWorkflowF(flowID)
}

func (di DI) NewContext() *Context {
	result := &Context{
		Name:      "root",
		Variables: map[string]schema.Schema{},
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

	result.Root = result
	return result
}
