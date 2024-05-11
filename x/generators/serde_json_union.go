package generators

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
)

func NewSerdeJSONUnion(union *shape.UnionLike) *SerdeJSONUnion {
	return &SerdeJSONUnion{
		union:                 union,
		skipImportsAndPackage: false,
		skipInitFunc:          false,
		pkgUsed: PkgMap{
			"json":   "encoding/json",
			"fmt":    "fmt",
			"shared": "github.com/widmogrod/mkunion/x/shared",
		},
	}
}

type SerdeJSONUnion struct {
	union                 *shape.UnionLike
	skipImportsAndPackage bool
	skipInitFunc          bool
	pkgUsed               PkgMap
}

func (g *SerdeJSONUnion) SkipImportsAndPackage(x bool) {
	g.skipImportsAndPackage = x
}

func (g *SerdeJSONUnion) SkipInitFunc(flag bool) *SerdeJSONUnion {
	g.skipInitFunc = flag
	return g
}

func (g *SerdeJSONUnion) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *SerdeJSONUnion) ExtractImports(x shape.Shape) PkgMap {
	pkgMap := shape.ExtractPkgImportNames(x)
	if pkgMap == nil {
		pkgMap = make(map[string]string)
	}

	// add default and necessary imports
	pkgMap = MergePkgMaps(pkgMap, g.pkgUsed)

	// remove self from importing
	delete(pkgMap, shape.ToGoPkgName(x))
	return pkgMap
}

func (g *SerdeJSONUnion) VariantName(x shape.Shape) string {
	return TemplateHelperShapeVariantToName(x)
}

func (g *SerdeJSONUnion) JSONVariantName(x shape.Shape) string {
	return shape.MatchShapeR1(
		x,
		func(y *shape.Any) string {
			panic(fmt.Errorf("generators.JSONVariantName: %T not suported", y))
		},
		func(y *shape.RefName) string {
			return fmt.Sprintf("%s.%s", y.PkgName, y.Name)
		},
		func(x *shape.PointerLike) string {
			return g.JSONVariantName(x.Type)
		},
		func(y *shape.AliasLike) string {
			return fmt.Sprintf("%s.%s", y.PkgName, y.Name)
		},
		func(y *shape.PrimitiveLike) string {
			panic(fmt.Errorf("generators.JSONVariantName: must be named %T", y))
		},
		func(y *shape.ListLike) string {
			panic(fmt.Errorf("generators.JSONVariantName: must be named %T", y))
		},
		func(y *shape.MapLike) string {
			panic(fmt.Errorf("generators.JSONVariantName: must be named %T", y))
		},
		func(y *shape.StructLike) string {
			return fmt.Sprintf("%s.%s", y.PkgName, y.Name)
		},
		func(y *shape.UnionLike) string {
			return fmt.Sprintf("%s.%s", y.PkgName, y.Name)
		},
	)
}

func (g *SerdeJSONUnion) Generate() ([]byte, error) {
	body := &bytes.Buffer{}

	// generate union type
	unionType, err := g.GenerateUnionType(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating union type; %w", err)
	}
	body.Write(unionType)

	fromFunc, err := g.GenerateUnionFromFunc(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating from func; %w", err)
	}
	body.Write(fromFunc)

	toFunc, err := g.GenerateUnionToFunc(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating to func; %w", err)
	}
	body.Write(toFunc)

	variantsFromFunc, err := g.GenerateVariantsFromToFunc(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating funcs for variant; %w", err)
	}
	body.Write(variantsFromFunc)

	head := &bytes.Buffer{}
	if !g.skipImportsAndPackage {
		head.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.union)))

		pkgMap := g.ExtractImports(g.union)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return nil, fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating imports %w", err)
		}
		head.WriteString(impPart)
	}

	if !g.skipInitFunc {
		inits := g.ExtractImportFuncs(g.union)
		varPart, err := g.GenerateInitFunc(inits)
		if err != nil {
			return nil, fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating func init(); %w", err)
		}
		head.WriteString(varPart)
	}

	if head.Len() > 0 {
		head.WriteString(body.String())
		return head.Bytes(), nil
	} else {
		return body.Bytes(), nil
	}
}

func (g *SerdeJSONUnion) Serde(x shape.Shape) string {
	serde := NewSerdeJSONTagged(x)
	serde.SkipImportsAndPackage(true)
	result, err := serde.Generate()
	if err != nil {
		panic(err)
	}

	g.pkgUsed = MergePkgMaps(g.pkgUsed, serde.ExtractImports(x))

	return result
}

func (g *SerdeJSONUnion) MatchFuncName(x *shape.UnionLike, returns int) string {
	return MatchUnionFuncName(x, returns)
}

func (g *SerdeJSONUnion) GenerateInitFunc(init []string) (string, error) {
	return GenerateInitFunc(init), nil
}

func (g *SerdeJSONUnion) ExtractImportFuncs(s shape.Shape) []string {
	result := []string{
		StrRegisterUnionFuncName(g.union.PkgName, s),
	}

	switch x := s.(type) {
	case *shape.UnionLike:
		for _, variant := range x.Variant {
			result = append(result, g.ExtractImportFuncs(variant)...)
		}
	}

	return result
}

