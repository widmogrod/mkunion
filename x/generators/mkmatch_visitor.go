package generators

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"go/ast"
)

const MkMatchTag = "mkmatch"

func NewMkMatchTaggedNodeVisitor() *MkMatchTaggedNodeVisitor {
	return &MkMatchTaggedNodeVisitor{
		matchBuilder: make(map[string]*MkMatchBuilder),
	}
}

type MkMatchTaggedNodeVisitor struct {
	matchBuilder map[string]*MkMatchBuilder
	pkgMap       PkgMap
}

func (f *MkMatchTaggedNodeVisitor) FromInferredInfo(inferred *shape.InferredInfo) {
	f.pkgMap = inferred.PackageNameToPackageImport()
	inferred.RunVisitorOnTaggedASTNodes(MkMatchTag, f.visitTaggedNode)
}

func (f *MkMatchTaggedNodeVisitor) Specs() []*MatchSpec {
	var specs []*MatchSpec
	for _, v := range f.matchBuilder {
		spec, err := v.Build()
		if err != nil {
			panic(err)
		}
		specs = append(specs, spec)
	}
	return specs
}

func (f *MkMatchTaggedNodeVisitor) visitTaggedNode(node *shape.NodeAndTag) {
	b := NewMkMatchBuilder()
	b.InitPkgMap(f.pkgMap)

	// If tag value is empty, the builder will use the interface name when walking the AST
	// Otherwise, use the provided tag value
	if node.Tag.Value != "" {
		b.name = node.Tag.Value
	}

	ast.Walk(b, node.Node)

	// After walking, the builder should have a name (either from tag or interface name)
	// Use that as the key for storing the builder
	key := b.name
	if key == "" {
		// This shouldn't happen, but as a fallback use the original tag value
		key = node.Tag.Value
		if key == "" {
			key = fmt.Sprintf("_unnamed_%p", b)
		}
	}
	f.matchBuilder[key] = b
}

func typeToString(t ast.Expr) string {
	switch t := t.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeToString(t.X)
	case *ast.SelectorExpr:
		return typeToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len != nil {
			return fmt.Sprintf("[%s]%s", t.Len, typeToString(t.Elt))
		}
		return "[]" + typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeToString(t.Key) + "]" + typeToString(t.Value)
	case *ast.IndexExpr:
		return typeToString(t.X) + "[" + typeToString(t.Index) + "]"
	default:
		panic(fmt.Sprintf("type %T is not supported", t))
	}
}
