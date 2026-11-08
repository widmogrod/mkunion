package generators

import (
	"bytes"
	"fmt"
	"strings"

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
//	func NewQueryHandler(onGetUser func(...) (*User, error), onCount ..., onTouch ...) QueryHandler
//
//	func (r *GetUser) Perform(ctx context.Context, h any) (any, error)   // only variants with f.Returns
//	func (r *Count) Perform(ctx context.Context, h any) (any, error)
//
// One interface per variant lets a handler be assembled from parts, and lets
// Perform ask only for the method it needs. QueryHandler is the whole union:
// adding a variant breaks every QueryHandler at compile time. HandleQuery is
// the function form, exhaustive like MatchQueryR2, with each arm typed by the
// variant's f.Returns; the answer comes back untyped because the arms do not
// share a type. NewQueryHandler is the same arms as a value, for the places
// that take a handler value. A variant without f.Returns answers with an error only, and
// gets no Perform: it is handled, but it is not an operation (see x/effect.Op).
//
// Every handler union also gets a decorator:
//
//	func AroundQuery(next QueryHandler, around func(ctx, op Query, call func(ctx) (any, error)) (any, error)) QueryHandler
//
// When every variant answers with a variant of one other union of the same
// package, Reply, the union is an RPC contract and three more things are
// generated, all typed by both unions:
//
//	func DispatchQuery(ctx, h QueryHandler, op Query) (Reply, error)        // the server side
//	func NewQueryClient(send func(ctx, Query) (Reply, error)) QueryHandler  // the client side
//	type QueryUnexpectedAnswer struct { Query Query; Reply Reply }          // a Reply that is not the declared answer
//
// The answer union is inferred: every variant has f.Returns and every answer
// is *V for a variant V of the same union. The tag option answers=Reply names
// it instead, and then a variant whose answer is not a Reply is an error at
// generation time, which is where a drifted contract should fail.
//
// Nothing generated has a default. A handler is complete or it does not compile.
type HandlerGenerator struct {
	union    *shape.UnionLike
	siblings []*shape.UnionLike
	pkgUsed  PkgMap
}

func NewHandlerGenerator(union *shape.UnionLike) *HandlerGenerator {
	return &HandlerGenerator{
		union:   union,
		pkgUsed: PkgMap{"context": "context", "fmt": "fmt"},
	}
}

// WithUnions tells the generator about the other unions of the same package, so
// it can find the one every answer belongs to. Without it no DispatchX or
// NewXClient is generated, and an answers= option is an error.
func (g *HandlerGenerator) WithUnions(unions []*shape.UnionLike) *HandlerGenerator {
	g.siblings = unions
	return g
}

// answersOption is the tag option that names the answer union: mkunion:"Query,answers=Reply".
const answersOption = "answers="

// handlerVariant is one union variant with its declared answer type.
// answer is empty when the variant does not embed f.Returns.
type handlerVariant struct {
	name   string // GetUser
	typ    string // GetUser, as written in this package
	answer string // *User, as written in this package; "" for none
	shape  shape.Shape
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
				hv.shape = r
				break
			}
		}
		result = append(result, hv)
	}

	return result, nil
}

// answerUnion is the union every variant's answer is a variant of, or nil.
// With the answers= option it is that union, and a variant that does not
// answer with one of its variants is an error. Without the option it is
// inferred, and silence is the answer when the variants do not agree.
func (g *HandlerGenerator) answerUnion(variants []handlerVariant) (*shape.UnionLike, error) {
	named := ""
	if tag, ok := g.union.Tags[shape.TagUnionName]; ok {
		for _, opt := range tag.Options {
			if strings.HasPrefix(opt, answersOption) {
				named = strings.TrimPrefix(opt, answersOption)
			}
		}
	}
	if named != "" {
		var answers *shape.UnionLike
		for _, u := range g.siblings {
			if u.Name == named && u.PkgImportName == g.union.PkgImportName {
				answers = u
			}
		}
		if answers == nil {
			return nil, fmt.Errorf("union %s: answers=%s names no union in package %s; the answer union must be declared in the same package", g.union.Name, named, shape.ToGoPkgName(g.union))
		}
		for _, v := range variants {
			if v.shape == nil {
				return nil, fmt.Errorf("union %s: variant %s has no f.Returns, but answers=%s needs every variant to answer with a %s", g.union.Name, v.name, named, named)
			}
			if !isVariantOf(v.shape, answers) {
				return nil, fmt.Errorf("union %s: variant %s answers with %s, which is not a variant of %s", g.union.Name, v.name, v.answer, named)
			}
		}
		return answers, nil
	}
	var found *shape.UnionLike
	for _, v := range variants {
		if v.shape == nil {
			return nil, nil
		}
		var owner *shape.UnionLike
		for _, u := range g.siblings {
			if u != g.union && isVariantOf(v.shape, u) {
				owner = u
			}
		}
		if owner == nil || (found != nil && found != owner) {
			return nil, nil
		}
		found = owner
	}
	return found, nil
}

