package generators

import (
	"bytes"
	_ "embed"
	"github.com/widmogrod/mkunion/x/shape"
	"text/template"
)

var (
	//go:embed visitor_default_generator.go.tmpl
	optionalVisitorTmpl string
)

func NewVisitorDefaultGenerator(union shape.UnionLike, helper *Helpers) *VisitorDefaultGenerator {
	types, _ := AdaptUnionToOldVersionOfGenerator(union)
	return &VisitorDefaultGenerator{
		Name:     union.Name,
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
