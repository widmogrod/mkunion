package shape

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/shared"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/mod/modfile"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func InferFromFile(filename string) (*InferredInfo, error) {
	if !path.IsAbs(filename) {
		cwd, _ := os.Getwd()
		filename = path.Join(cwd, filename)
	}

	result := &InferredInfo{
		fileName:             filename,
		pkgImportName:        tryToFindPkgImportName(filename),
		possibleVariantTypes: map[string][]string{},
		possibleTaggedTypes:  map[string]map[string]Tag{},
		shapes:               make(map[string]Shape),
		taggedNodes:          make(map[string][]*NodeAndTag),
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	ast.Walk(result, f)
	return result, nil
}

// tryToFindPkgImportName contains import name of the package
func tryToFindPkgImportName(filename string) string {
	log.Debugf("shape.tryToFindPkgImportName: looking for go.mod file in %s", filename)
	var toadd []string
	for {
		filename = path.Dir(filename)
		if filename == "." || filename == "/" {
			log.Debugf("shape.tryToFindPkgImportName: could not find go.mod file in %s, returning empty pkg name", filename)
			return ""
		}

		modpath := path.Join(filename, "go.mod")
		log.Debugf("shape.tryToFindPkgImportName: checking modpath %s", modpath)
		if _, err := os.Stat(modpath); err == nil {
			f, err := os.Open(modpath)
			defer f.Close()

			data, err := io.ReadAll(f)
			if err != nil {
				log.Debugf("shape.tryToFindPkgImportName: could not read go.mod file in %s, returning empty pkg name. %s", filename, err.Error())
				return ""
			}

			parsed, err := modfile.Parse(modpath, data, nil)
			if err != nil {
				log.Debugf("shape.tryToFindPkgImportName: could not parse go.mod file in %s, returning empty pkg name. %s", filename, err.Error())
				return ""
			}

			if parsed.Module == nil {
				log.Debugf("shape.tryToFindPkgImportName: could not find module name in go.mod file in %s, returning empty pkg name", filename)
				return ""
			}

			result := path.Join(append([]string{parsed.Module.Mod.Path}, toadd...)...)

			log.Infof("shape.tryToFindPkgImportName: found module name %s", result)
			return result
		} else {
			log.Warnf("shape.tryToFindPkgImportName: could not find go.mod file in %s, continuing with parent directory", filename)
		}

		toadd = append([]string{path.Base(filename)}, toadd...)
	}
}

func tryToFindPkgName(pkgImportName, defaultPkgName string) string {
	if pkgImportName == "" {
		return defaultPkgName
	}

	pkgPath, err := findPackagePath(pkgImportName)
	if err != nil {
		log.Debugf("shape.tryToFindPkgName: could not find package path for %s; %s", pkgImportName, err)
		return defaultPkgName
	}

	// open any go file (except _test.go) in the package and extract package name
	// this is a hack, but it works

	result := defaultPkgName
	err = filepath.WalkDir(
		pkgPath,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				log.Debugf("shape.tryToFindPkgName: could not walk %s; %s", path, err)
				// ignore errors
				return nil
			}

			if d.IsDir() {
				if pkgPath != path {
					return filepath.SkipDir
				}
				return nil
			}

			if filepath.Ext(path) != ".go" {
				log.Debugf("shape.tryToFindPkgName: skipping non-go file %s", path)
				return nil
			}

			if strings.HasSuffix(path, "_test.go") {
				log.Debugf("shape.tryToFindPkgName: skipping test file %s", path)
				return nil
			}

			pkgName, found := pkgNameFromFile(path)
			if !found {
				log.Debugf("shape.tryToFindPkgName: could not find package name in %s", path)
				return nil
			}

			result = pkgName
			return filepath.SkipAll
		},
	)

	if err != nil {
		log.Warnf("shape.tryToFindPkgName: could not find package name for %s; %s", pkgImportName, err)
		return defaultPkgName
	}

	return result
}

func pkgNameFromFile(path string) (string, bool) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", false
	}

	if f.Name == nil {
		return "", false
	}

	return f.Name.String(), true
}

var (
	matchGoGenerateExtractUnionName = regexp.MustCompile(`go:generate .* -{1,2}name=*\s*(\w+)`)
)

type InferredInfo struct {
	fileName                   string
	pkgName                    string
	pkgImportName              string
	possibleVariantTypes       map[string][]string
	shapes                     map[string]Shape
	packageNameToPackageImport map[string]string
	currentType                string
	possibleTaggedTypes        map[string]map[string]Tag
	taggedNodes                map[string][]*NodeAndTag
}

type NodeAndTag struct {
	Name string
	Node ast.Node
	Tag  Tag
}

type TagVisitor func(x *NodeAndTag)

func (f *InferredInfo) RunVisitorOnTaggedASTNodes(tagName string, visitor TagVisitor) {
	for _, node := range f.taggedNodes[tagName] {
		visitor(node)
	}
}

func (f *InferredInfo) FileName() string {
	return f.fileName
}

func (f *InferredInfo) PackageName() string {
	return f.pkgName
}

func (f *InferredInfo) PackageImportName() string {
	return f.pkgImportName
}

func (f *InferredInfo) PackageNameToPackageImport() map[string]string {
	return f.packageNameToPackageImport
}

