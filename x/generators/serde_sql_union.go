package generators

import (
	"bytes"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
)

func NewSerdeSQLUnion(union *shape.UnionLike) *SerdeSQLUnion {
	return &SerdeSQLUnion{
		union:                 union,
		skipImportsAndPackage: false,
		skipInitFunc:          false,
		pkgUsed: PkgMap{
			"sql":      "database/sql/driver",
			"fmt":      "fmt",
			"strconv":  "strconv",
			"strings":  "strings",
			"encoding": "encoding/json",
		},
	}
}

type SerdeSQLUnion struct {
	union                 *shape.UnionLike
	skipImportsAndPackage bool
	skipInitFunc          bool
	pkgUsed               PkgMap
}

func (g *SerdeSQLUnion) SkipImportsAndPackage(x bool) {
	g.skipImportsAndPackage = x
}

func (g *SerdeSQLUnion) SkipInitFunc(flag bool) *SerdeSQLUnion {
	g.skipInitFunc = flag
	return g
}

func (g *SerdeSQLUnion) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *SerdeSQLUnion) ExtractImports(x shape.Shape) PkgMap {
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

func (g *SerdeSQLUnion) VariantName(x shape.Shape) string {
	return TemplateHelperShapeVariantToName(x)
}

func (g *SerdeSQLUnion) Generate() ([]byte, error) {
	body := &bytes.Buffer{}

	// For experimental SQL support, we'll generate DDL and Scanner/Valuer interfaces
	body.WriteString("// Experimental SQL support - basic implementation\n\n")

	// Generate DDL comments
	ddl, err := g.GenerateUnionDDL(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeSQLUnion.Generate: when generating DDL; %w", err)
	}
	body.Write(ddl)

	// Generate SQL scanning/valuing functions
	scanFunc, err := g.GenerateUnionScanFunc(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeSQLUnion.Generate: when generating scan func; %w", err)
	}
	body.Write(scanFunc)

	valueFunc, err := g.GenerateUnionValueFunc(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeSQLUnion.Generate: when generating value func; %w", err)
	}
	body.Write(valueFunc)

	head := &bytes.Buffer{}
	if !g.skipImportsAndPackage {
		head.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.union)))

		pkgMap := g.ExtractImports(g.union)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return nil, fmt.Errorf("generators.SerdeSQLUnion.Generate: when generating imports %w", err)
		}
		head.WriteString(impPart)
	}

	if head.Len() > 0 {
		head.WriteString(body.String())
		return head.Bytes(), nil
	} else {
		return body.Bytes(), nil
	}
}

