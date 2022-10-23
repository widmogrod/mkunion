package mkunion

import (
	"github.com/fatih/structtag"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type InferredInfo struct {
	PackageName string
	Types       map[string]map[string][]Branching
	currentType string
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

func (f *InferredInfo) Visit(n ast.Node) ast.Visitor {
	switch t := n.(type) {
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

func InferFromFile(filename string) (*InferredInfo, error) {
	result := &InferredInfo{
		Types: map[string]map[string][]Branching{},
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		return nil, err
	}

	ast.Walk(result, f)
	return result, nil
}
