package mkunion

import (
	"bytes"
	"text/template"
)

type ReducerBreatheFirstGenerator struct {
	Name        variantName
	Types       []typeName
	PackageName string
	Branches    map[typeName][]Branching
}

var (
	breatheFirstTmpl = Header + `
package {{ .PackageName }}
{{ $root := . }}
{{- $name := .Name }}
var _ {{ $name }}Visitor = (*{{ $name }}BreatheFirstVisitor[any])(nil)

type {{ $name }}BreatheFirstVisitor[A any] struct {
	stop   bool
	result A
	reduce {{ $name }}Reducer[A]

	queue         []{{ $name }}
	visited       map[{{ $name }}]bool
	shouldExecute map[{{ $name }}]bool
}
{{ range $i, $type := .Types }}
func (d *{{ $name }}BreatheFirstVisitor[A]) Visit{{ . }}(v *{{ . }}) any {
	d.queue = append(d.queue, v)
{{- range (index $root.Branches $type) -}}
	{{- if .Lit}}
	d.queue = append(d.queue, v.{{ .Lit }})
	{{- else if .List }}
	for idx := range v.{{ .List }} {
		d.queue = append(d.queue, v.{{ .List }}[idx])
	}
	{{- else if .Map }}
	for idx, _ := range v.{{ .Map }} {
		d.queue = append(d.queue, v.{{ .Map }}[idx])
	}
	{{- end -}}
{{- end }}
	
	if d.shouldExecute[v] {
		d.shouldExecute[v] = false
		d.result, d.stop = d.reduce.Reduce{{ . }}(v, d.result)
	} else {
		d.execute()
	}
	return nil
}
{{ end }}
func (d *{{ $name }}BreatheFirstVisitor[A]) execute() {
	for len(d.queue) > 0 {
		if d.stop {
			return
		}

		i := d.pop()
		if d.visited[i] {
			continue
		}
		d.visited[i] = true
		d.shouldExecute[i] = true
		i.Accept(d)
	}

	return
}

func (d *{{ $name }}BreatheFirstVisitor[A]) pop() {{ $name }} {
	i := d.queue[0]
	d.queue = d.queue[1:]
	return i
}

func Reduce{{ $name }}BreatheFirst[A any](r {{ $name }}Reducer[A], v {{ $name }}, init A) A {
	reducer := &{{ $name }}BreatheFirstVisitor[A]{
		result:        init,
		reduce:        r,
		queue:         []{{ $name }}{v},
		visited:       make(map[{{ $name }}]bool),
		shouldExecute: make(map[{{ $name }}]bool),
	}

	_ = v.Accept(reducer)

	return reducer.result
}
`
)

var (
	renderBreatheFirst = template.Must(template.New("main").Parse(breatheFirstTmpl))
)

func (t *ReducerBreatheFirstGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := renderBreatheFirst.ExecuteTemplate(result, "main", t)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
