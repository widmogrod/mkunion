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
	traverseTmpl   string
	renderTraverse = template.Must(template.New("main").Parse(traverseTmpl))
)

type Branching struct {
	Lit  *string
	List *string
	Map  *string
}

type ReducerDepthFirstGenerator struct {
	Header      string
	Name        variantName
	Types       []typeName
	PackageName string
	Branches    map[typeName][]Branching
}

func (t *ReducerDepthFirstGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := renderTraverse.ExecuteTemplate(result, "main", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