func (f *InferredInfo) RetrieveUnions() []*UnionLike {
	var result []*UnionLike
	for _, shape := range f.RetrieveShapes() {
		if unionShape, ok := shape.(*UnionLike); ok {
			result = append(result, unionShape)
		}
	}

	return result
}

func (f *InferredInfo) RetrieveStruct(name string) *StructLike {
	result, ok := f.shapes[name].(*StructLike)
	if !ok {
		return nil
	}

	return result
}

func (f *InferredInfo) RetrieveUnion(name string) *UnionLike {
	variants := f.collectVariants(name)
	if len(variants) == 0 {
		return nil
	}

	expectedTypeParams := f.extractExpectedTypeParams(name)
	uniqueTypeParams := f.collectUniqueTypeParams(variants)

	if err := f.validateTypeParameters(name, expectedTypeParams, uniqueTypeParams, variants); err != nil {
		log.Error(err)
		return nil
	}

	return &UnionLike{
		Name:          name,
		PkgName:       f.pkgName,
		PkgImportName: f.pkgImportName,
		TypeParams:    uniqueTypeParams,
		Variant:       variants,
		Tags:          f.possibleTaggedTypes[name],
	}
}

// collectVariants collects and prepares union variants.
func (f *InferredInfo) collectVariants(unionName string) []Shape {
	var variants []Shape
	for _, variantName := range f.possibleVariantTypes[unionName] {
		variant := f.shapes[variantName]
		// Always inject the base union name (without type params) into variants
		injectTag(variant, TagUnionName, unionName)
		variants = append(variants, variant)
	}
	return variants
}

// extractExpectedTypeParams extracts expected type parameters from union tags.
func (f *InferredInfo) extractExpectedTypeParams(unionName string) []string {
	unionTags := f.possibleTaggedTypes[unionName]
	if unionTag, ok := unionTags["mkunion"]; ok {
		_, params := parseUnionNameWithTypeParams(unionTag.Value)
		return params
	}
	return nil
}

// collectUniqueTypeParams collects unique type parameters from variants.
func (f *InferredInfo) collectUniqueTypeParams(variants []Shape) []TypeParam {
	seen := make(map[string]bool)
	var unique []TypeParam

	for _, v := range variants {
		for _, param := range ExtractTypeParams(v) {
			if !seen[param.Name] {
				seen[param.Name] = true
				unique = append(unique, param)
			}
		}
	}

	return unique
}

// validateTypeParameters validates type parameters consistency between tag and variants.
func (f *InferredInfo) validateTypeParameters(unionName string, expected []string, unique []TypeParam, variants []Shape) error {
	if len(unique) == 0 {
		return nil // No generic variants, nothing to validate
	}

	if len(expected) == 0 {
		return fmt.Errorf(
			"%s: Union %q requires type params %s - change tag to //go:tag mkunion:%q",
			f.fileName, unionName, formatTypeParamsForTag(unique),
			unionName+formatTypeParamsForTag(unique))
	}

	if len(expected) != len(unique) {
		plural1 := ""
		if len(expected) != 1 {
			plural1 = "s"
		}
		plural2 := ""
		if len(unique) != 1 {
			plural2 = "s"
		}
		return fmt.Errorf(
			"%s: Union %q tag has %d param%s %v but variants use %d param%s %s",
			f.fileName, unionName, len(expected), plural1, expected,
			len(unique), plural2, formatTypeParamsForTag(unique))
	}

	return f.validateVariantParameters(unionName, expected, variants)
}

// validateVariantParameters validates each variant's type parameters.
func (f *InferredInfo) validateVariantParameters(unionName string, expected []string, variants []Shape) error {
	for _, variant := range variants {
		variantParams := ExtractTypeParams(variant)
		variantName := Name(variant)

		if err := f.validateVariantParamCount(unionName, variantName, expected, variantParams); err != nil {
			return err
		}

		if err := f.validateVariantParamNames(unionName, variantName, expected, variantParams); err != nil {
			return err
		}
	}

	return nil
}

// validateVariantParamCount validates variant has correct number of type parameters.
func (f *InferredInfo) validateVariantParamCount(unionName, variantName string, expected []string, actual []TypeParam) error {
	if len(actual) != len(expected) {
		return fmt.Errorf(
			"%s: Union %q variant %q has params %s but tag expects %v",
			f.fileName, unionName, variantName,
			formatTypeParamsForTag(actual), expected)
	}
	return nil
}

// validateVariantParamNames validates variant parameter names match by position.
func (f *InferredInfo) validateVariantParamNames(unionName, variantName string, expected []string, actual []TypeParam) error {
	for i, expectedName := range expected {
		if i < len(actual) && actual[i].Name != expectedName {
			return fmt.Errorf(
				"%s: Union %q variant %q param #%d: expected %q but found %q",
				f.fileName, unionName, variantName, i+1, expectedName, actual[i].Name)
		}
	}
	return nil
}

func (f *InferredInfo) RetrieveShapes() []Shape {
	shapes := make(map[string]Shape)

	ordered := make([]string, 0)
	for name, shape := range f.shapes {
		shapes[name] = shape
		ordered = append(ordered, name)
	}
	sort.Strings(ordered)

	var result []Shape
	unionNames := f.sortedPossibleUnionNames()
	for _, unionName := range unionNames {
		union := f.RetrieveUnion(unionName)
		if union == nil {
			continue
		}

		result = append(result, union)
		delete(shapes, unionName)
		for _, variantName := range f.possibleVariantTypes[unionName] {
			delete(shapes, variantName)
		}
	}

	for _, name := range ordered {
		if _, ok := shapes[name]; !ok {
			continue
		}
		result = append(result, shapes[name])
	}

	return result
}

