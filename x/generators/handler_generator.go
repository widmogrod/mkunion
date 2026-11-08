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
// Every handler union also gets a decorator and a name table:
//
//	func AroundQuery(next QueryHandler, around func(ctx, op Query, call func(ctx) (any, error)) (any, error)) QueryHandler
//	func QueryNames() []string                       // "query.GetUser", ... in declaration order
//	func NewQueryByName(name string) (Query, bool)   // the zero *GetUser for "query.GetUser" or "GetUser"
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
// Two more answer forms let a variant refuse with a value instead of an error:
//
//	Lookup struct{ f.Returns[f.Result[*Page, LookupRefusal]] }  // answers *Page, refuses with a LookupRefusal
//	Scan   struct{ f.Returns[ScanAnswer] }                      // answers with any ScanAnswer variant
//
// In both, every leaf must be a variant of the answer union, so it has a place
// on the wire: *Page is a Reply variant, and LookupRefusal and ScanAnswer are
// unions of the same file whose variants are Reply variants, usually through
// type aliases (type LookupMissing = Missing) because a struct is declared in
// one union block only. DispatchX unwraps the Result or the union onto the
// answer union; NewXClient rebuilds it from the Reply that comes back.
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
	// result is set when the answer is f.Result[A, E]: A and E as shapes and
	// as written in this package, and the Ok and Err types of the pair.
	result *resultAnswer
}

// resultAnswer is an answer of the form f.Result[A, E].
type resultAnswer struct {
	ok, refuse       shape.Shape
	okTyp, refuseTyp string
	okCase, errCase  string // *f.Ok[A, E], *f.Err[A, E]
}

const (
	resultName          = "Result"
	resultPkgImportName = "github.com/widmogrod/mkunion/f"
)

// resultOf reports whether answer is f.Result[A, E], and gives A and E.
func resultOf(answer shape.Shape) (a, e shape.Shape, ok bool) {
	ref, isRef := answer.(*shape.RefName)
	if !isRef || ref.Name != resultName || ref.PkgImportName != resultPkgImportName || len(ref.Indexed) != 2 {
		return nil, nil, false
	}
	return ref.Indexed[0], ref.Indexed[1], true
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
				if a, e, ok := resultOf(r); ok {
					ref := r.(*shape.RefName)
					okRef := &shape.RefName{Name: "Ok", PkgName: ref.PkgName, PkgImportName: ref.PkgImportName, Indexed: ref.Indexed}
					errRef := &shape.RefName{Name: "Err", PkgName: ref.PkgName, PkgImportName: ref.PkgImportName, Indexed: ref.Indexed}
					hv.result = &resultAnswer{
						ok: a, refuse: e,
						okTyp: shape.ToGoTypeName(a, root), refuseTyp: shape.ToGoTypeName(e, root),
						okCase: "*" + shape.ToGoTypeName(okRef, root), errCase: "*" + shape.ToGoTypeName(errRef, root),
					}
				}
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
		return g.namedAnswerUnion(named, variants)
	}
	return g.inferAnswerUnion(variants), nil
}

// namedAnswerUnion is the answers= union, after checking that every variant
// answers with one of its variants.
func (g *HandlerGenerator) namedAnswerUnion(named string, variants []handlerVariant) (*shape.UnionLike, error) {
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
		if err := g.checkAnswer(v, answers, named); err != nil {
			return nil, err
		}
	}
	return answers, nil
}

// checkAnswer is the error when v does not answer with a variant of answers:
// a plain *V must be a variant, a union answer must have only variants of
// answers as leaves, and an f.Result must hold both.
func (g *HandlerGenerator) checkAnswer(v handlerVariant, answers *shape.UnionLike, named string) error {
	if v.shape == nil {
		return fmt.Errorf("union %s: variant %s has no f.Returns, but answers=%s needs every variant to answer with a %s", g.union.Name, v.name, named, named)
	}
	if v.result != nil {
		if !isVariantOf(v.result.ok, answers) {
			return fmt.Errorf("union %s: variant %s answers with %s, whose Ok type %s is not a variant of %s", g.union.Name, v.name, v.answer, v.result.okTyp, named)
		}
		refusal := g.sibling(v.result.refuse)
		if refusal == nil {
			return fmt.Errorf("union %s: variant %s refuses with %s, which is not a union of this file", g.union.Name, v.name, v.result.refuseTyp)
		}
		if bad := variantNotIn(refusal, answers); bad != "" {
			return fmt.Errorf("union %s: variant %s refuses with %s, whose variant %s is not a variant of %s", g.union.Name, v.name, v.result.refuseTyp, bad, named)
		}
		return nil
	}
	if u := g.sibling(v.shape); u != nil {
		if bad := variantNotIn(u, answers); bad != "" {
			return fmt.Errorf("union %s: variant %s answers with %s, whose variant %s is not a variant of %s", g.union.Name, v.name, v.answer, bad, named)
		}
		return nil
	}
	if !isVariantOf(v.shape, answers) {
		return fmt.Errorf("union %s: variant %s answers with %s, which is not a variant of %s", g.union.Name, v.name, v.answer, named)
	}
	return nil
}

