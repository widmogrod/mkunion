package generators

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed visitor_default_generator.go.tmpl
	optionalVisitorTmpl string
)

func NewVisitorDefaultGenerator(name string, types []string, helper *Helpers) *VisitorDefaultGenerator {
	return &VisitorDefaultGenerator{
		Name:     name,
		Types:    types,
		Helper:   helper,
		template: template.Must(template.New("main").Funcs(helper.Func()).Parse(optionalVisitorTmpl)),
	}
}

type VisitorDefaultGenerator struct {
	Name     string
	Types    []string
	Helper   *Helpers
	template *template.Template
}

func (g *VisitorDefaultGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := g.template.ExecuteTemplate(result, "main", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