func (f *InferredInfo) sortedPossibleUnionNames() []string {
	unionNames := make([]string, len(f.possibleVariantTypes))
	for unionName := range f.possibleVariantTypes {
		unionNames = append(unionNames, unionName)
	}
	sort.Strings(unionNames)
	return unionNames
}

func (f *InferredInfo) RetrieveStructs() []*StructLike {
	var result []*StructLike
	for _, shape := range f.RetrieveShapes() {
		if structShape, ok := shape.(*StructLike); ok {
			result = append(result, structShape)
		}
	}

	return result
}

func (f *InferredInfo) RetrieveShapeNamedAs(name string) Shape {
	if result, ok := f.shapes[name]; ok {
		return result
	}

	res := f.RetrieveUnion(name)
	if res != nil {
		return res
	}

	return nil
}

func (f *InferredInfo) RetrieveShapeFromRef(x *RefName) Shape {
	shapes := f.RetrieveShapes()
	for _, shape := range shapes {
		// weak check
		if Name(shape) == Name(x) {
			return shape
		}
	}

	return nil
}

func (f *InferredInfo) RetrieveShapesTaggedAs(tagName string) []Shape {
	var result []Shape
	for _, shape := range f.RetrieveShapes() {
		tags := Tags(shape)
		if _, ok := tags[tagName]; ok {
			result = append(result, shape)
		}
	}

	return result
}

// removeTypeParams removes type parameters from a given string, returning the base name.
// Examples: "Option[A]" -> "Option", "Result[T, E]" -> "Result"
func removeTypeParams(name string) string {
	if i := strings.Index(name, "["); i != -1 {
		return name[:i]
	}
	return name
}

