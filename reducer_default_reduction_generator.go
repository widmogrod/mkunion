package mkunion

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed reducer_default_reduction_generator.go.tmpl
	defaultReductionTmpl string
	defaultReduction     = template.Must(template.New("main").Parse(defaultReductionTmpl))
)

type ReducerDefaultReductionGenerator struct {
	Header      string
	Name        variantName
	Types       []typeName
	PackageName string
}

func (t *ReducerDefaultReductionGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := defaultReduction.ExecuteTemplate(result, "main", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
