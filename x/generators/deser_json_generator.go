package generators

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
	"text/template"
)

var (
	//go:embed deser_json_generator.go.tmpl
	deserJSONTmpl string
)

func NewDeSerJSONGenerator(union *shape.UnionLike, helper *Helpers) *DeSerJSONGenerator {
	return &DeSerJSONGenerator{
		Union:    union,
		helper:   helper,
		template: template.Must(template.New("deser_json_generator.go.tmpl").Funcs(helper.Func()).Parse(deserJSONTmpl)),
	}
}

type DeSerJSONGenerator struct {
	Union    *shape.UnionLike
	helper   *Helpers
	template *template.Template
}

func (g *DeSerJSONGenerator) ident(d int) string {
	return strings.Repeat("\t", d)
}

func (g *DeSerJSONGenerator) padLeft(d int, s string) string {
	// pad each new line with \t
	return strings.ReplaceAll(s, "\n", "\n"+g.ident(d))
}

func (g *DeSerJSONGenerator) JSONFieldName(x shape.FieldLike) string {
	if x.Tags != nil {
		if v, ok := x.Tags["json"]; ok {
			return v.Value
		}
	}

	return x.Name
}

func (g *DeSerJSONGenerator) IsStruct(x shape.Shape) bool {
	_, ok := x.(*shape.StructLike)
	return ok
}

func (g *DeSerJSONGenerator) VariantName(x shape.Shape) string {
	return TemplateHelperShapeVariantToName(x)
}

