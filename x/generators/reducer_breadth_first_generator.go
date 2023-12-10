package generators

import (
	"bytes"
	_ "embed"
	"github.com/widmogrod/mkunion/x/shape"
	"text/template"
)

var (
	//go:embed reducer_breadth_first_generator.go.tmpl
	breadthFirstTmpl string
)

func NewReducerBreadthFirstGenerator(
	union shape.UnionLike,
	helper *Helpers,
) *ReducerBreadthFirstGenerator {
	types, branches := AdaptUnionToOldVersionOfGenerator(union)
	return &ReducerBreadthFirstGenerator{
		Name:     union.Name,
		Types:    types,
		Branches: branches,
		Helper:   helper,
		template: template.Must(template.New("reducer_breadth_first_generator.go.tmpl").Funcs(helper.Func()).Parse(breadthFirstTmpl)),
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
	err := t.template.ExecuteTemplate(result, "reducer_breadth_first_generator.go.tmpl", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
