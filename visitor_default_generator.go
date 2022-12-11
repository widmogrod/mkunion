package mkunion

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed visitor_default_generator.go.tmpl
	optionalVisitorTmpl   string
	optionalVisitorRender = template.Must(template.New("main").Parse(optionalVisitorTmpl))
)

type VisitorDefaultGenerator struct {
	Header      string
	Name        string
	Types       []string
	PackageName string
}

func (g *VisitorDefaultGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := optionalVisitorRender.ExecuteTemplate(result, "main", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
