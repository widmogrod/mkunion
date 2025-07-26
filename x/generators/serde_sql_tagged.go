package generators

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
)

const (
	unmarshalSQLMethodPrefix = "_unmarshalSQL"
	marshalSQLMethodPrefix   = "_marshalSQL"
)

func NewSerdeSQLTagged(shape shape.Shape) *SerdeSQLTagged {
	return &SerdeSQLTagged{
		shape:                         shape,
		skipImportsAndPackage:         false,
		didGenerateMarshalSQLMethod:   make(map[string]bool),
		didGenerateUnmarshalSQLMethod: make(map[string]bool),
		pkgUsed: PkgMap{
			"sql":      "database/sql/driver",
			"fmt":      "fmt",
			"strconv":  "strconv",
			"strings":  "strings",
			"encoding": "encoding/json",
		},
	}
}

type SerdeSQLTagged struct {
	shape                 shape.Shape
	skipImportsAndPackage bool

	didGenerateMarshalSQLMethod   map[string]bool
	didGenerateUnmarshalSQLMethod map[string]bool
	pkgUsed                       PkgMap
}

func (g *SerdeSQLTagged) SkipImportsAndPackage(flag bool) *SerdeSQLTagged {
	g.skipImportsAndPackage = flag
	return g
}

func (g *SerdeSQLTagged) Generate() (string, error) {
	body := &strings.Builder{}
	varPart, err := g.GenerateVarCasting(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeSQLTagged.Generate: when generating variable casting %w", err)
	}
	body.WriteString(varPart)

	if !shape.IsWeekAlias(g.shape) {
		scanPart, err := g.GenerateScan(g.shape)
		if err != nil {
			return "", fmt.Errorf("generators.SerdeSQLTagged.Generate: when generating Scan %w", err)
		}
		body.WriteString(scanPart)

		valuePart, err := g.GenerateValue(g.shape)
		if err != nil {
			return "", fmt.Errorf("generators.SerdeSQLTagged.Generate: when generating Value %w", err)
		}
		body.WriteString(valuePart)
	}

	head := &strings.Builder{}
	if !g.skipImportsAndPackage {
		head.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.shape)))

		pkgMap := g.ExtractImports(g.shape)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return "", fmt.Errorf("generators.SerdeSQLTagged.Generate: when generating imports %w", err)
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