func StrRegisterUnionFuncName(rootPkgName string, x shape.Shape) string {
	return fmt.Sprintf("shared.JSONMarshallerRegister(%q, %s, %s)",
		shape.ToGoTypeName(x, shape.WithPkgImportName(), shape.WithInstantiation()),
		StrFuncNameFromJSONInstantiated(rootPkgName, x),
		StrFuncNameToJSONInstantiated(rootPkgName, x),
	)
}

func (g *SerdeJSONUnion) FuncNameFromJSONInstantiated(x shape.Shape) string {
	return StrFuncNameFromJSONInstantiated(g.union.PkgName, x)
}

func (g *SerdeJSONUnion) FuncNameToJSONInstantiated(x shape.Shape) string {
	return StrFuncNameToJSONInstantiated(g.union.PkgName, x)
}

func (g *SerdeJSONUnion) instantiatef(x shape.Shape, template string) string {
	return StrInstantiatef(g.union.PkgName, x, template)
}

func StrFuncNameFromJSONInstantiated(rootPkgName string, x shape.Shape) string {
	return StrInstantiatef(rootPkgName, x, "%sFromJSON")
}

func StrFuncNameToJSONInstantiated(rootPkgName string, x shape.Shape) string {
	return StrInstantiatef(rootPkgName, x, "%sToJSON")
}

func StrInstantiatef(pkgName string, x shape.Shape, template string) string {
	typeParamTypes := shape.ToGoTypeParamsTypes(x)
	typeName := fmt.Sprintf(template, shape.Name(x))
	if len(typeParamTypes) == 0 {
		return typeName
	}

	instantiatedNames := make([]string, len(typeParamTypes))
	for i, t := range typeParamTypes {
		instantiatedNames[i] = shape.ToGoTypeName(t,
			shape.WithRootPackage(pkgName),
			shape.WithInstantiation(),
		)
	}

	return fmt.Sprintf("%s[%s]",
		typeName,
		strings.Join(instantiatedNames, ","),
	)
}

func (g *SerdeJSONUnion) FuncNameFromJSON(x shape.Shape) string {
	return g.parametrisedf(x, "%sFromJSON")
}

func (g *SerdeJSONUnion) FuncNameToSON(x shape.Shape) string {
	return g.parametrisedf(x, "%sToJSON")
}

func (g *SerdeJSONUnion) parametrisedf(x shape.Shape, template string) string {
	typeParamNames := shape.ToGoTypeParamsNames(x)
	typeName := fmt.Sprintf(template, shape.Name(x))
	if len(typeParamNames) == 0 {
		return typeName
	}

	return fmt.Sprintf("%s[%s]",
		typeName,
		strings.Join(typeParamNames, ","),
	)
}
func (g *SerdeJSONUnion) constructionf(x shape.Shape, template string) string {
	typeParams := shape.ExtractTypeParams(x)
	typeName := fmt.Sprintf(template, shape.Name(x))
	if len(typeParams) == 0 {
		return typeName
	}

	typeName += "["
	for i, t := range typeParams {
		if i > 0 {
			typeName += ", "
		}
		paramType := shape.ToGoTypeName(t.Type,
			shape.WithRootPackage(g.union.PkgName),
		)
		typeName += fmt.Sprintf("%s %s", t.Name, paramType)
	}
	typeName += "]"

	return typeName
}

func (g *SerdeJSONUnion) GenerateUnionType(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	body.WriteString(fmt.Sprintf("type %s struct {\n", g.constructionf(union, "%sUnionJSON")))
	body.WriteString(fmt.Sprintf("\tType string `json:\"$type,omitempty\"`\n"))
	for _, variant := range union.Variant {
		body.WriteString(fmt.Sprintf("\t%s json.RawMessage `json:\"%s,omitempty\"`\n",
			g.VariantName(variant),
			g.JSONVariantName(variant),
		))
	}
	body.WriteString(fmt.Sprintf("}\n\n"))

	return body.Bytes(), nil
}

func (g *SerdeJSONUnion) errorFuncContext(name string) string {
	return fmt.Sprintf(`%s.%s:`, g.union.PkgName, name)
}

