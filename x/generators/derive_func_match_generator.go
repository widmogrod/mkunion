package generators

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed derive_func_match_generator.go.tmpl
	deriveFuncMatchTmpl   string
	deriveFuncMatchRender = template.Must(template.New("derive_func_match_generator.go.tmpl").Funcs(map[string]any{
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
	}).Parse(deriveFuncMatchTmpl))
)

type DeriveFuncMatchGenerator struct {
	Header      string
	PackageName string
	MatchSpec   MatchSpec
}

func (g *DeriveFuncMatchGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := deriveFuncMatchRender.ExecuteTemplate(result, "derive_func_match_generator.go.tmpl", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
