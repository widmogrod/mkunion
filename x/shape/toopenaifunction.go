package shape

import (
	"fmt"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	log "github.com/sirupsen/logrus"
)

func ToOpenAIFunctionDefinition(name, desc string, in Shape) *openai.FunctionDefinition {
	return &openai.FunctionDefinition{
		Name:        name,
		Description: desc,
		Parameters:  toFunctionParameters(in),
	}
}

func toFunctionParameters(in Shape) *jsonschema.Definition {
	return MatchShapeR1(
		in,
		func(x *Any) *jsonschema.Definition {
			//TODO: this should be list of all possible types [object, string, number, boolean, null]
			log.Errorf("Any is not supported yet: %+v", x)
			//panic("not implemented")
			return &jsonschema.Definition{
				Type: jsonschema.Null,
			}
		},
		func(x *RefName) *jsonschema.Definition {
			// TODO: this should be list of all possible types [object, string, number, boolean, null]
			//log.Errorf("RefName is not supported yet: %+v", x)
			//panic("not implemented")
			return &jsonschema.Definition{
				Type: jsonschema.Null,
			}
		},
		func(x *PointerLike) *jsonschema.Definition {
			return toFunctionParameters(x.Type)
		},
		func(x *AliasLike) *jsonschema.Definition {
			return &jsonschema.Definition{
				Type: jsonschema.String,
			}
		},
		func(x *PrimitiveLike) *jsonschema.Definition {
			return MatchPrimitiveKindR1(
				x.Kind,
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
			)
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

				name := field.Name
				if field.Tags != nil {
					if v, ok := field.Tags["name"]; ok {
						name = v.Value
					}
				}

				properties[name] = *def
			}

			return &jsonschema.Definition{
				Type:       jsonschema.Object,
				Properties: properties,
				Required:   requireFields(x.Fields),
			}
		},
		func(x *UnionLike) *jsonschema.Definition {
			properties := map[string]jsonschema.Definition{}
			for _, variant := range x.Variant {
				def := toFunctionParameters(variant)
				variantName := toVariantName(variant)
				properties[variantName] = *def
			}

			return &jsonschema.Definition{
				Type:        jsonschema.Object,
				Description: "Each field is a variant of the union. Only one of them can be present in the object.",
				Properties:  properties,
			}
		},
	)
}

func toVariantName(x Shape) string {
	return MatchShapeR1(
		x,
		func(a *Any) string {
			return "any"
			//panic("not implemented")
		},
		func(x *RefName) string {
			//panic("not implemented")
			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
		func(x *PointerLike) string {
			return toVariantName(x.Type)
		},
		func(x *AliasLike) string {
			panic("not implemented")
		},
		func(x *PrimitiveLike) string {
			return MatchPrimitiveKindR1(
				x.Kind,
				func(x *BooleanLike) string {
					return "boolean"
				},
				func(x *StringLike) string {
					return "string"
				},
				func(x *NumberLike) string {
					return "number"
				},
			)
		},
		func(x *ListLike) string {
			return "list"
			//panic("not implemented")

		},
		func(x *MapLike) string {
			return "map"
			//panic("not implemented")

		},
		func(x *StructLike) string {
			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
		func(x *UnionLike) string {
			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
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

	return MatchGuardR1(
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
