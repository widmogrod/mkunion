package generators

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
)

func NewSerdeGraphQLTagged(shape shape.Shape) *SerdeGraphQLTagged {
	return &SerdeGraphQLTagged{
		shape:                 shape,
		skipImportsAndPackage: false,
		pkgUsed: PkgMap{
			"fmt":     "fmt",
			"strings": "strings",
			"strconv": "strconv",
		},
	}
}

type SerdeGraphQLTagged struct {
	shape                 shape.Shape
	skipImportsAndPackage bool
	pkgUsed               PkgMap
}

func (g *SerdeGraphQLTagged) SkipImportsAndPackage(flag bool) *SerdeGraphQLTagged {
	g.skipImportsAndPackage = flag
	return g
}

func (g *SerdeGraphQLTagged) Generate() (string, error) {
	body := &strings.Builder{}

	// Generate GraphQL schema for the type
	schemaComment, err := g.GenerateGraphQLSchema(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeGraphQLTagged.Generate: when generating GraphQL schema %w", err)
	}
	body.WriteString(schemaComment)

	// Generate resolver comment
	resolverComment, err := g.GenerateResolverComment(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeGraphQLTagged.Generate: when generating resolver comment %w", err)
	}
	body.WriteString(resolverComment)

	head := &strings.Builder{}
	if !g.skipImportsAndPackage {
		head.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.shape)))

		pkgMap := g.ExtractImports(g.shape)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return "", fmt.Errorf("generators.SerdeGraphQLTagged.Generate: when generating imports %w", err)
		}
		head.WriteString(impPart)
	}

	if head.Len() > 0 {
		head.WriteString(body.String())
		return head.String(), nil
	} else {
		return body.String(), nil
	}
}

func (g *SerdeGraphQLTagged) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *SerdeGraphQLTagged) ExtractImports(x shape.Shape) PkgMap {
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

func (g *SerdeGraphQLTagged) rootTypeName() string {
	return shape.ToGoTypeName(g.shape,
		shape.WithRootPackage(shape.ToGoPkgName(g.shape)),
	)
}

func (g *SerdeGraphQLTagged) GenerateGraphQLSchema(x shape.Shape) (string, error) {
	result := &strings.Builder{}
	
	typeName := g.rootTypeName()
	
	result.WriteString("/*\n")
	result.WriteString(fmt.Sprintf("GraphQL Schema for %s:\n\n", typeName))

	err := shape.MatchShapeR1(
		x,
		func(x *shape.Any) error {
			result.WriteString(fmt.Sprintf("scalar %s\n", typeName))
			return nil
		},
		func(x *shape.RefName) error {
			result.WriteString(fmt.Sprintf("# Reference to external type\n"))
			result.WriteString(fmt.Sprintf("scalar %s\n", typeName))
			return nil
		},
		func(x *shape.PointerLike) error {
			result.WriteString(fmt.Sprintf("# Pointer type - nullable\n"))
			return g.generateGraphQLTypeForShape(x.Type, typeName, result, true)
		},
		func(x *shape.AliasLike) error {
			result.WriteString(fmt.Sprintf("# Alias type\n"))
			return g.generateGraphQLTypeForShape(x.Type, typeName, result, false)
		},
		func(x *shape.PrimitiveLike) error {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) error {
					result.WriteString(fmt.Sprintf("scalar %s # Boolean type\n", typeName))
					return nil
				},
				func(x *shape.StringLike) error {
					result.WriteString(fmt.Sprintf("scalar %s # String type\n", typeName))
					return nil
				},
				func(x *shape.NumberLike) error {
					result.WriteString(fmt.Sprintf("scalar %s # Number type (Float)\n", typeName))
					return nil
				},
			)
		},
		func(x *shape.ListLike) error {
			if shape.IsBinary(x) {
				result.WriteString(fmt.Sprintf("scalar %s # Binary data (base64 encoded)\n", typeName))
			} else {
				elemType := g.shapeToGraphQLType(x.Element)
				result.WriteString(fmt.Sprintf("type %s {\n", typeName))
				result.WriteString(fmt.Sprintf("  items: [%s!]!\n", elemType))
				result.WriteString(fmt.Sprintf("}\n"))
			}
			return nil
		},
		func(x *shape.MapLike) error {
			keyType := g.shapeToGraphQLType(x.Key)
			valType := g.shapeToGraphQLType(x.Val)
			result.WriteString(fmt.Sprintf("type %sEntry {\n", typeName))
			result.WriteString(fmt.Sprintf("  key: %s!\n", keyType))
			result.WriteString(fmt.Sprintf("  value: %s!\n", valType))
			result.WriteString(fmt.Sprintf("}\n\n"))
			result.WriteString(fmt.Sprintf("type %s {\n", typeName))
			result.WriteString(fmt.Sprintf("  entries: [%sEntry!]!\n", typeName))
			result.WriteString(fmt.Sprintf("}\n"))
			return nil
		},
		func(x *shape.StructLike) error {
			result.WriteString(fmt.Sprintf("type %s {\n", typeName))
			for _, field := range x.Fields {
				fieldType := g.shapeToGraphQLType(field.Type)
				isRequired := !shape.IsPointer(field.Type)
				
				// Get GraphQL field name from tag or use field name
				gqlFieldName := shape.TagGetValue(field.Tags, "graphql", field.Name)
				if gqlFieldName == "" {
					gqlFieldName = strings.ToLower(field.Name[:1]) + field.Name[1:] // camelCase
				}
				
				if isRequired {
					result.WriteString(fmt.Sprintf("  %s: %s!\n", gqlFieldName, fieldType))
				} else {
					result.WriteString(fmt.Sprintf("  %s: %s\n", gqlFieldName, fieldType))
				}
			}
			result.WriteString(fmt.Sprintf("}\n"))
			return nil
		},
		func(x *shape.UnionLike) error {
			result.WriteString(fmt.Sprintf("# Union type - see union-specific schema\n"))
			result.WriteString(fmt.Sprintf("interface %s {\n", typeName))
			result.WriteString(fmt.Sprintf("  __typename: String!\n"))
			result.WriteString(fmt.Sprintf("}\n"))
			return nil
		},
	)

	if err != nil {
		return "", fmt.Errorf("generators.SerdeGraphQLTagged.GenerateGraphQLSchema: %w", err)
	}

	result.WriteString("*/\n\n")

	return result.String(), nil
}

