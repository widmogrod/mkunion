package generators

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
)

func NewSerdeJSONTagged(shape shape.Shape) *SerdeJSONTagged {
	return &SerdeJSONTagged{
		shape:                 shape,
		skipImportsAndPackage: false,
	}
}

type SerdeJSONTagged struct {
	shape                 shape.Shape
	skipImportsAndPackage bool
}

func (g *SerdeJSONTagged) SkipImportsAndPackage(flag bool) *SerdeJSONTagged {
	g.skipImportsAndPackage = flag
	return g
}

func (g *SerdeJSONTagged) Generate() (string, error) {
	result := &strings.Builder{}

	if !g.skipImportsAndPackage {
		result.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.shape)))

		pkgMap := g.ExtractImports(g.shape)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return "", fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating imports %w", err)
		}
		result.WriteString(impPart)
	}

	varPart, err := g.GenerateVarCasting(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating variable casting %w", err)
	}
	result.WriteString(varPart)

	marshalPart, err := g.GenerateMarshalJSON(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating marshal %w", err)

	}
	result.WriteString(marshalPart)

	unmarshalPart, err := g.GenerateUnmarshalJSON(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating unmarshal %w", err)
	}
	result.WriteString(unmarshalPart)

	return result.String(), nil
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
	g.InjectDefaultImports(pkgMap)
	// remove self from importing
	delete(pkgMap, shape.ToGoPkgName(x))
	return pkgMap
}

func (g *SerdeJSONTagged) InjectDefaultImports(pkgMap PkgMap) {
	pkgMap["fmt"] = "fmt"
	pkgMap["json"] = "encoding/json"
	pkgMap["shared"] = "github.com/widmogrod/mkunion/x/shared"
}

func (g *SerdeJSONTagged) GenerateVarCasting(x shape.Shape) (string, error) {
	return shape.MustMatchShapeR2(
		x,
		func(x *shape.Any) (string, error) {
			panic("not implemented")

		},
		func(x *shape.RefName) (string, error) {
			panic("not implemented")

		},
		func(x *shape.AliasLike) (string, error) {
			panic("not implemented")

		},
		func(x *shape.BooleanLike) (string, error) {
			panic("not implemented")

		},
		func(x *shape.StringLike) (string, error) {
			panic("not implemented")

		},
		func(x *shape.NumberLike) (string, error) {
			panic("not implemented")

		},
		func(x *shape.ListLike) (string, error) {
			panic("not implemented")

		},
		func(x *shape.MapLike) (string, error) {
			panic("not implemented")

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
			panic("not implemented")
		},
	)
}

