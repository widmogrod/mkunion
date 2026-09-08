package generators

import (
	"bytes"
	"fmt"

	"github.com/widmogrod/mkunion/x/shape"
)

// HandlerGenerator emits a typed handler for a union whose variants declare
// their answer type with an embedded f.Returns[R]. It is enabled with the
// `handler` union option: //go:tag mkunion:"Query,handler".
//
// For a union Query with variants GetUser{f.Returns[*User]} and Count{f.Returns[int]}
// it generates:
//
//	type QueryHandler interface {            // one typed method per variant
//		HandleGetUser(ctx context.Context, op *GetUser) (*User, error)
//		HandleCount(ctx context.Context, op *Count) (int, error)
//	}
//	type QueryOf[R any] interface {          // a Query that answers with R
//		Query
//		HandleQuery(ctx context.Context, h QueryHandler) (R, error)
//	}
//	func (r *GetUser) HandleQuery(...) (*User, error)   // *GetUser is a QueryOf[*User]
//	func QueryHandlerFunc(h QueryHandler) func(ctx context.Context, op Query) (any, error)
//	type QueryDefaults struct{}              // zero answers, embed and override
//
// Go interfaces cannot carry generic methods, so the per-variant answer type
// lives on QueryOf[R] and in the handler's method signatures. Adding a variant
// breaks every QueryHandler at compile time, which is the point.
type HandlerGenerator struct {
	union   *shape.UnionLike
	pkgUsed PkgMap
}

func NewHandlerGenerator(union *shape.UnionLike) *HandlerGenerator {
	return &HandlerGenerator{
		union:   union,
		pkgUsed: PkgMap{"context": "context"},
	}
}

// handlerVariant is one union variant with its declared answer type.
type handlerVariant struct {
	name   string // GetUser
	typ    string // GetUser, as written in this package
	answer string // *User, as written in this package
}

func (g *HandlerGenerator) variants() ([]handlerVariant, error) {
	if len(g.union.TypeParams) > 0 {
		return nil, fmt.Errorf("union %s has type parameters; the handler option needs a union without them, because Go interfaces cannot carry generic methods", g.union.Name)
	}

	root := shape.WithRootPkgName(shape.ToGoPkgName(g.union))
	result := make([]handlerVariant, 0, len(g.union.Variant))
	for _, v := range g.union.Variant {
		st, ok := v.(*shape.StructLike)
		if !ok {
			return nil, fmt.Errorf("union %s: variant %s is not a struct; the handler option needs struct variants that embed f.Returns[R]", g.union.Name, shape.ToGoTypeName(v))
		}

		var answer shape.Shape
		for _, field := range st.Fields {
			if r, ok := shape.ReturnsOf(field); ok {
				answer = r
				break
			}
		}
		if answer == nil {
			return nil, fmt.Errorf("union %s: variant %s does not embed f.Returns[R]; every variant must declare its answer type", g.union.Name, st.Name)
		}

		g.pkgUsed = MergePkgMaps(g.pkgUsed, shape.ExtractPkgImportNames(answer))

		result = append(result, handlerVariant{
			name:   TemplateHelperShapeVariantToName(v),
			typ:    shape.ToGoTypeName(v, root),
			answer: shape.ToGoTypeName(answer, root),
		})
	}

	return result, nil
}

// ExtractImports lists the packages the generated code names: context and
// the packages of every answer type. Call it after Generate.
func (g *HandlerGenerator) ExtractImports() PkgMap {
	pkgMap := PkgMap{}
	pkgMap = MergePkgMaps(pkgMap, g.pkgUsed)
	delete(pkgMap, shape.ToGoPkgName(g.union))
	return pkgMap
}

func (g *HandlerGenerator) Generate() ([]byte, error) {
	variants, err := g.variants()
	if err != nil {
		return nil, fmt.Errorf("generators.HandlerGenerator.Generate: %w", err)
	}

	name := g.union.Name
	out := &bytes.Buffer{}

	fmt.Fprintf(out, "// %sHandler answers every %s with the type it declares in f.Returns.\n", name, name)
	fmt.Fprintf(out, "// Adding a variant to %s breaks every %sHandler at compile time.\n", name, name)
	fmt.Fprintf(out, "type %sHandler interface {\n", name)
	for _, v := range variants {
		fmt.Fprintf(out, "\tHandle%s(ctx context.Context, op *%s) (%s, error)\n", v.name, v.typ, v.answer)
	}
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "// %sOf is a %s that answers with R.\n", name, name)
	fmt.Fprintf(out, "type %sOf[R any] interface {\n", name)
	fmt.Fprintf(out, "\t%s\n", name)
	fmt.Fprintf(out, "\tHandle%s(ctx context.Context, h %sHandler) (R, error)\n", name, name)
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "var (\n")
	for _, v := range variants {
		fmt.Fprintf(out, "\t_ %sOf[%s] = (*%s)(nil)\n", name, v.answer, v.typ)
	}
	fmt.Fprintf(out, ")\n\n")

	for _, v := range variants {
		fmt.Fprintf(out, "func (r *%s) Handle%s(ctx context.Context, h %sHandler) (%s, error) {\n", v.typ, name, name, v.answer)
		fmt.Fprintf(out, "\treturn h.Handle%s(ctx, r)\n", v.name)
		fmt.Fprintf(out, "}\n\n")
	}

	fmt.Fprintf(out, "// %sHandlerFunc adapts a typed %sHandler to a plain function over the union.\n", name, name)
	fmt.Fprintf(out, "// The answer is the type the variant declares; only its static type is lost.\n")
	fmt.Fprintf(out, "func %sHandlerFunc(h %sHandler) func(ctx context.Context, op %s) (any, error) {\n", name, name, name)
	fmt.Fprintf(out, "\treturn func(ctx context.Context, op %s) (any, error) {\n", name)
	fmt.Fprintf(out, "\t\treturn %s(op,\n", MatchUnionFuncName(g.union, 2))
	for _, v := range variants {
		fmt.Fprintf(out, "\t\t\tfunc(x *%s) (any, error) { return x.Handle%s(ctx, h) },\n", v.typ, name)
	}
	fmt.Fprintf(out, "\t\t)\n")
	fmt.Fprintf(out, "\t}\n")
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "// %sDefaults answers every %s with the zero value of its declared type.\n", name, name)
	fmt.Fprintf(out, "// Embed it in a handler and override only the methods you care about.\n")
	fmt.Fprintf(out, "type %sDefaults struct{}\n\n", name)
	fmt.Fprintf(out, "var _ %sHandler = %sDefaults{}\n\n", name, name)
	for _, v := range variants {
		fmt.Fprintf(out, "func (%sDefaults) Handle%s(context.Context, *%s) (%s, error) {\n", name, v.name, v.typ, v.answer)
		fmt.Fprintf(out, "\tvar zero %s\n", v.answer)
		fmt.Fprintf(out, "\treturn zero, nil\n")
		fmt.Fprintf(out, "}\n\n")
	}

	return out.Bytes(), nil
}
