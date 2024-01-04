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

func (g *VisitorGenerator) GenerateVisitorInterfaces(x *shape.UnionLike) ([]byte, error) {
	result := &bytes.Buffer{}

	result.WriteString(fmt.Sprintf("type %sVisitor interface {\n", x.Name))
	for _, v := range x.Variant {
		result.WriteString(fmt.Sprintf("\tVisit%s(v *%s) any\n", g.variantName(v), g.variantName(v)))
	}
	result.WriteString("}\n\n")

	result.WriteString(fmt.Sprintf("type %s interface {\n", x.Name))
	result.WriteString(fmt.Sprintf("\tAccept%s(g %sVisitor) any\n", x.Name, x.Name))
	result.WriteString("}\n\n")

	return result.Bytes(), nil
}

func (g *VisitorGenerator) GenerateVisitorImplementation(x *shape.UnionLike) ([]byte, error) {
	result := &bytes.Buffer{}

	// generate var assertions to interface for each method
	result.WriteString(fmt.Sprintf("var (\n"))
	for _, v := range x.Variant {
		result.WriteString(fmt.Sprintf("\t_ %s = (*%s)(nil)\n", x.Name, g.variantName(v)))
	}
	result.WriteString(fmt.Sprintf(")\n\n"))

	// generate method implementation
	for _, v := range x.Variant {
		result.WriteString(fmt.Sprintf("func (r *%s) Accept%s(v %sVisitor) any { return v.Visit%s(r) }\n",
			g.variantName(v), x.Name, x.Name, g.variantName(v)))
	}

	result.WriteString("\n")

	return result.Bytes(), nil
}

func MatchUnionFuncName(x *shape.UnionLike, returns int) string {
	return fmt.Sprintf("Match%sR%d", x.Name, returns)
}

func (g *VisitorGenerator) GenerateMatchFunctions(x *shape.UnionLike, returns int) ([]byte, error) {
	result := &bytes.Buffer{}

	for ; returns > 0; returns-- {
		outTypes := g.mkOutTypeNames(returns)
		result.WriteString(fmt.Sprintf("func %s[%s any](\n", MatchUnionFuncName(x, returns), outTypes))
		result.WriteString(fmt.Sprintf("\tx %s,\n", x.Name))
		for i, v := range x.Variant {
			result.WriteString(fmt.Sprintf("\tf%d func(x *%s) (%s),\n", i+1, g.variantName(v), outTypes))
		}

		result.WriteString(fmt.Sprintf(") (%s) {\n", outTypes))

		result.WriteString(fmt.Sprintf("\tswitch v := x.(type) {\n"))
		for i, v := range x.Variant {
			result.WriteString(fmt.Sprintf("\tcase *%s:\n", g.variantName(v)))
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
	result.WriteString(fmt.Sprintf("func %s(\n", MatchUnionFuncName(x, 0)))
	result.WriteString(fmt.Sprintf("\tx %s,\n", x.Name))
	for i, v := range x.Variant {
		result.WriteString(fmt.Sprintf("\tf%d func(x *%s),\n", i+1, g.variantName(v)))
	}
	result.WriteString(fmt.Sprintf(") {\n"))
	result.WriteString(fmt.Sprintf("\tswitch v := x.(type) {\n"))
	for i, v := range x.Variant {
		result.WriteString(fmt.Sprintf("\tcase *%s:\n", g.variantName(v)))
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

func (g *VisitorGenerator) mkOutTypeName(i int) string {
	return fmt.Sprintf("T%d", i)
}

func (g *VisitorGenerator) variantName(x shape.Shape) string {
	return TemplateHelperShapeVariantToName(x)
}
