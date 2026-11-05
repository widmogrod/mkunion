package generators

import (
	"bytes"
	"fmt"

	"github.com/widmogrod/mkunion/x/shape"
)

// HandlerGenerator emits a typed handler for a union in which at least one
// variant declares its answer type with an embedded f.Returns[R]. There is no
// tag option: the marker is the switch.
//
// For a union Query with variants GetUser{f.Returns[*User]}, Count{f.Returns[int]}
// and Touch{} (no marker) it generates:
//
//	type QueryGetUserHandler interface { HandleGetUser(ctx, op *GetUser) (*User, error) }
//	type QueryCountHandler   interface { HandleCount(ctx, op *Count) (int, error) }
//	type QueryTouchHandler   interface { HandleTouch(ctx, op *Touch) error }
//	type QueryHandler interface { QueryGetUserHandler; QueryCountHandler; QueryTouchHandler }
//
//	func HandleQuery(ctx, op Query, onGetUser func(...) (*User, error), onCount ..., onTouch ...) (any, error)
//
//	func (r *GetUser) Perform(ctx context.Context, h any) (any, error)   // only variants with f.Returns
//	func (r *Count) Perform(ctx context.Context, h any) (any, error)
//
// One interface per variant lets a handler be assembled from parts, and lets
// Perform ask only for the method it needs. QueryHandler is the whole union:
// adding a variant breaks every QueryHandler at compile time. HandleQuery is
// the function form, exhaustive like MatchQueryR2, with each arm typed by the
// variant's f.Returns; the answer comes back untyped because the arms do not
// share a type. A variant without f.Returns answers with an error only, and
// gets no Perform: it is handled, but it is not an operation (see x/effect.Op).
//
// Nothing generated has a default. A handler is complete or it does not compile.
type HandlerGenerator struct {
	union   *shape.UnionLike
	pkgUsed PkgMap
}

func NewHandlerGenerator(union *shape.UnionLike) *HandlerGenerator {
	return &HandlerGenerator{
		union:   union,
		pkgUsed: PkgMap{"context": "context", "fmt": "fmt"},
	}
}

// handlerVariant is one union variant with its declared answer type.
// answer is empty when the variant does not embed f.Returns.
type handlerVariant struct {
	name   string // GetUser
	typ    string // GetUser, as written in this package
	answer string // *User, as written in this package; "" for none
}

// results is the method's result list: "(R, error)" or "error".
func (v handlerVariant) results() string {
	if v.answer == "" {
		return "error"
	}
	return fmt.Sprintf("(%s, error)", v.answer)
}

func (g *HandlerGenerator) variants() ([]handlerVariant, error) {
	if len(g.union.TypeParams) > 0 {
		return nil, fmt.Errorf("union %s has type parameters; a union with f.Returns must not have them, because Go interfaces cannot carry generic methods", g.union.Name)
	}

	root := shape.WithRootPkgName(shape.ToGoPkgName(g.union))
	result := make([]handlerVariant, 0, len(g.union.Variant))
	for _, v := range g.union.Variant {
		st, ok := v.(*shape.StructLike)
		if !ok {
			return nil, fmt.Errorf("union %s: variant %s is not a struct; a union with f.Returns needs struct variants", g.union.Name, shape.ToGoTypeName(v))
		}

		hv := handlerVariant{
			name: TemplateHelperShapeVariantToName(v),
			typ:  shape.ToGoTypeName(v, root),
		}
		for _, field := range st.Fields {
			if r, ok := shape.ReturnsOf(field); ok {
				g.pkgUsed = MergePkgMaps(g.pkgUsed, shape.ExtractPkgImportNames(r))
				hv.answer = shape.ToGoTypeName(r, root)
				break
			}
		}
		result = append(result, hv)
	}

	return result, nil
}

// ExtractImports lists the packages the generated code names: context, fmt
// and the packages of every answer type. Call it after Generate.
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

	fmt.Fprintf(out, "// One handler interface per %s variant, so a handler can be assembled from\n", name)
	fmt.Fprintf(out, "// parts. A variant with f.Returns answers with that type; one without answers\n")
	fmt.Fprintf(out, "// with an error only.\n")
	fmt.Fprintf(out, "type (\n")
	for _, v := range variants {
		fmt.Fprintf(out, "\t%s%sHandler interface {\n", name, v.name)
		fmt.Fprintf(out, "\t\tHandle%s(ctx context.Context, op *%s) %s\n", v.name, v.typ, v.results())
		fmt.Fprintf(out, "\t}\n")
	}
	fmt.Fprintf(out, ")\n\n")

	fmt.Fprintf(out, "// %sHandler handles every %s. Adding a variant to %s breaks every\n", name, name, name)
	fmt.Fprintf(out, "// %sHandler at compile time.\n", name)
	fmt.Fprintf(out, "type %sHandler interface {\n", name)
	for _, v := range variants {
		fmt.Fprintf(out, "\t%s%sHandler\n", name, v.name)
	}
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "// Handle%s hands op to the arm for its variant. Each arm is typed by the\n", name)
	fmt.Fprintf(out, "// variant's f.Returns; the answer comes back untyped because the arms do not\n")
	fmt.Fprintf(out, "// share a type. Exhaustive: every arm must be given.\n")
	fmt.Fprintf(out, "func Handle%s(\n", name)
	fmt.Fprintf(out, "\tctx context.Context,\n")
	fmt.Fprintf(out, "\top %s,\n", name)
	for _, v := range variants {
		fmt.Fprintf(out, "\ton%s func(ctx context.Context, op *%s) %s,\n", v.name, v.typ, v.results())
	}
	fmt.Fprintf(out, ") (any, error) {\n")
	fmt.Fprintf(out, "\treturn %s(op,\n", MatchUnionFuncName(g.union, 2))
	for _, v := range variants {
		if v.answer == "" {
			fmt.Fprintf(out, "\t\tfunc(x *%s) (any, error) { return nil, on%s(ctx, x) },\n", v.typ, v.name)
		} else {
			fmt.Fprintf(out, "\t\tfunc(x *%s) (any, error) { return on%s(ctx, x) },\n", v.typ, v.name)
		}
	}
	fmt.Fprintf(out, "\t)\n")
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "// Perform lets a variant with f.Returns be performed by any handler value that\n")
	fmt.Fprintf(out, "// has its Handle method, so operations from several unions can share one\n")
	fmt.Fprintf(out, "// program and one handler (see x/effect: Op, OpOf, Fx). The answer's static\n")
	fmt.Fprintf(out, "// type is carried by f.Returns.Ret, not by Perform.\n")
	for _, v := range variants {
		if v.answer == "" {
			continue
		}
		fmt.Fprintf(out, "func (r *%s) Perform(ctx context.Context, h any) (any, error) {\n", v.typ)
		fmt.Fprintf(out, "\ttyped, ok := h.(%s%sHandler)\n", name, v.name)
		fmt.Fprintf(out, "\tif !ok {\n")
		fmt.Fprintf(out, "\t\treturn nil, fmt.Errorf(\"%s: handler %%T does not implement %s%sHandler\", h)\n", shape.ToGoPkgName(g.union), name, v.name)
		fmt.Fprintf(out, "\t}\n")
		fmt.Fprintf(out, "\treturn typed.Handle%s(ctx, r)\n", v.name)
		fmt.Fprintf(out, "}\n\n")
	}

	return out.Bytes(), nil
}
