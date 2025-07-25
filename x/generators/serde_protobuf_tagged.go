package generators

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
)

const (
	unmarshalProtobufMethodPrefix = "_unmarshalProtobuf"
	marshalProtobufMethodPrefix   = "_marshalProtobuf"
)

func NewSerdeProtobufTagged(shape shape.Shape) *SerdeProtobufTagged {
	return &SerdeProtobufTagged{
		shape:                              shape,
		skipImportsAndPackage:              false,
		didGenerateMarshalProtobufMethod:   make(map[string]bool),
		didGenerateUnmarshalProtobufMethod: make(map[string]bool),
		pkgUsed: PkgMap{
			"proto":  "google.golang.org/protobuf/proto",
			"fmt":    "fmt",
			"binary": "encoding/binary",
		},
	}
}

type SerdeProtobufTagged struct {
	shape                 shape.Shape
	skipImportsAndPackage bool

	didGenerateMarshalProtobufMethod   map[string]bool
	didGenerateUnmarshalProtobufMethod map[string]bool
	pkgUsed                            PkgMap
}

func (g *SerdeProtobufTagged) SkipImportsAndPackage(flag bool) *SerdeProtobufTagged {
	g.skipImportsAndPackage = flag
	return g
}

func (g *SerdeProtobufTagged) Generate() (string, error) {
	body := &strings.Builder{}
	varPart, err := g.GenerateVarCasting(g.shape)
	if err != nil {
		return "", fmt.Errorf("generators.SerdeProtobufTagged.Generate: when generating variable casting %w", err)
	}
	body.WriteString(varPart)

	if !shape.IsWeekAlias(g.shape) {
		marshalPart, err := g.GenerateMarshalProtobuf(g.shape)
		if err != nil {
			return "", fmt.Errorf("generators.SerdeProtobufTagged.Generate: when generating marshal %w", err)
		}
		body.WriteString(marshalPart)

		unmarshalPart, err := g.GenerateUnmarshalProtobuf(g.shape)
		if err != nil {
			return "", fmt.Errorf("generators.SerdeProtobufTagged.Generate: when generating unmarshal %w", err)
		}
		body.WriteString(unmarshalPart)
	}

	head := &strings.Builder{}
	if !g.skipImportsAndPackage {
		head.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.shape)))

		pkgMap := g.ExtractImports(g.shape)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return "", fmt.Errorf("generators.SerdeProtobufTagged.Generate: when generating imports %w", err)
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

func (g *SerdeProtobufTagged) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *SerdeProtobufTagged) ExtractImports(x shape.Shape) PkgMap {
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

func (g *SerdeProtobufTagged) GenerateVarCasting(x shape.Shape) (string, error) {
	return shape.MatchShapeR2(
		x,
		func(x *shape.Any) (string, error) {
			panic("not implemented any var casting for protobuf")
		},
		func(x *shape.RefName) (string, error) {
			panic("not implemented ref var casting for protobuf")
		},
		func(x *shape.PointerLike) (string, error) {
			panic("not implemented pointer var casting for protobuf")
		},
		func(x *shape.AliasLike) (string, error) {
			result := &strings.Builder{}
			result.WriteString("var (\n")
			result.WriteString("\t_ proto.Message = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString(")\n\n")

			return result.String(), nil
		},
		func(x *shape.PrimitiveLike) (string, error) {
			panic("not implemented primitive var casting for protobuf")
		},
		func(x *shape.ListLike) (string, error) {
			panic("not implemented list var casting for protobuf")
		},
		func(x *shape.MapLike) (string, error) {
			panic("not implemented map var casting for protobuf")
		},
		func(x *shape.StructLike) (string, error) {
			result := &strings.Builder{}
			result.WriteString("var (\n")
			result.WriteString("\t_ proto.Message = (*")
			result.WriteString(shape.ToGoTypeName(x,
				shape.WithInstantiation(),
				shape.WithRootPackage(shape.ToGoPkgName(x)),
			))
			result.WriteString(")(nil)\n")
			result.WriteString(")\n\n")

			return result.String(), nil
		},
		func(x *shape.UnionLike) (string, error) {
			panic("not implemented union var casting for protobuf")
		},
	)
}

func (g *SerdeProtobufTagged) rootTypeName() string {
	return shape.ToGoTypeName(g.shape,
		shape.WithRootPackage(shape.ToGoPkgName(g.shape)),
	)
}

func (g *SerdeProtobufTagged) methodNameWithPrefix(x shape.Shape, prefix string) string {
	typeName := shape.ToGoTypeName(x,
		shape.WithRootPackage(shape.ToGoPkgName(g.shape)),
	)

	name := fmt.Sprintf("%s%s", prefix, removeNonAlpha.Replace(typeName))
	return name
}

func (g *SerdeProtobufTagged) GenerateMarshalProtobuf(x shape.Shape) (string, error) {
	result := &strings.Builder{}
	result.WriteString(fmt.Sprintf("func (r *%s) Marshal() ([]byte, error) {\n", g.rootTypeName()))
	result.WriteString(fmt.Sprintf("\tif r == nil {\n"))
	result.WriteString(fmt.Sprintf("\t\treturn nil, nil\n"))
	result.WriteString(fmt.Sprintf("\t}\n"))
	result.WriteString(fmt.Sprintf("\treturn proto.Marshal(r)\n"))
	result.WriteString("}\n\n")

	return result.String(), nil
}

func (g *SerdeProtobufTagged) GenerateUnmarshalProtobuf(x shape.Shape) (string, error) {
	result := &strings.Builder{}
	result.WriteString(fmt.Sprintf("func (r *%s) Unmarshal(data []byte) error {\n", g.rootTypeName()))
	result.WriteString(fmt.Sprintf("\tif len(data) == 0 {\n"))
	result.WriteString(fmt.Sprintf("\t\treturn nil\n"))
	result.WriteString(fmt.Sprintf("\t}\n"))
	result.WriteString(fmt.Sprintf("\treturn proto.Unmarshal(data, r)\n"))
	result.WriteString("}\n\n")

	return result.String(), nil
}