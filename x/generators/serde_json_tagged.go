package generators

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
)

const (
	unmarshalJSONMethodPrefix = "_unmarshalJSON"
	marshalJSONMethodPrefix   = "_marshalJSON"
)

func NewSerdeJSONTagged(shape shape.Shape) *SerdeJSONTagged {
	return &SerdeJSONTagged{
		shape:                          shape,
		skipImportsAndPackage:          false,
		didGenerateMarshalJSONMethod:   make(map[string]bool),
		didGenerateUnmarshalJSONMethod: make(map[string]bool),
		pkgUsed: PkgMap{
			"json": "encoding/json",
			"fmt":  "fmt",
		},
	}
}

type SerdeJSONTagged struct {
	shape                 shape.Shape
	skipImportsAndPackage bool

	didGenerateMarshalJSONMethod   map[string]bool
	didGenerateUnmarshalJSONMethod map[string]bool
	pkgUsed                        PkgMap
}

func (g *SerdeJSONTagged) SkipImportsAndPackage(flag bool) *SerdeJSONTagged {
	g.skipImportsAndPackage = flag
	return g
}

func (g *SerdeJSONTagged) Generate() (string, error) {

	body := &strings.Builder{}
	varPart, err := g.GenerateVarCasting(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating variable casting %w", err)
	}
	body.WriteString(varPart)

	marshalPart, err := g.GenerateMarshalJSON(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating marshal %w", err)

	}
	body.WriteString(marshalPart)

	unmarshalPart, err := g.GenerateUnmarshalJSON(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating unmarshal %w", err)
	}
	body.WriteString(unmarshalPart)

	head := &strings.Builder{}
	if !g.skipImportsAndPackage {
		head.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.shape)))

		pkgMap := g.ExtractImports(g.shape)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return "", fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating imports %w", err)
		}
		head.WriteString(impPart)
	}

	if head.Len() > 0 {
		head.WriteString(body.String())
		return head.String(), nil
	} else {
		return body.String(), nil
	}
}