func (g *SerdeJSONTagged) GenerateMarshalJSON(x shape.Shape) (string, error) {
	errorContext := fmt.Sprintf(`%s.MarshalJSON:`, shape.ToGoTypeName(x))

	return shape.MustMatchShapeR2(
		x,
		func(y *shape.Any) (string, error) {
			panic("not implemented")

		},
		func(y *shape.RefName) (string, error) {
			panic("not implemented")

		},
		func(y *shape.AliasLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.BooleanLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.StringLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.NumberLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.ListLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.MapLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.StructLike) (string, error) {
			typeName := shape.ToGoTypeName(y, shape.WithRootPackage(shape.ToGoPkgName(x)))

			result := &strings.Builder{}
			result.WriteString(fmt.Sprintf("func (r *%s) MarshalJSON() ([]byte, error) {\n", typeName))
			result.WriteString("\tvar err error\n")
			result.WriteString("\tresult := make(map[string]json.RawMessage)\n\n")

			for _, field := range y.Fields {
				fieldTypeName := shape.WrapPointerIf(shape.ToGoTypeName(field.Type, shape.WithRootPackage(shape.ToGoPkgName(x))), field.IsPointer)

				switch y := field.Type.(type) {
				case *shape.ListLike:
					elementTypeName := shape.WrapPointerIf(shape.ToGoTypeName(y.Element, shape.WithRootPackage(shape.ToGoPkgName(x))), field.IsPointer)

					result.WriteString(fmt.Sprintf("\tfield%s := make([]json.RawMessage, len(r.%s))\n", field.Name, field.Name))
					result.WriteString(fmt.Sprintf("\tfor i, v := range r.%s {\n", field.Name))
					result.WriteString(fmt.Sprintf("\t\tfield%s[i], err = shared.JSONMarshal[%s](v)\n", field.Name, elementTypeName))
					result.WriteString("\t\tif err != nil {\n")
					result.WriteString("\t\t\treturn nil, fmt.Errorf(\"" + errorContext + " field " + field.Name + "[%d]; %w\", i, err)\n")
					result.WriteString("\t\t}\n")
					result.WriteString("\t}\n")
					result.WriteString(fmt.Sprintf("\tresult[\"%s\"], err = json.Marshal(field%s)\n", field.Name, field.Name))
					result.WriteString("\tif err != nil {\n")
					result.WriteString("\t\treturn nil, fmt.Errorf(\"" + errorContext + " field " + field.Name + "; %w\", err)\n")
					result.WriteString("\t}\n")

				case *shape.MapLike:
					keyTypeName := shape.WrapPointerIf(shape.ToGoTypeName(y.Key, shape.WithRootPackage(shape.ToGoPkgName(x))), field.IsPointer)
					valTypeName := shape.WrapPointerIf(shape.ToGoTypeName(y.Val, shape.WithRootPackage(shape.ToGoPkgName(x))), field.IsPointer)

					result.WriteString(fmt.Sprintf("\tfield%s := make(map[string]json.RawMessage)\n", field.Name))
					result.WriteString(fmt.Sprintf("\tfor k, v := range r.%s {\n", field.Name))
					result.WriteString(fmt.Sprintf("\t\tvar key any\n"))
					result.WriteString(fmt.Sprintf("\t\tkey, ok := any(k).(string)\n"))
					result.WriteString(fmt.Sprintf("\t\tif !ok {\n"))
					result.WriteString(fmt.Sprintf("\t\t\tkey, err = shared.JSONMarshal[%s](k)\n", keyTypeName))
					result.WriteString("\t\t\tif err != nil {\n")
					result.WriteString("\t\t\t\treturn nil, fmt.Errorf(\"" + errorContext + " field " + field.Name + "[%#v] key decoding; %w\", key, err)\n")
					result.WriteString("\t\t\t}\n")
					result.WriteString("\t\t\tkey = string(key.([]byte))\n")
					result.WriteString("\t\t}\n\n")
					result.WriteString(fmt.Sprintf("\t\tfield%s[key.(string)], err = shared.JSONMarshal[%s](v)\n", field.Name, valTypeName))
					result.WriteString("\t\tif err != nil {\n")
					result.WriteString("\t\t\treturn nil, fmt.Errorf(\"" + errorContext + " field " + field.Name + "[%#v] value decoding %#v; %w\", key, v, err)\n")
					result.WriteString("\t\t}\n")
					result.WriteString("\t}\n")
					result.WriteString(fmt.Sprintf("\tresult[\"%s\"], err = json.Marshal(field%s)\n", field.Name, field.Name))
					result.WriteString("\tif err != nil {\n")
					result.WriteString("\t\treturn nil, fmt.Errorf(\"" + errorContext + " field " + field.Name + "; %w\", err)\n")
					result.WriteString("\t}\n")

				default:
					result.WriteString(fmt.Sprintf("\tfield%s, err := shared.JSONMarshal[%s](r.%s)\n", field.Name, fieldTypeName, field.Name))
					result.WriteString("\tif err != nil {\n")
					result.WriteString("\t\treturn nil, fmt.Errorf(\"" + errorContext + " field " + field.Name + "; %w\", err)\n")
					result.WriteString("\t}\n")
					result.WriteString(fmt.Sprintf("\tresult[\"%s\"] = field%s\n", field.Name, field.Name))
				}
				result.WriteString("\n")
			}

			result.WriteString("\toutput, err := json.Marshal(result)\n")
			result.WriteString("\tif err != nil {\n")
			result.WriteString("\t\treturn nil, fmt.Errorf(\"" + errorContext + " final step; %w\", err)\n")
			result.WriteString("\t}\n")
			result.WriteString("\n")
			result.WriteString("\treturn output, nil\n")
			result.WriteString("}\n\n")

			return result.String(), nil
		},
		func(x *shape.UnionLike) (string, error) {
			panic("not implemented")
		},
	)
}

