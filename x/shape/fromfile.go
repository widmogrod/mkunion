package shape

import (
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/shared"
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
		PkgImportName:        tryToPkgImportName(filename),
		possibleVariantTypes: map[string][]string{},
		shapes:               make(map[string]*StructLike),
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
	possibleVariantTypes       map[string][]string
	shapes                     map[string]*StructLike
	packageNameToPackageImport map[string]string
	currentType                string
}

func (f *InferredInfo) PossibleUnionTypes() []string {
	result := make([]string, 0)
	for unionName := range f.possibleVariantTypes {
		result = append(result, unionName)
	}
	return result
}

func (f *InferredInfo) PossibleVariantsTypes(unionName string) []string {
	return f.possibleVariantTypes[unionName]
}

func (f *InferredInfo) StructShapeWith(name string) *StructLike {
	return f.shapes[name]
}

func (f *InferredInfo) RetrieveUnions() []*UnionLike {
	result := make([]*UnionLike, 0)
	for unionName := range f.possibleVariantTypes {
		union := f.RetrieveUnion(unionName)
		result = append(result, &union)
	}

	return result
}

func (f *InferredInfo) RetrieveUnion(name string) UnionLike {
	var variants []Shape
	for _, variant := range f.possibleVariantTypes[name] {
		variants = append(variants, f.shapes[variant])
	}

	return UnionLike{
		Name:          name,
		PkgName:       f.PackageName,
		PkgImportName: f.PkgImportName,
		Variant:       variants,
	}
}

func (f *InferredInfo) RetrieveStruct() []*StructLike {
	structs := make(map[string]*StructLike)
	for _, structShape := range f.shapes {
		structs[structShape.Name] = structShape
	}

	for union, variants := range f.possibleVariantTypes {
		delete(structs, union)
		for _, variant := range variants {
			delete(structs, variant)
		}
	}

	result := make([]*StructLike, 0)
	for _, x := range structs {
		result = append(result, x)
	}

	return result
}

func (f *InferredInfo) Visit(n ast.Node) ast.Visitor {
	switch t := n.(type) {
	case *ast.GenDecl:
		comment := shared.Comment(t.Doc)
		if !strings.Contains(comment, shared.Program) {
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
		f.currentType = t.Name.Name

	case *ast.StructType:
		if t.Struct.IsValid() {
			if _, ok := f.shapes[f.currentType]; !ok {
				f.shapes[f.currentType] = &StructLike{
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
						structShape.Fields = append(structShape.Fields, &FieldLike{
							Name: typ.Name,
							Type: FromAst(typ, InjectPkgName(f.PkgImportName, f.PackageName)),
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

					var typ Shape
					switch ttt := field.Type.(type) {
					// selectors in struct, means that we are using type from other package
					case *ast.SelectorExpr:
						if ident, ok := ttt.X.(*ast.Ident); ok {
							pkgName := ident.Name
							pkgImportName := f.packageNameToPackageImport[pkgName]
							typ = FromAst(ttt.Sel, InjectPkgName(pkgImportName, pkgName))
						} else {
							log.Infof("infer_defaults: unknown selector X in  %s.%s: %T\n", f.currentType, fieldName.Name, ttt)
							typ = FromAst(ttt, InjectPkgName(f.PkgImportName, f.PackageName))
						}
					// this is reference to other struct in the same package or other package
					case *ast.StarExpr:
						if selector, ok := ttt.X.(*ast.SelectorExpr); ok {
							// other package
							if ident, ok := selector.X.(*ast.Ident); ok {
								pkgName := ident.Name
								pkgImportName := f.packageNameToPackageImport[pkgName]
								typ = FromAst(selector.Sel, InjectPkgName(pkgImportName, pkgName))
							} else {
								// same package
								log.Infof("infer_defaults: unknown star X in  %s.%s: %T\n", f.currentType, fieldName.Name, ttt)
								typ = FromAst(ttt, InjectPkgName(f.PkgImportName, f.PackageName))
							}
						} else {
							// same package
							log.Infof("infer_defaults: unknown star-else in  %s.%s: %T\n", f.currentType, fieldName.Name, ttt)
							typ = FromAst(ttt, InjectPkgName(f.PkgImportName, f.PackageName))
						}

					case *ast.Ident, *ast.ArrayType, *ast.MapType, *ast.StructType:
						typ = FromAst(ttt, InjectPkgName(f.PkgImportName, f.PackageName))

					default:
						log.Warnf("infer_defaults: unknown ast type in  %s.%s: %T\n", f.currentType, fieldName.Name, ttt)
						typ = &Any{}
					}

					tag := ""
					if field.Tag != nil {
						tag = field.Tag.Value
					}

					tags := ExtractTags(tag)
					desc := TagsToDesc(tags)
					guard := TagsToGuard(tags)

					structShape.Fields = append(structShape.Fields, &FieldLike{
						Name:      fieldName.Name,
						Type:      typ,
						Desc:      desc,
						Guard:     guard,
						Tags:      tags,
						IsPointer: IsStarExpr(field.Type),
					})
				}
			}

			f.shapes[f.currentType] = structShape

			log.Infof("infer_defaults: struct %s: %s\n", f.currentType, ToStr(structShape))
		}
	}

	return f
}
