package generators

import (
	"bytes"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
)

func NewSerdeGraphQLUnion(union *shape.UnionLike) *SerdeGraphQLUnion {
	return &SerdeGraphQLUnion{
		union:                 union,
		skipImportsAndPackage: false,
		skipInitFunc:          false,
		pkgUsed: PkgMap{
			"fmt":     "fmt",
			"strings": "strings",
			"context": "context",
		},
	}
}

type SerdeGraphQLUnion struct {
	union                 *shape.UnionLike
	skipImportsAndPackage bool
	skipInitFunc          bool
	pkgUsed               PkgMap
}

func (g *SerdeGraphQLUnion) SkipImportsAndPackage(x bool) {
	g.skipImportsAndPackage = x
}

func (g *SerdeGraphQLUnion) SkipInitFunc(flag bool) *SerdeGraphQLUnion {
	g.skipInitFunc = flag
	return g
}

func (g *SerdeGraphQLUnion) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *SerdeGraphQLUnion) ExtractImports(x shape.Shape) PkgMap {
	pkgMap := shape.ExtractPkgImportNames(x)
	if pkgMap == nil {
		pkgMap = make(map[string]string)
	}

	// add default and necessary imports
	pkgMap = MergePkgMaps(pkgMap, g.pkgUsed)

	// remove self from importing
	delete(pkgMap, shape.ToGoPkgName(x))
	return pkgMap
}

func (g *SerdeGraphQLUnion) VariantName(x shape.Shape) string {
	return TemplateHelperShapeVariantToName(x)
}

func (g *SerdeGraphQLUnion) Generate() ([]byte, error) {
	body := &bytes.Buffer{}

	// For experimental GraphQL support, we'll generate schema and resolver templates
	body.WriteString("// Experimental GraphQL support - schema and resolver templates\n\n")

	// Generate GraphQL schema
	schema, err := g.GenerateUnionSchema(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeGraphQLUnion.Generate: when generating union schema; %w", err)
	}
	body.Write(schema)

	// Generate resolver templates
	resolvers, err := g.GenerateUnionResolvers(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeGraphQLUnion.Generate: when generating resolvers; %w", err)
	}
	body.Write(resolvers)

	head := &bytes.Buffer{}
	if !g.skipImportsAndPackage {
		head.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.union)))

		pkgMap := g.ExtractImports(g.union)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return nil, fmt.Errorf("generators.SerdeGraphQLUnion.Generate: when generating imports %w", err)
		}
		head.WriteString(impPart)
	}

	if head.Len() > 0 {
		head.WriteString(body.String())
		return head.Bytes(), nil
	} else {
		return body.Bytes(), nil
	}
}

func (g *SerdeGraphQLUnion) constructionf(x shape.Shape, template string) string {
	typeParams := shape.ExtractTypeParams(x)
	typeName := fmt.Sprintf(template, shape.Name(x))
	if len(typeParams) == 0 {
		return typeName
	}

	typeName += "["
	for i, t := range typeParams {
		if i > 0 {
			typeName += ", "
		}
		paramType := shape.ToGoTypeName(t.Type,
			shape.WithRootPackage(g.union.PkgName),
		)
		typeName += fmt.Sprintf("%s %s", t.Name, paramType)
	}
	typeName += "]"

	return typeName
}

func (g *SerdeGraphQLUnion) parametrisedf(x shape.Shape, template string) string {
	typeParamNames := shape.ToGoTypeParamsNames(x)
	typeName := fmt.Sprintf(template, shape.Name(x))
	if len(typeParamNames) == 0 {
		return typeName
	}

	return fmt.Sprintf("%s[%s]",
		typeName,
		strings.Join(typeParamNames, ","),
	)
}