// isVariantOf reports whether answer is *V for a variant V of u.
func isVariantOf(answer shape.Shape, u *shape.UnionLike) bool {
	ptr, ok := answer.(*shape.PointerLike)
	if !ok {
		return false
	}
	ref, ok := ptr.Type.(*shape.RefName)
	if !ok {
		return false
	}
	for _, v := range u.Variant {
		st, ok := v.(*shape.StructLike)
		if ok && st.Name == ref.Name && st.PkgImportName == ref.PkgImportName {
			return true
		}
	}
	return false
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

	funcs := lowerFirst(name) + "HandlerFuncs"
	fmt.Fprintf(out, "// New%sHandler builds a %sHandler from one function per variant: the arms\n", name, name)
	fmt.Fprintf(out, "// of Handle%s as a value. Exhaustive: every arm must be given. Use it where a\n", name)
	fmt.Fprintf(out, "// handler is a value, such as effect.Interpret, or a field of a struct that\n")
	fmt.Fprintf(out, "// embeds the handlers of several packages.\n")
	fmt.Fprintf(out, "func New%sHandler(\n", name)
	for _, v := range variants {
		fmt.Fprintf(out, "\ton%s func(ctx context.Context, op *%s) %s,\n", v.name, v.typ, v.results())
	}
	fmt.Fprintf(out, ") %sHandler {\n", name)
	fmt.Fprintf(out, "\treturn &%s{", funcs)
	for i, v := range variants {
		if i > 0 {
			fmt.Fprintf(out, ", ")
		}
		fmt.Fprintf(out, "on%s: on%s", v.name, v.name)
	}
	fmt.Fprintf(out, "}\n")
	fmt.Fprintf(out, "}\n\n")

	fmt.Fprintf(out, "type %s struct {\n", funcs)
	for _, v := range variants {
		fmt.Fprintf(out, "\ton%s func(ctx context.Context, op *%s) %s\n", v.name, v.typ, v.results())
	}
	fmt.Fprintf(out, "}\n\n")
	for _, v := range variants {
		fmt.Fprintf(out, "func (h *%s) Handle%s(ctx context.Context, op *%s) %s {\n", funcs, v.name, v.typ, v.results())
		fmt.Fprintf(out, "\treturn h.on%s(ctx, op)\n", v.name)
		fmt.Fprintf(out, "}\n\n")
	}

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

	g.generateAround(out, variants)

	answers, err := g.answerUnion(variants)
	if err != nil {
		return nil, fmt.Errorf("generators.HandlerGenerator.Generate: %w", err)
	}
	if answers != nil {
		g.generateRPC(out, variants, answers)
	}

	return out.Bytes(), nil
}

// generateAround emits AroundX: one function that decorates every method of a
// handler, for logging, metrics or deadlines that do not care which variant.
func (g *HandlerGenerator) generateAround(out *bytes.Buffer, variants []handlerVariant) {
	name := g.union.Name
	fmt.Fprintf(out, "// Around%s decorates every method of next with around. around gets the\n", name)
	fmt.Fprintf(out, "// operation as the %s union and a call that runs it on next, and returns the\n", name)
	fmt.Fprintf(out, "// answer untyped; Around%s casts it back to the variant's declared type. One\n", name)
	fmt.Fprintf(out, "// function wraps all variants: logging, metrics, deadlines.\n")
	fmt.Fprintf(out, "func Around%s(next %sHandler, around func(ctx context.Context, op %s, call func(ctx context.Context) (any, error)) (any, error)) %sHandler {\n", name, name, name, name)
	fmt.Fprintf(out, "\treturn New%sHandler(\n", name)
	for _, v := range variants {
		fmt.Fprintf(out, "\t\tfunc(ctx context.Context, op *%s) %s {\n", v.typ, v.results())
		if v.answer == "" {
			fmt.Fprintf(out, "\t\t\t_, err := around(ctx, op, func(ctx context.Context) (any, error) { return nil, next.Handle%s(ctx, op) })\n", v.name)
			fmt.Fprintf(out, "\t\t\treturn err\n")
		} else {
			fmt.Fprintf(out, "\t\t\tout, err := around(ctx, op, func(ctx context.Context) (any, error) { return next.Handle%s(ctx, op) })\n", v.name)
			fmt.Fprintf(out, "\t\t\tanswer, _ := out.(%s)\n", v.answer)
			fmt.Fprintf(out, "\t\t\treturn answer, err\n")
		}
		fmt.Fprintf(out, "\t\t},\n")
	}
	fmt.Fprintf(out, "\t)\n")
	fmt.Fprintf(out, "}\n\n")
}