func (g *DeSerJSONGenerator) JSONVariantName(x shape.Shape) string {
	return shape.MustMatchShape(
		x,
		func(y *shape.Any) string {
			panic(fmt.Errorf("generators.JSONVariantName: %T not suported", y))
		},
		func(y *shape.RefName) string {
			return fmt.Sprintf("%s.%s", y.PkgName, y.Name)
		},
		func(x *shape.AliasLike) string {
			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
		func(y *shape.BooleanLike) string {
			panic(fmt.Errorf("generators.JSONVariantName: must be named %T", y))
		},
		func(y *shape.StringLike) string {
			panic(fmt.Errorf("generators.JSONVariantName: must be named %T", y))
		},
		func(y *shape.NumberLike) string {
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

func (g *DeSerJSONGenerator) SupportNativeJSONMarshal(x shape.Shape) bool {
	return shape.MustMatchShape(
		x,
		func(y *shape.Any) bool {
			return false
		},
		func(y *shape.RefName) bool {
			return false
		},
		func(y *shape.AliasLike) bool {
			return false
		},
		func(y *shape.BooleanLike) bool {
			return false
		},
		func(y *shape.StringLike) bool {
			return false
		},
		func(y *shape.NumberLike) bool {
			return false
		},
		func(y *shape.ListLike) bool {
			return false
		},
		func(y *shape.MapLike) bool {
			return false
		},
		func(y *shape.StructLike) bool {
			return true
		},
		func(y *shape.UnionLike) bool {
			return false
		},
	)
}

func (g *DeSerJSONGenerator) pkgNameAndImport(x *shape.UnionLike) string {
	if x.PkgImportName == g.Union.PkgImportName {
		return ""
	}

	g.helper.RenderImport(x.PkgImportName)
	return fmt.Sprintf("%s.", x.PkgName)
}

func (g *DeSerJSONGenerator) UnmarshalTemplate(field *shape.FieldLike, depth int) string {
	return shape.MustMatchShape(
		field.Type,
		func(x *shape.Any) string {
			return g.padLeft(depth, fmt.Sprintf("return json.Unmarshal(value, &result.%s)", field.Name))
		},
		func(x *shape.RefName) string {
			// check if reference is Union,
			// if yes, then we need to use function from the package called {VariantName}FromJSON
			y, _ := shape.LookupShape(x)
			z, ok := y.(*shape.UnionLike)
			if ok {
				pkgName := g.pkgNameAndImport(z)

				result := bytes.Buffer{}
				result.WriteString(fmt.Sprintf("res, err := %s%sFromJSON(value)\n", pkgName, z.Name))
				result.WriteString(fmt.Sprintf("if err != nil {\n"))
				result.WriteString(fmt.Sprintf("\treturn fmt.Errorf(\"%s._FromJSON: field %s %%w\", err)\n", g.Union.PkgName, z.Name))
				result.WriteString(fmt.Sprintf("}\n"))
				result.WriteString(fmt.Sprintf("result.%s = res\n", field.Name))
				result.WriteString(fmt.Sprintf("return nil"))

				return g.padLeft(depth+1, result.String())
			}

			return g.padLeft(depth, fmt.Sprintf("return json.Unmarshal(value, &result.%s)", field.Name))
		},
		func(x *shape.AliasLike) string {
			panic(fmt.Errorf("generators.UnmarshalTemplate: %T not suported", x))
		},
		func(x *shape.BooleanLike) string {
			return g.padLeft(depth, fmt.Sprintf("return json.Unmarshal(value, &result.%s)", field.Name))
		},
		func(x *shape.StringLike) string {
			return g.padLeft(depth, fmt.Sprintf("return json.Unmarshal(value, &result.%s)", field.Name))
		},
		func(x *shape.NumberLike) string {
			return g.padLeft(depth, fmt.Sprintf("return json.Unmarshal(value, &result.%s)", field.Name))
		},
		func(x *shape.ListLike) string {
			ref, ok := x.Element.(*shape.RefName)
			if ok {
				y, _ := shape.LookupShape(ref)
				z, ok := y.(*shape.UnionLike)
				if ok {
					pkgName := g.pkgNameAndImport(z)

					result := bytes.Buffer{}
					result.WriteString(fmt.Sprintf("res, err := shared.JSONToListWithDeserializer(value, result.%s, %s%sFromJSON)\n", field.Name, pkgName, z.Name))
					result.WriteString(fmt.Sprintf("if err != nil {\n"))
					result.WriteString(fmt.Sprintf("\treturn fmt.Errorf(\"%s._FromJSON: field %s %%w\", err)\n", g.Union.PkgName, z.Name))
					result.WriteString(fmt.Sprintf("}\n"))
					result.WriteString(fmt.Sprintf("result.%s = res\n", field.Name))
					result.WriteString(fmt.Sprintf("return nil"))

					return g.padLeft(depth+1, result.String())
				}
			}

			return g.padLeft(depth, fmt.Sprintf("return json.Unmarshal(value, &result.%s)", field.Name))
		},
		func(x *shape.MapLike) string {
			ref, ok := x.Val.(*shape.RefName)
			if ok {
				y, _ := shape.LookupShape(ref)
				z, ok := y.(*shape.UnionLike)
				if ok {
					pkgName := g.pkgNameAndImport(z)

					result := bytes.Buffer{}
					result.WriteString(fmt.Sprintf("res, err := shared.JSONToMapWithDeserializer(value, result.%s, %s%sFromJSON)\n", field.Name, pkgName, z.Name))
					result.WriteString(fmt.Sprintf("if err != nil {\n"))
					result.WriteString(fmt.Sprintf("\treturn fmt.Errorf(\"%s._FromJSON: field %s %%w\", err)\n", g.Union.PkgName, z.Name))
					result.WriteString(fmt.Sprintf("}\n"))
					result.WriteString(fmt.Sprintf("result.%s = res\n", field.Name))
					result.WriteString(fmt.Sprintf("return nil"))

					return g.padLeft(depth+1, result.String())
				}
			}

			return g.padLeft(depth, fmt.Sprintf("return json.Unmarshal(value, &result.%s)", field.Name))
		},
		func(x *shape.StructLike) string {
			return g.padLeft(depth, fmt.Sprintf("return json.Unmarshal(value, &result.%s)", field.Name))
		},
		func(x *shape.UnionLike) string {
			return g.padLeft(depth, fmt.Sprintf("return json.Unmarshal(value, &result.%s)", field.Name))
		},
	)
}

func (g *DeSerJSONGenerator) MarshalTemplate(field *shape.FieldLike, depth int) string {
	return shape.MustMatchShape(
		field.Type,
		func(x *shape.Any) string {
			return g.padLeft(depth, fmt.Sprintf("json.Marshal(x.%s)", field.Name))
		},
		func(x *shape.RefName) string {
			// check if reference is Union,
			y, _ := shape.LookupShape(x)
			if z, ok := y.(*shape.UnionLike); ok {
				pkgName := g.pkgNameAndImport(z)
				return g.padLeft(depth+1, fmt.Sprintf("%s%sToJSON(x.%s)", pkgName, z.Name, field.Name))
			}

			return g.padLeft(depth, fmt.Sprintf("json.Marshal(x.%s)", field.Name))
		},
		func(x *shape.AliasLike) string {
			panic(fmt.Errorf("generators.MarshalTemplate: %T not suported", x))
		},
		func(x *shape.BooleanLike) string {
			return g.padLeft(depth, fmt.Sprintf("json.Marshal(x.%s)", field.Name))
		},
		func(x *shape.StringLike) string {
			return g.padLeft(depth, fmt.Sprintf("json.Marshal(x.%s)", field.Name))
		},
		func(x *shape.NumberLike) string {
			return g.padLeft(depth, fmt.Sprintf("json.Marshal(x.%s)", field.Name))
		},
		func(x *shape.ListLike) string {
			ref, ok := x.Element.(*shape.RefName)
			if ok {
				y, _ := shape.LookupShape(ref)
				z, ok := y.(*shape.UnionLike)
				if ok {
					pkgName := g.pkgNameAndImport(z)
					return g.padLeft(depth+1, fmt.Sprintf("shared.JSONListFromSerializer(x.%s, %s%sToJSON)", field.Name, pkgName, z.Name))
				}
			}

			return g.padLeft(depth, fmt.Sprintf("json.Marshal(x.%s)", field.Name))
		},
		func(x *shape.MapLike) string {
			ref, ok := x.Val.(*shape.RefName)
			if ok {
				y, _ := shape.LookupShape(ref)
				z, ok := y.(*shape.UnionLike)
				if ok {
					pkgName := g.pkgNameAndImport(z)
					return g.padLeft(depth+1, fmt.Sprintf("shared.JSONMapFromSerializer(x.%s, %s%sToJSON)", field.Name, pkgName, z.Name))
				}
			}

			return g.padLeft(depth, fmt.Sprintf("json.Marshal(x.%s)", field.Name))
		},
		func(x *shape.StructLike) string {
			return g.padLeft(depth, fmt.Sprintf("json.Marshal(x.%s)", field.Name))
		},
		func(x *shape.UnionLike) string {
			return g.padLeft(depth, fmt.Sprintf("json.Marshal(x.%s)", field.Name))
		},
	)
}

func (g *DeSerJSONGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}
	err := g.template.ExecuteTemplate(result, "deser_json_generator.go.tmpl", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
