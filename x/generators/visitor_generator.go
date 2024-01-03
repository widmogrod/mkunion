package generators

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"text/template"
)

var (
	//go:embed visitor_generator.go.tmpl
	visitorTmpl string
)

func NewVisitorGenerator(union *shape.UnionLike) *VisitorGenerator {
	return &VisitorGenerator{
		Union:    union,
		template: template.Must(template.New("visitor_generator.go.tmpl").Parse(visitorTmpl)),
		pkgUsed: PkgMap{
			"f": "github.com/widmogrod/mkunion/f",
		},
	}
}

type VisitorGenerator struct {
	Union                 *shape.UnionLike
	template              *template.Template
	skipImportsAndPackage bool
	pkgUsed               PkgMap
}

func (g *VisitorGenerator) SkipImportsAndPackage(flag bool) {
	g.skipImportsAndPackage = flag
}

func (g *VisitorGenerator) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *VisitorGenerator) ExtractImports(x shape.Shape) PkgMap {
	// add default and necessary imports
	pkgMap := g.pkgUsed

	// remove self from importing
	delete(pkgMap, shape.ToGoPkgName(x))
	return pkgMap
}

func (g *VisitorGenerator) VariantName(x shape.Shape) string {
	return TemplateHelperShapeVariantToName(x)
}
func (g *VisitorGenerator) GenIntSlice(from, to int) []int {
	var result []int
	for i := from; i <= to; i++ {
		result = append(result, i)
	}
	return result
}
func (g *VisitorGenerator) Add(a, b int) int {
	return a + b
}

func (g *VisitorGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}

	if !g.skipImportsAndPackage {
		result.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.Union)))

		pkgMap := g.ExtractImports(g.Union)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return nil, fmt.Errorf("generators.VisitorGenerator.Generate: when generating imports; %w", err)
		}
		result.WriteString(impPart)
	}

	err := g.template.ExecuteTemplate(result, "visitor_generator.go.tmpl", g)

	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
