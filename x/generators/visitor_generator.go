package generators

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
)

func NewVisitorGenerator(union *shape.UnionLike) *VisitorGenerator {
	return &VisitorGenerator{
		union: union,
	}
}

type VisitorGenerator struct {
	union                 *shape.UnionLike
	skipImportsAndPackage bool
}

func (g *VisitorGenerator) SkipImportsAndPackage(flag bool) {
	g.skipImportsAndPackage = flag
}

func (g *VisitorGenerator) Generate() ([]byte, error) {
	body := &bytes.Buffer{}
	result, err := g.GenerateVisitorInterfaces(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.VisitorGenerator.Generate: when generating visitor interfaces; %w", err)
	}
	body.Write(result)

	result, err = g.GenerateVisitorImplementation(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.VisitorGenerator.Generate: when generating visitor methods; %w", err)
	}
	body.Write(result)

	result, err = g.GenerateMatchFunctions(g.union, 3)
	if err != nil {
		return nil, fmt.Errorf("generators.VisitorGenerator.Generate: when generating match functions; %w", err)
	}
	body.Write(result)

	header := &bytes.Buffer{}
	if !g.skipImportsAndPackage {
		header.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.union)))
	}

	if header.Len() > 0 {
		return append(header.Bytes(), body.Bytes()...), nil
	} else {
		return body.Bytes(), nil
	}
}

func (g *VisitorGenerator) variantType(x shape.Shape) string {
	return shape.ToGoTypeName(x,
		shape.WithRootPkgName(shape.ToGoPkgName(g.union)))
}

func (g *VisitorGenerator) variantTypeInstantiated(x shape.Shape) string {
	return shape.ToGoTypeName(x,
		shape.WithRootPkgName(shape.ToGoPkgName(g.union)),
		shape.WithInstantiation())
}

func (g *VisitorGenerator) GenerateVisitorInterfaces(x *shape.UnionLike) ([]byte, error) {
	result := &bytes.Buffer{}

	result.WriteString(fmt.Sprintf("type %sVisitor%s interface {\n", x.Name, g.typeParamsNames(x)))
	for _, v := range x.Variant {
		result.WriteString(fmt.Sprintf("\tVisit%s(v *%s) any\n", g.variantName(v), g.variantType(v)))
	}
	result.WriteString("}\n\n")

	result.WriteString(fmt.Sprintf("type %s%s interface {\n", x.Name, g.typeParamsNames(x)))
	result.WriteString(fmt.Sprintf("\tAccept%s(g %sVisitor%s) any\n", x.Name, x.Name, g.typeParamsNamesWithoutType(x)))
	result.WriteString("}\n\n")

	return result.Bytes(), nil
}

func (g *VisitorGenerator) GenerateVisitorImplementation(x *shape.UnionLike) ([]byte, error) {
	result := &bytes.Buffer{}

	// generate var assertions to interface for each method
	result.WriteString(fmt.Sprintf("var (\n"))
	for _, v := range x.Variant {
		result.WriteString(fmt.Sprintf("\t_ %s%s = (*%s)(nil)\n", x.Name, g.typeParamsNamesInstantiated(x), g.variantTypeInstantiated(v)))
	}
	result.WriteString(fmt.Sprintf(")\n\n"))

	// generate method implementation
	for _, v := range x.Variant {
		result.WriteString(fmt.Sprintf("func (r *%s) Accept%s(v %sVisitor%s) any { return v.Visit%s(r) }\n",
			g.variantType(v), x.Name, x.Name, g.typeParamsNamesWithoutType(x), g.variantName(v)))
	}

	result.WriteString("\n")

	return result.Bytes(), nil
}

func MatchUnionFuncName(x *shape.UnionLike, returns int) string {
	return fmt.Sprintf("Match%sR%d", x.Name, returns)
}

