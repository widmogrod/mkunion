package mkunion

import (
	"bytes"
	"text/template"
)

type (
	VisitorDefaultGenerator struct {
		Name        string
		Types       []string
		PackageName string
	}
)

var (
	optionalVisitorTmpl = Header + `
package {{ .PackageName }}
{{ $name := .Name }}
type {{ $name }}DefaultVisitor[A any] struct {
	Default A
	{{- range .Types }}
	On{{ . }} func(x *{{ . }}) A
	{{- end }}
}

{{- range .Types }}
func (t *{{ $name }}DefaultVisitor[A]) Visit{{ . }}(v *{{ . }}) any {
	if t.On{{ . }} != nil {
		return t.On{{ . }}(v)
	}
	return t.Default
}
{{- end }}
`
)

var (
	optionalVisitorRender = template.Must(template.New("main").Parse(optionalVisitorTmpl))
)

func (g *VisitorDefaultGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := optionalVisitorRender.ExecuteTemplate(result, "main", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