func (f *InferredInfo) Visit(n ast.Node) ast.Visitor {
	opt := f.optionAST()
	switch t := n.(type) {
	case *ast.GenDecl:
		if t.Tok != token.TYPE {
			return f
		}

		// detect declaration of union type
		// either as comment
		// //go:generate mkunion -name=Example
		// //go:tag mkunion:"Example"
		tags := ExtractDocumentTags(t.Doc)

		for tname, tvalue := range tags {
			f.taggedNodes[tname] = append(f.taggedNodes[tname], &NodeAndTag{
				Name: tname,
				Node: t,
				Tag:  tvalue,
			})
		}

		// detect single declaration of type with a comment block
		// // some comment
		// type A struct {}
		if t.Lparen == 0 && t.Rparen == 0 && len(t.Specs) == 1 {
			switch s := t.Specs[0].(type) {
			case *ast.TypeSpec:
				// extract individual tags for each of variant
				f.possibleTaggedTypes[s.Name.Name] = tags
				return f
			}
		}

		// when there are more than one spec block,
		// it means that we are dealing with union (by convention)

		// register tags for specific type inside block:
		// type (
		//   ...
		// )
		for _, spec := range t.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				// extract individual tags for each of variant
				f.possibleTaggedTypes[s.Name.Name] = ExtractDocumentTags(s.Doc)
			}
		}

		unionName := ""
		if unionTag, ok := tags["mkunion"]; ok {
			unionName = removeTypeParams(unionTag.Value)
		} else {
			comment := shared.Comment(t.Doc)
			names := matchGoGenerateExtractUnionName.FindStringSubmatch(comment)
			if len(names) < 2 {
				return f
			}
			unionName = names[1]
		}

		// It's impossible to have type Option interface{} and type Option[A] interface{} in same package
		// Register union tags under the base name (without type params)
		f.possibleTaggedTypes[unionName] = tags

		// start capturing possible variants
		if _, ok := f.possibleVariantTypes[unionName]; !ok {
			f.possibleVariantTypes[unionName] = make([]string, 0)
		}

		for _, spec := range t.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				// register possible variant for union
				// NOTE: this is only convention that unions must be declared as type group specification:
				// type (
				// 	Variant2 struct {}
				//	Variant2 int
				//)
				f.possibleVariantTypes[unionName] = append(f.possibleVariantTypes[unionName], s.Name.Name)
			}
		}

		return f

	case *ast.File:
		if t.Name != nil {
			f.pkgName = t.Name.String()
		}

		f.packageNameToPackageImport = map[string]string{
			f.pkgName: f.pkgImportName,
		}
		for _, imp := range t.Imports {
			pkgImportName := strings.Trim(imp.Path.Value, "\"")
			if imp.Name != nil {
				f.packageNameToPackageImport[imp.Name.String()] = pkgImportName
			} else {
				defaultPkgName := path.Base(pkgImportName)
				pkgName := tryToFindPkgName(pkgImportName, defaultPkgName)
				f.packageNameToPackageImport[pkgName] = pkgImportName
			}
		}

	case *ast.TypeSpec:
		f.currentType = t.Name.Name

		// Detect named literal types like:
		// type A string
		// type B int
		// type C bool
		switch next := t.Type.(type) {
		case *ast.Ident:
			switch next.Name {
			case "string":
				f.shapes[f.currentType] = &AliasLike{
					Name:          f.currentType,
					PkgName:       f.pkgName,
					PkgImportName: f.pkgImportName,
					IsAlias:       IsASTAlias(t),
					Type:          &PrimitiveLike{Kind: &StringLike{}},
					Tags:          f.possibleTaggedTypes[f.currentType],
				}

			case "int", "int8", "int16", "int32", "int64",
				"uint", "uint8", "uint16", "uint32", "uint64",
				"float64", "float32", "byte", "rune":
				f.shapes[f.currentType] = &AliasLike{
					Name:          f.currentType,
					PkgName:       f.pkgName,
					PkgImportName: f.pkgImportName,
					IsAlias:       IsASTAlias(t),
					Type: &PrimitiveLike{
						Kind: &NumberLike{
							Kind: TypeStringToNumberKindMap[next.Name],
						},
					},
					Tags: f.possibleTaggedTypes[f.currentType],
				}

			case "bool":
				f.shapes[f.currentType] = &AliasLike{
					Name:          f.currentType,
					PkgName:       f.pkgName,
					PkgImportName: f.pkgImportName,
					IsAlias:       IsASTAlias(t),
					Type:          &PrimitiveLike{Kind: &BooleanLike{}},
					Tags:          f.possibleTaggedTypes[f.currentType],
				}

			default:
				// alias type from the same package
				// example
				//  type A ListOf
				f.shapes[f.currentType] = &AliasLike{
					Name:          f.currentType,
					PkgName:       f.pkgName,
					PkgImportName: f.pkgImportName,
					TypeParams:    f.extractTypeParams(t.TypeParams),
					IsAlias:       IsASTAlias(t),
					Type: &RefName{
						Name:          next.Name,
						PkgName:       f.pkgName,
						PkgImportName: f.pkgImportName,
					},
					Tags: f.possibleTaggedTypes[f.currentType],
				}
			}

		case *ast.SelectorExpr:
			// alias type from other packages
			// example:
			//  type A time.Time
			f.shapes[f.currentType] = &AliasLike{
				Name:          f.currentType,
				PkgName:       f.pkgName,
				PkgImportName: f.pkgImportName,
				TypeParams:    f.extractTypeParams(t.TypeParams),
				IsAlias:       IsASTAlias(t),
				Type:          f.selectExrToShape(next),
				Tags:          f.possibleTaggedTypes[f.currentType],
			}

		case *ast.IndexExpr:
			// alias of type that has one type params initialized
			// example:
			//  type A ListOf[any]
			//  type B ListOf[ListOf[any]]
			f.shapes[f.currentType] = &AliasLike{
				Name:          f.currentType,
				PkgName:       f.pkgName,
				PkgImportName: f.pkgImportName,
				TypeParams:    f.extractTypeParams(t.TypeParams),
				IsAlias:       IsASTAlias(t),
				Type:          FromAST(next, opt...),
				Tags:          f.possibleTaggedTypes[f.currentType],
			}

		case *ast.IndexListExpr:
			// alias of type that has two type params initialized
			// example:
			//  type A ListOf2[string, int]
			//  type B ListOf2[ListOf2[string, int], ListOf2[string, int]]
			f.shapes[f.currentType] = &AliasLike{
				Name:          f.currentType,
				PkgName:       f.pkgName,
				PkgImportName: f.pkgImportName,
				TypeParams:    f.extractTypeParams(t.TypeParams),
				IsAlias:       IsASTAlias(t),
				Type:          FromAST(next, opt...),
				Tags:          f.possibleTaggedTypes[f.currentType],
			}

		case *ast.MapType:
			// example:
			//  type A map[string]string
			//  type B map[string]ListOf2[string, int]
			f.shapes[f.currentType] = &AliasLike{
				Name:          f.currentType,
				PkgName:       f.pkgName,
				PkgImportName: f.pkgImportName,
				TypeParams:    f.extractTypeParams(t.TypeParams),
				IsAlias:       IsASTAlias(t),
				Type: &MapLike{
					Key: FromAST(next.Key, opt...),
					Val: FromAST(next.Value, opt...),
					//KeyIsPointer: IsStarExpr(next.Key),
					//ValIsPointer: IsStarExpr(next.Term),
				},
				Tags: f.possibleTaggedTypes[f.currentType],
			}

		case *ast.ArrayType:
			f.shapes[f.currentType] = &AliasLike{
				Name:          f.currentType,
				PkgName:       f.pkgName,
				PkgImportName: f.pkgImportName,
				TypeParams:    f.extractTypeParams(t.TypeParams),
				IsAlias:       IsASTAlias(t),
				Type: &ListLike{
					Element: FromAST(next.Elt, opt...),
					//ElementIsPointer: IsStarExpr(next.Elt),
					ArrayLen: tryGetArrayLen(next.Len),
				},
				Tags: f.possibleTaggedTypes[f.currentType],
			}

		case *ast.StructType:
			f.shapes[f.currentType] = &StructLike{
				Name:          f.currentType,
				PkgName:       f.pkgName,
				PkgImportName: f.pkgImportName,
				TypeParams:    f.extractTypeParams(t.TypeParams),
				Tags:          f.possibleTaggedTypes[f.currentType],
			}

		case *ast.StarExpr:
			// example:
			//  type A *string
			//  type B = *int
			f.shapes[f.currentType] = &AliasLike{
				Name:          f.currentType,
				PkgName:       f.pkgName,
				PkgImportName: f.pkgImportName,
				TypeParams:    f.extractTypeParams(t.TypeParams),
				IsAlias:       IsASTAlias(t),
				Type: &PointerLike{
					Type: FromAST(next.X, opt...),
				},
			}
		}

	case *ast.StructType:
		if !t.Struct.IsValid() {
			break
		}

		structShape, ok := f.shapes[f.currentType].(*StructLike)
		if !ok {
			log.Warnf("shape.InferFromFile: could not cast %s to StructLike", f.currentType)
			return f
		}

		for _, field := range t.Fields.List {
			// this happens when field is embedded in struct
			// something like `type A struct { B }`
			if len(field.Names) == 0 {
				switch typ := field.Type.(type) {
				case *ast.Ident:
					structShape.Fields = append(structShape.Fields, &FieldLike{
						Name: typ.Name,
						Type: FromAST(typ, opt...),
					})
					break
				default:
					log.Warnf("shape.InferFromFile: unknown ast type embedded in struct: %T\n", typ)
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
					typ = f.selectExrToShape(ttt)
				// this is reference to other struct in the same package or other package
				case *ast.StarExpr:
					if selector, ok := ttt.X.(*ast.SelectorExpr); ok {
						typ = f.selectExrToShape(selector)
						typ = &PointerLike{
							Type: typ,
						}
					} else {
						typ = FromAST(ttt, opt...)
					}

				case *ast.IndexExpr, *ast.Ident, *ast.ArrayType, *ast.MapType, *ast.StructType:
					typ = FromAST(ttt, opt...)

				default:
					log.Warnf("shape.InferFromFile: unknown ast type in  %s.%s: %T\n", f.currentType, fieldName.Name, ttt)
					typ = &Any{}
				}

				typ = CleanTypeThatAreOvershadowByTypeParam(typ, structShape.TypeParams)

				tag := ""
				if field.Tag != nil {
					tag = field.Tag.Value
				}

				tags := ExtractTags(tag)
				desc := TagsToDesc(tags)
				guard := TagsToGuard(tags)

				structShape.Fields = append(structShape.Fields, &FieldLike{
					Name:  fieldName.Name,
					Type:  typ,
					Desc:  desc,
					Guard: guard,
					Tags:  tags,
				})
			}
		}

		f.shapes[f.currentType] = structShape
		log.Infof("shape.InferFromFile: struct %s: %s\n", f.currentType, ToStr(structShape))
	}

	return f
}

