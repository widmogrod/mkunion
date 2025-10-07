package generators

import (
	"bytes"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"sort"
	"strings"
)

var (
	ErrNotSupported = fmt.Errorf("type not supported for generation")
)

func NewShapeTagged(shape shape.Shape) *ShapeTagged {
	return &ShapeTagged{
		shape:                 shape,
		skipImportsAndPackage: false,
		skipInitFunc:          false,
		pkgUsed: PkgMap{
			"shape": "github.com/widmogrod/mkunion/x/shape",
		},
	}
}

type ShapeTagged struct {
	shape                 shape.Shape
	skipImportsAndPackage bool
	skipInitFunc          bool
	pkgUsed               PkgMap
}

func (g *ShapeTagged) SkipImportsAndPackage(flag bool) *ShapeTagged {
	g.skipImportsAndPackage = flag
	return g
}

func (g *ShapeTagged) SkipInitFunc(flag bool) *ShapeTagged {
	g.skipInitFunc = flag
	return g
}

func (g *ShapeTagged) Generate() (string, error) {
	body := &bytes.Buffer{}
	marshalPart, err := g.generateShapeFunc(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.ShapeTagged.Generate: when generating shape func; %w", err)

	}
	marshalPart = g.removeShapePkgIfShapePkg(marshalPart)
	body.WriteString(marshalPart)

	head := &strings.Builder{}
	if !g.skipImportsAndPackage {
		head.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.shape)))

		pkgMap := g.ExtractImports(g.shape)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return "", fmt.Errorf("generators.ShapeTagged.Generate: when generating imports; %w", err)
		}
		impPart = g.removeShapePkgIfShapePkg(impPart)
		head.WriteString(impPart)
	}

	if !g.skipInitFunc {
		inits := g.ExtractImportFuncs(g.shape)
		varPart, err := g.GenerateInitFunc(inits)
		if err != nil {
			return "", fmt.Errorf("generators.ShapeTagged.Generate: when generating func init(); %w", err)
		}
		varPart = g.removeShapePkgIfShapePkg(varPart)
		head.WriteString(varPart)
	}

	if head.Len() > 0 {
		head.WriteString(body.String())
		return head.String(), nil
	} else {
		return body.String(), nil
	}
}

func (g *ShapeTagged) removeShapePkgIfShapePkg(x string) string {
	if shape.ToGoPkgName(g.shape) == "shape" {
		return strings.ReplaceAll(x, "shape.", "")
	}

	return x
}

func (g *ShapeTagged) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *ShapeTagged) ExtractImports(x shape.Shape) PkgMap {
	// add default and necessary imports
	pkgMap := g.pkgUsed

	// remove self from importing
	delete(pkgMap, shape.ToGoPkgName(x))
	return pkgMap
}

func (g *ShapeTagged) ExtractImportFuncs(s shape.Shape) []string {
	name, supports := TypeNameIfSupports(s)
	if !supports {
		return nil
	}

	result := []string{
		g.removeShapePkgIfShapePkg(fmt.Sprintf("shape.Register(%sShape())", name)),
	}

	switch x := s.(type) {
	case *shape.UnionLike:
		for _, variant := range x.Variant {
			result = append(result, g.ExtractImportFuncs(variant)...)
		}
	}

	return result
}

func (g *ShapeTagged) GenerateInitFunc(init []string) (string, error) {
	return GenerateInitFunc(init), nil
}

func (g *ShapeTagged) generateShapeFunc(s shape.Shape) (string, error) {
	switch x := s.(type) {
	case *shape.UnionLike:
		result := &bytes.Buffer{}
		// union func name is composed from those functions
		union, err := g.generateUnionFunc(x)
		if err != nil {
			return "", fmt.Errorf("generators.ShapeTagged.generateShapeFunc: when generating union func; %w", err)
		}

		result.WriteString("\n")
		result.WriteString(union)

		// for each variant generate a func
		for _, variant := range x.Variant {
			partial, err := g.generateVariantFunc(variant)
			if err != nil {
				return "", fmt.Errorf("generators.ShapeTagged.generateShapeFunc: when generating variant func; %w", err)
			}

			result.WriteString("\n")
			result.WriteString(partial)
		}

		return result.String(), nil
	}

	return g.generateVariantFunc(s)
}

