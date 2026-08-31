package shapeopenai

import (
	"fmt"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/shape"
)

func ToOpenAIFunctionDefinition(name, desc string, in shape.Shape) *openai.FunctionDefinition {
	return &openai.FunctionDefinition{
		Name:        name,
		Description: desc,
		Parameters:  toFunctionParameters(in),
	}
}

func toFunctionParameters(in shape.Shape) *jsonschema.Definition {
	return shape.MatchShapeR1(
		in,
		func(x *shape.Any) *jsonschema.Definition {
			//TODO: this should be list of all possible types [object, string, number, boolean, null]
			log.Errorf("shape.Any is not supported yet: %+v", x)
			//panic("not implemented")
			return &jsonschema.Definition{
				Type: jsonschema.Null,
			}
		},
		func(x *shape.RefName) *jsonschema.Definition {
			// TODO: this should be list of all possible types [object, string, number, boolean, null]
			//log.Errorf("shape.RefName is not supported yet: %+v", x)
			//panic("not implemented")
			return &jsonschema.Definition{
				Type: jsonschema.Null,
			}
		},
		func(x *shape.PointerLike) *jsonschema.Definition {
			return toFunctionParameters(x.Type)
		},
		func(x *shape.AliasLike) *jsonschema.Definition {
			return &jsonschema.Definition{
				Type: jsonschema.String,
			}
		},
		func(x *shape.PrimitiveLike) *jsonschema.Definition {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) *jsonschema.Definition {
					return &jsonschema.Definition{
						Type: jsonschema.Boolean,
					}
				},
				func(x *shape.StringLike) *jsonschema.Definition {
					return &jsonschema.Definition{
						Type: jsonschema.String,
					}
				},
				func(x *shape.NumberLike) *jsonschema.Definition {
					return &jsonschema.Definition{
						Type: jsonschema.Number,
					}
				},
			)
		},
		func(x *shape.ListLike) *jsonschema.Definition {
			return &jsonschema.Definition{
				Type:  jsonschema.Array,
				Items: toFunctionParameters(x.Element),
			}
		},
		func(x *shape.MapLike) *jsonschema.Definition {
			return &jsonschema.Definition{
				Type: jsonschema.Object,
				// TODO: this should be list of all possible types [object, string, number, boolean, null]
				//AdditionalProperties: toFunctionParameters(x.Val),
			}
		},
		func(x *shape.StructLike) *jsonschema.Definition {
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
		func(x *shape.UnionLike) *jsonschema.Definition {
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

func toVariantName(x shape.Shape) string {
	return shape.MatchShapeR1(
		x,
		func(a *shape.Any) string {
			return "any"
			//panic("not implemented")
		},
		func(x *shape.RefName) string {
			//panic("not implemented")
			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
		func(x *shape.PointerLike) string {
			return toVariantName(x.Type)
		},
		func(x *shape.AliasLike) string {
			panic("not implemented")
		},
		func(x *shape.PrimitiveLike) string {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) string {
					return "boolean"
				},
				func(x *shape.StringLike) string {
					return "string"
				},
				func(x *shape.NumberLike) string {
					return "number"
				},
			)
		},
		func(x *shape.ListLike) string {
			return "list"
			//panic("not implemented")

		},
		func(x *shape.MapLike) string {
			return "map"
			//panic("not implemented")

		},
		func(x *shape.StructLike) string {
			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
		func(x *shape.UnionLike) string {
			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
	)

}

func requireFields(fields []*shape.FieldLike) []string {
	var result []string
	for _, field := range fields {
		if _, ok := field.Guard.(*shape.Required); ok {
			result = append(result, field.Name)
		}
	}
	return result
}

func toOpenAIFieldName(guard shape.Guard, field *jsonschema.Definition) *jsonschema.Definition {
	if guard == nil {
		return field
	}

	return shape.MatchGuardR1(
		guard,
		func(y *shape.Enum) *jsonschema.Definition {
			field.Enum = y.Val
			return field
		},
		func(y *shape.Required) *jsonschema.Definition {
			return field
		},
		func(y *shape.AndGuard) *jsonschema.Definition {
			for _, guard := range y.L {
				field = toOpenAIFieldName(guard, field)
			}
			return field
		},
	)
}