func (g *VisitorGenerator) GenerateMatchFunctions(x *shape.UnionLike, returns int) ([]byte, error) {
	result := &bytes.Buffer{}

	typeName := shape.ToGoTypeName(x, shape.WithRootPkgName(shape.ToGoPkgName(g.union)))
	genTypes := g.mkOutTypeNamesForTypeParams(x.TypeParams)

	for ; returns > 0; returns-- {
		outTypes := g.mkOutTypeNames(returns)
		if genTypes != "" {
			result.WriteString(fmt.Sprintf("func %s[%s, %s any](\n", MatchUnionFuncName(x, returns), genTypes, outTypes))
		} else {
			result.WriteString(fmt.Sprintf("func %s[%s any](\n", MatchUnionFuncName(x, returns), outTypes))
		}
		result.WriteString(fmt.Sprintf("\tx %s,\n", typeName))
		for i, v := range x.Variant {
			result.WriteString(fmt.Sprintf("\tf%d func(x *%s)", i+1, g.variantType(v)))
			if returns == 1 {
				result.WriteString(fmt.Sprintf(" %s,\n", outTypes))
			} else {
				result.WriteString(fmt.Sprintf(" (%s),\n", outTypes))
			}
		}

		if returns == 1 {
			result.WriteString(fmt.Sprintf(") %s {\n", outTypes))
		} else {
			result.WriteString(fmt.Sprintf(") (%s) {\n", outTypes))
		}

		result.WriteString(fmt.Sprintf("\tswitch v := x.(type) {\n"))
		for i, v := range x.Variant {
			result.WriteString(fmt.Sprintf("\tcase *%s:\n", g.variantType(v)))
			result.WriteString(fmt.Sprintf("\t\treturn f%d(v)\n", i+1))
		}
		result.WriteString(fmt.Sprintf("\t}\n"))

		// init zero values
		for i := 0; i < returns; i++ {
			result.WriteString(fmt.Sprintf("\tvar result%d %s\n", i+1, g.mkOutTypeName(i)))
		}

		// return zero values
		result.WriteString(fmt.Sprintf("\treturn "))
		for i := 0; i < returns; i++ {
			if i > 0 {
				result.WriteString(", ")
			}
			result.WriteString(fmt.Sprintf("result%d", i+1))
		}
		result.WriteString(fmt.Sprintf("\n"))

		result.WriteString(fmt.Sprintf("}\n\n"))
	}

	// render match function with no return values
	if genTypes != "" {
		result.WriteString(fmt.Sprintf("func %s[%s](\n", MatchUnionFuncName(x, 0), genTypes))
	} else {
		result.WriteString(fmt.Sprintf("func %s(\n", MatchUnionFuncName(x, 0)))
	}
	result.WriteString(fmt.Sprintf("\tx %s,\n", typeName))
	for i, v := range x.Variant {
		result.WriteString(fmt.Sprintf("\tf%d func(x *%s),\n", i+1, g.variantType(v)))
	}
	result.WriteString(fmt.Sprintf(") {\n"))
	result.WriteString(fmt.Sprintf("\tswitch v := x.(type) {\n"))
	for i, v := range x.Variant {
		result.WriteString(fmt.Sprintf("\tcase *%s:\n", g.variantType(v)))
		result.WriteString(fmt.Sprintf("\t\tf%d(v)\n", i+1))
	}
	result.WriteString(fmt.Sprintf("\t}\n"))
	result.WriteString(fmt.Sprintf("}\n"))

	return result.Bytes(), nil
}

func (g *VisitorGenerator) mkOutTypeNames(returns int) string {
	var result string
	for i := 0; i < returns; i++ {
		if i > 0 {
			result += ", "
		}
		result += g.mkOutTypeName(i)
	}
	return result
}

func (g *VisitorGenerator) mkOutTypeNamesForTypeParams(x []shape.TypeParam) string {
	if len(x) == 0 {
		return ""
	}

	rootPackage := shape.WithRootPkgName(shape.ToGoPkgName(g.union))

	var result string
	for i := 0; i < len(x); i++ {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s %s", x[i].Name, shape.ToGoTypeName(x[i].Type, rootPackage))
	}

	return result
}

func (g *VisitorGenerator) mkOutTypeName(i int) string {
	return fmt.Sprintf("T%d", i)
}

func (g *VisitorGenerator) variantName(x shape.Shape) string {
	return TemplateHelperShapeVariantToName(x)
}

func (g *VisitorGenerator) typeParamsNames(x *shape.UnionLike) string {
	rootPackage := shape.WithRootPkgName(shape.ToGoPkgName(g.union))

	var result string
	for _, typeParam := range x.TypeParams {
		if len(result) > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s %s", typeParam.Name, shape.ToGoTypeName(typeParam.Type, rootPackage))
	}

	if len(result) > 0 {
		result = "[" + result + "]"
	}

	return result
}

func (g *VisitorGenerator) typeParamsNamesInstantiated(x *shape.UnionLike) string {
	rootPackage := shape.WithRootPkgName(shape.ToGoPkgName(g.union))

	var result string
	for _, typeParam := range x.TypeParams {
		if len(result) > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s", shape.ToGoTypeName(typeParam.Type, rootPackage))
	}

	if len(result) > 0 {
		result = "[" + result + "]"
	}

	return result
}

func (g *VisitorGenerator) typeParamsNamesWithoutType(x *shape.UnionLike) string {
	var result string

	for _, typeParam := range x.TypeParams {
		if len(result) > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s", typeParam.Name)
	}

	if len(result) > 0 {
		result = "[" + result + "]"
	}

	return result
}