func (g *ShapeTagged) generateUnionFunc(s *shape.UnionLike) (string, error) {
	name, supports := TypeNameIfSupports(s)
	if !supports {
		return "", fmt.Errorf("generators.ShapeTagged.generateUnionFunc: %w", ErrNotSupported)
	}

	result := &bytes.Buffer{}
	fmt.Fprintf(result, "func %sShape() shape.Shape {\n", name)
	fmt.Fprintf(result, "\treturn &shape.UnionLike{\n")
	fmt.Fprintf(result, "\t\tName: \"%s\",\n", s.Name)
	fmt.Fprintf(result, "\t\tPkgName: \"%s\",\n", s.PkgName)
	fmt.Fprintf(result, "\t\tPkgImportName: \"%s\",\n", s.PkgImportName)

	if len(s.TypeParams) > 0 {
		fmt.Fprintf(result, "\t\tTypeParams: []shape.TypeParam{\n")
		for _, param := range s.TypeParams {
			fmt.Fprintf(result, "%s,\n", padLeftTabs(3, TypeParamToString(param)))
		}
		fmt.Fprintf(result, "\t\t},\n")
	}

	fmt.Fprintf(result, "\t\tVariant: []shape.Shape{\n")
	for _, variant := range s.Variant {
		variantName, supports := TypeNameIfSupports(variant)
		if !supports {
			return "", fmt.Errorf("generators.ShapeTagged.generateUnionFunc: variant %v does not have name; %w", variant, ErrNotSupported)
		}
		fmt.Fprintf(result, "\t\t\t%sShape(),\n", variantName)
	}
	fmt.Fprintf(result, "\t\t},\n")
	fmt.Fprintf(result, "\t}\n")
	fmt.Fprintf(result, "}\n")

	return result.String(), nil
}

func (g *ShapeTagged) generateVariantFunc(s shape.Shape) (string, error) {
	name, supports := TypeNameIfSupports(s)
	if !supports {
		return "", fmt.Errorf("generators.ShapeTagged.generateVariantFunc: %w", ErrNotSupported)
	}

	result := &bytes.Buffer{}
	fmt.Fprintf(result, "func %sShape() shape.Shape {\n", name)
	fmt.Fprintf(result, "\treturn %s\n", padLeftTabs2(1, ShapeToString(s)))
	fmt.Fprintf(result, "}\n")

	return result.String(), nil
}

