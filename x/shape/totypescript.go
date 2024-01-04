package shape

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"strings"
)

type (
	TypeScriptOptions struct {
		currentPkgName       string
		currentPkgImportName packageImportName
		imports              map[packageName]packageImportName
	}

	packageImportName = string
	packageName       = string
	shapeName         = string
)

func (o *TypeScriptOptions) IsCurrentPkgName(pkgName string) bool {
	if pkgName == "" {
		return true
	}

	return o.currentPkgName == pkgName
}

func (o *TypeScriptOptions) NeedsToImportPkgName(pkg packageName, imp packageImportName) {
	if o.imports == nil {
		o.imports = make(map[packageName]packageImportName)
	}

	o.imports[pkg] = imp
}

func ToTypeScriptOptimisation(x Shape) Shape {
	return MatchShapeR1(
		x,
		func(x *Any) Shape {
			return x
		},
		func(x *RefName) Shape {
			for i, name := range x.Indexed {
				x.Indexed[i] = ToTypeScriptOptimisation(name)
			}
			return x
		},
		func(x *AliasLike) Shape {
			x.Type = ToTypeScriptOptimisation(x.Type)
			return x
		},
		func(x *PrimitiveLike) Shape {
			return x
		},
		func(x *ListLike) Shape {
			// do forward lookup and detect if we can optimise and convert to string
			switch y := x.Element.(type) {
			case *PrimitiveLike:
				switch z := y.Kind.(type) {
				case *NumberLike:
					switch z.Kind.(type) {
					// byte is uint8
					// rune is int32
					case *UInt8, *Int32:
						return &PrimitiveLike{
							Kind: &StringLike{},
						}
					}
				}
			}

			x.Element = ToTypeScriptOptimisation(x.Element)
			return x
		},
		func(x *MapLike) Shape {
			x.Val = ToTypeScriptOptimisation(x.Val)
			x.Key = ToTypeScriptOptimisation(x.Key)
			return x
		},
		func(x *StructLike) Shape {
			for _, field := range x.Fields {
				field.Type = ToTypeScriptOptimisation(field.Type)
			}
			for _, param := range x.TypeParams {
				param.Type = ToTypeScriptOptimisation(param.Type)
			}
			return x
		},
		func(x *UnionLike) Shape {
			for _, variant := range x.Variant {
				variant = ToTypeScriptOptimisation(variant)
			}
			return x
		},
	)
}

func ToTypeScript(x Shape, option *TypeScriptOptions) string {
	x = ToTypeScriptOptimisation(x)
	return MatchShapeR1(
		x,
		func(x *Any) string {
			return "any"
		},
		func(x *RefName) string {
			prefix := ""
			if !option.IsCurrentPkgName(x.PkgName) {
				if x.PkgName == "" {
					return x.Name
				}

				prefix = fmt.Sprintf("%s.", x.PkgName)
				option.NeedsToImportPkgName(x.PkgName, x.PkgImportName)
			}

			if len(x.Indexed) > 0 {
				var names []string
				for _, name := range x.Indexed {
					names = append(names, toTypeTypeScriptTypeName(name, option))
				}
				return fmt.Sprintf("%s%s<%s>", prefix, x.Name, strings.Join(names, ", "))
			}

			return fmt.Sprintf("%s%s", prefix, x.Name)
		},
		func(x *AliasLike) string {
			return fmt.Sprintf("export type %s = %s\n", x.Name, ToTypeScript(x.Type, option))
		},
		func(x *PrimitiveLike) string {
			return MatchPrimitiveKindR1(
				x.Kind,
				func(x *BooleanLike) string {
					return "boolean"
				},
				func(x *StringLike) string {
					return "string"
				},
				func(x *NumberLike) string {
					return "number"
				},
			)
		},
		func(x *ListLike) string {
			return fmt.Sprintf("%s[]", ToTypeScript(x.Element, option))
		},
		func(x *MapLike) string {
			return fmt.Sprintf("{[key: %s]: %s}", ToTypeScript(x.Key, option), ToTypeScript(x.Val, option))
		},
		func(x *StructLike) string {
			result := &strings.Builder{}
			_, _ = fmt.Fprintf(result, "export type %s", x.Name)
			if len(x.TypeParams) > 0 {
				_, _ = fmt.Fprintf(result, "<")
				for i, params := range x.TypeParams {
					if i > 0 {
						_, _ = fmt.Fprintf(result, ", ")
					}
					_, _ = fmt.Fprintf(result, "%s", params.Name)

				}
				_, _ = fmt.Fprintf(result, ">")
			}

			_, _ = fmt.Fprintf(result, " = {")
			if len(x.Fields) > 0 {
				_, _ = fmt.Fprintf(result, "\n")
				for _, field := range x.Fields {
					result.WriteString(fmt.Sprintf("\t%s?: %s,\n", field.Name, ToTypeScript(field.Type, option)))
				}
			}
			result.WriteString("}")
			result.WriteString("\n")

			return result.String()
		},
		func(x *UnionLike) string {
			result := &strings.Builder{}
			// build union type in typescript
			_, _ = fmt.Fprintf(result, "export type %s = ", x.Name)
			result.WriteString(toTypeTypeScriptTypeName(x, option))

			result.WriteString("\n")

			return result.String()
		},
	)
}

