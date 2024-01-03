package generators

import (
	_ "embed"
	"github.com/widmogrod/mkunion/x/shape"
)

func NewShapeGenerator(union *shape.UnionLike) *ShapeGenerator {
	core := NewShapeTagged(union)
	return &ShapeGenerator{
		gen: core,
	}
}

type ShapeGenerator struct {
	gen *ShapeTagged
}

func (g *ShapeGenerator) SkipImportsAndPackage(flag bool) {
	g.gen.SkipImportsAndPackage(flag)
}

func (g *ShapeGenerator) SkipInitFunc(flag bool) {
	g.gen.SkipInitFunc(flag)
}

func (g *ShapeGenerator) GenerateImports(pkgMap PkgMap) (string, error) {
	return g.gen.GenerateImports(pkgMap)
}

func (g *ShapeGenerator) ExtractImports(x shape.Shape) PkgMap {
	return g.gen.ExtractImports(x)
}

func (g *ShapeGenerator) ExtractImportFuncs(s shape.Shape) []string {
	return g.gen.ExtractImportFuncs(s)
}

func (g *ShapeGenerator) GenerateInitFunc(init []string) (string, error) {
	return g.gen.GenerateInitFunc(init)
}

func (g *ShapeGenerator) Generate() ([]byte, error) {
	result, err := g.gen.Generate()
	if err != nil {
		return nil, err
	}

	return []byte(result), nil
}
