package generators

import (
	"bytes"
	_ "embed"
	"github.com/widmogrod/mkunion/x/shape"
	"text/template"
)

var (
	//go:embed reducer_default_reduction_generator.go.tmpl
	defaultReductionTmpl string
)

func NewReducerDefaultReductionGenerator(
	union shape.UnionLike,
	helper *Helpers,
) *ReducerDefaultReductionGenerator {
	types, _ := AdaptUnionToOldVersionOfGenerator(union)
	return &ReducerDefaultReductionGenerator{
		Name:     union.Name,
		Types:    types,
		Helper:   helper,
		template: template.Must(template.New("reducer_default_reduction_generator.go.tmpl").Funcs(helper.Func()).Parse(defaultReductionTmpl)),
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
	err := t.template.ExecuteTemplate(result, "reducer_default_reduction_generator.go.tmpl", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
