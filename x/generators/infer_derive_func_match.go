package generators

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shared"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"strings"
)

func InferDeriveFuncMatchFromFile(filename string) (*InferredDeriveFuncMatchInfo, error) {
	result := &InferredDeriveFuncMatchInfo{
		matchBuilder: make(map[string]*MatchBuilder),
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
	matchGoGenerateExtractFuncName = regexp.MustCompile(`go:generate .* -{1,2}name=(\w+)`)
)

type InferredDeriveFuncMatchInfo struct {
	PackageName  string
	matchBuilder map[string]*MatchBuilder
}

func (f *InferredDeriveFuncMatchInfo) MatchSpec(name string) (*MatchSpec, error) {
	if _, ok := f.matchBuilder[name]; !ok {
		return nil, fmt.Errorf("no match builder with name: %v", name)
	}
	return f.matchBuilder[name].Build()
}

func (f *InferredDeriveFuncMatchInfo) Visit(n ast.Node) ast.Visitor {
	switch t := n.(type) {
	case *ast.GenDecl:
		comment := shared.Comment(t.Doc)
		if !strings.Contains(comment, "match") {
			return f
		}

		if t.Tok != token.TYPE {
			return f
		}

		names := matchGoGenerateExtractFuncName.FindStringSubmatch(comment)
		if len(names) < 2 {
			return f
		}

		builderName := names[1]
		if _, ok := f.matchBuilder[builderName]; ok {
			panic(fmt.Sprintf("duplicated match builder name: %v", builderName))
		}

		f.matchBuilder[builderName] = NewMatchBuilder()

		for _, spec := range t.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				err := f.matchBuilder[builderName].SetName(s.Name.Name)
				if err != nil {
					panic(err)
				}

				// for each type param register as input in builder
				var inputs []string
				for _, param := range s.TypeParams.List {
					if len(param.Names) > 0 {
						// types can have params like [a, b int]
						// where there is same type for multiple params
						// here we need to add multiple attribs
						for range param.Names {
							inputs = append(inputs, typeToString(param.Type))
						}
					} else {
						inputs = append(inputs, typeToString(param.Type))
					}
				}
				err = f.matchBuilder[builderName].SetInputs(inputs...)
				if err != nil {
					panic(err)
				}

				switch t := s.Type.(type) {
				case *ast.InterfaceType:
					for _, method := range t.Methods.List {
						if fn, ok := method.Type.(*ast.FuncType); ok {
							var attribs []string
							for _, param := range fn.Params.List {
								switch p := param.Type.(type) {
								case *ast.Ident, *ast.SelectorExpr, *ast.StarExpr, *ast.ArrayType, *ast.MapType:
									if len(param.Names) > 0 {
										// functions can have params like (a, b int)
										// where there is same type for multiple params
										// here we need to add multiple attribs
										for range param.Names {
											attribs = append(attribs, typeToString(p))
										}
									} else {
										attribs = append(attribs, typeToString(p))
									}
								default:
									panic(fmt.Sprintf("type in matchign function is not supported %T", p))
								}
							}
							err := f.matchBuilder[builderName].AddCase(method.Names[0].Name, attribs...)
							if err != nil {
								panic(err)
							}
						}
					}
				}
			}
		}
		return f

	case *ast.File:
		if t.Name != nil {
			f.PackageName = t.Name.String()
		}
	}

	return f
}

func typeToString(t ast.Expr) string {
	switch t := t.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len != nil {
			return fmt.Sprintf("[%s]%s", t.Len, typeToString(t.Elt))
		}
		return "[]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	default:
		panic(fmt.Sprintf("type %T is not supported", t))
	}
}