func (g *SerdeJSONTagged) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *SerdeJSONTagged) ExtractImports(x shape.Shape) PkgMap {
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

func (g *SerdeJSONTagged) GenerateVarCasting(x shape.Shape) (string, error) {
	return shape.MustMatchShapeR2(
		x,
		func(x *shape.Any) (string, error) {
			panic("not implemented any var casting")

		},
		func(x *shape.RefName) (string, error) {
			panic("not implemented ref var casting")

		},
		func(x *shape.AliasLike) (string, error) {
			result := &strings.Builder{}
			result.WriteString("var (\n")
			result.WriteString("\t_ json.Unmarshaler = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString("\t_ json.Marshaler   = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString(")\n\n")

			return result.String(), nil

		},
		func(x *shape.BooleanLike) (string, error) {
			panic("not implemented boolean var casting")

		},
		func(x *shape.StringLike) (string, error) {
			panic("not implemented string var casting")

		},
		func(x *shape.NumberLike) (string, error) {
			panic("not implemented number var casting")

		},
		func(x *shape.ListLike) (string, error) {
			panic("not implemented list var casting")

		},
		func(x *shape.MapLike) (string, error) {
			panic("not implemented map var casting")
		},
		func(x *shape.StructLike) (string, error) {
			result := &strings.Builder{}
			result.WriteString("var (\n")
			result.WriteString("\t_ json.Unmarshaler = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString("\t_ json.Marshaler   = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString(")\n\n")

			return result.String(), nil
		},
		func(x *shape.UnionLike) (string, error) {
			panic("not implemented union var casting")
		},
	)
}

func (g *SerdeJSONTagged) GenerateMarshalJSON(x shape.Shape) (string, error) {
	result := &strings.Builder{}
	result.WriteString(fmt.Sprintf("func (r *%s) MarshalJSON() ([]byte, error) {\n", g.rootTypeName()))
	result.WriteString(fmt.Sprintf("\tif r == nil {\n"))
	result.WriteString(fmt.Sprintf("\t\treturn nil, nil\n"))
	result.WriteString(fmt.Sprintf("\t}\n"))
	result.WriteString(fmt.Sprintf("\treturn r.%s(*r)\n", g.methodNameWithPrefix(x, marshalJSONMethodPrefix)))
	result.WriteString("}\n")

	methods, err := g.GenerateMarshalJSONMethods(x)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSON: %w", err)
	}
	result.WriteString(methods)

	return result.String(), nil
}

var removeNonAlpha = strings.NewReplacer(
	" ", "",
	".", "_",
	"*", "Ptr",
	"[]", "Slice",
	"[", "Lb_",
	"]", "_bL",
	"(", "Lp_",
	")", "_pL",
	",", "Comma",
	"=", "Equal",
	"~", "Tilda",
)

// g.toGoAlphaName return name of type that could be use as part of method or function name
func (g *SerdeJSONTagged) toGoAlphaName(x shape.Shape) string {
	typeName := shape.ToGoTypeName(x,
		shape.WithRootPackage(shape.ToGoPkgName(g.shape)),
	)

	return removeNonAlpha.Replace(typeName)
}

func (g *SerdeJSONTagged) rootPkgName() string {
	return shape.ToGoPkgName(g.shape)
}

func (g *SerdeJSONTagged) rootTypeName() string {
	return shape.ToGoTypeName(g.shape,
		shape.WithRootPackage(shape.ToGoPkgName(g.shape)),
	)
}

func (g *SerdeJSONTagged) errorContext(name string) string {
	return fmt.Sprintf(`%s: %s.%s:`, g.rootPkgName(), g.rootTypeName(), name)
}

func (g *SerdeJSONTagged) methodNameWithPrefix(x shape.Shape, prefix string) string {
	name := fmt.Sprintf("%s%s", prefix, g.toGoAlphaName(x))
	return name
}

func (g *SerdeJSONTagged) GenerateMarshalJSONMethods(x shape.Shape) (string, error) {
	// prevent infinite recursion
	methodName := g.methodNameWithPrefix(x, marshalJSONMethodPrefix)
	if g.didGenerateMarshalJSONMethod[methodName] {
		return "", nil
	} else {
		g.didGenerateMarshalJSONMethod[methodName] = true
	}

	rootTypeName := g.rootTypeName()
	typeName := shape.ToGoTypeName(x, shape.WithRootPackage(shape.ToGoPkgName(g.shape)))
	errorContext := g.errorContext(methodName)

	methodWrap := func(body *strings.Builder) (string, error) {
		result := &strings.Builder{}
		result.WriteString(fmt.Sprintf("func (r *%s) %s(x %s) ([]byte, error) {\n", rootTypeName, methodName, typeName))
		result.WriteString(padLeftTabs(1, body.String()))
		result.WriteString("}\n")
		return result.String(), nil
	}

	return shape.MustMatchShapeR2(
		x,
		func(y *shape.Any) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("result, err := json.Marshal(x)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.RefName) (string, error) {
			g.pkgUsed["shared"] = "github.com/widmogrod/mkunion/x/shared"

			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("result, err := shared.JSONMarshal[%s](x)\n", typeName))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.AliasLike) (string, error) {
			aliasTypeName := shape.ToGoTypeName(y.Type, shape.WithRootPackage(shape.ToGoPkgName(g.shape)))

			if y.IsAlias {
				g.pkgUsed["shared"] = "github.com/widmogrod/mkunion/x/shared"

				body := &strings.Builder{}
				body.WriteString(fmt.Sprintf("result, err := shared.JSONMarshal[%s](x)\n", aliasTypeName))
				body.WriteString(fmt.Sprintf("if err != nil {\n"))
				body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s; %%w\", err)\n", errorContext))
				body.WriteString(fmt.Sprintf("}\n"))
				body.WriteString(fmt.Sprintf("return result, nil\n"))
				return methodWrap(body)
			}

			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("return r.%s(%s(x))\n",
				g.methodNameWithPrefix(y.Type, marshalJSONMethodPrefix),
				aliasTypeName,
			))

			result, err := methodWrap(body)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: alias wrapping; %w", err)
			}

			methods, err := g.GenerateMarshalJSONMethods(y.Type)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: alias methods; %w", err)
			}

			return result + methods, nil
		},
		func(y *shape.BooleanLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("result, err := json.Marshal(x)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.StringLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("result, err := json.Marshal(x)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.NumberLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("result, err := json.Marshal(x)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.ListLike) (string, error) {
			body := &strings.Builder{}

			if shape.IsBinary(y) {
				body.WriteString(fmt.Sprintf("result, err := json.Marshal(x)\n"))
				body.WriteString(fmt.Sprintf("if err != nil {\n"))
				body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s; %%w\", err)\n", errorContext))
				body.WriteString(fmt.Sprintf("}\n"))
				body.WriteString(fmt.Sprintf("return result, nil\n"))
				return methodWrap(body)
			}

			body.WriteString(fmt.Sprintf("partial := make([]json.RawMessage, len(x))\n"))
			body.WriteString(fmt.Sprintf("for i, v := range x {\n"))
			body.WriteString(fmt.Sprintf("\titem, err := r.%s(v)\n", g.methodNameWithPrefix(y.Element, marshalJSONMethodPrefix)))
			body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			body.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s at index %%d; %%w\", i, err)\n", errorContext))
			body.WriteString(fmt.Sprintf("\t}\n"))
			body.WriteString(fmt.Sprintf("\tpartial[i] = item\n"))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("result, err := json.Marshal(partial)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))

			result, err := methodWrap(body)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: list wrapping; %w", err)
			}

			methods, err := g.GenerateMarshalJSONMethods(y.Element)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: list methods; %w", err)
			}

			return result + methods, nil
		},
		func(y *shape.MapLike) (string, error) {
			keyTypeName := shape.ToGoTypeName(y.Key, shape.WithRootPackage(shape.ToGoPkgName(g.shape)))
			isKeyString := shape.IsString(y.Key) || shape.IsBinary(y.Key)

			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("partial := make(map[string]json.RawMessage)\n"))

			if !isKeyString {
				body.WriteString(fmt.Sprintf("var err error\n"))
				body.WriteString(fmt.Sprintf("var keyType %s\n", keyTypeName))
				body.WriteString(fmt.Sprintf("_, isString := any(keyType).(string)\n"))
			}

			body.WriteString(fmt.Sprintf("for k, v := range x {\n"))
			if isKeyString {
				body.WriteString(fmt.Sprintf("\tkey := string(k)\n"))
			} else {
				body.WriteString(fmt.Sprintf("\tvar key []byte\n"))
				body.WriteString(fmt.Sprintf("\tif isString {\n"))
				body.WriteString(fmt.Sprintf("\t\tkey = []byte(any(k).(string))\n"))
				body.WriteString(fmt.Sprintf("\t} else {\n"))
				body.WriteString(fmt.Sprintf("\t\tkey, err = r.%s(k)\n", g.methodNameWithPrefix(y.Key, marshalJSONMethodPrefix)))
				body.WriteString(fmt.Sprintf("\t\tif err != nil {\n"))
				body.WriteString(fmt.Sprintf("\t\t\treturn nil, fmt.Errorf(\"%s key; %%w\", err)\n", errorContext))
				body.WriteString(fmt.Sprintf("\t\t}\n"))
				body.WriteString(fmt.Sprintf("\t}\n"))
			}
			body.WriteString(fmt.Sprintf("\tvalue, err := r.%s(v)\n", g.methodNameWithPrefix(y.Val, marshalJSONMethodPrefix)))
			body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			body.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s value; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("\t}\n"))
			body.WriteString(fmt.Sprintf("\tpartial[string(key)] = value\n"))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("result, err := json.Marshal(partial)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))

			result, err := methodWrap(body)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: map wrapping; %w", err)
			}

			keyMethods, err := g.GenerateMarshalJSONMethods(y.Key)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: key methods; %w", err)
			}

			valMethods, err := g.GenerateMarshalJSONMethods(y.Val)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: value methods; %w", err)
			}

			return result + keyMethods + valMethods, nil
		},
		func(y *shape.StructLike) (string, error) {
			body := &strings.Builder{}

			if y.IsPointer {
				body.WriteString(fmt.Sprintf("if x == nil {\n"))
				body.WriteString(fmt.Sprintf("\treturn nil, nil\n"))
				body.WriteString(fmt.Sprintf("}\n"))
			}

			body.WriteString(fmt.Sprintf("partial := make(map[string]json.RawMessage)\n"))
			body.WriteString(fmt.Sprintf("var err error\n"))
			for _, field := range y.Fields {
				jsonFieldName := shape.TagGetValue(field.Tags, "json", field.Name)

				if field.IsPointer {
					body.WriteString(fmt.Sprintf("if x.%s != nil {\n", field.Name))
					body.WriteString(fmt.Sprintf("\tvar field%s []byte\n", field.Name))
					body.WriteString(fmt.Sprintf("\tfield%s, err = r.%s(x.%s)\n", field.Name, g.methodNameWithPrefix(field.Type, marshalJSONMethodPrefix), field.Name))
					body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
					body.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s field name %s; %%w\", err)\n", errorContext, field.Name))
					body.WriteString(fmt.Sprintf("\t}\n"))
					body.WriteString(fmt.Sprintf("\tpartial[\"%s\"] = field%s\n", jsonFieldName, field.Name))
					body.WriteString(fmt.Sprintf("}\n"))
				} else {
					body.WriteString(fmt.Sprintf("var field%s []byte\n", field.Name))
					body.WriteString(fmt.Sprintf("field%s, err = r.%s(x.%s)\n", field.Name, g.methodNameWithPrefix(field.Type, marshalJSONMethodPrefix), field.Name))
					body.WriteString(fmt.Sprintf("if err != nil {\n"))
					body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s field name %s; %%w\", err)\n", errorContext, field.Name))
					body.WriteString(fmt.Sprintf("}\n"))
					body.WriteString(fmt.Sprintf("partial[\"%s\"] = field%s\n", jsonFieldName, field.Name))
				}
			}
			body.WriteString(fmt.Sprintf("result, err := json.Marshal(partial)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s struct; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))

			result, err := methodWrap(body)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: struct wrapping; %w", err)
			}

			methods := ""
			for _, field := range y.Fields {
				fieldMethods, err := g.GenerateMarshalJSONMethods(field.Type)
				if err != nil {
					return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: field %s methods; %w", field.Name, err)
				}
				methods += fieldMethods
			}

			return result + methods, nil
		},
		func(y *shape.UnionLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("return %sToJSON(x)\n", typeName))
			return methodWrap(body)
		},
	)
}