func NewTypeScriptRenderer() *TypeScriptRenderer {
	return &TypeScriptRenderer{
		imports:    make(map[packageName]*TypeScriptOptions),
		contents:   make(map[packageName]*strings.Builder),
		shapeAdded: make(map[shapeName]bool),
	}
}

type TypeScriptRenderer struct {
	imports    map[packageName]*TypeScriptOptions
	contents   map[packageName]*strings.Builder
	shapeAdded map[shapeName]bool
}

func (r *TypeScriptRenderer) AddShape(x Shape) {
	if x == nil {
		return
	}

	// don't add shape twice
	key := ToGoTypeName(x, WithPkgImportName())
	if r.shapeAdded[key] {
		return
	}
	r.shapeAdded[key] = true

	MatchShapeR0(
		x,
		func(x *Any) {
			log.Infof("totypescript: AddShape Any is not supported")
		},
		func(x *RefName) {
			contents := r.initContentsFor(x.PkgImportName)
			options := r.initImportsFor(x.PkgName, x.PkgImportName)

			res := ToTypeScript(x, options)
			contents.WriteString(res)
			contents.WriteString("\n")
		},
		func(x *AliasLike) {
			contents := r.initContentsFor(x.PkgImportName)
			options := r.initImportsFor(x.PkgName, x.PkgImportName)

			res := ToTypeScript(x, options)
			contents.WriteString(res)
			contents.WriteString("\n")
		},
		func(x *PrimitiveLike) {
			log.Infof("totypescript: AddShape PrimitiveLike is not supported")
		},
		func(x *ListLike) {
			log.Infof("totypescript: AddShape ListLike is not supported")
		},
		func(x *MapLike) {
			log.Infof("totypescript: AddShape MapLike is not supported")
		},
		func(x *StructLike) {
			contents := r.initContentsFor(x.PkgImportName)
			options := r.initImportsFor(x.PkgName, x.PkgImportName)

			res := ToTypeScript(x, options)
			contents.WriteString(res)
			contents.WriteString("\n")

		},
		func(x *UnionLike) {
			contents := r.initContentsFor(x.PkgImportName)
			options := r.initImportsFor(x.PkgName, x.PkgImportName)

			res := ToTypeScript(x, options)
			contents.WriteString(res)
			contents.WriteString("\n")
		},
	)

	r.FollowRef(x)
}

func (r *TypeScriptRenderer) FollowRef(x Shape) {
	refs := ExtractRefs(x)
	for _, ref := range refs {
		log.Debugf("totypescript: FollowRef %s", ToGoTypeName(ref))
		x, found := LookupShapeOnDisk(ref)
		if found {
			r.AddShape(x)
		}
	}
}

func (r *TypeScriptRenderer) FollowImports() {
	for _, options := range r.imports {
		for _, imp := range options.imports {
			log.Debugf("totypescript: FollowImports %s", imp)
			shapes := LookupPkgShapeOnDisk(imp)
			for _, shape := range shapes {
				r.AddShape(shape)
			}
		}
	}
}

func (r *TypeScriptRenderer) initImportsFor(pkgName, pkgImportName string) *TypeScriptOptions {
	if _, ok := r.imports[pkgImportName]; ok {
		return r.imports[pkgImportName]
	}

	r.imports[pkgImportName] = &TypeScriptOptions{
		currentPkgName:       pkgName,
		currentPkgImportName: pkgImportName,
		imports:              make(map[packageName]packageImportName),
	}

	return r.imports[pkgImportName]
}

