package workflow

import (
	"github.com/widmogrod/mkunion/x/schema"
)

type (
	Function func(args *FunctionInput) (*FunctionOutput, error)

	//FunctionDef struct {
	//	Name string
	//	Input schema.ShapeDef
	//	Output schema.ShapeDef
	//}
	FunctionInput struct {
		CallbackID string
		Args       []schema.Schema
		//ArgsDef schema.TypeDef
	}
	FunctionOutput struct {
		Result schema.Schema
	}
)
