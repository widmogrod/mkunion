package mkunion

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed schema_generator.go.tmpl
	visitorSchemaTmpl string
	renderSchema      = template.Must(template.New("main").Parse(visitorSchemaTmpl))
)

type SchemaGenerator struct {
	Header      string
	Types       []string
	Name        string
	PackageName string
}

func (g *SchemaGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := renderSchema.ExecuteTemplate(result, "main", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
