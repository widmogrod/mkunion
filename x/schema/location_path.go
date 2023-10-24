package schema

import (
	"github.com/alecthomas/participle/v2"
	"strings"
)

var (
	pathParser = participle.MustBuild[PathAst](
		participle.Unquote("String", "Char", "RawString"),
	)
)

func ParseLocation(input string) ([]Location, error) {
	// Parse the input and build a Predicate value
	ast, err := pathParser.ParseString("", strings.TrimSpace(input))
	if err != nil {
		return nil, err
	}

	// Convert the AST to a Predicate value
	return ast.ToLocation()
}

type PathAst struct {
	Parts []Part `@@ ( "." @@ )*`
}

func (ast PathAst) ToLocation() ([]Location, error) {
	var parts []Location
	for _, part := range ast.Parts {
		parts = append(parts, part.ToLocation()...)
	}
	return parts, nil
}

type Part struct {
	Location string `@Ident`
	Acc      []Acc  `("[" @@ "]")*`
}

type Acc struct {
	Name  *string `@(String|Char|RawString)`
	Index *int    `| @Int`
	Any   bool    `| @("*") `
}

func (p Part) ToLocation() []Location {
	var result []Location
	result = append(result, &LocationField{Name: p.Location})
	for _, a := range p.Acc {
		result = append(result, a.ToAccessor())
	}
	return result
}

func (a Acc) ToAccessor() Location {
	if a.Name != nil {
		return &LocationField{Name: *a.Name}
	}

	if a.Index != nil {
		return &LocationIndex{Index: *a.Index}
	}

	return &LocationAnything{}
}