func ShapeToString(x shape.Shape) string {
	return shape.MatchShapeR1(
		x,
		func(x *shape.Any) string {
			return `&shape.Any{}`
		},
		func(x *shape.RefName) string {
			result := &bytes.Buffer{}

			fmt.Fprintf(result, "&shape.RefName{\n")
			fmt.Fprintf(result, "\tName: %q,\n", x.Name)
			fmt.Fprintf(result, "\tPkgName: %q,\n", x.PkgName)
			fmt.Fprintf(result, "\tPkgImportName: %q,\n", x.PkgImportName)

			//if x.IsPointer {
			//	fmt.Fprintf(result, "\tIsPointer: %v,\n", x.IsPointer)
			//}

			if len(x.Indexed) > 0 {
				fmt.Fprintf(result, "\tIndexed: []shape.Shape{\n")
				for _, indexed := range x.Indexed {
					fmt.Fprintf(result, "%s,\n", padLeftTabs(2, ShapeToString(indexed)))
				}
				fmt.Fprintf(result, "\t},\n")
			}

			fmt.Fprintf(result, "}")

			return result.String()
		},
		func(x *shape.PointerLike) string {
			result := &bytes.Buffer{}

			fmt.Fprintf(result, "&shape.PointerLike{\n")
			fmt.Fprintf(result, "\tType: %s,\n", padLeftTabs2(1, ShapeToString(x.Type)))
			fmt.Fprintf(result, "}")

			return result.String()
		},
		func(x *shape.AliasLike) string {
			result := &bytes.Buffer{}

			fmt.Fprintf(result, "&shape.AliasLike{\n")
			fmt.Fprintf(result, "\tName: %q,\n", x.Name)
			fmt.Fprintf(result, "\tPkgName: %q,\n", x.PkgName)
			fmt.Fprintf(result, "\tPkgImportName: %q,\n", x.PkgImportName)

			if len(x.TypeParams) > 0 {
				fmt.Fprintf(result, "\tTypeParams: []shape.TypeParam{\n")
				for _, param := range x.TypeParams {
					fmt.Fprintf(result, "%s,\n", padLeftTabs(2, TypeParamToString(param)))
				}
				fmt.Fprintf(result, "\t},\n")
			}

			if len(x.Tags) > 0 {
				fmt.Fprintf(result, "\tTags: %s,\n", padLeftTabs2(1, TagsToStr(x.Tags)))
			}
			if x.IsAlias {
				fmt.Fprintf(result, "\tIsAlias: %v,\n", x.IsAlias)
			}
			fmt.Fprintf(result, "\tType: %s,\n", padLeftTabs2(1, ShapeToString(x.Type)))
			fmt.Fprintf(result, "}")

			return result.String()
		},
		func(x *shape.PrimitiveLike) string {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) string {
					return "&shape.PrimitiveLike{Kind: &shape.BooleanLike{}}"
				},
				func(x *shape.StringLike) string {
					return "&shape.PrimitiveLike{Kind: &shape.StringLike{}}"
				},
				func(x *shape.NumberLike) string {
					result := &bytes.Buffer{}

					fmt.Fprintf(result, "&shape.PrimitiveLike{\n")
					fmt.Fprintf(result, "\tKind: &shape.NumberLike{")
					if x.Kind != nil {
						fmt.Fprintf(result, "\n")
						fmt.Fprintf(result, "\t\tKind: &%s,\n", KindToGoName(x.Kind))
					}
					fmt.Fprintf(result, "\t},\n")
					fmt.Fprintf(result, "}")

					return result.String()
				},
			)
		},
		func(x *shape.ListLike) string {
			result := &bytes.Buffer{}

			fmt.Fprintf(result, "&shape.ListLike{\n")
			fmt.Fprintf(result, "\tElement: %s,\n", padLeftTabs2(1, ShapeToString(x.Element)))
			if x.ArrayLen != nil {
				fmt.Fprintf(result, "\tArrayLen: shape.Ptr(%d),\n", *x.ArrayLen)
			}
			fmt.Fprintf(result, "}")

			return result.String()
		},
		func(x *shape.MapLike) string {
			result := &bytes.Buffer{}

			fmt.Fprintf(result, "&shape.MapLike{\n")
			fmt.Fprintf(result, "\tKey: %s,\n", padLeftTabs2(1, ShapeToString(x.Key)))
			fmt.Fprintf(result, "\tVal: %s,\n", padLeftTabs2(1, ShapeToString(x.Val)))
			fmt.Fprintf(result, "}")

			return result.String()
		},
		func(x *shape.StructLike) string {
			result := &bytes.Buffer{}

			fmt.Fprintf(result, "&shape.StructLike{\n")
			fmt.Fprintf(result, "\tName: %q,\n", x.Name)
			fmt.Fprintf(result, "\tPkgName: %q,\n", x.PkgName)
			fmt.Fprintf(result, "\tPkgImportName: %q,\n", x.PkgImportName)

			if len(x.TypeParams) > 0 {
				fmt.Fprintf(result, "\tTypeParams: []shape.TypeParam{\n")
				for _, param := range x.TypeParams {
					fmt.Fprintf(result, "%s,\n", padLeftTabs(2, TypeParamToString(param)))
				}
				fmt.Fprintf(result, "\t},\n")
			}

			if len(x.Fields) > 0 {
				fmt.Fprintf(result, "\tFields: []*shape.FieldLike{\n")
				for _, field := range x.Fields {
					fmt.Fprintf(result, "%s,\n", padLeftTabs(2, FieldLikeToString(field, true)))
				}
				fmt.Fprintf(result, "\t},\n")
			}

			if len(x.Tags) > 0 {
				fmt.Fprintf(result, "\tTags: %s,\n", padLeftTabs2(1, TagsToStr(x.Tags)))
			}

			fmt.Fprintf(result, "}")

			return result.String()
		},
		func(x *shape.UnionLike) string {
			result := &bytes.Buffer{}

			fmt.Fprintf(result, "&shape.UnionLike{\n")
			fmt.Fprintf(result, "\tName: %q,\n", x.Name)
			fmt.Fprintf(result, "\tPkgName: %q,\n", x.PkgName)
			fmt.Fprintf(result, "\tPkgImportName: %q,\n", x.PkgImportName)

			if len(x.TypeParams) > 0 {
				fmt.Fprintf(result, "\tTypeParams: []shape.TypeParam{\n")
				for _, param := range x.TypeParams {
					fmt.Fprintf(result, "%s,\n", padLeftTabs(2, TypeParamToString(param)))
				}
				fmt.Fprintf(result, "\t},\n")
			}

			if len(x.Variant) > 0 {
				fmt.Fprintf(result, "\tVariant: []shape.Shape{\n")
				for _, variant := range x.Variant {
					fmt.Fprintf(result, "%s,\n", padLeftTabs(3, ShapeToString(variant)))
				}
				fmt.Fprintf(result, "\t},\n")
			}

			fmt.Fprintf(result, "}")

			return result.String()
		},
	)
}