func (g *SerdeGraphQLUnion) GenerateUnionSchema(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	unionName := shape.Name(union)

	body.WriteString("/*\n")
	body.WriteString(fmt.Sprintf("GraphQL Schema for %s Union:\n\n", unionName))

	// Generate interface definition
	body.WriteString(fmt.Sprintf("interface %s {\n", unionName))
	body.WriteString("  __typename: String!\n")
	body.WriteString("}\n\n")

	// Generate union definition
	body.WriteString(fmt.Sprintf("union %sUnion = ", unionName))
	variantNames := make([]string, len(union.Variant))
	for i, variant := range union.Variant {
		variantNames[i] = g.VariantName(variant)
	}
	body.WriteString(strings.Join(variantNames, " | "))
	body.WriteString("\n\n")

	// Generate individual variant types
	for _, variant := range union.Variant {
		variantName := g.VariantName(variant)
		body.WriteString(fmt.Sprintf("type %s implements %s {\n", variantName, unionName))
		body.WriteString("  __typename: String!\n")
		
		// Add fields based on variant type
		err := shape.MatchShapeR1(
			variant,
			func(x *shape.Any) error {
				body.WriteString("  data: JSON\n")
				return nil
			},
			func(x *shape.RefName) error {
				body.WriteString(fmt.Sprintf("  # Fields from %s\n", x.Name))
				return nil
			},
			func(x *shape.PointerLike) error {
				return g.generateVariantFields(x.Type, body)
			},
			func(x *shape.AliasLike) error {
				return g.generateVariantFields(x.Type, body)
			},
			func(x *shape.PrimitiveLike) error {
				return shape.MatchPrimitiveKindR1(
					x.Kind,
					func(x *shape.BooleanLike) error {
						body.WriteString("  value: Boolean!\n")
						return nil
					},
					func(x *shape.StringLike) error {
						body.WriteString("  value: String!\n")
						return nil
					},
					func(x *shape.NumberLike) error {
						body.WriteString("  value: Float!\n")
						return nil
					},
				)
			},
			func(x *shape.ListLike) error {
				body.WriteString("  items: [JSON!]!\n")
				return nil
			},
			func(x *shape.MapLike) error {
				body.WriteString("  entries: [JSON!]!\n")
				return nil
			},
			func(x *shape.StructLike) error {
				return g.generateStructFields(x, body)
			},
			func(x *shape.UnionLike) error {
				body.WriteString("  # Nested union type\n")
				body.WriteString("  data: JSON\n")
				return nil
			},
		)
		
		if err != nil {
			return nil, fmt.Errorf("generators.SerdeGraphQLUnion.GenerateUnionSchema: variant %s; %w", variantName, err)
		}
		
		body.WriteString("}\n\n")
	}

	// Generate queries and mutations
	body.WriteString("extend type Query {\n")
	body.WriteString(fmt.Sprintf("  get%s(id: ID!): %s\n", unionName, unionName))
	body.WriteString(fmt.Sprintf("  list%ss: [%s!]!\n", unionName, unionName))
	body.WriteString("}\n\n")

	body.WriteString("extend type Mutation {\n")
	for _, variant := range union.Variant {
		variantName := g.VariantName(variant)
		body.WriteString(fmt.Sprintf("  create%s(input: %sInput!): %s\n", variantName, variantName, variantName))
	}
	body.WriteString("}\n\n")

	// Generate input types
	for _, variant := range union.Variant {
		variantName := g.VariantName(variant)
		body.WriteString(fmt.Sprintf("input %sInput {\n", variantName))
		
		err := shape.MatchShapeR1(
			variant,
			func(x *shape.Any) error {
				body.WriteString("  data: JSON\n")
				return nil
			},
			func(x *shape.RefName) error {
				body.WriteString("  # Input fields based on referenced type\n")
				return nil
			},
			func(x *shape.PointerLike) error {
				return g.generateInputFields(x.Type, body)
			},
			func(x *shape.AliasLike) error {
				return g.generateInputFields(x.Type, body)
			},
			func(x *shape.PrimitiveLike) error {
				return shape.MatchPrimitiveKindR1(
					x.Kind,
					func(x *shape.BooleanLike) error {
						body.WriteString("  value: Boolean!\n")
						return nil
					},
					func(x *shape.StringLike) error {
						body.WriteString("  value: String!\n")
						return nil
					},
					func(x *shape.NumberLike) error {
						body.WriteString("  value: Float!\n")
						return nil
					},
				)
			},
			func(x *shape.ListLike) error {
				body.WriteString("  items: [JSON!]!\n")
				return nil
			},
			func(x *shape.MapLike) error {
				body.WriteString("  entries: [JSON!]!\n")
				return nil
			},
			func(x *shape.StructLike) error {
				return g.generateStructInputFields(x, body)
			},
			func(x *shape.UnionLike) error {
				body.WriteString("  data: JSON\n")
				return nil
			},
		)
		
		if err != nil {
			return nil, fmt.Errorf("generators.SerdeGraphQLUnion.GenerateUnionSchema: input %s; %w", variantName, err)
		}
		
		body.WriteString("}\n\n")
	}

	body.WriteString("*/\n\n")

	return body.Bytes(), nil
}

func (g *SerdeGraphQLUnion) generateVariantFields(s shape.Shape, body *bytes.Buffer) error {
	return shape.MatchShapeR1(
		s,
		func(x *shape.Any) error {
			body.WriteString("  data: JSON\n")
			return nil
		},
		func(x *shape.RefName) error {
			body.WriteString(fmt.Sprintf("  # Fields from %s\n", x.Name))
			return nil
		},
		func(x *shape.PointerLike) error {
			return g.generateVariantFields(x.Type, body)
		},
		func(x *shape.AliasLike) error {
			return g.generateVariantFields(x.Type, body)
		},
		func(x *shape.PrimitiveLike) error {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) error {
					body.WriteString("  value: Boolean\n")
					return nil
				},
				func(x *shape.StringLike) error {
					body.WriteString("  value: String\n")
					return nil
				},
				func(x *shape.NumberLike) error {
					body.WriteString("  value: Float\n")
					return nil
				},
			)
		},
		func(x *shape.ListLike) error {
			body.WriteString("  items: [JSON!]\n")
			return nil
		},
		func(x *shape.MapLike) error {
			body.WriteString("  entries: [JSON!]\n")
			return nil
		},
		func(x *shape.StructLike) error {
			return g.generateStructFields(x, body)
		},
		func(x *shape.UnionLike) error {
			body.WriteString("  data: JSON\n")
			return nil
		},
	)
}