// inferAnswerUnion is the one sibling union every variant answers into, or
// nil when a variant has no answer or the variants do not agree.
func (g *HandlerGenerator) inferAnswerUnion(variants []handlerVariant) *shape.UnionLike {
	var found *shape.UnionLike
	for _, v := range variants {
		if v.shape == nil {
			return nil
		}
		owner := g.ownerOf(v)
		if owner == nil || (found != nil && found != owner) {
			return nil
		}
		found = owner
	}
	return found
}

// ownerOf is the sibling union v answers into, or nil.
func (g *HandlerGenerator) ownerOf(v handlerVariant) *shape.UnionLike {
	var owner *shape.UnionLike
	for _, u := range g.siblings {
		if u == g.union {
			continue
		}
		switch {
		case v.result != nil:
			refusal := g.sibling(v.result.refuse)
			if isVariantOf(v.result.ok, u) && refusal != nil && refusal != u && variantNotIn(refusal, u) == "" {
				owner = u
			}
		case g.sibling(v.shape) != nil:
			if a := g.sibling(v.shape); a != u && variantNotIn(a, u) == "" {
				owner = u
			}
		case isVariantOf(v.shape, u):
			owner = u
		}
	}
	return owner
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
	return hasLeaf(u, ref.Name, ref.PkgImportName)
}

// sibling is the union of this file that answer names, or nil.
func (g *HandlerGenerator) sibling(answer shape.Shape) *shape.UnionLike {
	ref, ok := answer.(*shape.RefName)
	if !ok {
		return nil
	}
	for _, u := range g.siblings {
		if u.Name == ref.Name && u.PkgImportName == ref.PkgImportName {
			return u
		}
	}
	return nil
}

// leafOf is the struct a variant stands for: itself, or for a type alias
// (type LookupMissing = Missing) the struct it aliases. A struct is declared
// in one union block, so an alias is how one struct is a variant of two unions.
func leafOf(v shape.Shape) (name, pkgImportName string, ok bool) {
	switch x := v.(type) {
	case *shape.StructLike:
		return x.Name, x.PkgImportName, true
	case *shape.AliasLike:
		if ref, isRef := x.Type.(*shape.RefName); isRef && x.IsAlias {
			return ref.Name, ref.PkgImportName, true
		}
	}
	return "", "", false
}

// hasLeaf reports whether u has a variant that is, or aliases, the named struct.
func hasLeaf(u *shape.UnionLike, name, pkgImportName string) bool {
	for _, v := range u.Variant {
		if n, p, ok := leafOf(v); ok && n == name && p == pkgImportName {
			return true
		}
	}
	return false
}

