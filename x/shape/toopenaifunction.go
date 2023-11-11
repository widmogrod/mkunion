package shape

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func ToOpenAIFunctionDefinition(name, desc string, in Shape) openai.FunctionDefinition {
	return openai.FunctionDefinition{
		Name:        name,
		Description: desc,
		Parameters:  toFunctionParameters(in),
	}
}

func toFunctionParameters(in Shape) *jsonschema.Definition {
	return MustMatchShape(
		in,
		func(x *Any) *jsonschema.Definition {
			//TODO: this should be list of all possible types [object, string, number, boolean, null]
			panic("not implemented")
			//return &jsonschema.Definition{
			//	Type: jsonschema.Null,
			//}
		},
		func(x *RefName) *jsonschema.Definition {
			// TODO: this should be list of all possible types [object, string, number, boolean, null]
			panic("not implemented")
			//return &jsonschema.Definition{
			//	Type: jsonschema.Null,
			//}
		},
		func(x *BooleanLike) *jsonschema.Definition {
			return &jsonschema.Definition{
				Type: jsonschema.Boolean,
			}
		},
		func(x *StringLike) *jsonschema.Definition {
			return &jsonschema.Definition{
				Type: jsonschema.String,
			}
		},
		func(x *NumberLike) *jsonschema.Definition {
			return &jsonschema.Definition{
				Type: jsonschema.Number,
			}
		},
		func(x *ListLike) *jsonschema.Definition {
			return &jsonschema.Definition{
				Type:  jsonschema.Array,
				Items: toFunctionParameters(x.Element),
			}
		},
		func(x *MapLike) *jsonschema.Definition {
			return &jsonschema.Definition{
				Type: jsonschema.Object,
				// TODO: this should be list of all possible types [object, string, number, boolean, null]
				//AdditionalProperties: toFunctionParameters(x.Val),
			}
		},
		func(x *StructLike) *jsonschema.Definition {
			properties := map[string]jsonschema.Definition{}
			for _, field := range x.Fields {
				def := toOpenAIFieldName(field.Guard, toFunctionParameters(field.Type))
				if field.Desc != nil {
					def.Description = *field.Desc
				}

				properties[field.Name] = *def
			}

			return &jsonschema.Definition{
				Type:       jsonschema.Object,
				Properties: properties,
				Required:   requireFields(x.Fields),
			}
		},
		func(x *UnionLike) *jsonschema.Definition {
			panic("not implemented")
			//return &jsonschema.Definition{
			//	OneOf: toFunctionParameters(x.Variant),
			//}
		},
	)
}

func requireFields(fields []*FieldLike) []string {
	var result []string
	for _, field := range fields {
		if _, ok := field.Guard.(*Required); ok {
			result = append(result, field.Name)
		}
	}
	return result
}

func toOpenAIFieldName(guard Guard, field *jsonschema.Definition) *jsonschema.Definition {
	if guard == nil {
		return field
	}

	return MustMatchGuard(
		guard,
		func(y *Enum) *jsonschema.Definition {
			field.Enum = y.Val
			return field
		},
		func(y *Required) *jsonschema.Definition {
			return field
		},
		func(y *AndGuard) *jsonschema.Definition {
			for _, guard := range y.L {
				field = toOpenAIFieldName(guard, field)
			}
			return field
		},
	)
}
