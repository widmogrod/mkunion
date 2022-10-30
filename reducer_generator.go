package mkunion

import (
	"bytes"
	"text/template"
)

func PtrStr(x string) *string {
	return &x
}

type typeName = string
type variantName = string

type Branching struct {
	Lit  *string
	List *string
	Map  *string
}

type ReducerGenerator struct {
	Name        variantName
	Types       []typeName
	PackageName string
	Branches    map[typeName][]Branching
}

var (
	traverseTmpl = Header + `
package {{ .PackageName }}
{{ $root := . }}
{{- $name := .Name }}
type (
	{{ $name }}Reducer[A any] interface {
		{{- range .Types }}
		Reduce{{ . }}(x *{{ . }}, agg A) (result A, stop bool)
		{{- end }}
	}
)

type {{ $name }}DepthFirstVisitor[A any] struct {
	stop   bool
	result A
	reduce {{ $name }}Reducer[A]
}

var _ {{ $name }}Visitor = (*{{ $name }}DepthFirstVisitor[any])(nil)
{{ range $i, $type := .Types }}
func (d *{{ $name }}DepthFirstVisitor[A]) Visit{{ . }}(v *{{ . }}) any {
	d.result, d.stop = d.reduce.Reduce{{ . }}(v, d.result)
	if d.stop {
		return nil
	}
	
	{{- range (index $root.Branches $type) -}}
	{{- if .Lit}}
	if _ = v.{{ .Lit }}.Accept(d); d.stop {
		return nil
	}
	{{- else if .List }}
	for idx := range v.{{ .List }} {
		if _ = v.{{ .List }}[idx].Accept(d); d.stop {
			return nil
		}
	}
	{{- else if .Map }}
	for idx, _ := range v.{{ .Map }} {
		if _ = v.{{ .Map }}[idx].Accept(d); d.stop {
			return nil
		}
	}
	{{- end -}}
	{{- end }}

	return nil
}
{{ end }}
func Reduce{{ $name }}DepthFirst[A any](r {{ $name }}Reducer[A], v {{ $name }}, init A) A {
	reducer := &{{ $name }}DepthFirstVisitor[A]{
		result: init,
		reduce: r,
	}

	_ = v.Accept(reducer)

	return reducer.result
}

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
	renderTraverse = template.Must(template.New("main").Parse(traverseTmpl))
)

func (t *ReducerGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := renderTraverse.ExecuteTemplate(result, "main", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
