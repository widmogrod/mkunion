package mkunion

import (
	"github.com/fatih/structtag"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/shape"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/mod/modfile"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
)

func InferFromFile(filename string) (*InferredInfo, error) {
	if !path.IsAbs(filename) {
		cwd, _ := os.Getwd()
		filename = path.Join(cwd, filename)
	}

	result := &InferredInfo{
		Types:                map[string]map[string][]Branching{},
		PkgImportName:        tryToPkgImportName(filename),
		possibleVariantTypes: map[string][]string{},
		shapes:               make(map[string]*shape.StructLike),
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	ast.Walk(result, f)
	return result, nil
}

// tryToPkgImportName contains import name of the package
func tryToPkgImportName(filename string) string {
	var toadd []string
	for {
		filename = path.Dir(filename)
		if filename == "." || filename == "/" {
			log.Infof("infer_defaults: could not find go.mod file in %s, returning empty pkg name", filename)
			return ""
		}

		modpath := path.Join(filename, "go.mod")
		if _, err := os.Stat(modpath); err == nil {
			f, err := os.Open(modpath)
			defer f.Close()

			data, err := io.ReadAll(f)
			if err != nil {
				log.Infof("infer_defaults: could not read go.mod file in %s, returning empty pkg name. %s", filename, err.Error())
				return ""
			}

			parsed, err := modfile.Parse(modpath, data, nil)
			if err != nil {
				log.Infof("infer_defaults: could not parse go.mod file in %s, returning empty pkg name. %s", filename, err.Error())
				return ""
			}

			if parsed.Module == nil {
				log.Infof("infer_defaults: could not find module name in go.mod file in %s, returning empty pkg name", filename)
				return ""
			}

			return path.Join(append([]string{parsed.Module.Mod.Path}, toadd...)...)
		}

		toadd = append([]string{path.Base(filename)}, toadd...)
	}
}

var (
	matchGoGenerateExtractUnionName = regexp.MustCompile(`go:generate .* -{1,2}name=(\w+)`)
)

type InferredInfo struct {
	PackageName                string
	PkgImportName              string
	Types                      map[string]map[string][]Branching
	currentType                string
	possibleVariantTypes       map[string][]string
	shapes                     map[string]*shape.StructLike
	packageNameToPackageImport map[string]string
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

func (f *InferredInfo) StructShapeWith(name string) *shape.StructLike {
	return f.shapes[name]
}

func (f *InferredInfo) RetrieveUnions() []*shape.UnionLike {
	result := make([]*shape.UnionLike, 0)
	for unionName, possibleVariants := range f.possibleVariantTypes {
		var variants []*shape.StructLike
		for _, variant := range possibleVariants {
			variants = append(variants, f.shapes[variant])
		}
		result = append(result, &shape.UnionLike{
			Name:          unionName,
			PkgName:       f.PackageName,
			PkgImportName: f.PkgImportName,
			Variant:       variants,
		})
	}

	return result
}

func (f *InferredInfo) RetrieveStruct() []*shape.StructLike {
	structs := make(map[string]*shape.StructLike)
	for _, structShape := range f.shapes {
		structs[structShape.Name] = structShape
	}

	for union, variants := range f.possibleVariantTypes {
		delete(structs, union)
		for _, variant := range variants {
			delete(structs, variant)
		}
	}

	result := make([]*shape.StructLike, 0)
	for _, x := range structs {
		result = append(result, x)
	}

	return result
}

func (f *InferredInfo) Visit(n ast.Node) ast.Visitor {
	switch t := n.(type) {
	case *ast.GenDecl:
		comment := comment(t.Doc)
		if !strings.Contains(comment, Program) {
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

		f.packageNameToPackageImport = make(map[string]string)
		for _, imp := range t.Imports {
			if imp.Name != nil {
				f.packageNameToPackageImport[imp.Name.String()] = strings.Trim(imp.Path.Value, "\"")
			} else {
				f.packageNameToPackageImport[path.Base(strings.Trim(imp.Path.Value, "\""))] = strings.Trim(imp.Path.Value, "\"")
			}
		}

	case *ast.TypeSpec:
		f.Types[t.Name.Name] = make(map[string][]Branching)
		f.currentType = t.Name.Name

	case *ast.StructType:
		if t.Struct.IsValid() {
			if _, ok := f.shapes[f.currentType]; !ok {
				f.shapes[f.currentType] = &shape.StructLike{
					Name:          f.currentType,
					PkgName:       f.PackageName,
					PkgImportName: f.PkgImportName,
				}
			}

			structShape := f.shapes[f.currentType]
			for _, field := range t.Fields.List {
				// this happens when field is embedded in struct
				// something like `type A struct { B }`
				if len(field.Names) == 0 {
					switch typ := field.Type.(type) {
					case *ast.Ident:
						structShape.Fields = append(structShape.Fields, &shape.FieldLike{
							Name: typ.Name,
							Type: shape.FromAst(typ, shape.InjectPkgName(f.PkgImportName, f.PackageName)),
						})
						break
					default:
						log.Warnf("infer_defaults: unknown ast type embedded in struct: %T\n", typ)
						continue
					}
				}

				for _, fieldName := range field.Names {
					if !fieldName.IsExported() {
						continue
					}

					var typ shape.Shape
					switch ttt := field.Type.(type) {
					// selectors in struct, means that we are using type from other package
					case *ast.SelectorExpr:
						if ident, ok := ttt.X.(*ast.Ident); ok {
							pkgName := ident.Name
							pkgImportName := f.packageNameToPackageImport[pkgName]
							typ = shape.FromAst(ttt.Sel, shape.InjectPkgName(pkgImportName, pkgName))
						} else {
							log.Infof("infer_defaults: unknown selector X in  %s.%s: %T\n", f.currentType, fieldName.Name, ttt)
							typ = shape.FromAst(ttt, shape.InjectPkgName(f.PkgImportName, f.PackageName))
						}
					// this is reference to other struct in the same package or other package
					case *ast.StarExpr:
						if selector, ok := ttt.X.(*ast.SelectorExpr); ok {
							// other package
							if ident, ok := selector.X.(*ast.Ident); ok {
								pkgName := ident.Name
								pkgImportName := f.packageNameToPackageImport[pkgName]
								typ = shape.FromAst(selector.Sel, shape.InjectPkgName(pkgImportName, pkgName))
							} else {
								// same package
								log.Infof("infer_defaults: unknown star X in  %s.%s: %T\n", f.currentType, fieldName.Name, ttt)
								typ = shape.FromAst(ttt, shape.InjectPkgName(f.PkgImportName, f.PackageName))
							}
						} else {
							// same package
							log.Infof("infer_defaults: unknown star-else in  %s.%s: %T\n", f.currentType, fieldName.Name, ttt)
							typ = shape.FromAst(ttt, shape.InjectPkgName(f.PkgImportName, f.PackageName))
						}

					case *ast.Ident, *ast.ArrayType, *ast.MapType, *ast.StructType:
						typ = shape.FromAst(ttt, shape.InjectPkgName(f.PkgImportName, f.PackageName))

					default:
						log.Warnf("infer_defaults: unknown ast type in  %s.%s: %T\n", f.currentType, fieldName.Name, ttt)
						typ = &shape.Any{}
					}

					structShape.Fields = append(structShape.Fields, &shape.FieldLike{
						Name: fieldName.Name,
						Type: typ,
					})
				}
			}

			f.shapes[f.currentType] = structShape

			log.Infof("infer_defaults: struct %s: %s\n", f.currentType, shape.ToStr(structShape))
		}

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

// comment implementation was copied from func (g *CommentGroup) Text() string
// and reduced return all comments lines. In contrast to original implementation
// that skips declaration comments like "//go:generate" which is important
// for inferring union types.
func comment(g *ast.CommentGroup) string {
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