func KindToGoName(kind shape.NumberKind) string {
	return shape.MatchNumberKindR1(
		kind,
		func(x *shape.UInt) string {
			return "shape.UInt{}"
		},
		func(x *shape.UInt8) string {
			return "shape.UInt8{}"
		},
		func(x *shape.UInt16) string {
			return "shape.UInt16{}"
		},
		func(x *shape.UInt32) string {
			return "shape.UInt32{}"
		},
		func(x *shape.UInt64) string {
			return "shape.UInt64{}"
		},
		func(x *shape.Int) string {
			return "shape.Int{}"
		},
		func(x *shape.Int8) string {
			return "shape.Int8{}"
		},
		func(x *shape.Int16) string {
			return "shape.Int16{}"
		},
		func(x *shape.Int32) string {
			return "shape.Int32{}"
		},
		func(x *shape.Int64) string {
			return "shape.Int64{}"
		},
		func(x *shape.Float32) string {
			return "shape.Float32{}"
		},
		func(x *shape.Float64) string {
			return "shape.Float64{}"
		},
	)
}

func TypeParamToString(x shape.TypeParam) string {
	return fmt.Sprintf(`shape.TypeParam{
	Name: %q,
	Type: %s,
}`, x.Name, padLeftTabs2(2, ShapeToString(x.Type)))
}

func FieldLikeToString(x *shape.FieldLike, removeTypeName bool) string {
	result := &bytes.Buffer{}

	if removeTypeName {
		fmt.Fprintf(result, "{\n")
	} else {
		fmt.Fprintf(result, "&shape.FieldLike{\n")
	}

	fmt.Fprintf(result, "\tName: %q,\n", x.Name)
	fmt.Fprintf(result, "\tType: %s,\n", padLeftTabs2(1, ShapeToString(x.Type)))
	if x.Desc != nil {
		fmt.Fprintf(result, "\tDesc: %s,\n", PtrToString(x.Desc))
	}
	if x.Guard != nil {
		fmt.Fprintf(result, "\tGuard: %s,\n", padLeftTabs2(1, GuardToString(x.Guard)))
	}
	if len(x.Tags) > 0 {
		fmt.Fprintf(result, "\tTags: %s,\n", padLeftTabs2(1, TagsToStr(x.Tags)))
	}
	fmt.Fprintf(result, "}")

	return result.String()
}

func GuardToString(x shape.Guard) string {
	if x == nil {
		return "nil"
	}

	return shape.MatchGuardR1(
		x,
		func(x *shape.Enum) string {
			result := &bytes.Buffer{}

			fmt.Fprintf(result, "&shape.Enum{\n")
			if len(x.Val) > 0 {
				fmt.Fprintf(result, "\tVal: []string{\n")
				for _, val := range x.Val {
					fmt.Fprintf(result, "\t\t%q,\n", val)
				}
				fmt.Fprintf(result, "\t},\n")
			}
			fmt.Fprintf(result, "}")

			return result.String()
		},
		func(x *shape.Required) string {
			return "&shape.Required{}"
		},
		func(x *shape.AndGuard) string {
			result := &bytes.Buffer{}

			fmt.Fprintf(result, "&shape.AndGuard{\n")
			if len(x.L) > 0 {
				fmt.Fprintf(result, "\tGuards: []shape.Guard{\n")
				for _, guard := range x.L {
					fmt.Fprintf(result, "%s,\n", padLeftTabs(2, GuardToString(guard)))
				}
				fmt.Fprintf(result, "\t},\n")
			}
			fmt.Fprintf(result, "}")

			return result.String()
		},
	)
}

func PtrToString(x *string) string {
	if x == nil {
		return "nil"
	}

	return fmt.Sprintf(`shape.Ptr(%q)`, *x)
}

func TagsToStr(x map[string]shape.Tag) string {
	result := &bytes.Buffer{}

	fmt.Fprintf(result, "map[string]shape.Tag{\n")

	// Sort keys for deterministic output
	keys := make([]string, 0, len(x))
	for k := range x {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := x[k]
		fmt.Fprintf(result, "\t%q: %s,\n", k, padLeftTabs2(1, TagToStr(v, true)))
	}
	fmt.Fprintf(result, "}")

	return result.String()
}

func TagToStr(x shape.Tag, removeTypeName bool) string {
	result := &bytes.Buffer{}

	if removeTypeName {
		fmt.Fprintf(result, "{\n")
	} else {
		fmt.Fprintf(result, "shape.Tag{\n")
	}

	fmt.Fprintf(result, "\tValue: %q,\n", x.Value)

	if len(x.Options) > 0 {
		fmt.Fprintf(result, "\tOptions: []string{\n")
		for _, option := range x.Options {
			fmt.Fprintf(result, "\t\t%q,\n", option)
		}
		fmt.Fprintf(result, "\t},\n")
	}

	fmt.Fprintf(result, "}")

	return result.String()
}