func (g *SerdeGraphQLTagged) generateGraphQLTypeForShape(s shape.Shape, typeName string, result *strings.Builder, nullable bool) error {
	baseType := g.shapeToGraphQLType(s)
	if nullable {
		result.WriteString(fmt.Sprintf("scalar %s # Nullable %s\n", typeName, baseType))
	} else {
		result.WriteString(fmt.Sprintf("scalar %s # %s\n", typeName, baseType))
	}
	return nil
}

func (g *SerdeGraphQLTagged) shapeToGraphQLType(s shape.Shape) string {
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

func (g *SerdeGraphQLTagged) GenerateResolverComment(x shape.Shape) (string, error) {
	result := &strings.Builder{}
	
	typeName := g.rootTypeName()
	
	result.WriteString("/*\n")
	result.WriteString(fmt.Sprintf("Example GraphQL Resolver for %s:\n\n", typeName))
	result.WriteString("// In your resolver file:\n")
	result.WriteString(fmt.Sprintf("func (r *queryResolver) Get%s(ctx context.Context, id string) (*%s, error) {\n", typeName, typeName))
	result.WriteString(fmt.Sprintf("    // Your resolver logic here\n"))
	result.WriteString(fmt.Sprintf("    return &%s{}, nil\n", typeName))
	result.WriteString(fmt.Sprintf("}\n\n"))
	result.WriteString(fmt.Sprintf("func (r *mutationResolver) Create%s(ctx context.Context, input %sInput) (*%s, error) {\n", typeName, typeName, typeName))
	result.WriteString(fmt.Sprintf("    // Your mutation logic here\n"))
	result.WriteString(fmt.Sprintf("    return &%s{}, nil\n", typeName))
	result.WriteString(fmt.Sprintf("}\n"))
	result.WriteString("*/\n\n")

	return result.String(), nil
}