// generateRPC emits the request/answer pairing: DispatchX for the server,
// NewXClient for the client, XUnexpectedAnswer for an answer of the wrong variant.
func (g *HandlerGenerator) generateRPC(out *bytes.Buffer, variants []handlerVariant, answers *shape.UnionLike) {
	name := g.union.Name
	root := shape.WithRootPkgName(shape.ToGoPkgName(g.union))
	ans := shape.ToGoTypeName(answers, root)
	pkg := shape.ToGoPkgName(g.union)

	fmt.Fprintf(out, "// Dispatch%s runs op on h and returns the answer as a %s: every %s\n", name, ans, name)
	fmt.Fprintf(out, "// variant answers with a %s variant, so the arms share one type. It is the\n", ans)
	fmt.Fprintf(out, "// server side of the %s contract; New%sClient is the client side.\n", name, name)
	fmt.Fprintf(out, "func Dispatch%s(ctx context.Context, h %sHandler, op %s) (%s, error) {\n", name, name, name, ans)
	fmt.Fprintf(out, "\treturn %s[%s, error](op,\n", MatchUnionFuncName(g.union, 2), ans)
	for _, v := range variants {
		fmt.Fprintf(out, "\t\tfunc(x *%s) (%s, error) { return h.Handle%s(ctx, x) },\n", v.typ, ans, v.name)
	}
	fmt.Fprintf(out, "\t)\n")
	fmt.Fprintf(out, "}\n\n")

	client := lowerFirst(name) + "Client"
	fmt.Fprintf(out, "// New%sClient builds the %sHandler whose every method sends the operation\n", name, name)
	fmt.Fprintf(out, "// through send and expects the answer the variant declared with f.Returns; any\n")
	fmt.Fprintf(out, "// other %s comes back as a *%sUnexpectedAnswer. send is the transport: an\n", ans, name)
	fmt.Fprintf(out, "// HTTP post, or Dispatch%s bound to a handler for an in-process client.\n", name)
	fmt.Fprintf(out, "func New%sClient(send func(ctx context.Context, op %s) (%s, error)) %sHandler {\n", name, name, ans, name)
	fmt.Fprintf(out, "\treturn &%s{send: send}\n", client)
	fmt.Fprintf(out, "}\n\n")
	fmt.Fprintf(out, "type %s struct {\n", client)
	fmt.Fprintf(out, "\tsend func(ctx context.Context, op %s) (%s, error)\n", name, ans)
	fmt.Fprintf(out, "}\n\n")
	for _, v := range variants {
		fmt.Fprintf(out, "func (c *%s) Handle%s(ctx context.Context, op *%s) (%s, error) {\n", client, v.name, v.typ, v.answer)
		fmt.Fprintf(out, "\tvar zero %s\n", v.answer)
		fmt.Fprintf(out, "\tres, err := c.send(ctx, op)\n")
		fmt.Fprintf(out, "\tif err != nil {\n")
		fmt.Fprintf(out, "\t\treturn zero, err\n")
		fmt.Fprintf(out, "\t}\n")
		fmt.Fprintf(out, "\tif answer, ok := res.(%s); ok {\n", v.answer)
		fmt.Fprintf(out, "\t\treturn answer, nil\n")
		fmt.Fprintf(out, "\t}\n")
		fmt.Fprintf(out, "\treturn zero, &%sUnexpectedAnswer{%s: op, %s: res}\n", name, name, ans)
		fmt.Fprintf(out, "}\n\n")
	}

	fmt.Fprintf(out, "// %sUnexpectedAnswer is the error New%sClient returns when the answer is a\n", name, name)
	fmt.Fprintf(out, "// %s variant other than the one the operation declared. Unwrap returns the\n", ans)
	fmt.Fprintf(out, "// %s when it is an error itself, so errors.As reaches it.\n", ans)
	fmt.Fprintf(out, "type %sUnexpectedAnswer struct {\n", name)
	fmt.Fprintf(out, "\t%s %s\n", name, name)
	fmt.Fprintf(out, "\t%s %s\n", ans, ans)
	fmt.Fprintf(out, "}\n\n")
	fmt.Fprintf(out, "func (e *%sUnexpectedAnswer) Error() string {\n", name)
	fmt.Fprintf(out, "\tif err, ok := e.%s.(error); ok {\n", ans)
	fmt.Fprintf(out, "\t\treturn fmt.Sprintf(\"%s: %%T answered with %%T: %%v\", e.%s, e.%s, err)\n", pkg, name, ans)
	fmt.Fprintf(out, "\t}\n")
	fmt.Fprintf(out, "\treturn fmt.Sprintf(\"%s: %%T answered with %%T\", e.%s, e.%s)\n", pkg, name, ans)
	fmt.Fprintf(out, "}\n\n")
	fmt.Fprintf(out, "func (e *%sUnexpectedAnswer) Unwrap() error {\n", name)
	fmt.Fprintf(out, "\tif err, ok := e.%s.(error); ok {\n", ans)
	fmt.Fprintf(out, "\t\treturn err\n")
	fmt.Fprintf(out, "\t}\n")
	fmt.Fprintf(out, "\treturn nil\n")
	fmt.Fprintf(out, "}\n\n")
}

// lowerFirst is the unexported spelling of an exported name.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
