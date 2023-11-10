package mkunion

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed schema_generator.go.tmpl
	visitorSchemaTmpl string
)

func NewSchemaGenerator(name string, types []string, helper *Helpers) *SchemaGenerator {
	return &SchemaGenerator{
		Name:     name,
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