func (g *SerdeJSONTagged) GenerateUnmarshalJSON(x shape.Shape) (string, error) {
	result := &strings.Builder{}
	result.WriteString(fmt.Sprintf("func (r *%s) UnmarshalJSON(data []byte) error {\n", g.rootTypeName()))
	result.WriteString(fmt.Sprintf("\tresult, err := r.%s(data)\n", g.methodNameWithPrefix(x, unmarshalJSONMethodPrefix)))
	result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
	result.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"%s %%w\", err)\n", g.errorContext("UnmarshalJSON")))
	result.WriteString(fmt.Sprintf("\t}\n"))
	result.WriteString(fmt.Sprintf("\t*r = result\n"))
	result.WriteString(fmt.Sprintf("\treturn nil\n"))
	result.WriteString(fmt.Sprintf("}\n"))

	methods, err := g.GenerateUnmarshalJSONMethods(x)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSON: %w", err)
	}
	result.WriteString(methods)

	return result.String(), nil
}

func (g *SerdeJSONTagged) GenerateUnmarshalJSONMethods(x shape.Shape) (string, error) {
	// prevent infinite recursion
	methodName := g.methodNameWithPrefix(x, unmarshalJSONMethodPrefix)
	if g.didGenerateUnmarshalJSONMethod[methodName] {
		return "", nil
	} else {
		g.didGenerateUnmarshalJSONMethod[methodName] = true
	}

	rootTypeName := g.rootTypeName()
	typeName := shape.ToGoTypeName(x, shape.WithRootPackage(shape.ToGoPkgName(g.shape)))
	errorContext := g.errorContext(methodName)

	methodWrap := func(body *strings.Builder) (string, error) {
		result := &strings.Builder{}
		result.WriteString(fmt.Sprintf("func (r *%s) %s(data []byte) (%s, error) {\n", rootTypeName, methodName, typeName))
		result.WriteString(padLeftTabs(1, body.String()))
		result.WriteString("}\n")
		return result.String(), nil
	}

	return shape.MustMatchShapeR2(
		x,
		func(y *shape.Any) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("var result %s\n", typeName))
			body.WriteString(fmt.Sprintf("err := json.Unmarshal(data, &result)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native any unwrap; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.RefName) (string, error) {
			g.pkgUsed["shared"] = "github.com/widmogrod/mkunion/x/shared"

			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("result, err := shared.JSONUnmarshal[%s](data)\n", typeName))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native ref unwrap; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.AliasLike) (string, error) {
			aliasTypeName := shape.ToGoTypeName(y.Type, shape.WithRootPackage(shape.ToGoPkgName(g.shape)))

			if y.IsAlias {
				g.pkgUsed["shared"] = "github.com/widmogrod/mkunion/x/shared"

				body := &strings.Builder{}
				body.WriteString(fmt.Sprintf("result, err := shared.JSONUnmarshal[%s](data)\n", aliasTypeName))
				body.WriteString(fmt.Sprintf("if err != nil {\n"))
				body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native ref unwrap; %%w\", err)\n", errorContext))
				body.WriteString(fmt.Sprintf("}\n"))
				body.WriteString(fmt.Sprintf("return result, nil\n"))
				return methodWrap(body)
			}

			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("var result %s\n", typeName))
			body.WriteString(fmt.Sprintf("intermidiary, err := r.%s(data)\n", g.methodNameWithPrefix(y.Type, unmarshalJSONMethodPrefix)))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s alias; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("result = %s(intermidiary)\n", typeName))
			body.WriteString(fmt.Sprintf("return result, nil\n"))

			result, err := methodWrap(body)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: alias wrapping; %w", err)
			}

			methods, err := g.GenerateUnmarshalJSONMethods(y.Type)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: alias methods; %w", err)
			}

			return result + methods, nil
		},
		func(y *shape.BooleanLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("var result %s\n", typeName))
			body.WriteString(fmt.Sprintf("err := json.Unmarshal(data, &result)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native boolean unwrap; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.StringLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("var result %s\n", typeName))
			body.WriteString(fmt.Sprintf("err := json.Unmarshal(data, &result)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native string unwrap; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.NumberLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("var result %s\n", typeName))
			body.WriteString(fmt.Sprintf("err := json.Unmarshal(data, &result)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native number unwrap; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))
			return methodWrap(body)
		},
		func(y *shape.ListLike) (string, error) {
			body := &strings.Builder{}

			if shape.IsBinary(y) {
				body.WriteString(fmt.Sprintf("var result %s\n", typeName))
				body.WriteString(fmt.Sprintf("err := json.Unmarshal(data, &result)\n"))
				body.WriteString(fmt.Sprintf("if err != nil {\n"))
				body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native list unwrap; %%w\", err)\n", errorContext))
				body.WriteString(fmt.Sprintf("}\n"))
				body.WriteString(fmt.Sprintf("return result, nil\n"))
				return methodWrap(body)
			}

			if y.ArrayLen != nil {
				body.WriteString(fmt.Sprintf("result := %s{}\n", typeName))
			} else {
				body.WriteString(fmt.Sprintf("result := make(%s, 0)\n", typeName))
			}

			body.WriteString(fmt.Sprintf("var partial []json.RawMessage\n"))
			body.WriteString(fmt.Sprintf("err := json.Unmarshal(data, &partial)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native list unwrap; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))

			body.WriteString(fmt.Sprintf("for i, v := range partial {\n"))
			body.WriteString(fmt.Sprintf("\titem, err := r.%s(v)\n", g.methodNameWithPrefix(y.Element, unmarshalJSONMethodPrefix)))
			body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			body.WriteString(fmt.Sprintf("\t\treturn result, fmt.Errorf(\"%s at index %%d; %%w\", i, err)\n", errorContext))
			body.WriteString(fmt.Sprintf("\t}\n"))

			if y.ArrayLen != nil {
				body.WriteString(fmt.Sprintf("\tresult[i] = item\n"))
			} else {
				body.WriteString(fmt.Sprintf("\tresult = append(result, item)\n"))
			}

			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))

			result, err := methodWrap(body)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: list wrapping; %w", err)
			}

			methods, err := g.GenerateUnmarshalJSONMethods(y.Element)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: list methods; %w", err)
			}

			return result + methods, nil
		},
		func(y *shape.MapLike) (string, error) {
			keyTypeName := shape.ToGoTypeName(y.Key, shape.WithRootPackage(shape.ToGoPkgName(g.shape)))
			isKeyString := shape.IsString(y.Key) || shape.IsBinary(y.Key)

			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("var partial map[string]json.RawMessage\n"))
			body.WriteString(fmt.Sprintf("err := json.Unmarshal(data, &partial)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s native map unwrap; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("result := make(%s)\n", typeName))

			if !isKeyString {
				body.WriteString(fmt.Sprintf("var keyType %s\n", keyTypeName))
				body.WriteString(fmt.Sprintf("_, isString := any(keyType).(string)\n"))
			}

			body.WriteString(fmt.Sprintf("for k, v := range partial {\n"))
			if isKeyString {
				body.WriteString(fmt.Sprintf("\tkey := string(k)\n"))
			} else {
				body.WriteString(fmt.Sprintf("\tvar key %s\n", keyTypeName))
				body.WriteString(fmt.Sprintf("\tif isString {\n"))
				body.WriteString(fmt.Sprintf("\t\tkey = any(k).(%s)\n", keyTypeName))
				body.WriteString(fmt.Sprintf("\t} else {\n"))
				body.WriteString(fmt.Sprintf("\t\tkey, err = r.%s([]byte(k))\n", g.methodNameWithPrefix(y.Key, unmarshalJSONMethodPrefix)))
				body.WriteString(fmt.Sprintf("\t\tif err != nil {\n"))
				body.WriteString(fmt.Sprintf("\t\t\treturn nil, fmt.Errorf(\"%s key; %%w\", err)\n", errorContext))
				body.WriteString(fmt.Sprintf("\t\t}\n"))
				body.WriteString(fmt.Sprintf("\t}\n"))
			}
			body.WriteString(fmt.Sprintf("\tvalue, err := r.%s(v)\n", g.methodNameWithPrefix(y.Val, unmarshalJSONMethodPrefix)))
			body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			body.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s value; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("\t}\n"))
			body.WriteString(fmt.Sprintf("\tresult[key] = value\n"))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return result, nil\n"))

			result, err := methodWrap(body)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: map wrapping; %w", err)
			}

			keyMethods, err := g.GenerateUnmarshalJSONMethods(y.Key)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: key methods; %w", err)
			}

			valMethods, err := g.GenerateUnmarshalJSONMethods(y.Val)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: value methods; %w", err)
			}

			return result + keyMethods + valMethods, nil
		},
		func(y *shape.StructLike) (string, error) {
			body := &strings.Builder{}
			if shape.IsPointer(y) {
				body.WriteString(fmt.Sprintf("if len(data) == 0 {\n"))
				body.WriteString(fmt.Sprintf("\treturn nil, nil\n"))
				body.WriteString(fmt.Sprintf("}\n"))
				body.WriteString(fmt.Sprintf("if string(data[:4]) == \"null\" {\n"))
				body.WriteString(fmt.Sprintf("\treturn nil, nil\n"))
				body.WriteString(fmt.Sprintf("}\n"))

				body.WriteString(fmt.Sprintf("result := new(%s)\n", shape.UnwrapPointer(typeName)))
			} else {
				body.WriteString(fmt.Sprintf("result := %s{}\n", typeName))
			}
			body.WriteString(fmt.Sprintf("var partial map[string]json.RawMessage\n"))
			body.WriteString(fmt.Sprintf("err := json.Unmarshal(data, &partial)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native struct unwrap; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			for _, field := range y.Fields {
				jsonFieldName := shape.TagGetValue(field.Tags, "json", field.Name)

				body.WriteString(fmt.Sprintf("if field%s, ok := partial[\"%s\"]; ok {\n", field.Name, jsonFieldName))
				body.WriteString(fmt.Sprintf("\tresult.%s, err = r.%s(field%s)\n", field.Name, g.methodNameWithPrefix(field.Type, unmarshalJSONMethodPrefix), field.Name))
				body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
				body.WriteString(fmt.Sprintf("\t\treturn result, fmt.Errorf(\"%s field %s; %%w\", err)\n", errorContext, field.Name))
				body.WriteString(fmt.Sprintf("\t}\n"))
				body.WriteString(fmt.Sprintf("}\n"))
			}

			body.WriteString(fmt.Sprintf("return result, nil\n"))

			result, err := methodWrap(body)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: struct wrapping; %w", err)
			}

			methods := ""
			for _, field := range y.Fields {
				fieldMethods, err := g.GenerateUnmarshalJSONMethods(field.Type)
				if err != nil {
					return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: field %s methods; %w", field.Name, err)
				}
				methods += fieldMethods
			}

			return result + methods, nil
		},
		func(y *shape.UnionLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("return %sFromJSON(data)\n", typeName))
			return methodWrap(body)
		},
	)
}
