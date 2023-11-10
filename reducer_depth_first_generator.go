package mkunion

import (
	"bytes"
	_ "embed"
	"text/template"
)

func PtrStr(x string) *string {
	return &x
}

type typeName = string
type variantName = string

var (
	//go:embed reducer_depth_first_generator.go.tmpl
	traverseTmpl string
)

type Branching struct {
	Lit  *string
	List *string
	Map  *string
}

func NewReducerDepthFirstGenerator(
	name variantName,
	types []typeName,
	branches map[typeName][]Branching,
	helper *Helpers,
) *ReducerDepthFirstGenerator {
	return &ReducerDepthFirstGenerator{
		Name:     name,
		Types:    types,
		Branches: branches,
		Helper:   helper,
		template: template.Must(template.New("main").Funcs(helper.Func()).Parse(traverseTmpl)),
	}
}

type ReducerDepthFirstGenerator struct {
	Name     variantName
	Types    []typeName
	Branches map[typeName][]Branching
	Helper   *Helpers
	template *template.Template
}

func (t *ReducerDepthFirstGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := t.template.ExecuteTemplate(result, "main", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