func (g *SerdeSQLTagged) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *SerdeSQLTagged) ExtractImports(x shape.Shape) PkgMap {
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

func (g *SerdeSQLTagged) GenerateVarCasting(x shape.Shape) (string, error) {
	return shape.MatchShapeR2(
		x,
		func(x *shape.Any) (string, error) {
			panic("not implemented any var casting for SQL")
		},
		func(x *shape.RefName) (string, error) {
			panic("not implemented ref var casting for SQL")
		},
		func(x *shape.PointerLike) (string, error) {
			panic("not implemented pointer var casting for SQL")
		},
		func(x *shape.AliasLike) (string, error) {
			result := &strings.Builder{}
			result.WriteString("var (\n")
			result.WriteString("\t_ sql.Scanner = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString("\t_ sql.Valuer = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString(")\n\n")

			return result.String(), nil
		},
		func(x *shape.PrimitiveLike) (string, error) {
			panic("not implemented primitive var casting for SQL")
		},
		func(x *shape.ListLike) (string, error) {
			panic("not implemented list var casting for SQL")
		},
		func(x *shape.MapLike) (string, error) {
			panic("not implemented map var casting for SQL")
		},
		func(x *shape.StructLike) (string, error) {
			result := &strings.Builder{}
			result.WriteString("var (\n")
			result.WriteString("\t_ sql.Scanner = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString("\t_ sql.Valuer = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString(")\n\n")

			return result.String(), nil
		},
		func(x *shape.UnionLike) (string, error) {
			panic("not implemented union var casting for SQL")
		},
	)
}

func (g *SerdeSQLTagged) rootTypeName() string {
	return shape.ToGoTypeName(g.shape,
		shape.WithRootPackage(shape.ToGoPkgName(g.shape)),
	)
}

func (g *SerdeSQLTagged) GenerateScan(x shape.Shape) (string, error) {
	result := &strings.Builder{}
	result.WriteString(fmt.Sprintf("// Scan implements sql.Scanner interface for SQL database operations\n"))
	result.WriteString(fmt.Sprintf("func (r *%s) Scan(value interface{}) error {\n", g.rootTypeName()))
	result.WriteString(fmt.Sprintf("\tif value == nil {\n"))
	result.WriteString(fmt.Sprintf("\t\treturn nil\n"))
	result.WriteString(fmt.Sprintf("\t}\n"))
	result.WriteString(fmt.Sprintf("\n"))
	result.WriteString(fmt.Sprintf("\tswitch v := value.(type) {\n"))
	result.WriteString(fmt.Sprintf("\tcase string:\n"))
	result.WriteString(fmt.Sprintf("\t\t// For complex types, assume JSON encoding\n"))
	result.WriteString(fmt.Sprintf("\t\treturn encoding.Unmarshal([]byte(v), r)\n"))
	result.WriteString(fmt.Sprintf("\tcase []byte:\n"))
	result.WriteString(fmt.Sprintf("\t\t// For complex types, assume JSON encoding\n"))
	result.WriteString(fmt.Sprintf("\t\treturn encoding.Unmarshal(v, r)\n"))
	result.WriteString(fmt.Sprintf("\tdefault:\n"))
	result.WriteString(fmt.Sprintf("\t\treturn fmt.Errorf(\"cannot scan %%T into %s\", value)\n", g.rootTypeName()))
	result.WriteString(fmt.Sprintf("\t}\n"))
	result.WriteString(fmt.Sprintf("}\n\n"))

	return result.String(), nil
}

func (g *SerdeSQLTagged) GenerateValue(x shape.Shape) (string, error) {
	result := &strings.Builder{}
	result.WriteString(fmt.Sprintf("// Value implements sql.Valuer interface for SQL database operations\n"))
	result.WriteString(fmt.Sprintf("func (r *%s) Value() (sql.Value, error) {\n", g.rootTypeName()))
	result.WriteString(fmt.Sprintf("\tif r == nil {\n"))
	result.WriteString(fmt.Sprintf("\t\treturn nil, nil\n"))
	result.WriteString(fmt.Sprintf("\t}\n"))
	result.WriteString(fmt.Sprintf("\n"))

	// Generate basic SQL value conversion based on type
	err := shape.MatchShapeR1(
		x,
		func(x *shape.Any) error {
			result.WriteString(fmt.Sprintf("\t// For any type, convert to JSON string\n"))
			result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
			result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
			result.WriteString(fmt.Sprintf("\t}\n"))
			result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
			return nil
		},
		func(x *shape.RefName) error {
			result.WriteString(fmt.Sprintf("\t// For referenced type, convert to JSON string\n"))
			result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
			result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
			result.WriteString(fmt.Sprintf("\t}\n"))
			result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
			return nil
		},
		func(x *shape.PointerLike) error {
			result.WriteString(fmt.Sprintf("\t// For pointer type, convert to JSON string\n"))
			result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
			result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
			result.WriteString(fmt.Sprintf("\t}\n"))
			result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
			return nil
		},
		func(x *shape.AliasLike) error {
			// Check if underlying type is a simple type
			return shape.MatchShapeR1(
				x.Type,
				func(y *shape.Any) error {
					result.WriteString(fmt.Sprintf("\treturn encoding.Marshal(r)\n"))
					return nil
				},
				func(y *shape.RefName) error {
					result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
					result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
					result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
					result.WriteString(fmt.Sprintf("\t}\n"))
					result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
					return nil
				},
				func(y *shape.PointerLike) error {
					result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
					result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
					result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
					result.WriteString(fmt.Sprintf("\t}\n"))
					result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
					return nil
				},
				func(y *shape.AliasLike) error {
					result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
					result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
					result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
					result.WriteString(fmt.Sprintf("\t}\n"))
					result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
					return nil
				},
				func(y *shape.PrimitiveLike) error {
					return shape.MatchPrimitiveKindR1(
						y.Kind,
						func(y *shape.BooleanLike) error {
							result.WriteString(fmt.Sprintf("\treturn bool(*r), nil\n"))
							return nil
						},
						func(y *shape.StringLike) error {
							result.WriteString(fmt.Sprintf("\treturn string(*r), nil\n"))
							return nil
						},
						func(y *shape.NumberLike) error {
							result.WriteString(fmt.Sprintf("\treturn float64(*r), nil\n"))
							return nil
						},
					)
				},
				func(y *shape.ListLike) error {
					result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
					result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
					result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
					result.WriteString(fmt.Sprintf("\t}\n"))
					result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
					return nil
				},
				func(y *shape.MapLike) error {
					result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
					result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
					result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
					result.WriteString(fmt.Sprintf("\t}\n"))
					result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
					return nil
				},
				func(y *shape.StructLike) error {
					result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
					result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
					result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
					result.WriteString(fmt.Sprintf("\t}\n"))
					result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
					return nil
				},
				func(y *shape.UnionLike) error {
					result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
					result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
					result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
					result.WriteString(fmt.Sprintf("\t}\n"))
					result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
					return nil
				},
			)
		},
		func(x *shape.PrimitiveLike) error {
			return shape.MatchPrimitiveKindR1(
				x.Kind,
				func(x *shape.BooleanLike) error {
					result.WriteString(fmt.Sprintf("\treturn bool(*r), nil\n"))
					return nil
				},
				func(x *shape.StringLike) error {
					result.WriteString(fmt.Sprintf("\treturn string(*r), nil\n"))
					return nil
				},
				func(x *shape.NumberLike) error {
					result.WriteString(fmt.Sprintf("\treturn float64(*r), nil\n"))
					return nil
				},
			)
		},
		func(x *shape.ListLike) error {
			result.WriteString(fmt.Sprintf("\t// For slice/array, convert to JSON string\n"))
			result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
			result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
			result.WriteString(fmt.Sprintf("\t}\n"))
			result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
			return nil
		},
		func(x *shape.MapLike) error {
			result.WriteString(fmt.Sprintf("\t// For map, convert to JSON string\n"))
			result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
			result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
			result.WriteString(fmt.Sprintf("\t}\n"))
			result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
			return nil
		},
		func(x *shape.StructLike) error {
			result.WriteString(fmt.Sprintf("\t// For struct, convert to JSON string\n"))
			result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
			result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
			result.WriteString(fmt.Sprintf("\t}\n"))
			result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
			return nil
		},
		func(x *shape.UnionLike) error {
			result.WriteString(fmt.Sprintf("\t// For union, convert to JSON string\n"))
			result.WriteString(fmt.Sprintf("\tdata, err := encoding.Marshal(r)\n"))
			result.WriteString(fmt.Sprintf("\tif err != nil {\n"))
			result.WriteString(fmt.Sprintf("\t\treturn nil, err\n"))
			result.WriteString(fmt.Sprintf("\t}\n"))
			result.WriteString(fmt.Sprintf("\treturn string(data), nil\n"))
			return nil
		},
	)

	if err != nil {
		return "", fmt.Errorf("generators.SerdeSQLTagged.GenerateValue: %w", err)
	}

	result.WriteString(fmt.Sprintf("}\n\n"))

	return result.String(), nil
}
