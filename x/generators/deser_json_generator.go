package generators

import (
	"bytes"
	_ "embed"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"text/template"
)

var (
	//go:embed deser_json_generator.go.tmpl
	deserJSONTmpl string
)

func NewDeSerJSONGenerator(union *shape.UnionLike, helper *Helpers) *DeSerJSONGenerator {
	return &DeSerJSONGenerator{
		Union:                 union,
		helper:                helper,
		template:              template.Must(template.New("deser_json_generator.go.tmpl").Funcs(helper.Func()).Parse(deserJSONTmpl)),
		skipImportsAndPackage: false,
	}
}

type DeSerJSONGenerator struct {
	Union                 *shape.UnionLike
	helper                *Helpers
	template              *template.Template
	skipImportsAndPackage bool
}

func (g *DeSerJSONGenerator) SkipImportsAndPackage(x bool) {
	g.skipImportsAndPackage = x
}

func (g *DeSerJSONGenerator) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *DeSerJSONGenerator) ExtractImports(x shape.Shape) PkgMap {
	pkgMap := shape.ExtractPkgImportNames(x)
	if pkgMap == nil {
		pkgMap = make(map[string]string)
	}

	// add default and necessary imports
	defaults := g.defaultImportsFor(x)
	pkgMap = MergePkgMaps(pkgMap, defaults)

	// remove self from importing
	delete(pkgMap, shape.ToGoPkgName(x))
	return pkgMap
}

func (g *DeSerJSONGenerator) defaultImportsFor(x shape.Shape) PkgMap {
	return map[string]string{
		"json":   "encoding/json",
		"fmt":    "fmt",
		"shared": "github.com/widmogrod/mkunion/x/shared",
	}
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
func (g *DeSerJSONGenerator) OptionallyImport(x string) string {
	hasStruct := false
	for _, f := range g.Union.Variant {
		if g.IsStruct(f) {
			hasStruct = true
			break
		}
	}

	if hasStruct {
		return g.helper.RenderImport(x)
	}

	return ""
}

func (g *DeSerJSONGenerator) Generate() ([]byte, error) {
	result := &bytes.Buffer{}

	if !g.skipImportsAndPackage {
		result.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.Union)))

		pkgMap := g.ExtractImports(g.Union)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return nil, fmt.Errorf("generators.SerdeJSONTagged.Generate: when generating imports %w", err)
		}
		result.WriteString(impPart)
	}

	err := g.template.ExecuteTemplate(result, "deser_json_generator.go.tmpl", g)
	if err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}

func (g *DeSerJSONGenerator) Serde(x shape.Shape) string {
	//switch x.(type) {
	//case *shape.AliasLike:
	//	return ""
	//}

	serde := NewSerdeJSONTagged(x)
	serde.SkipImportsAndPackage(true)
	result, err := serde.Generate()
	if err != nil {
		panic(err)
	}

	return result
}
