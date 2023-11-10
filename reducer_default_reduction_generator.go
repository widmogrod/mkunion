package mkunion

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed reducer_default_reduction_generator.go.tmpl
	defaultReductionTmpl string
)

func NewReducerDefaultReductionGenerator(
	name variantName,
	types []typeName,
	helper *Helpers,
) *ReducerDefaultReductionGenerator {
	return &ReducerDefaultReductionGenerator{
		Name:     name,
		Types:    types,
		Helper:   helper,
		template: template.Must(template.New("main").Funcs(helper.Func()).Parse(defaultReductionTmpl)),
	}
}

type ReducerDefaultReductionGenerator struct {
	Name     variantName
	Types    []typeName
	Helper   *Helpers
	template *template.Template
}

func (t *ReducerDefaultReductionGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := t.template.ExecuteTemplate(result, "main", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
