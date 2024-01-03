package generators

import (
	_ "embed"
	"github.com/widmogrod/mkunion/x/shape"
)

func NewShapeUnion(union *shape.UnionLike) *ShapeUnion {
	core := NewShapeTagged(union)
	return &ShapeUnion{
		gen: core,
	}
}

type ShapeUnion struct {
	gen *ShapeTagged
}

func (g *ShapeUnion) SkipImportsAndPackage(flag bool) {
	g.gen.SkipImportsAndPackage(flag)
}

func (g *ShapeUnion) SkipInitFunc(flag bool) {
	g.gen.SkipInitFunc(flag)
}

func (g *ShapeUnion) GenerateImports(pkgMap PkgMap) (string, error) {
	return g.gen.GenerateImports(pkgMap)
}

func (g *ShapeUnion) ExtractImports(x shape.Shape) PkgMap {
	return g.gen.ExtractImports(x)
}

func (g *ShapeUnion) ExtractImportFuncs(s shape.Shape) []string {
	return g.gen.ExtractImportFuncs(s)
}

func (g *ShapeUnion) GenerateInitFunc(init []string) (string, error) {
	return g.gen.GenerateInitFunc(init)
}

func (g *ShapeUnion) Generate() ([]byte, error) {
	result, err := g.gen.Generate()
	if err != nil {
		return nil, err
	}

	return []byte(result), nil
}