func (g *SerdeJSONUnion) GenerateUnionFromFunc(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	errorContext := g.errorFuncContext(g.parametrisedf(union, "%sFromJSON"))

	body.WriteString(fmt.Sprintf("func %s(x []byte) (%s, error) {\n",
		g.constructionf(union, "%sFromJSON"),
		g.parametrisedf(union, "%s"),
	))
	body.WriteString(fmt.Sprintf("\tif x == nil || len(x) == 0 {\n"))
	body.WriteString(fmt.Sprintf("\t\treturn nil, nil\n"))
	body.WriteString(fmt.Sprintf("\t}\n"))
	body.WriteString(fmt.Sprintf("\tif string(x[:4]) == \"null\" {\n"))
	body.WriteString(fmt.Sprintf("\t\treturn nil, nil\n"))
	body.WriteString(fmt.Sprintf("\t}\n"))
	body.WriteString(fmt.Sprintf("\tvar data %s\n", g.parametrisedf(union, "%sUnionJSON")))
	body.WriteString(fmt.Sprintf("\terr := json.Unmarshal(x, &data)\n"))
	body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
	body.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s %%w\", err)\n", errorContext))
	body.WriteString(fmt.Sprintf("\t}\n\n"))
	body.WriteString(fmt.Sprintf("\tswitch data.Type {\n"))
	for _, variant := range union.Variant {
		body.WriteString(fmt.Sprintf("\tcase %q:\n", g.JSONVariantName(variant)))
		body.WriteString(fmt.Sprintf("\t\treturn %s(data.%s)\n",
			g.FuncNameFromJSON(variant),
			g.VariantName(variant),
		))
	}
	body.WriteString(fmt.Sprintf("\t}\n\n"))

	for i, variant := range union.Variant {
		if i > 0 {
			body.WriteString(fmt.Sprintf(" else "))
		} else {
			body.WriteString(fmt.Sprintf("\t"))
		}

		body.WriteString(fmt.Sprintf("if data.%s != nil {\n", g.VariantName(variant)))
		body.WriteString(fmt.Sprintf("\t\treturn %s(data.%s)\n",
			g.FuncNameFromJSON(variant),
			g.VariantName(variant),
		))
		body.WriteString(fmt.Sprintf("\t}"))
	}
	body.WriteString(fmt.Sprintf("\n"))

	body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s unknown type: %%s\", data.Type)\n", errorContext))
	body.WriteString(fmt.Sprintf("}\n\n"))

	return body.Bytes(), nil
}

func (g *SerdeJSONUnion) GenerateUnionToFunc(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	errorContext := g.errorFuncContext(g.parametrisedf(union, "%sToJSON"))

	body.WriteString(fmt.Sprintf("func %s(x %s) ([]byte, error) {\n",
		g.constructionf(union, "%sToJSON"),
		g.parametrisedf(union, "%s"),
	))
	body.WriteString(fmt.Sprintf("\tif x == nil {\n"))
	body.WriteString(fmt.Sprintf("\t\treturn []byte(`null`), nil\n"))
	body.WriteString(fmt.Sprintf("\t}\n"))

	body.WriteString(fmt.Sprintf("\treturn %s(\n", MatchUnionFuncName(union, 2)))
	body.WriteString(fmt.Sprintf("\t\tx,\n"))

	for _, variant := range union.Variant {
		body.WriteString(fmt.Sprintf("\t\tfunc (y *%s) ([]byte, error) {\n", g.parametrisedf(variant, "%s")))
		body.WriteString(fmt.Sprintf("\t\t\tbody, err := %s(y)\n", g.FuncNameToSON(variant)))
		body.WriteString(fmt.Sprintf("\t\t\tif err != nil {\n"))
		body.WriteString(fmt.Sprintf("\t\t\t\treturn nil, fmt.Errorf(\"%s %%w\", err)\n", errorContext))
		body.WriteString(fmt.Sprintf("\t\t\t}\n"))
		body.WriteString(fmt.Sprintf("\t\t\treturn json.Marshal(%s{\n", g.parametrisedf(union, "%sUnionJSON")))
		body.WriteString(fmt.Sprintf("\t\t\t\tType: %q,\n", g.JSONVariantName(variant)))
		body.WriteString(fmt.Sprintf("\t\t\t\t%s: body,\n", g.VariantName(variant)))
		body.WriteString(fmt.Sprintf("\t\t\t})\n"))
		body.WriteString(fmt.Sprintf("\t\t},\n"))
	}
	body.WriteString(fmt.Sprintf("\t)\n"))
	body.WriteString(fmt.Sprintf("}\n\n"))

	return body.Bytes(), nil
}

func (g *SerdeJSONUnion) GenerateVariantsFromToFunc(x *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	for _, variant := range x.Variant {
		errorContext := g.errorFuncContext(g.parametrisedf(variant, "%sFromJSON"))

		// from json func
		body.WriteString(fmt.Sprintf("func %s(x []byte) (*%s, error) {\n",
			g.constructionf(variant, "%sFromJSON"),
			g.parametrisedf(variant, "%s"),
		))
		body.WriteString(fmt.Sprintf("\tresult := new(%s)\n", g.parametrisedf(variant, "%s")))
		body.WriteString(fmt.Sprintf("\terr := result.UnmarshalJSON(x)\n"))
		body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
		body.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s %%w\", err)\n", errorContext))
		body.WriteString(fmt.Sprintf("\t}\n"))
		body.WriteString(fmt.Sprintf("\treturn result, nil\n"))
		body.WriteString(fmt.Sprintf("}\n\n"))

		// to json func
		body.WriteString(fmt.Sprintf("func %s(x *%s) ([]byte, error) {\n",
			g.constructionf(variant, "%sToJSON"),
			g.parametrisedf(variant, "%s"),
		))
		body.WriteString(fmt.Sprintf("\treturn x.MarshalJSON()\n"))
		body.WriteString(fmt.Sprintf("}\n\n"))

		body.WriteString(g.Serde(variant))
		body.WriteString("\n")
	}

	return body.Bytes(), nil
}