func CleanTypeThatAreOvershadowByTypeParam(typ Shape, params []TypeParam) Shape {
	return MatchShapeR1(
		typ,
		func(x *Any) Shape {
			return x
		},
		func(x *RefName) Shape {
			if nameExistsInParams(x.Name, params) {
				x.PkgName = ""
				x.PkgImportName = ""
			}

			for i, name := range x.Indexed {
				x.Indexed[i] = CleanTypeThatAreOvershadowByTypeParam(name, params)
			}

			return x
		},
		func(x *PointerLike) Shape {
			x.Type = CleanTypeThatAreOvershadowByTypeParam(x.Type, params)
			return x
		},
		func(x *AliasLike) Shape {
			if nameExistsInParams(x.Name, params) {
				x.PkgName = ""
				x.PkgImportName = ""
			}

			return x
		},
		func(x *PrimitiveLike) Shape {
			return x
		},
		func(x *ListLike) Shape {
			x.Element = CleanTypeThatAreOvershadowByTypeParam(x.Element, params)
			return x
		},
		func(x *MapLike) Shape {
			x.Key = CleanTypeThatAreOvershadowByTypeParam(x.Key, params)
			x.Val = CleanTypeThatAreOvershadowByTypeParam(x.Val, params)
			return x
		},
		func(x *StructLike) Shape {
			if nameExistsInParams(x.Name, params) {
				x.PkgName = ""
				x.PkgImportName = ""
			}

			for _, field := range x.Fields {
				field.Type = CleanTypeThatAreOvershadowByTypeParam(field.Type, params)
			}
			return x
		},
		func(x *UnionLike) Shape {
			if nameExistsInParams(x.Name, params) {
				x.PkgName = ""
				x.PkgImportName = ""
			}

			for _, variant := range x.Variant {
				variant = CleanTypeThatAreOvershadowByTypeParam(variant, params)
			}
			return x
		},
	)
}

func IndexWith(y Shape, ref *RefName) Shape {
	var typeParams []TypeParam
	switch x := y.(type) {
	case *StructLike:
		typeParams = x.TypeParams
	case *AliasLike:
		typeParams = x.TypeParams
	case *UnionLike:
		typeParams = x.TypeParams
	default:
		return y
	}

	z := y

	if len(typeParams) != len(ref.Indexed) ||
		len(typeParams) == 0 {
		return z
	}

	params := make(map[string]Shape, len(typeParams))
	for i, param := range typeParams {
		params[param.Name] = ref.Indexed[i]
	}

	return InstantiateTypeThatAreOvershadowByTypeParam(z, params)
}

