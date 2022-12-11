package mkunion

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed reducer_breadth_first_generator.go.tmpl
	breadthFirstTmpl   string
	renderBreadthFirst = template.Must(template.New("main").Parse(breadthFirstTmpl))
)

type ReducerBreadthFirstGenerator struct {
	Header      string
	Name        variantName
	Types       []typeName
	PackageName string
	Branches    map[typeName][]Branching
}

func (t *ReducerBreadthFirstGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := renderBreadthFirst.ExecuteTemplate(result, "main", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
