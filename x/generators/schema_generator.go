package generators

import (
	"bytes"
	_ "embed"
	"github.com/widmogrod/mkunion/x/shape"
	"text/template"
)

var (
	//go:embed schema_generator.go.tmpl
	visitorSchemaTmpl string
)

func NewSchemaGenerator(union shape.UnionLike, helper *Helpers) *SchemaGenerator {
	return &SchemaGenerator{
		Union:    union,
		template: template.Must(template.New("schema_generator.go.tmpl").Funcs(helper.Func()).Parse(visitorSchemaTmpl)),
	}
}

type SchemaGenerator struct {
	Union    shape.UnionLike
	template *template.Template
}

func (g *SchemaGenerator) VariantName(x shape.Shape) string {
	return TemplateHelperShapeVariantToName(x)
}

func (g *SchemaGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := g.template.ExecuteTemplate(result, "schema_generator.go.tmpl", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
