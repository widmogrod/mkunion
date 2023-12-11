package workflow

import (
	"github.com/widmogrod/mkunion/x/schema"
)

type Function func(args *FunctionInput) (*FunctionOutput, error)

//go:generate go run ../../cmd/mkunion/main.go -name=FunctionDSL
type (
	//FunctionDef struct {
	//	Name string
	//	Input schema.ShapeDef
	//	Output schema.ShapeDef
	//}
	FunctionInput struct {
		// Name acts as unique function ID
		Name string
		// CallbackID is used to identify callback function, and when its set
		// it means that function is async, and should return result by calling callback endpoint with CallbackID
		CallbackID string
		Args       []schema.Schema

		//ArgsDef schema.TypeDef
	}
	FunctionOutput struct {
		Result schema.Schema
	}
)
