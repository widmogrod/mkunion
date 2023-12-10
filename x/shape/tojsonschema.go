package shape

import (
	"strings"
)

// ToJsonSchema converts a Shape to a JSON Schema
// following specification https://json-schema.org/specification
func ToJsonSchema(s Shape) string {
	return toJsonSchema(s, map[string]string{}, 0, nil)
}

func toJsonSchema(s Shape, definitions map[string]string, depth int, desc *string) string {
	return MustMatchShape(
		s,
		func(x *Any) string {
			return `{"type": "any"` + toDescription(desc) + `}`
		},
		func(x *RefName) string {
			return `{"$ref": "#/$defs/` + x.Name + `"` + toDescription(desc) + `}`
		},
		func(x *BooleanLike) string {
			return `{"type": "boolean"` + toDescription(desc) + `}`
		},
		func(x *StringLike) string {
			return `{"type": "string"` + toDescription(desc) + `}`
		},
		func(x *NumberLike) string {
			return `{"type": "number"` + toDescription(desc) + `}`
		},
		func(x *ListLike) string {
			return `{"type": "array", "items": ` + toJsonSchema(x.Element, definitions, depth+1, nil) + toDefinitions(definitions, depth) + toDescription(desc) + `}`
		},
		func(x *MapLike) string {
			return `{"type": "object", "additionalProperties": ` + toJsonSchema(x.Val, definitions, depth+1, nil) + toDefinitions(definitions, depth) + toDescription(desc) + `}`
		},
		func(x *StructLike) string {
			var properties []string
			for _, field := range x.Fields {
				properties = append(properties, `"`+field.Name+`": `+toJsonSchema(field.Type, definitions, depth+1, field.Desc))
			}

			if _, ok := definitions[x.Name]; !ok {
				definitions[x.Name] = `{"type": "object", "properties": {` + strings.Join(properties, ", ") + `}}`
			}

			if depth > 0 {
				return `{"$ref": "#/$defs/` + x.Name + `"` + toDescription(desc) + `}`
			}

			return `{"$ref": "#/$defs/` + x.Name + `"` + toDefinitions(definitions, depth) + toDescription(desc) + `}`
		},
		func(x *UnionLike) string {
			var oneOf []string
			for _, variant := range x.Variant {
				oneOf = append(oneOf, toJsonSchema(variant, definitions, depth+1, nil))
			}

			if _, ok := definitions[x.Name]; !ok {
				definitions[x.Name] = `{"oneOf": [` + strings.Join(oneOf, ", ") + `]` + toDescription(desc) + `}`
			}

			if depth > 0 {
				return `{"$ref": "#/$defs/` + x.Name + `"` + toDescription(desc) + `}`
			}

			return `{"$ref": "#/$defs/` + x.Name + `"` + toDefinitions(definitions, depth) + toDescription(desc) + `}`
		},
	)
}

func toDescription(desc *string) string {
	if desc == nil {
		return ""
	}

	return `, "description": "` + *desc + `"`
}

func toDefinitions(definitions map[string]string, depth int) string {
	if depth != 0 {
		return ""
	}

	if len(definitions) == 0 {
		return ""
	}

	var result []string
	for name, schema := range definitions {
		result = append(result, `"`+name+`": `+schema)
	}

	return `, "$defs": {` + strings.Join(result, ", ") + `}, "$schema": "https://json-schema.org/draft/2020-12/schema"`
}
