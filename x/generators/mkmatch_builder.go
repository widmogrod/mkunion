package generators

import (
	"fmt"
	"go/ast"
	"go/token"
)

func NewMkMatchBuilder() *MkMatchBuilder {
	return &MkMatchBuilder{
		usePackageNames: make(map[string]bool),
	}
}

type MkMatchBuilder struct {
	name       string
	inputTypes []string
	cases      [][]string
	names      []string

	usePackageNames map[string]bool
	knownPkgMap     PkgMap
}

func (b *MkMatchBuilder) InitPkgMap(pkgMap PkgMap) {
	b.knownPkgMap = pkgMap
}

func (b *MkMatchBuilder) Visit(node ast.Node) ast.Visitor {
	switch t := node.(type) {
	case *ast.GenDecl:
		if t.Tok != token.TYPE {
			return nil
		}

		return b

	case *ast.TypeSpec:
		s := t
		err := b.SetName(s.Name.Name)
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
		err = b.SetInputs(inputs...)
		if err != nil {
			panic(err)
		}

		return b

	case *ast.InterfaceType:
		for _, method := range t.Methods.List {
			if fn, ok := method.Type.(*ast.FuncType); ok {
				var attribs []string
				for _, param := range fn.Params.List {
					switch p := param.Type.(type) {
					case *ast.Ident, *ast.SelectorExpr, *ast.StarExpr, *ast.ArrayType, *ast.MapType:
						b.extractPackageName(p)

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
				err := b.AddCase(method.Names[0].Name, attribs...)
				if err != nil {
					panic(err)
				}
			}
		}

	}

	return b
}

func (b *MkMatchBuilder) extractPackageName(node ast.Node) {
	switch t := node.(type) {
	case *ast.StarExpr:
		b.extractPackageName(t.X)

	case *ast.ArrayType:
		b.extractPackageName(t.Elt)

	case *ast.MapType:
		b.extractPackageName(t.Key)
		b.extractPackageName(t.Value)

	case *ast.TypeSpec:
		// indexed type like type None[A any] struct{}
		if t.TypeParams == nil {
			return
		}

		for _, param := range t.TypeParams.List {
			b.extractPackageName(param.Type)
		}

	case *ast.IndexExpr:
		b.extractPackageName(t.X)
		b.extractPackageName(t.Index)

	case *ast.IndexListExpr:
		b.extractPackageName(t.X)
		for _, index := range t.Indices {
			b.extractPackageName(index)
		}

	case *ast.SelectorExpr:
		// example: time.Duration
		if ident, ok := t.X.(*ast.Ident); ok {
			pkgName := ident.Name
			b.usePackageNames[pkgName] = true
		}
	}
}

func (b *MkMatchBuilder) SetName(name string) error {
	// If no name is set yet (empty string from tag), use the interface name
	// If name is "-", still allow override (this is the special skip case)
	// If name is already set from the tag, keep it unless it's "-"
	if b.name == "" || b.name == "-" {
		b.name = name
	}

	return nil
}

func (b *MkMatchBuilder) SetInputs(types ...string) error {
	if len(types) == 0 {
		return fmt.Errorf("mkmatch: list of type parameters is required")
	}

	if b.inputTypes == nil {
		b.inputTypes = types
	} else {
		return fmt.Errorf("mkmatch: cannot declare type parameters more than once")
	}

	return nil
}

func (b *MkMatchBuilder) AddCase(name string, inputs ...string) error {
	if len(inputs) == 0 {
		return fmt.Errorf("mkmatch: matching case %s must have at least %d arguments", name, len(b.inputTypes))
	}

	if len(inputs) != len(b.inputTypes) {
		return fmt.Errorf("mkmatch: matching case %s must have same number of function arguments as number of type params", name)
	}

	// check if there are no duplicates in other cases
	for cid, caseInputs := range b.cases {
		same := len(caseInputs)
		for i, input := range caseInputs {
			if input == inputs[i] {
				same--
			}
		}
		if same == 0 {
			return fmt.Errorf("mkmatch: matching case %s cannot have duplicate argument names", b.names[cid])
		}
	}
	b.cases = append(b.cases, inputs)

	// check if there are no duplicates in names
	for _, caseName := range b.names {
		if caseName == name {
			return fmt.Errorf("mkmatch: cannot have duplicate; case name: %s", caseName)
		}
	}
	b.names = append(b.names, name)

	return nil
}

type MatchSpec struct {
	Name        string
	Names       []string
	Inputs      []string
	Cases       [][]string
	UsedPackMap PkgMap
}

func (b *MkMatchBuilder) Build() (*MatchSpec, error) {
	if b.name == "" {
		return nil, fmt.Errorf("mkmatch: type match must have name")
	}

	if len(b.cases) == 0 {
		return nil, fmt.Errorf("mkmatch: type match must have at least one case")
	}

	pkgMap := make(PkgMap)

	for pkgName := range b.usePackageNames {
		if pkg, ok := b.knownPkgMap[pkgName]; ok {
			pkgMap[pkgName] = pkg
		} else {
			return nil, fmt.Errorf("mkmatch: cannot find package import path for name %s", pkgName)
		}
	}

	return &MatchSpec{
		Name:        b.name,
		Names:       b.names,
		Inputs:      b.inputTypes,
		Cases:       b.cases,
		UsedPackMap: pkgMap,
	}, nil
}
