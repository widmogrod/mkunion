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
	types, _ := AdaptUnionToOldVersionOfGenerator(union)
	return &SchemaGenerator{
		Name:     union.Name,
		Types:    types,
		Helper:   helper,
		template: template.Must(template.New("main").Funcs(helper.Func()).Parse(visitorSchemaTmpl)),
	}
}

type SchemaGenerator struct {
	Types    []string
	Name     string
	Helper   *Helpers
	template *template.Template
}

func (g *SchemaGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := g.template.ExecuteTemplate(result, "main", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