func InstantiateTypeThatAreOvershadowByTypeParam(typ Shape, replacement map[string]Shape) Shape {
	return MatchShapeR1(
		typ,
		func(x *Any) Shape {
			return x
		},
		func(x *RefName) Shape {
			if result, err := replacement[x.Name]; err {
				return result
			}

			result := &RefName{
				Name:          x.Name,
				PkgName:       x.PkgName,
				PkgImportName: x.PkgImportName,
			}
			for _, name := range x.Indexed {
				result.Indexed = append(result.Indexed, InstantiateTypeThatAreOvershadowByTypeParam(name, replacement))
			}

			return result
		},
		func(x *PointerLike) Shape {
			result := &PointerLike{
				Type: InstantiateTypeThatAreOvershadowByTypeParam(x.Type, replacement),
			}
			return result
		},
		func(x *AliasLike) Shape {
			result := &AliasLike{
				Name:          x.Name,
				PkgName:       x.PkgName,
				PkgImportName: x.PkgImportName,
				IsAlias:       x.IsAlias,
				Type:          InstantiateTypeThatAreOvershadowByTypeParam(x.Type, replacement),
				Tags:          x.Tags,
			}

			// change names of type params, to represent substitution
			for _, param := range x.TypeParams {
				param := param
				if rep, ok := replacement[param.Name]; ok {
					param.Name = ToGoTypeName(rep, WithRootPackage(x.PkgName))
					param.Type = rep
				}

				result.TypeParams = append(result.TypeParams, param)
			}

			return result
		},
		func(x *PrimitiveLike) Shape {
			return x
		},
		func(x *ListLike) Shape {
			result := &ListLike{
				Element:  InstantiateTypeThatAreOvershadowByTypeParam(x.Element, replacement),
				ArrayLen: x.ArrayLen,
			}
			return result
		},
		func(x *MapLike) Shape {
			result := &MapLike{
				Key: InstantiateTypeThatAreOvershadowByTypeParam(x.Key, replacement),
				Val: InstantiateTypeThatAreOvershadowByTypeParam(x.Val, replacement),
			}
			return result
		},
		func(x *StructLike) Shape {
			result := &StructLike{
				Name:          x.Name,
				PkgName:       x.PkgName,
				PkgImportName: x.PkgImportName,
				Tags:          x.Tags,
			}

			// change names of type params, to represent substitution
			for _, param := range x.TypeParams {
				param := param
				if rep, ok := replacement[param.Name]; ok {
					param.Name = ToGoTypeName(rep, WithRootPackage(x.PkgName))
					param.Type = rep
				}

				result.TypeParams = append(result.TypeParams, param)
			}

			for _, field := range x.Fields {
				result.Fields = append(result.Fields, &FieldLike{
					Name:  field.Name,
					Type:  InstantiateTypeThatAreOvershadowByTypeParam(field.Type, replacement),
					Desc:  field.Desc,
					Guard: field.Guard,
					Tags:  field.Tags,
				})
			}
			return result
		},
		func(x *UnionLike) Shape {
			result := &UnionLike{
				Name:          x.Name,
				PkgName:       x.PkgName,
				PkgImportName: x.PkgImportName,
				Tags:          x.Tags,
			}
			// change names of type params, to represent substitution
			for _, param := range x.TypeParams {
				param := param
				if rep, ok := replacement[param.Name]; ok {
					param.Name = ToGoTypeName(rep, WithRootPackage(x.PkgName))
					param.Type = rep
				}

				result.TypeParams = append(result.TypeParams, param)
			}

			for _, variant := range x.Variant {
				result.Variant = append(result.Variant, InstantiateTypeThatAreOvershadowByTypeParam(variant, replacement))
			}
			return result
		},
	)
}

// nameExistsInParams checks if a type parameter name exists in the given list.
func nameExistsInParams(name string, params []TypeParam) bool {
	for _, param := range params {
		if param.Name == name {
			return true
		}
	}
	return false
}

func (f *InferredInfo) optionAST() []FromASTOption {
	return []FromASTOption{
		InjectPkgName(f.pkgName),
		InjectPkgImportName(f.packageNameToPackageImport),
	}
}

func IsASTAlias(t *ast.TypeSpec) bool {
	return t.Assign != 0
}

func (f *InferredInfo) selectExrToShape(ttt *ast.SelectorExpr) Shape {
	if ident, ok := ttt.X.(*ast.Ident); ok {
		pkgName := ident.Name
		return FromAST(ttt.Sel, InjectPkgName(pkgName), InjectPkgImportName(f.packageNameToPackageImport))
	}

	return FromAST(ttt, f.optionAST()...)
}

func (f *InferredInfo) extractTypeParams(params *ast.FieldList) []TypeParam {
	if params == nil {
		return nil
	}

	var result []TypeParam
	for _, param := range params.List {
		typ := FromAST(param.Type,
			InjectPkgImportName(f.packageNameToPackageImport),
			InjectPkgName(f.pkgName),
		)

		if len(param.Names) == 0 {
			result = append(result, TypeParam{
				Name: param.Type.(*ast.Ident).Name,
				Type: typ,
			})
			continue
		}

		for _, name := range param.Names {
			result = append(result, TypeParam{
				Name: name.Name,
				Type: typ,
			})
		}
	}

	return result
}

func tryGetArrayLen(expr ast.Expr) *int {
	if expr == nil {
		return nil
	}

	switch x := expr.(type) {
	case *ast.BasicLit:
		if x.Kind == token.INT {
			n, _ := strconv.Atoi(x.Value)
			return &n
		}
	}

	return nil
}

