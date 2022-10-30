package mkunion

import (
	"bytes"
	"text/template"
)

type ReducerDefaultReductionGenerator struct {
	Name        variantName
	Types       []typeName
	PackageName string
}

var (
	defaultReductionTmpl = Header + `
package {{ .PackageName }}
{{ $root := . }}
{{- $name := .Name }}
var _ {{ $name }}Reducer[any] = (*{{ $name }}DefaultReduction[any])(nil)

type (
	{{ $name }}DefaultReduction[A any] struct {
		PanicOnFallback bool
		DefaultStopReduction bool
		{{- range .Types }}
		On{{ . }} func(x *{{ . }}, agg A) (result A, stop bool)
		{{- end }}
	}
)
{{ range $i, $type := .Types }}
func (t *{{ $name }}DefaultReduction[A]) Reduce{{ $type }}(x *{{ $type }}, agg A) (result A, stop bool) {
	if t.On{{ $type }} != nil {
		return t.On{{ $type }}(x, agg)
	}
	if t.PanicOnFallback {
		panic("no fallback allowed on undefined ReduceBranch")
	}
	return agg, t.DefaultStopReduction
}
{{ end }}`
)

var (
	defaultReduction = template.Must(template.New("main").Parse(defaultReductionTmpl))
)

func (t *ReducerDefaultReductionGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := defaultReduction.ExecuteTemplate(result, "main", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