func (g *SerdeGraphQLUnion) generateInputFields(s shape.Shape, body *bytes.Buffer) error {
	return g.generateVariantFields(s, body) // Same logic for now
}

func (g *SerdeGraphQLUnion) generateStructFields(s *shape.StructLike, body *bytes.Buffer) error {
	for _, field := range s.Fields {
		fieldType := g.shapeToGraphQLType(field.Type)
		isRequired := !shape.IsPointer(field.Type)
		
		gqlFieldName := shape.TagGetValue(field.Tags, "graphql", field.Name)
		if gqlFieldName == "" {
			gqlFieldName = strings.ToLower(field.Name[:1]) + field.Name[1:] // camelCase
		}
		
		if isRequired {
			body.WriteString(fmt.Sprintf("  %s: %s!\n", gqlFieldName, fieldType))
		} else {
			body.WriteString(fmt.Sprintf("  %s: %s\n", gqlFieldName, fieldType))
		}
	}
	return nil
}

func (g *SerdeGraphQLUnion) generateStructInputFields(s *shape.StructLike, body *bytes.Buffer) error {
	return g.generateStructFields(s, body) // Same logic for now
}

func (g *SerdeGraphQLUnion) shapeToGraphQLType(s shape.Shape) string {
	return shape.MatchShapeR1(
		s,
		func(x *shape.Any) string {
			return "JSON"
		},
		func(x *shape.RefName) string {
			return x.Name
		},
		func(x *shape.PointerLike) string {
			return g.shapeToGraphQLType(x.Type)
		},
		func(x *shape.AliasLike) string {
			return shape.Name(x)
		},
		func(x *shape.PrimitiveLike) string {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) string {
					return "Boolean"
				},
				func(x *shape.StringLike) string {
					return "String"
				},
				func(x *shape.NumberLike) string {
					return "Float"
				},
			)
		},
		func(x *shape.ListLike) string {
			if shape.IsBinary(x) {
				return "String" // base64 encoded
			}
			elemType := g.shapeToGraphQLType(x.Element)
			return fmt.Sprintf("[%s!]", elemType)
		},
		func(x *shape.MapLike) string {
			return "JSON" // Maps are typically JSON in GraphQL
		},
		func(x *shape.StructLike) string {
			return shape.Name(x)
		},
		func(x *shape.UnionLike) string {
			return shape.Name(x)
		},
	)
}

func (g *SerdeGraphQLUnion) GenerateUnionResolvers(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	unionName := shape.Name(union)

	body.WriteString("/*\n")
	body.WriteString(fmt.Sprintf("Example GraphQL Resolvers for %s Union:\n\n", unionName))
	body.WriteString("// In your resolver file:\n\n")

	// Generate query resolvers
	body.WriteString(fmt.Sprintf("func (r *queryResolver) Get%s(ctx context.Context, id string) (%s, error) {\n", unionName, unionName))
	body.WriteString("    // Your query logic here\n")
	body.WriteString("    // Return appropriate variant based on data\n")
	body.WriteString("    return nil, nil\n")
	body.WriteString("}\n\n")

	body.WriteString(fmt.Sprintf("func (r *queryResolver) List%ss(ctx context.Context) ([]%s, error) {\n", unionName, unionName))
	body.WriteString("    // Your list logic here\n")
	body.WriteString("    return nil, nil\n")
	body.WriteString("}\n\n")

	// Generate mutation resolvers for each variant
	for _, variant := range union.Variant {
		variantName := g.VariantName(variant)
		body.WriteString(fmt.Sprintf("func (r *mutationResolver) Create%s(ctx context.Context, input %sInput) (*%s, error) {\n", variantName, variantName, variantName))
		body.WriteString("    // Your mutation logic here\n")
		body.WriteString(fmt.Sprintf("    return &%s{}, nil\n", variantName))
		body.WriteString("}\n\n")
	}

	// Generate type resolver for union
	body.WriteString(fmt.Sprintf("func (r *Resolver) %s() %sResolver {\n", unionName, unionName))
	body.WriteString(fmt.Sprintf("    return &%sResolver{r}\n", strings.ToLower(unionName)))
	body.WriteString("}\n\n")

	body.WriteString(fmt.Sprintf("type %sResolver struct{ *Resolver }\n\n", strings.ToLower(unionName)))

	body.WriteString(fmt.Sprintf("func (r *%sResolver) __resolveType(obj interface{}) (string, error) {\n", strings.ToLower(unionName)))
	body.WriteString("    switch obj.(type) {\n")
	for _, variant := range union.Variant {
		variantName := g.VariantName(variant)
		body.WriteString(fmt.Sprintf("    case *%s:\n", variantName))
		body.WriteString(fmt.Sprintf("        return \"%s\", nil\n", variantName))
	}
	body.WriteString("    default:\n")
	body.WriteString("        return \"\", fmt.Errorf(\"unknown type\")\n")
	body.WriteString("    }\n")
	body.WriteString("}\n")

	body.WriteString("*/\n\n")

	return body.Bytes(), nil
}