func NewIndexTypeInDir(dir string) (*IndexedTypeWalker, error) {
	if !path.IsAbs(dir) {
		cwd, _ := os.Getwd()
		dir = path.Join(cwd, dir)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("shape.NewIndexTypeInDir: %s does not exist", dir)
	}

	result := &IndexedTypeWalker{
		indexedShapes:              make(map[string]Shape),
		packageNameToPackageImport: make(map[string]string),
		knownTypeNamesInPackage:    map[string]struct{}{},
		pkgName:                    "", // will be set in ast.Walk
		pkgImportName:              tryToFindPkgImportName(path.Join(dir, "shape_index_type_in_dir.go")),
	}

	fset := token.NewFileSet()

	err := filepath.WalkDir(dir,
		func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				if path == dir {
					return nil
				}
				return filepath.SkipDir
			}

			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("could not parse file %s; %w", path, err)
			}

			ast.Walk(result, f)

			return nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("shape.NewIndexTypeInDir: %w", err)
	}

	return result, nil
}

func newIndexedTypeWalkerWithContentBody(x string, opts ...func(x *IndexedTypeWalker)) *IndexedTypeWalker {
	walker := &IndexedTypeWalker{
		indexedShapes:              make(map[string]Shape),
		packageNameToPackageImport: make(map[string]string),
		knownTypeNamesInPackage:    map[string]struct{}{},
	}

	for _, opt := range opts {
		opt(walker)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "_new_walker.go", []byte(x), parser.ParseComments)
	if err != nil {
		panic(err)
	}

	ast.Walk(walker, f)

	return walker
}

type IndexedTypeWalker struct {
	filterGenericTypes         []string
	indexedShapes              map[string]Shape
	packageNameToPackageImport map[string]string

	knownTypeNamesInPackage map[string]struct{}

	pkgName       string
	pkgImportName string
	packageTags   map[string]Tag
}

func (walker *IndexedTypeWalker) PackageName() string {
	return walker.pkgName
}

func (walker *IndexedTypeWalker) PkgMap() map[string]string {
	return walker.packageNameToPackageImport
}

func (walker *IndexedTypeWalker) SetPkgImportName(pkgImportName string) {
	walker.pkgImportName = pkgImportName
}

func (walker *IndexedTypeWalker) IndexedShapes() map[string]Shape {
	return walker.indexedShapes
}

func (walker *IndexedTypeWalker) PackageTags() map[string]Tag {
	return walker.packageTags
}

func (walker *IndexedTypeWalker) ExpandedShapes() map[string]Shape {
	// find if given shape is variant, of an union or union
	// if is variant, then append to list other variants of the union, and index them by the same type
	// if is union, then append to list all variants of the union, and index them by the same type
	result := make(map[string]Shape, len(walker.indexedShapes))

	for name, shape := range walker.IndexedShapes() {
		result[name] = shape

		ref, ok := shape.(*RefName)
		if !ok {
			continue
		}

		newShape, found := LookupShapeOnDisk(ref)
		if !found {
			log.Debugf("shape.ExpandedShapes: not found during ref lookup: %s", name)
			continue
		}

		union, ok := newShape.(*UnionLike)
		if !ok {
			if unionRef := RetrieveVariantTypeRef(newShape); unionRef != nil {
				foundShape, _ := LookupShapeOnDisk(unionRef)
				union, _ = foundShape.(*UnionLike)
			}
		}

		if union == nil {
			continue
		}

		for _, variant := range union.Variant {
			if IsWeekAlias(variant) {
				continue
			}

			newVariant := IndexWith(variant, ref)
			name := ToGoTypeName(newVariant)
			result[name] = newVariant
		}
	}

	return result
}

