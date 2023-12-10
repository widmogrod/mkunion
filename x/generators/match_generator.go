package generators

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"
)

var (
	//go:embed match_generator.go.tmpl
	matchTmpl   string
	renderMatch = template.Must(template.New("match_generator.go.tmpl").Funcs(map[string]any{
		"GenIntSlice": func(from, to int) []int {
			var result []int
			for i := from; i <= to; i++ {
				result = append(result, i)
			}
			return result
		},
		"Repeat": func(s string, n int) []string {
			var result []string
			for i := 0; i < n; i++ {
				result = append(result, s)
			}
			return result
		},
		"Join": func(s []string, sep string) string {
			return strings.Join(s, sep)
		},
	}).Parse(matchTmpl))
)

type FunctionMatchGenerator struct {
	Header      string
	PackageName string
	MaxSize     int
}

func (t *FunctionMatchGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := renderMatch.ExecuteTemplate(result, "match_generator.go.tmpl", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