// variantNotIn names the first variant of sub whose leaf is not a variant of
// u, or "" when every variant of sub has a place in u.
func variantNotIn(sub, u *shape.UnionLike) string {
	for _, v := range sub.Variant {
		n, p, ok := leafOf(v)
		if !ok || !hasLeaf(u, n, p) {
			return shape.ToGoTypeName(v, shape.WithRootPkgName(shape.ToGoPkgName(sub)))
		}
	}
	return ""
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
	g.generateByName(out, variants)

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

// generateByName emits XNames and NewXByName: the variants' $type names in
// declaration order, and the zero value of the variant with a given name, for
// a transport that picks the variant from a URL rather than from the body
// (see x/rpc.Mount), and for a server's catalogue of what it accepts.
func (g *HandlerGenerator) generateByName(out *bytes.Buffer, variants []handlerVariant) {
	name := g.union.Name
	pkg := shape.ToGoPkgName(g.union)
	fmt.Fprintf(out, "// %sNames lists the $type name of every %s variant, in declaration order:\n", name, name)
	fmt.Fprintf(out, "// what a server accepts, and the routes of a per-variant transport.\n")
	fmt.Fprintf(out, "func %sNames() []string {\n", name)
	fmt.Fprintf(out, "\treturn []string{")
	for i, v := range variants {
		if i > 0 {
			fmt.Fprintf(out, ", ")
		}
		fmt.Fprintf(out, "%q", pkg+"."+v.name)
	}
	fmt.Fprintf(out, "}\n")
	fmt.Fprintf(out, "}\n\n")
	fmt.Fprintf(out, "// New%sByName returns the zero value of the variant whose $type name is name,\n", name)
	fmt.Fprintf(out, "// with or without the package prefix (%q or %q), and false for\n", pkg+"."+variants[0].name, variants[0].name)
	fmt.Fprintf(out, "// a name that is not a %s. A transport decodes a bare payload into it.\n", name)
	fmt.Fprintf(out, "func New%sByName(name string) (%s, bool) {\n", name, name)
	fmt.Fprintf(out, "\tswitch name {\n")
	for _, v := range variants {
		fmt.Fprintf(out, "\tcase %q, %q:\n", pkg+"."+v.name, v.name)
		fmt.Fprintf(out, "\t\treturn &%s{}, true\n", v.typ)
	}
	fmt.Fprintf(out, "\t}\n")
	fmt.Fprintf(out, "\treturn nil, false\n")
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
		switch {
		case v.result != nil:
			// Ok carries the answer variant; Err carries a refusal union whose
			// every variant is an answer variant, so its Match lifts it.
			fmt.Fprintf(out, "\t\tfunc(x *%s) (%s, error) {\n", v.typ, ans)
			fmt.Fprintf(out, "\t\t\tres, err := h.Handle%s(ctx, x)\n", v.name)
			fmt.Fprintf(out, "\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n")
			fmt.Fprintf(out, "\t\t\tswitch r := res.(type) {\n")
			fmt.Fprintf(out, "\t\t\tcase %s:\n\t\t\t\treturn r.Value, nil\n", v.result.okCase)
			fmt.Fprintf(out, "\t\t\tcase %s:\n", v.result.errCase)
			fmt.Fprintf(out, "\t\t\t\tif r.Error == nil {\n\t\t\t\t\treturn nil, fmt.Errorf(\"%s: %s refused with a nil %s\")\n\t\t\t\t}\n", pkg, v.name, v.result.refuseTyp)
			fmt.Fprintf(out, "\t\t\t\treturn %s, nil\n", g.liftUnion(g.sibling(v.result.refuse), "r.Error", ans, root))
			fmt.Fprintf(out, "\t\t\t}\n")
			fmt.Fprintf(out, "\t\t\treturn nil, fmt.Errorf(\"%s: %s answered with %%T, want %s\", res)\n", pkg, v.name, v.answer)
			fmt.Fprintf(out, "\t\t},\n")
		case g.sibling(v.shape) != nil:
			fmt.Fprintf(out, "\t\tfunc(x *%s) (%s, error) {\n", v.typ, ans)
			fmt.Fprintf(out, "\t\t\tres, err := h.Handle%s(ctx, x)\n", v.name)
			fmt.Fprintf(out, "\t\t\tif err != nil {\n\t\t\t\treturn nil, err\n\t\t\t}\n")
			fmt.Fprintf(out, "\t\t\tif res == nil {\n\t\t\t\treturn nil, fmt.Errorf(\"%s: %s answered with a nil %s\")\n\t\t\t}\n", pkg, v.name, v.answer)
			fmt.Fprintf(out, "\t\t\treturn %s, nil\n", g.liftUnion(g.sibling(v.shape), "res", ans, root))
			fmt.Fprintf(out, "\t\t},\n")
		default:
			fmt.Fprintf(out, "\t\tfunc(x *%s) (%s, error) { return h.Handle%s(ctx, x) },\n", v.typ, ans, v.name)
		}
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
		if v.result != nil {
			fmt.Fprintf(out, "\tif answer, ok := res.(%s); ok {\n", v.result.okTyp)
			fmt.Fprintf(out, "\t\treturn &%s{Value: answer}, nil\n", strings.TrimPrefix(v.result.okCase, "*"))
			fmt.Fprintf(out, "\t}\n")
			fmt.Fprintf(out, "\tif refusal, ok := res.(%s); ok {\n", v.result.refuseTyp)
			fmt.Fprintf(out, "\t\treturn &%s{Error: refusal}, nil\n", strings.TrimPrefix(v.result.errCase, "*"))
			fmt.Fprintf(out, "\t}\n")
		} else {
			fmt.Fprintf(out, "\tif answer, ok := res.(%s); ok {\n", v.answer)
			fmt.Fprintf(out, "\t\treturn answer, nil\n")
			fmt.Fprintf(out, "\t}\n")
		}
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

// liftUnion is the expression that turns value, a sub of the answer union
// ans (every variant of sub is a variant of ans), into an ans: the generated
// Match of sub with one identity arm per variant, so the compiler checks that
// each variant is an ans.
func (g *HandlerGenerator) liftUnion(sub *shape.UnionLike, value, ans string, root shape.ToGoTypeNameOptionFunc) string {
	arms := make([]string, 0, len(sub.Variant))
	for _, v := range sub.Variant {
		arms = append(arms, fmt.Sprintf("func(x *%s) %s { return x }", shape.ToGoTypeName(v, root), ans))
	}
	return fmt.Sprintf("%s[%s](%s, %s)", MatchUnionFuncName(sub, 1), ans, value, strings.Join(arms, ", "))
}

// lowerFirst is the unexported spelling of an exported name.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