func (walker *IndexedTypeWalker) Visit(n ast.Node) ast.Visitor {
	switch t := n.(type) {
	case *ast.File:
		if t.Name != nil {
			walker.pkgName = t.Name.String()
		}

		walker.packageNameToPackageImport[walker.pkgName] = walker.pkgImportName
		walker.packageTags = MergeTagsInto(walker.packageTags, ExtractDocumentTags(t.Doc))

		for _, imp := range t.Imports {
			pkgImportName := strings.Trim(imp.Path.Value, "\"")
			if imp.Name != nil {
				walker.packageNameToPackageImport[imp.Name.String()] = pkgImportName
			} else {
				defaultPkgName := path.Base(pkgImportName)
				pkgName := tryToFindPkgName(pkgImportName, defaultPkgName)
				walker.packageNameToPackageImport[pkgName] = pkgImportName
			}
		}

	case *ast.TypeSpec:
		walker.knownTypeNamesInPackage[t.Name.Name] = struct{}{}

		prev := walker.filterGenericTypes
		walker.filterGenericTypes = walker.typeParamNames(t.TypeParams)

		// for named types try to register them as indexed types
		// if they are indexed types
		if t.TypeParams != nil {
			for _, param := range t.TypeParams.List {
				walker.registerIndexedShape(param.Type)
			}
		}

		// for the actual details of the type
		ast.Walk(walker, t.Type)

		walker.filterGenericTypes = prev
		return nil

	case *ast.ValueSpec:
		if t.Type != nil {
			walker.registerIndexedShape(t.Type)
		}

		if t.Values != nil {
			for _, value := range t.Values {
				walker.registerIndexedShape(value)
			}
		}

		for _, val := range t.Values {
			ast.Walk(walker, val)
		}

		return nil

	case *ast.CompositeLit:
		walker.registerIndexedShape(t.Type)

	case *ast.FuncDecl:
		fun := t.Type

		prev := walker.filterGenericTypes
		if t.Recv != nil {
			walker.filterGenericTypes = walker.guessParamNamesReceiver(t.Recv)
			for _, param := range t.Recv.List {
				walker.registerIndexedShape(param.Type)
			}
		} else {
			walker.filterGenericTypes = walker.typeParamNames(fun.TypeParams)
		}

		if fun.TypeParams != nil {
			for _, param := range fun.TypeParams.List {
				walker.registerIndexedShape(param.Type)
			}
		}

		// walk function type params (type params of the function)
		if fun.Params != nil {
			for _, param := range fun.Params.List {
				walker.registerIndexedShape(param.Type)
			}
		}

		// walk function results (return params of the function)
		if fun.Results != nil {
			for _, result := range fun.Results.List {
				walker.registerIndexedShape(result.Type)
			}
		}

		// walk function body
		ast.Walk(walker, t.Body)

		walker.filterGenericTypes = prev
		return nil

	case *ast.FuncType:
		// for list of params
		// for list of results
		// attempt to find indexed types
		// and register them as indexed types

		if t.Params != nil {
			for _, param := range t.Params.List {
				walker.registerIndexedShape(param.Type)
			}
		}

		if t.Results != nil {
			for _, result := range t.Results.List {
				walker.registerIndexedShape(result.Type)
			}
		}

		return nil

	case *ast.CallExpr:
		switch fun := t.Fun.(type) {
		case *ast.IndexExpr:
			// this is a function call with type params
			// in such situations we're interested in indexed values, not function name
			// example:
			//  shared.JSONMarshal[ListOf[int]](...)
			switch fun.Index.(type) {
			case *ast.IndexExpr, *ast.IndexListExpr:
				walker.registerIndexedShape(fun.Index)
			}

			// func arguments can have indexed type, so iterate them
			for _, arg := range t.Args {
				// function argument could be other function, so let's walk it
				ast.Walk(walker, arg)
			}

			// we're done here, dont traverse deeper
			return nil

		case *ast.IndexListExpr:
			// this is a function call with type params
			// in such situations we're interested in indexed values, not function name
			// example:
			//  shared.JSONMarshal[ListOf2[int,any]](...)
			for _, arg := range fun.Indices {
				switch arg := arg.(type) {
				case *ast.IndexExpr, *ast.IndexListExpr:
					walker.registerIndexedShape(arg)
				}
			}

			// func arguments can have indexed type, so iterate them
			for _, arg := range t.Args {
				// function argument could be other function, so let's walk it
				ast.Walk(walker, arg)
			}

			// we're done here, dont traverse deeper
			return nil

		default:
			// func arguments can have indexed type, so iterate them
			for _, arg := range t.Args {
				// function argument could be other function, so let's walk it
				ast.Walk(walker, arg)
			}

			// we're done here, dont traverse deeper
			return nil
		}
	}

	return walker
}

func (walker *IndexedTypeWalker) typeParamNames(x ast.Node) []string {
	switch t := x.(type) {
	case *ast.FieldList:
		if t == nil {
			return nil
		}

		if t.List == nil {
			return nil
		}

		var result []string
		for _, param := range t.List {
			for _, name := range param.Names {
				result = append(result, name.Name)
			}
		}
		return result
	}

	panic(fmt.Sprintf("typeParamNames: unknown type %T", x))
}

func (walker *IndexedTypeWalker) guessParamNamesReceiver(x *ast.FieldList) []string {
	if x == nil {
		return nil
	}

	if x.List == nil {
		return nil
	}

	var result []string
	for _, param := range x.List {
		types := []ast.Expr{param.Type}
		for {
			typ := types[0]
			types = types[1:]

			switch t := typ.(type) {
			case *ast.StarExpr:
				types = append(types, t.X)
			case *ast.IndexExpr:
				types = append(types, t.Index)
			case *ast.IndexListExpr:
				for _, index := range t.Indices {
					types = append(types, index)
				}
			case *ast.Ident:
				if _, ok := walker.knownTypeNamesInPackage[t.Name]; !ok {
					result = append(result, t.Name)
				}
			}

			if len(types) == 0 {
				break
			}
		}
	}

	return result
}

func (walker *IndexedTypeWalker) registerIndexedShape(arg ast.Node) {
	switch arg.(type) {
	case *ast.IndexExpr, *ast.IndexListExpr, *ast.StarExpr:
		options := []FromASTOption{
			InjectPkgName(walker.pkgName),
			InjectPkgImportName(walker.packageNameToPackageImport),
		}
		indexed := FromAST(arg, options...)

		// if is pointer unwrap it
		ptr, ok := indexed.(*PointerLike)
		if ok {
			indexed = ptr.Type
		}

		if len(walker.filterGenericTypes) > 0 {
			indexedName := Name(indexed)
			for _, name := range walker.filterGenericTypes {
				if name == indexedName {
					// we extracted type parameter, not interested in it
					return
				}

				typeParams := ExtractIndexedTypes(indexed)
				for {
					if len(typeParams) == 0 {
						break
					}

					tp := typeParams[0]
					typeParams = typeParams[1:]
					if Name(tp) == name {
						// we extracted type parameter, not interested in it
						return
					}

					typeParams = append(typeParams, ExtractIndexedTypes(tp)...)
				}
			}
		}

		name := ToGoTypeName(indexed)

		if _, ok := walker.indexedShapes[name]; !ok {
			walker.indexedShapes[name] = indexed
		}
	}
}