func (g *SerdeSQLUnion) constructionf(x shape.Shape, template string) string {
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

func (g *SerdeSQLUnion) parametrisedf(x shape.Shape, template string) string {
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

func (g *SerdeSQLUnion) GenerateUnionDDL(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	// Generate DDL comment for creating table
	tableName := strings.ToLower(shape.Name(union))
	body.WriteString(fmt.Sprintf("/*\n"))
	body.WriteString(fmt.Sprintf("-- SQL DDL for %s union type\n", shape.Name(union)))
	body.WriteString(fmt.Sprintf("-- Recommended table structure:\n\n"))
	body.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", tableName))
	body.WriteString(fmt.Sprintf("    id SERIAL PRIMARY KEY,\n"))
	body.WriteString(fmt.Sprintf("    type VARCHAR(50) NOT NULL,\n"))
	body.WriteString(fmt.Sprintf("    data JSON NOT NULL,\n"))
	body.WriteString(fmt.Sprintf("    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,\n"))
	body.WriteString(fmt.Sprintf("    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n"))
	body.WriteString(fmt.Sprintf(");\n\n"))

	body.WriteString(fmt.Sprintf("-- Valid type values:\n"))
	for _, variant := range union.Variant {
		variantName := strings.ToLower(g.VariantName(variant))
		body.WriteString(fmt.Sprintf("-- '%s' for %s\n", variantName, g.VariantName(variant)))
	}

	body.WriteString(fmt.Sprintf("\n-- Example queries:\n"))
	body.WriteString(fmt.Sprintf("-- SELECT * FROM %s WHERE type = 'branch';\n", tableName))
	body.WriteString(fmt.Sprintf("-- SELECT * FROM %s WHERE data->>'field' = 'value';\n", tableName))
	body.WriteString(fmt.Sprintf("*/\n\n"))

	return body.Bytes(), nil
}

func (g *SerdeSQLUnion) GenerateUnionScanFunc(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	unionType := g.parametrisedf(union, "%s")
	errorContext := fmt.Sprintf("%s.%sScanSQL:", g.union.PkgName, shape.Name(union))

	body.WriteString(fmt.Sprintf("// %sScanSQL scans a union type from SQL result\n", shape.Name(union)))
	body.WriteString(fmt.Sprintf("func %sScanSQL(unionType, jsonData string) (%s, error) {\n",
		g.constructionf(union, "%s"),
		unionType,
	))

	body.WriteString(fmt.Sprintf("\tswitch strings.ToLower(unionType) {\n"))
	for _, variant := range union.Variant {
		variantName := strings.ToLower(g.VariantName(variant))
		body.WriteString(fmt.Sprintf("\tcase %q:\n", variantName))
		body.WriteString(fmt.Sprintf("\t\tvar result %s\n", g.parametrisedf(variant, "%s")))
		body.WriteString(fmt.Sprintf("\t\terr := encoding.Unmarshal([]byte(jsonData), &result)\n"))
		body.WriteString(fmt.Sprintf("\t\tif err != nil {\n"))
		body.WriteString(fmt.Sprintf("\t\t\treturn nil, fmt.Errorf(\"%s %s; %%w\", err)\n", errorContext, variantName))
		body.WriteString(fmt.Sprintf("\t\t}\n"))
		body.WriteString(fmt.Sprintf("\t\treturn &result, nil\n"))
	}
	body.WriteString(fmt.Sprintf("\tdefault:\n"))
	body.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s unknown type: %%s\", unionType)\n", errorContext))
	body.WriteString(fmt.Sprintf("\t}\n"))
	body.WriteString(fmt.Sprintf("}\n\n"))

	return body.Bytes(), nil
}

func (g *SerdeSQLUnion) GenerateUnionValueFunc(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	unionType := g.parametrisedf(union, "%s")
	errorContext := fmt.Sprintf("%s.%sToSQL:", g.union.PkgName, shape.Name(union))

	body.WriteString(fmt.Sprintf("// %sToSQL converts union type to SQL type and JSON data\n", shape.Name(union)))
	body.WriteString(fmt.Sprintf("func %sToSQL(x %s) (string, string, error) {\n",
		g.constructionf(union, "%s"),
		unionType,
	))
	body.WriteString(fmt.Sprintf("\tif x == nil {\n"))
	body.WriteString(fmt.Sprintf("\t\treturn \"\", \"\", nil\n"))
	body.WriteString(fmt.Sprintf("\t}\n\n"))

	body.WriteString(fmt.Sprintf("\treturn %s(\n", MatchUnionFuncName(union, 3)))
	body.WriteString(fmt.Sprintf("\t\tx,\n"))

	for _, variant := range union.Variant {
		variantName := strings.ToLower(g.VariantName(variant))
		body.WriteString(fmt.Sprintf("\t\tfunc (y *%s) (string, string, error) {\n", g.parametrisedf(variant, "%s")))
		body.WriteString(fmt.Sprintf("\t\t\tdata, err := encoding.Marshal(y)\n"))
		body.WriteString(fmt.Sprintf("\t\t\tif err != nil {\n"))
		body.WriteString(fmt.Sprintf("\t\t\t\treturn \"\", \"\", fmt.Errorf(\"%s %s; %%w\", err)\n", errorContext, variantName))
		body.WriteString(fmt.Sprintf("\t\t\t}\n"))
		body.WriteString(fmt.Sprintf("\t\t\treturn %q, string(data), nil\n", variantName))
		body.WriteString(fmt.Sprintf("\t\t},\n"))
	}
	body.WriteString(fmt.Sprintf("\t)\n"))
	body.WriteString(fmt.Sprintf("}\n\n"))

	return body.Bytes(), nil
}