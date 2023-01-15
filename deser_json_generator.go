package mkunion

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed deser_json_generator.go.tmpl
	visitorDeSerJsonTmpl string
	renderDeSerJson      = template.Must(template.New("main").Funcs(map[string]any{
		"GenIntSlice": func(from, to int) []int {
			var result []int
			for i := from; i <= to; i++ {
				result = append(result, i)
			}
			return result
		},
		"Add": func(a, b int) int {
			return a + b
		},
	}).Parse(visitorDeSerJsonTmpl))
)

type DeSerJsonGenerator struct {
	Header      string
	Types       []string
	Name        string
	PackageName string
}

func (g *DeSerJsonGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := renderDeSerJson.ExecuteTemplate(result, "main", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