func (r *TypeScriptRenderer) initContentsFor(pkgImportName string) *strings.Builder {
	if _, ok := r.contents[pkgImportName]; !ok {
		r.contents[pkgImportName] = &strings.Builder{}
	}

	return r.contents[pkgImportName]
}

func (r *TypeScriptRenderer) WriteToDir(dir string) error {
	for pkgImportName, content := range r.contents {
		imports := r.imports[pkgImportName]
		if imports == nil {
			continue
		}

		importsContent := &strings.Builder{}
		for pkg, imp := range imports.imports {
			_, err := fmt.Fprintf(importsContent, "//eslint-disable-next-line\n")
			_, err = fmt.Fprintf(importsContent, "import * as %s from '%s'\n", pkg, r.normaliseImport(imp))
			if err != nil {
				return fmt.Errorf("totypescript: WriteToDir failed to write imports: %w", err)
			}
		}

		_, err := fmt.Fprintf(content, "\n%s", importsContent.String())
		if err != nil {
			return fmt.Errorf("totypescript: WriteToDir failed to write imports: %w", err)
		}

		header := "//generated by mkunion\n"
		err = r.writeToFile(dir, r.normaliseImport(pkgImportName), header+content.String())
		if err != nil {
			return fmt.Errorf("totypescript: WriteToDir failed to write file %s: %w", dir, err)
		}
	}

	return nil
}

func (r *TypeScriptRenderer) writeToFile(dir string, name packageName, content string) error {
	filename := path.Join(dir, fmt.Sprintf("%s.ts", name))
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (r *TypeScriptRenderer) normaliseImport(imp packageImportName) string {
	replace := strings.NewReplacer("/", "_", ".", "_")
	result := replace.Replace(imp)
	result = "./" + result
	return result
}

func toTypeTypeScriptTypeName(variant Shape, option *TypeScriptOptions) string {
	return MatchShapeR1(
		variant,
		func(x *Any) string {
			return "any"
		},
		func(x *RefName) string {
			prefix := ""
			if !option.IsCurrentPkgName(x.PkgName) {
				prefix = fmt.Sprintf("%s.", x.PkgName)
				option.NeedsToImportPkgName(x.PkgName, x.PkgImportName)
			}

			if len(x.Indexed) > 0 {
				var names []string
				for _, name := range x.Indexed {
					names = append(names, toTypeTypeScriptTypeName(name, option))
				}
				return fmt.Sprintf("%s%s<%s>", prefix, x.Name, strings.Join(names, ", "))
			}
			return prefix + x.Name
		},
		func(x *AliasLike) string {
			//typeName := toTypeTypeScriptTypeName(x.Type, option)
			typeName := x.Name
			typeNameFul := fmt.Sprintf("%s.%s", x.PkgName, x.Name)

			result := &strings.Builder{}
			result.WriteString("{\n")
			_, _ = fmt.Fprintf(result, "\t"+`"$type"?: "%s",`+"\n", typeNameFul)
			_, _ = fmt.Fprintf(result, "\t"+`"%s": %s`, typeNameFul, typeName)
			result.WriteString("\n}")

			return result.String()
		},
		func(x *PrimitiveLike) string {
			return MatchPrimitiveKindR1(
				x.Kind,
				func(x *BooleanLike) string {
					return "boolean"
				},
				func(x *StringLike) string {
					return "string"
				},
				func(x *NumberLike) string {
					return "number"
				},
			)
		},
		func(x *ListLike) string {
			return fmt.Sprintf("%s[]", ToTypeScript(x.Element, option))
		},
		func(x *MapLike) string {
			return fmt.Sprintf("{[key: %s]: %s}", ToTypeScript(x.Key, option), ToTypeScript(x.Val, option))
		},
		func(x *StructLike) string {
			result := &strings.Builder{}
			typeName := x.Name
			typeNameFul := fmt.Sprintf("%s.%s", x.PkgName, x.Name)

			result.WriteString("{\n")
			_, _ = fmt.Fprintf(result, "\t"+`"$type"?: "%s",`+"\n", typeNameFul)
			_, _ = fmt.Fprintf(result, "\t"+`"%s": %s`, typeNameFul, typeName)
			result.WriteString("\n}")

			return result.String()

		},
		func(x *UnionLike) string {
			result := &strings.Builder{}
			for idx, variant := range x.Variant {
				if idx > 0 {
					result.WriteString(" | ")
				}
				result.WriteString(toTypeTypeScriptTypeName(variant, option))
			}
			return result.String()
		},
	)
}
