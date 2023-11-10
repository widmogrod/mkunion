package mkunion

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed reducer_breadth_first_generator.go.tmpl
	breadthFirstTmpl string
)

func NewReducerBreadthFirstGenerator(
	name variantName,
	types []typeName,
	branches map[typeName][]Branching,
	helper *Helpers,
) *ReducerBreadthFirstGenerator {
	return &ReducerBreadthFirstGenerator{
		Name:     name,
		Types:    types,
		Branches: branches,
		Helper:   helper,
		template: template.Must(template.New("main").Funcs(helper.Func()).Parse(breadthFirstTmpl)),
	}
}

type ReducerBreadthFirstGenerator struct {
	Name     variantName
	Types    []typeName
	Branches map[typeName][]Branching
	Helper   *Helpers
	template *template.Template
}

func (t *ReducerBreadthFirstGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := t.template.ExecuteTemplate(result, "main", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
