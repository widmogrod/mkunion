package breaking

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
)

// Locator finds source positions for the findings. Shapes carry no
// positions, so the current files are parsed once more and types, union
// tags and fields are indexed by name.
type Locator struct {
	types  map[string]position // "Search", and union names from //go:tag mkunion:"Request"
	fields map[string]position // "Search.Query"; embedded fields by their type name, "GetInfo.Returns"
	first  string              // the first file, for findings with nothing left to point at
}

type position struct {
	file string
	line int
}

var unionTagRe = regexp.MustCompile(`go:tag\s+mkunion:"([A-Za-z_][A-Za-z0-9_]*)`)

// Locate parses the files; paths are kept as given so findings print them
// the way the user wrote them.
func Locate(files []string) (*Locator, error) {
	l := &Locator{types: map[string]position{}, fields: map[string]position{}}
	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		if l.first == "" {
			l.first = file
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			l.indexDecl(fset, gen)
		}
	}
	return l, nil
}

func (l *Locator) indexDecl(fset *token.FileSet, gen *ast.GenDecl) {
	l.indexUnionTag(fset, gen.Doc, gen.Pos())
	for _, spec := range gen.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}
		l.indexUnionTag(fset, ts.Doc, ts.Pos())
		l.types[ts.Name.Name] = at(fset, ts.Pos())
		st, ok := ts.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			continue
		}
		for _, field := range st.Fields.List {
			for _, name := range fieldNames(field) {
				l.fields[ts.Name.Name+"."+name] = at(fset, field.Pos())
			}
		}
	}
}

func (l *Locator) indexUnionTag(fset *token.FileSet, doc *ast.CommentGroup, pos token.Pos) {
	if doc == nil {
		return
	}
	for _, c := range doc.List {
		if m := unionTagRe.FindStringSubmatch(c.Text); m != nil {
			l.types[m[1]] = at(fset, pos)
		}
	}
}

func fieldNames(field *ast.Field) []string {
	if len(field.Names) > 0 {
		var names []string
		for _, n := range field.Names {
			names = append(names, n.Name)
		}
		return names
	}
	if name, ok := embeddedName(field.Type); ok {
		return []string{name}
	}
	return nil
}

func embeddedName(expr ast.Expr) (string, bool) {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name, true
	case *ast.SelectorExpr:
		return x.Sel.Name, true
	case *ast.StarExpr:
		return embeddedName(x.X)
	case *ast.IndexExpr:
		return embeddedName(x.X)
	case *ast.IndexListExpr:
		return embeddedName(x.X)
	}
	return "", false
}

func at(fset *token.FileSet, pos token.Pos) position {
	p := fset.Position(pos)
	return position{file: p.Filename, line: p.Line}
}

// Type is where a type or union is declared; the first file with no line
// when it is gone. A nil Locator knows nothing.
func (l *Locator) Type(name string) (string, int) {
	if l == nil {
		return "", 0
	}
	if p, ok := l.types[name]; ok {
		return p.file, p.line
	}
	return l.first, 0
}

// Field is where a field is declared, else where its type is.
func (l *Locator) Field(typ, field string) (string, int) {
	if l == nil {
		return "", 0
	}
	if p, ok := l.fields[typ+"."+field]; ok {
		return p.file, p.line
	}
	return l.Type(typ)
}
