package generators

import (
	"bytes"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"strings"
)

func NewSerdeProtobufUnion(union *shape.UnionLike) *SerdeProtobufUnion {
	return &SerdeProtobufUnion{
		union:                 union,
		skipImportsAndPackage: false,
		skipInitFunc:          false,
		pkgUsed: PkgMap{
			"proto":  "google.golang.org/protobuf/proto",
			"fmt":    "fmt",
			"binary": "encoding/binary",
		},
	}
}

type SerdeProtobufUnion struct {
	union                 *shape.UnionLike
	skipImportsAndPackage bool
	skipInitFunc          bool
	pkgUsed               PkgMap
}

func (g *SerdeProtobufUnion) SkipImportsAndPackage(x bool) {
	g.skipImportsAndPackage = x
}

func (g *SerdeProtobufUnion) SkipInitFunc(flag bool) *SerdeProtobufUnion {
	g.skipInitFunc = flag
	return g
}

func (g *SerdeProtobufUnion) GenerateImports(pkgMap PkgMap) (string, error) {
	return GenerateImports(pkgMap), nil
}

func (g *SerdeProtobufUnion) ExtractImports(x shape.Shape) PkgMap {
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

func (g *SerdeProtobufUnion) VariantName(x shape.Shape) string {
	return TemplateHelperShapeVariantToName(x)
}

func (g *SerdeProtobufUnion) Generate() ([]byte, error) {
	body := &bytes.Buffer{}

	// For experimental protobuf support, we'll generate a simple enum-based approach
	body.WriteString("// Experimental Protobuf support - basic implementation\n\n")

	// Generate protobuf message type
	unionType, err := g.GenerateUnionMessage(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeProtobufUnion.Generate: when generating union message; %w", err)
	}
	body.Write(unionType)

	fromFunc, err := g.GenerateUnionFromFunc(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeProtobufUnion.Generate: when generating from func; %w", err)
	}
	body.Write(fromFunc)

	toFunc, err := g.GenerateUnionToFunc(g.union)
	if err != nil {
		return nil, fmt.Errorf("generators.SerdeProtobufUnion.Generate: when generating to func; %w", err)
	}
	body.Write(toFunc)

	head := &bytes.Buffer{}
	if !g.skipImportsAndPackage {
		head.WriteString(fmt.Sprintf("package %s\n\n", shape.ToGoPkgName(g.union)))

		pkgMap := g.ExtractImports(g.union)
		impPart, err := g.GenerateImports(pkgMap)
		if err != nil {
			return nil, fmt.Errorf("generators.SerdeProtobufUnion.Generate: when generating imports %w", err)
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

func (g *SerdeProtobufUnion) constructionf(x shape.Shape, template string) string {
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

func (g *SerdeProtobufUnion) parametrisedf(x shape.Shape, template string) string {
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

func (g *SerdeProtobufUnion) GenerateUnionMessage(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	// Generate an enum for the union type
	body.WriteString(fmt.Sprintf("type %sProtoType int32\n\n", shape.Name(union)))
	body.WriteString("const (\n")
	body.WriteString(fmt.Sprintf("\t%sProtoType_UNKNOWN %sProtoType = 0\n", shape.Name(union), shape.Name(union)))

	for i, variant := range union.Variant {
		body.WriteString(fmt.Sprintf("\t%sProtoType_%s %sProtoType = %d\n",
			shape.Name(union),
			strings.ToUpper(g.VariantName(variant)),
			shape.Name(union),
			i+1,
		))
	}
	body.WriteString(")\n\n")

	// Generate message struct
	body.WriteString(fmt.Sprintf("type %sProtoMessage struct {\n", shape.Name(union)))
	body.WriteString(fmt.Sprintf("\tType %sProtoType `protobuf:\"varint,1,opt,name=type,proto3,enum=%sProtoType\" json:\"type,omitempty\"`\n", shape.Name(union), shape.Name(union)))

	for i, variant := range union.Variant {
		fieldNum := i + 2 // Start from 2 since type is field 1
		body.WriteString(fmt.Sprintf("\t%s []byte `protobuf:\"bytes,%d,opt,name=%s,proto3\" json:\"%s,omitempty\"`\n",
			g.VariantName(variant),
			fieldNum,
			strings.ToLower(g.VariantName(variant)),
			strings.ToLower(g.VariantName(variant)),
		))
	}
	body.WriteString("}\n\n")

	return body.Bytes(), nil
}

func (g *SerdeProtobufUnion) GenerateUnionFromFunc(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	errorContext := fmt.Sprintf("%s.%sFromProtobuf:", g.union.PkgName, shape.Name(union))

	body.WriteString(fmt.Sprintf("func %sFromProtobuf(data []byte) (%s, error) {\n",
		g.constructionf(union, "%s"),
		g.parametrisedf(union, "%s"),
	))
	body.WriteString(fmt.Sprintf("\tif len(data) == 0 {\n"))
	body.WriteString(fmt.Sprintf("\t\treturn nil, nil\n"))
	body.WriteString(fmt.Sprintf("\t}\n"))
	body.WriteString(fmt.Sprintf("\tvar msg %sProtoMessage\n", shape.Name(union)))
	body.WriteString(fmt.Sprintf("\terr := proto.Unmarshal(data, &msg)\n"))
	body.WriteString(fmt.Sprintf("\tif err != nil {\n"))
	body.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s %%w\", err)\n", errorContext))
	body.WriteString(fmt.Sprintf("\t}\n\n"))

	body.WriteString(fmt.Sprintf("\tswitch msg.Type {\n"))
	for i, variant := range union.Variant {
		body.WriteString(fmt.Sprintf("\tcase %sProtoType_%s:\n",
			shape.Name(union),
			strings.ToUpper(g.VariantName(variant)),
		))
		body.WriteString(fmt.Sprintf("\t\tvar result %s\n", g.parametrisedf(variant, "%s")))
		body.WriteString(fmt.Sprintf("\t\terr := proto.Unmarshal(msg.%s, &result)\n", g.VariantName(variant)))
		body.WriteString(fmt.Sprintf("\t\tif err != nil {\n"))
		body.WriteString(fmt.Sprintf("\t\t\treturn nil, fmt.Errorf(\"%s variant %d; %%w\", err)\n", errorContext, i))
		body.WriteString(fmt.Sprintf("\t\t}\n"))
		body.WriteString(fmt.Sprintf("\t\treturn &result, nil\n"))
	}
	body.WriteString(fmt.Sprintf("\tdefault:\n"))
	body.WriteString(fmt.Sprintf("\t\treturn nil, fmt.Errorf(\"%s unknown type: %%d\", msg.Type)\n", errorContext))
	body.WriteString(fmt.Sprintf("\t}\n"))
	body.WriteString(fmt.Sprintf("}\n\n"))

	return body.Bytes(), nil
}

func (g *SerdeProtobufUnion) GenerateUnionToFunc(union *shape.UnionLike) ([]byte, error) {
	body := &bytes.Buffer{}

	errorContext := fmt.Sprintf("%s.%sToProtobuf:", g.union.PkgName, shape.Name(union))

	body.WriteString(fmt.Sprintf("func %sToProtobuf(x %s) ([]byte, error) {\n",
		g.constructionf(union, "%s"),
		g.parametrisedf(union, "%s"),
	))
	body.WriteString(fmt.Sprintf("\tif x == nil {\n"))
	body.WriteString(fmt.Sprintf("\t\treturn nil, nil\n"))
	body.WriteString(fmt.Sprintf("\t}\n\n"))

	body.WriteString(fmt.Sprintf("\treturn %s(\n", MatchUnionFuncName(union, 2)))
	body.WriteString(fmt.Sprintf("\t\tx,\n"))

	for i, variant := range union.Variant {
		body.WriteString(fmt.Sprintf("\t\tfunc (y *%s) ([]byte, error) {\n", g.parametrisedf(variant, "%s")))
		body.WriteString(fmt.Sprintf("\t\t\tdata, err := proto.Marshal(y)\n"))
		body.WriteString(fmt.Sprintf("\t\t\tif err != nil {\n"))
		body.WriteString(fmt.Sprintf("\t\t\t\treturn nil, fmt.Errorf(\"%s variant %d; %%w\", err)\n", errorContext, i))
		body.WriteString(fmt.Sprintf("\t\t\t}\n"))
		body.WriteString(fmt.Sprintf("\t\t\tmsg := &%sProtoMessage{\n", shape.Name(union)))
		body.WriteString(fmt.Sprintf("\t\t\t\tType: %sProtoType_%s,\n",
			shape.Name(union),
			strings.ToUpper(g.VariantName(variant)),
		))
		body.WriteString(fmt.Sprintf("\t\t\t\t%s: data,\n", g.VariantName(variant)))
		body.WriteString(fmt.Sprintf("\t\t\t}\n"))
		body.WriteString(fmt.Sprintf("\t\t\treturn proto.Marshal(msg)\n"))
		body.WriteString(fmt.Sprintf("\t\t},\n"))
	}
	body.WriteString(fmt.Sprintf("\t)\n"))
	body.WriteString(fmt.Sprintf("}\n\n"))

	return body.Bytes(), nil
}
