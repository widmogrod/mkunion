package mkunion

import (
	"bytes"
	"text/template"
)

const (
	Program = "mkunion"
	Header  = `// Code generated by ` + Program + `. DO NOT EDIT.`
)

type Generator interface {
	Generate() ([]byte, error)
}

type (
	VisitorGenerator struct {
		Types       []string
		Name        string
		PackageName string
	}
)

var (
	tmpl = Header + `
package {{ .PackageName }}
{{ $name := .Name }}
type {{ $name }}Visitor interface {
	{{- range .Types }}
	Visit{{ . }}(v *{{ . }}) any
	{{- end }}
}

type {{ $name }} interface {
	Accept(g {{ $name }}Visitor) any
}
{{ range .Types }}
func (r *{{ . }}) Accept(v {{ $name }}Visitor) any { return v.Visit{{ . }}(r) }
{{- end }}

var (
	{{- range .Types }}
	_ {{ $name }} = (*{{ . }})(nil)
	{{- end }}
)

type {{ $name }}OneOf struct {
{{- range .Types }}
	{{ . }} *{{ . }} ` + "`json:\",omitempty\"`" + `
{{- end }}
}

func (r *{{ $name }}OneOf) Accept(v {{ $name }}Visitor) any {
	switch {
{{- range .Types }}
	case r.{{ . }} != nil:
		return v.Visit{{ . }}(r.{{ . }})
{{- end }}
	default:
		panic("unexpected")
	}
}

var _ {{ $name }} = (*{{ $name }}OneOf)(nil)

type map{{ $name }}ToOneOf struct{}
{{ range .Types }}
func (t *map{{ $name }}ToOneOf) Visit{{ . }}(v *{{ . }}) any { return &{{ $name }}OneOf{ {{- . }}: v} }
{{- end }}

var defaultMap{{ $name }}ToOneOf {{ $name }}Visitor = &map{{ $name }}ToOneOf{}

func Map{{ $name }}ToOneOf(v {{ $name }}) *{{ $name }}OneOf {
	return v.Accept(defaultMap{{ $name }}ToOneOf).(*{{ $name }}OneOf)
}
`
)

var (
	render = template.Must(template.New("main").Parse(tmpl))
)

func (g *VisitorGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := render.ExecuteTemplate(result, "main", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}