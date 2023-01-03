package mkunion

import (
	"github.com/fatih/structtag"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

func InferFromFile(filename string) (*InferredInfo, error) {
	result := &InferredInfo{
		Types:                map[string]map[string][]Branching{},
		possibleVariantTypes: map[string][]string{},
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	ast.Walk(result, f)
	return result, nil
}

var (
	matchGoGenerateExtractUnionName = regexp.MustCompile(`go:generate .* -{1,2}name=(\w+)`)
)

type InferredInfo struct {
	PackageName          string
	Types                map[string]map[string][]Branching
	currentType          string
	possibleVariantTypes map[string][]string
}

func (f *InferredInfo) ForVariantType(name string, types []string) map[string][]Branching {
	result := make(map[string][]Branching)
	for _, t := range types {
		result[t] = nil
		for vn, fields := range f.Types[t] {
			if vn == name {
				result[t] = fields
			}
		}
	}

	return result
}

func (f *InferredInfo) PossibleVariantsTypes(unionName string) []string {
	return f.possibleVariantTypes[unionName]
}

// comment implementation was copied from func (g *CommentGroup) Text() string
// and reduced return all comments lines. In contrast to original implementation
// that skips declaration comments like "//go:generate" which is important
// for inferring union types.
func (f *InferredInfo) comment(g *ast.CommentGroup) string {
	if g == nil {
		return ""
	}
	comments := make([]string, len(g.List))
	for i, c := range g.List {
		comments[i] = c.Text
	}

	lines := make([]string, 0, 10) // most comments are less than 10 lines
	for _, c := range comments {
		// Remove comment markers.
		// The parser has given us exactly the comment text.
		switch c[1] {
		case '/':
			//-style comment (no newline at the end)
			c = c[2:]
			if len(c) == 0 {
				// empty line
				break
			}
			if c[0] == ' ' {
				// strip first space - required for Example tests
				c = c[1:]
				break
			}
		case '*':
			/*-style comment */
			c = c[2 : len(c)-2]
		}

		lines = append(lines, c)
	}

	return strings.Join(lines, "\n")
}

func (f *InferredInfo) Visit(n ast.Node) ast.Visitor {
	switch t := n.(type) {
	case *ast.GenDecl:
		comment := f.comment(t.Doc)
		if !strings.Contains(comment, "mkunion") {
			return f
		}
		if t.Tok != token.TYPE {
			return f
		}

		names := matchGoGenerateExtractUnionName.FindStringSubmatch(comment)
		if len(names) < 2 {
			return f
		}

		unionName := names[1]
		if _, ok := f.possibleVariantTypes[unionName]; !ok {
			f.possibleVariantTypes[unionName] = make([]string, 0)
		}

		for _, spec := range t.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				f.possibleVariantTypes[unionName] = append(f.possibleVariantTypes[unionName], s.Name.Name)
			}
		}
		return f

	case *ast.FuncDecl:
		return nil

	case *ast.File:
		if t.Name != nil {
			f.PackageName = t.Name.String()
		}

	case *ast.TypeSpec:
		f.Types[t.Name.Name] = make(map[string][]Branching)
		f.currentType = t.Name.Name

	case *ast.StructType:
		for _, field := range t.Fields.List {
			traverseOption := true
			if field.Tag != nil {
				traverseOption = f.traverseOption(field.Tag.Value)
			}

			isList := false
			isMap := false

			var id *ast.Ident
			var ok bool
			if arr, okArr := field.Type.(*ast.ArrayType); okArr {
				id, ok = arr.Elt.(*ast.Ident)
				isList = true
			} else if m, okMap := field.Type.(*ast.MapType); okMap {
				id, ok = m.Value.(*ast.Ident)
				isMap = true
			} else {
				id, ok = field.Type.(*ast.Ident)
			}

			if !ok {
				continue
			}
			fieldName := id.Name

			if !traverseOption {
				continue
			}

			for _, ff := range field.Names {
				fieldTypeName := fieldName
				branch := Branching{Lit: PtrStr(ff.Name)}
				if isList {
					branch = Branching{List: PtrStr(ff.Name)}
				} else if isMap {
					branch = Branching{Map: PtrStr(ff.Name)}
				}
				f.Types[f.currentType][fieldTypeName] = append(f.Types[f.currentType][fieldTypeName], branch)
			}
		}
	}

	return f
}

func (f *InferredInfo) traverseOption(tag string) bool {
	st, err := structtag.Parse(strings.Trim(tag, "`"))
	if err != nil {
		return true
	}

	opts, err := st.Get("mkunion")
	if err != nil {
		return true
	}

	return !opts.HasOption("notraverse")
}
