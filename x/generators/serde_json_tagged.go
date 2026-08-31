package generators

import (
	"encoding/json"
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

	if !shape.IsWeekAlias(g.shape) {
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
	}

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
	return shape.MatchShapeR2(
		x,
		func(x *shape.Any) (string, error) {
			panic("not implemented any var casting")

		},
		func(x *shape.RefName) (string, error) {
			panic("not implemented ref var casting")

		},
		func(x *shape.PointerLike) (string, error) {
			panic("not implemented pointer var casting")
		},
		func(x *shape.AliasLike) (string, error) {
			result := &strings.Builder{}
			result.WriteString("var (\n")
			result.WriteString("\t_ json.Unmarshaler = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPkgName(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString("\t_ json.Marshaler   = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPkgName(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString(")\n\n")

			return result.String(), nil

		},
		func(x *shape.PrimitiveLike) (string, error) {
			panic("not implemented primitive var casting")
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
				shape.WithRootPkgName(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString("\t_ json.Marshaler   = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPkgName(shape.ToGoPkgName(x)),
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

// fieldJSONInfo interprets the field's json tag the same way encoding/json does:
// json:"-" skips the field, json:"name" renames it, and the omitempty option
// drops the field when its value is empty.
func fieldJSONInfo(field *shape.FieldLike) (name string, skip bool, omitEmpty bool) {
	name = shape.TagGetValue(field.Tags, "json", field.Name)
	if name == "-" {
		return "", true, false
	}
	return name, false, shape.TagHasOption(field.Tags, "json", "omitempty")
}

// omitEmptyCond returns a Go expression that is true when the field value is
// non-empty in the encoding/json omitempty sense. It returns "" when the type
// has no notion of emptiness (structs and unresolved references are always
// emitted, matching encoding/json behaviour for structs).
func omitEmptyCond(expr string, t shape.Shape) string {
	return shape.MatchShapeR1(
		t,
		func(*shape.Any) string { return expr + " != nil" },
		func(*shape.RefName) string { return "" },
		func(*shape.PointerLike) string { return expr + " != nil" },
		func(y *shape.AliasLike) string { return omitEmptyCond(expr, y.Type) },
		func(y *shape.PrimitiveLike) string {
			return shape.MatchPrimitiveKindR1(
				y.Kind,
				func(*shape.BooleanLike) string { return expr + " != false" },
				func(*shape.StringLike) string { return expr + ` != ""` },
				func(*shape.NumberLike) string { return expr + " != 0" },
			)
		},
		func(*shape.ListLike) string { return "len(" + expr + ") != 0" },
		func(*shape.MapLike) string { return "len(" + expr + ") != 0" },
		func(*shape.StructLike) string { return "" },
		func(*shape.UnionLike) string { return expr + " != nil" },
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
		shape.WithRootPkgName(shape.ToGoPkgName(g.shape)),
	)

	return removeNonAlpha.Replace(typeName)
}

func (g *SerdeJSONTagged) rootPkgName() string {
	return shape.ToGoPkgName(g.shape)
}

func (g *SerdeJSONTagged) rootTypeName() string {
	return shape.ToGoTypeName(g.shape,
		shape.WithRootPkgName(shape.ToGoPkgName(g.shape)),
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

	if shape.IsWeekAlias(x) {
		return "", nil
	}

	rootTypeName := g.rootTypeName()
	typeName := shape.ToGoTypeName(x, shape.WithRootPkgName(shape.ToGoPkgName(g.shape)))
	errorContext := g.errorContext(methodName)

	methodWrap := func(body *strings.Builder) (string, error) {
		result := &strings.Builder{}
		result.WriteString(fmt.Sprintf("func (r *%s) %s(x %s) ([]byte, error) {\n", rootTypeName, methodName, typeName))
		result.WriteString(padLeftTabs(1, body.String()))
		result.WriteString("}\n")
		return result.String(), nil
	}

	return shape.MatchShapeR2(
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
		func(y *shape.PointerLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("if x == nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, nil\n"))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return r.%s(*x)\n", g.methodNameWithPrefix(y.Type, marshalJSONMethodPrefix)))

			result, _ := methodWrap(body)

			methods, err := g.GenerateMarshalJSONMethods(y.Type)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: alias methods; %w", err)
			}

			return result + methods, nil
		},
		func(y *shape.AliasLike) (string, error) {
			aliasTypeName := shape.ToGoTypeName(y.Type, shape.WithRootPkgName(shape.ToGoPkgName(g.shape)))

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

			result, _ := methodWrap(body)

			methods, err := g.GenerateMarshalJSONMethods(y.Type)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: alias methods; %w", err)
			}

			return result + methods, nil
		},
		func(x *shape.PrimitiveLike) (string, error) {
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

			result, _ := methodWrap(body)

			methods, err := g.GenerateMarshalJSONMethods(y.Element)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: list methods; %w", err)
			}

			return result + methods, nil
		},
		func(y *shape.MapLike) (string, error) {
			keyTypeName := shape.ToGoTypeName(y.Key, shape.WithRootPkgName(shape.ToGoPkgName(g.shape)))
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

			result, _ := methodWrap(body)

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
			var emitted []*shape.FieldLike
			for _, field := range y.Fields {
				_, skip, _ := fieldJSONInfo(field)
				if !skip {
					emitted = append(emitted, field)
				}
			}

			body := &strings.Builder{}

			if len(emitted) == 0 {
				body.WriteString("return []byte(\"{}\"), nil\n")
				return methodWrap(body)
			}

			g.pkgUsed["bytes"] = "bytes"

			body.WriteString(fmt.Sprintf("buf := bytes.Buffer{}\n"))
			body.WriteString(fmt.Sprintf("buf.WriteByte('{')\n"))
			body.WriteString(fmt.Sprintf("var err error\n"))
			for _, field := range emitted {
				jsonFieldName, _, omitEmpty := fieldJSONInfo(field)
				keyJSON, err := json.Marshal(jsonFieldName)
				if err != nil {
					return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateMarshalJSONMethods: field %s json key; %w", field.Name, err)
				}

				inner := &strings.Builder{}
				inner.WriteString(fmt.Sprintf("var field%s []byte\n", field.Name))
				inner.WriteString(fmt.Sprintf("field%s, err = r.%s(x.%s)\n", field.Name, g.methodNameWithPrefix(field.Type, marshalJSONMethodPrefix), field.Name))
				inner.WriteString(fmt.Sprintf("if err != nil {\n"))
				inner.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s field name %s; %%w\", err)\n", errorContext, field.Name))
				inner.WriteString(fmt.Sprintf("}\n"))
				inner.WriteString(fmt.Sprintf("if len(field%s) == 0 {\n", field.Name))
				inner.WriteString(fmt.Sprintf("\tfield%s = []byte(\"null\")\n", field.Name))
				inner.WriteString(fmt.Sprintf("}\n"))
				inner.WriteString(fmt.Sprintf("if buf.Len() > 1 {\n"))
				inner.WriteString(fmt.Sprintf("\tbuf.WriteByte(',')\n"))
				inner.WriteString(fmt.Sprintf("}\n"))
				inner.WriteString(fmt.Sprintf("buf.WriteString(%q)\n", string(keyJSON)+":"))
				inner.WriteString(fmt.Sprintf("buf.Write(field%s)\n", field.Name))

				cond := ""
				if omitEmpty {
					cond = omitEmptyCond("x."+field.Name, field.Type)
				}
				if cond != "" {
					body.WriteString(fmt.Sprintf("if %s {\n", cond))
					body.WriteString(padLeftTabs(1, inner.String()))
					body.WriteString(fmt.Sprintf("}\n"))
				} else {
					body.WriteString(inner.String())
				}
			}
			body.WriteString(fmt.Sprintf("buf.WriteByte('}')\n"))
			body.WriteString(fmt.Sprintf("return buf.Bytes(), nil\n"))

			result, _ := methodWrap(body)

			methods := ""
			for _, field := range emitted {
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

	if shape.IsWeekAlias(x) {
		return "", nil
	}

	rootTypeName := g.rootTypeName()
	typeName := shape.ToGoTypeName(x, shape.WithRootPkgName(shape.ToGoPkgName(g.shape)))
	errorContext := g.errorContext(methodName)

	methodWrap := func(body *strings.Builder) (string, error) {
		result := &strings.Builder{}
		result.WriteString(fmt.Sprintf("func (r *%s) %s(data []byte) (%s, error) {\n", rootTypeName, methodName, typeName))
		result.WriteString(padLeftTabs(1, body.String()))
		result.WriteString("}\n")
		return result.String(), nil
	}

	return shape.MatchShapeR2(
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
		func(y *shape.PointerLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("if len(data) == 0 {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, nil\n"))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("if string(data[:4]) == \"null\" {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, nil\n"))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("result, err := r.%s(data)\n", g.methodNameWithPrefix(y.Type, unmarshalJSONMethodPrefix)))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn nil, fmt.Errorf(\"%s pointer; %%w\", err)\n", errorContext))
			body.WriteString(fmt.Sprintf("}\n"))
			body.WriteString(fmt.Sprintf("return &result, nil\n"))

			result, _ := methodWrap(body)

			methods, err := g.GenerateUnmarshalJSONMethods(y.Type)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: alias methods; %w", err)
			}

			return result + methods, nil
		},
		func(y *shape.AliasLike) (string, error) {
			aliasTypeName := shape.ToGoTypeName(y.Type, shape.WithRootPkgName(shape.ToGoPkgName(g.shape)))

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

			result, _ := methodWrap(body)

			methods, err := g.GenerateUnmarshalJSONMethods(y.Type)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: alias methods; %w", err)
			}

			return result + methods, nil
		},
		func(x *shape.PrimitiveLike) (string, error) {
			body := &strings.Builder{}
			body.WriteString(fmt.Sprintf("var result %s\n", typeName))
			body.WriteString(fmt.Sprintf("err := json.Unmarshal(data, &result)\n"))
			body.WriteString(fmt.Sprintf("if err != nil {\n"))
			body.WriteString(fmt.Sprintf("\treturn result, fmt.Errorf(\"%s native primitive unwrap; %%w\", err)\n", errorContext))
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

			result, _ := methodWrap(body)

			methods, err := g.GenerateUnmarshalJSONMethods(y.Element)
			if err != nil {
				return "", fmt.Errorf("generators.SerdeJSONTagged.GenerateUnmarshalJSONMethods: list methods; %w", err)
			}

			return result + methods, nil
		},
		func(y *shape.MapLike) (string, error) {
			keyTypeName := shape.ToGoTypeName(y.Key, shape.WithRootPkgName(shape.ToGoPkgName(g.shape)))
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

			result, _ := methodWrap(body)

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
				jsonFieldName, skip, _ := fieldJSONInfo(field)
				if skip {
					continue
				}

				body.WriteString(fmt.Sprintf("if field%s, ok := partial[\"%s\"]; ok {\n", field.Name, jsonFieldName))
				body.WriteString(fmt.Sprintf("\tresult.%s, err = r.%s(field%s)\n", field.Name, g.methodNameWithPrefix(field.Type, unmarshalJSONMethodPrefix), field.Name))
				body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
				body.WriteString(fmt.Sprintf("\t\treturn result, fmt.Errorf(\"%s field %s; %%w\", err)\n", errorContext, field.Name))
				body.WriteString(fmt.Sprintf("\t}\n"))
				body.WriteString(fmt.Sprintf("}\n"))
			}

			body.WriteString(fmt.Sprintf("return result, nil\n"))

			result, _ := methodWrap(body)

			methods := ""
			for _, field := range y.Fields {
				if _, skip, _ := fieldJSONInfo(field); skip {
					continue
				}
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