func (g *SerdeJSONTagged) GenerateUnmarshalJSON(x shape.Shape) (string, error) {
	errorContext := fmt.Sprintf(`%s.UnmarshalJSON:`, shape.ToGoTypeName(x))

	return shape.MustMatchShapeR2(
		x,
		func(y *shape.Any) (string, error) {
			panic("not implemented")

		},
		func(y *shape.RefName) (string, error) {
			panic("not implemented")

		},
		func(y *shape.AliasLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.BooleanLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.StringLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.NumberLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.ListLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.MapLike) (string, error) {
			panic("not implemented")

		},
		func(y *shape.StructLike) (string, error) {
			typeName := shape.ToGoTypeName(y, shape.WithRootPackage(shape.ToGoPkgName(x)))

			result := &strings.Builder{}
			result.WriteString(fmt.Sprintf("func (r *%s) UnmarshalJSON(bytes []byte) error {\n", typeName))
			result.WriteString("\treturn shared.JSONParseObject(bytes, func(key string, bytes []byte) error {\n")
			result.WriteString("\t\tswitch key {\n")

			for _, field := range y.Fields {
				fieldTypeName := shape.WrapPointerIf(shape.ToGoTypeName(field.Type, shape.WithRootPackage(shape.ToGoPkgName(x))), field.IsPointer)

				result.WriteString(fmt.Sprintf("\t\tcase \"%s\":\n", field.Name))

				switch y := field.Type.(type) {
				case *shape.ListLike:
					elementTypeName := shape.WrapPointerIf(shape.ToGoTypeName(y.Element, shape.WithRootPackage(shape.ToGoPkgName(x))), field.IsPointer)

					result.WriteString(fmt.Sprintf("\t\t\terr := shared.JSONParseList(bytes, func(index int, bytes []byte) error {\n"))
					result.WriteString(fmt.Sprintf("\t\t\t\titem, err := shared.JSONUnmarshal[%s](bytes)\n", elementTypeName))
					result.WriteString(fmt.Sprintf("\t\t\t\tif err != nil {\n"))
					result.WriteString("\t\t\t\t\treturn fmt.Errorf(\"" + errorContext + " field " + field.Name + "[%d]; %w\", index, err)\n")
					result.WriteString(fmt.Sprintf("\t\t\t\t}\n"))
					result.WriteString(fmt.Sprintf("\t\t\t\tr.%s = append(r.%s, item)\n", field.Name, field.Name))
					result.WriteString(fmt.Sprintf("\t\t\t\treturn nil\n"))
					result.WriteString(fmt.Sprintf("\t\t\t})\n"))
					result.WriteString(fmt.Sprintf("\t\t\tif err != nil {\n"))
					result.WriteString("\t\t\t\treturn fmt.Errorf(\"" + errorContext + " field " + field.Name + "; %w\", err)\n")
					result.WriteString(fmt.Sprintf("\t\t\t}\n"))
					result.WriteString(fmt.Sprintf("\t\t\treturn nil\n"))

				case *shape.MapLike:
					keyTypeName := shape.WrapPointerIf(shape.ToGoTypeName(y.Key, shape.WithRootPackage(shape.ToGoPkgName(x))), field.IsPointer)
					valTypeName := shape.WrapPointerIf(shape.ToGoTypeName(y.Val, shape.WithRootPackage(shape.ToGoPkgName(x))), field.IsPointer)

					result.WriteString(fmt.Sprintf("\t\t\tr.%s = make(map[%s]%s)\n", field.Name, keyTypeName, valTypeName))
					result.WriteString(fmt.Sprintf("\t\t\terr := shared.JSONParseObject(bytes, func(rawKey string, bytes []byte) error {\n"))
					result.WriteString(fmt.Sprintf("\t\t\t\titem, err := shared.JSONUnmarshal[%s](bytes)\n", valTypeName))
					result.WriteString(fmt.Sprintf("\t\t\t\tif err != nil {\n"))
					result.WriteString("\t\t\t\t\treturn fmt.Errorf(\"key=%s to type=%T item error;  %w\", bytes, item, err)\n")
					result.WriteString(fmt.Sprintf("\t\t\t\t}\n"))
					result.WriteString(fmt.Sprintf("\n"))
					result.WriteString(fmt.Sprintf("\t\t\t\tvar key2 %s\n", keyTypeName))
					result.WriteString(fmt.Sprintf("\t\t\t\tif _, ok := any(key2).(string); !ok {\n"))
					result.WriteString(fmt.Sprintf("\t\t\t\t\tvar err error\n"))
					result.WriteString(fmt.Sprintf("\t\t\t\t\tkey2, err = shared.JSONUnmarshal[%s]([]byte(rawKey))\n", keyTypeName))
					result.WriteString(fmt.Sprintf("\t\t\t\t\tif err != nil {\n"))
					result.WriteString("\t\t\t\t\t\treturn fmt.Errorf(\"key=%s to type=%T key error; %w\", rawKey, key2, err)\n")
					result.WriteString("\t\t\t\t\t}\n")
					result.WriteString("\t\t\t\t} else {\n")
					result.WriteString(fmt.Sprintf("\t\t\t\t\tkey2 = any(rawKey).(%s)\n", keyTypeName))
					result.WriteString("\t\t\t\t}\n")
					result.WriteString("\n")
					result.WriteString(fmt.Sprintf("\t\t\t\tr.%s[key2] = item\n", field.Name))
					result.WriteString(fmt.Sprintf("\t\t\t\treturn nil\n"))
					result.WriteString(fmt.Sprintf("\t\t\t})\n"))
					result.WriteString(fmt.Sprintf("\t\t\tif err != nil {\n"))
					result.WriteString("\t\t\t\treturn fmt.Errorf(\"" + errorContext + " field " + field.Name + "; %w\", err)\n")
					result.WriteString(fmt.Sprintf("\t\t\t}\n"))
					result.WriteString(fmt.Sprintf("\t\t\treturn nil\n"))

				default:
					result.WriteString("\t\t\tvar err error\n")
					result.WriteString(fmt.Sprintf("\t\t\tr.%s, err = shared.JSONUnmarshal[%s](bytes)\n", field.Name, fieldTypeName))
					result.WriteString("\t\t\tif err != nil {\n")
					result.WriteString("\t\t\t\treturn fmt.Errorf(\"" + errorContext + " field " + field.Name + "; %w\", err)\n")
					result.WriteString("\t\t\t}\n")
					result.WriteString("\t\t\treturn nil\n")

				}

				result.WriteString("\n")
			}

			context := shape.ToGoTypeName(x)

			result.WriteString("\t\t}\n\n")
			result.WriteString("\t\treturn fmt.Errorf(\"" + context + ".UnmarshalJSON: unknown key: %s\", key)\n")
			result.WriteString("\t})\n")
			result.WriteString("}\n")

			return result.String(), nil
		},
		func(x *shape.UnionLike) (string, error) {
			panic("not implemented")
		},
	)
}

func padLeftTabs(n int, s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat("\t", n) + line
	}
	return strings.Join(lines, "\n")
}
