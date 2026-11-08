package shape

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

// TypeScriptRenderer writes TypeScript for the shapes it is given: one file
// per Go package, named after the package (api → api.ts), with one exported
// type per shape, plus mkunion.ts with what every file shares.
//
// TypeScript has no decoder of its own: JSON.parse returns the value and the
// types only describe it. So the declarations say exactly what the generated
// Go serde writes (docs/examples/json.md): a union is
// {"$type": "api.X", "api.X": {...}} with both keys present; struct fields use
// their json tag names; an omitempty field is optional; a nil pointer or
// union is null, so such a value is `T | null`; time.Time is an RFC 3339
// string, []byte a base64 string or null, map keys strings, any is unknown.
//
// Two encoders write Go structs and they disagree on nil slices and maps: the
// generated serde, which writes every union variant, every struct that holds
// a union and every type tagged serde:"json", writes [] and {}; encoding/json,
// which writes the other structs, writes null. The types follow the encoder
// of each struct, so a slice is `T[]` in the first and `T[] | null` in the
// second. A hand-written MarshalJSON is not seen. What the Go decoder also
// accepts on input (no $type, a missing or null payload) is wider than these
// types and is not described.
//
// A variant that embeds f.Returns[R] is listed in the union's <Union>Answer
// and <Union>Payload tables; when every answer is a variant of one union, a
// send() overload is emitted that wraps the payload as the request union,
// hands it to a transport and unwraps the typed answer.
//
// A type named Record or Promise gets a trailing underscore: those are the
// only globals the generated types use. Every other Go name is kept, Error
// included, since a module is a namespace (import * as api from "./api";
// api.Error) and the prelude is imported as one too (mkunion.RFC3339).
type TypeScriptRenderer struct {
	packages map[string]*tsPackage // by package import name
	known    map[string]Shape      // by "importName.Name", every shape seen, variants included
	added    map[string]bool
	errs     []string
}

type tsPackage struct {
	name       string
	importName string
	unions     []*UnionLike
	structs    []*StructLike
	aliases    []*AliasLike
}

const tsPreludeFile = "mkunion"

func NewTypeScriptRenderer() *TypeScriptRenderer {
	return &TypeScriptRenderer{
		packages: map[string]*tsPackage{},
		known:    map[string]Shape{},
		added:    map[string]bool{},
	}
}

// AddShape queues a named shape and everything it refers to. Union variants
// are rendered with their union, so adding a variant adds the union.
func (r *TypeScriptRenderer) AddShape(x Shape) {
	if x == nil {
		return
	}
	key := ToGoTypeName(x, WithPkgImportName())
	if r.added[key] {
		return
	}
	r.added[key] = true

	switch y := x.(type) {
	case *UnionLike:
		pkg := r.pkg(y.PkgName, y.PkgImportName)
		pkg.unions = append(pkg.unions, y)
		r.know(y)
		for _, v := range y.Variant {
			r.know(v)
			r.added[ToGoTypeName(v, WithPkgImportName())] = true
		}
	case *StructLike:
		if r.addUnionOfVariant(y) {
			return
		}
		pkg := r.pkg(y.PkgName, y.PkgImportName)
		pkg.structs = append(pkg.structs, y)
		r.know(y)
	case *AliasLike:
		if r.addUnionOfVariant(y) {
			return
		}
		pkg := r.pkg(y.PkgName, y.PkgImportName)
		pkg.aliases = append(pkg.aliases, y)
		r.know(y)
	case *PointerLike:
		r.AddShape(y.Type)
		return
	default:
		log.Infof("totypescript: AddShape: %T is not a named type, skipped", x)
		return
	}

	r.FollowRef(x)
}

// addUnionOfVariant adds the union a variant belongs to and reports whether
// the shape was a variant.
func (r *TypeScriptRenderer) addUnionOfVariant(x Shape) bool {
	ref := RetrieveVariantTypeRef(x)
	if ref == nil {
		return false
	}
	if union, ok := r.resolve(ref); ok {
		r.AddShape(union)
	} else {
		r.errs = append(r.errs, fmt.Sprintf("totypescript: %s is a variant of %s.%s, which was not found", Name(x), ref.PkgName, ref.Name))
	}
	return true
}

// FollowRef adds the shapes x refers to. Builtins, time.Time and the
// f.Returns marker are not shapes to render.
func (r *TypeScriptRenderer) FollowRef(x Shape) {
	for _, ref := range ExtractRefs(x) {
		if !r.renderable(ref) {
			continue
		}
		if s, ok := r.resolve(ref); ok {
			r.AddShape(s)
		} else {
			log.Debugf("totypescript: FollowRef: %s not found", ToGoTypeName(ref))
		}
	}
}

func (r *TypeScriptRenderer) renderable(ref *RefName) bool {
	switch ref.PkgImportName {
	case "", "time", returnsPkgImportName:
		return false
	}
	return true
}

func (r *TypeScriptRenderer) pkg(name, importName string) *tsPackage {
	if p, ok := r.packages[importName]; ok {
		return p
	}
	p := &tsPackage{name: name, importName: importName}
	r.packages[importName] = p
	return p
}

func (r *TypeScriptRenderer) know(x Shape) {
	r.known[knownKey(ToGoPkgImportName(x), Name(x))] = x
}

// resolve finds the shape a reference points to: among the shapes already
// seen, else on disk.
func (r *TypeScriptRenderer) resolve(ref *RefName) (Shape, bool) {
	if s, ok := r.known[knownKey(ref.PkgImportName, ref.Name)]; ok {
		return s, true
	}
	s, ok := LookupShapeOnDisk(ref)
	if ok {
		r.know(s)
	}
	return s, ok
}

// Render returns the TypeScript files by name.
func (r *TypeScriptRenderer) Render() (map[string]string, error) {
	files := map[string]string{}
	if len(r.packages) == 0 {
		return files, nil
	}

	importNames := make([]string, 0, len(r.packages))
	for importName := range r.packages {
		importNames = append(importNames, importName)
	}
	sort.Strings(importNames)

	fileOf := map[string]string{tsPreludeFile + ".ts": "the mkunion prelude"}
	for _, importName := range importNames {
		pkg := r.packages[importName]
		fileName := pkg.name + ".ts"
		if other, taken := fileOf[fileName]; taken {
			r.errs = append(r.errs, fmt.Sprintf("totypescript: packages %s and %s both render as %s", other, importName, fileName))
			continue
		}
		fileOf[fileName] = importName

		w := &tsWriter{r: r, pkg: pkg, imports: map[string]string{}, prelude: map[string]bool{}}
		files[fileName] = w.renderPackage()
	}
	files[tsPreludeFile+".ts"] = tsPrelude

	if len(r.errs) > 0 {
		return files, fmt.Errorf("%s", strings.Join(r.errs, "\n"))
	}
	return files, nil
}

// WriteToDir renders and writes the files into dir, creating it when needed.
func (r *TypeScriptRenderer) WriteToDir(dir string) error {
	files, err := r.Render()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("totypescript: WriteToDir: %w", err)
	}
	for name, content := range files {
		if err := os.WriteFile(path.Join(dir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("totypescript: WriteToDir %s: %w", name, err)
		}
	}
	return nil
}

// tsWriter renders one package.
type tsWriter struct {
	r       *TypeScriptRenderer
	pkg     *tsPackage
	b       strings.Builder
	imports map[string]string // Go packages this file refers to: name → import name
	prelude map[string]bool   // names used from mkunion.ts
}

func (w *tsWriter) line(indent int, format string, args ...any) {
	w.b.WriteString(strings.Repeat("  ", indent))
	fmt.Fprintf(&w.b, format, args...)
	w.b.WriteString("\n")
}

func (w *tsWriter) blank() {
	w.b.WriteString("\n")
}

func (w *tsWriter) fail(format string, args ...any) {
	w.r.errs = append(w.r.errs, "totypescript: "+fmt.Sprintf(format, args...))
}

func (w *tsWriter) renderPackage() string {
	sort.Slice(w.pkg.unions, func(i, j int) bool { return w.pkg.unions[i].Name < w.pkg.unions[j].Name })
	sort.Slice(w.pkg.structs, func(i, j int) bool { return w.pkg.structs[i].Name < w.pkg.structs[j].Name })
	sort.Slice(w.pkg.aliases, func(i, j int) bool { return w.pkg.aliases[i].Name < w.pkg.aliases[j].Name })

	w.line(0, "/** Go package %s (%s). */", w.pkg.name, w.pkg.importName)

	for _, union := range w.pkg.unions {
		w.blank()
		w.renderUnion(union)
		for _, v := range union.Variant {
			w.blank()
			switch y := v.(type) {
			case *StructLike:
				w.renderStruct(y)
			case *AliasLike:
				w.renderAlias(y)
			default:
				w.fail("%s.%s has a variant of kind %T, which has no TypeScript form", union.PkgName, union.Name, v)
			}
		}
	}
	for _, st := range w.pkg.structs {
		w.blank()
		w.renderStruct(st)
	}
	for _, alias := range w.pkg.aliases {
		w.blank()
		w.renderAlias(alias)
	}

	var sendOps []tsOperation
	for _, union := range w.pkg.unions {
		if UnionDeclaresReturns(union) {
			w.blank()
			sendOps = append(sendOps, w.renderOperations(union)...)
		}
	}
	if len(sendOps) > 0 {
		w.blank()
		w.renderSend(sendOps)
	}

	// The imports are known only now.
	var header strings.Builder
	header.WriteString("// Code generated by mkunion. DO NOT EDIT.\n")
	header.WriteString("/* eslint-disable */\n")
	if len(w.imports) > 0 || len(w.prelude) > 0 {
		header.WriteString("\n")
	}
	// The prelude is a namespace too, so its names cannot clash with the package's; a
	// type-only import when nothing from it runs.
	switch {
	case w.prelude["UnexpectedAnswer"]:
		fmt.Fprintf(&header, "import * as %s from \"./%s\";\n", tsPreludeFile, tsPreludeFile)
	case len(w.prelude) > 0:
		fmt.Fprintf(&header, "import type * as %s from \"./%s\";\n", tsPreludeFile, tsPreludeFile)
	}
	importedPkgs := make([]string, 0, len(w.imports))
	for name := range w.imports {
		importedPkgs = append(importedPkgs, name)
	}
	sort.Strings(importedPkgs)
	for _, name := range importedPkgs {
		fmt.Fprintf(&header, "import type * as %s from \"./%s\";\n", name, name)
	}
	header.WriteString("\n")

	return header.String() + w.b.String()
}

func (w *tsWriter) renderAlias(x *AliasLike) {
	// The null a nil value of the alias may be is written where the alias is used.
	w.line(0, "export type %s%s = %s;", tsTypeName(x.Name), tsTypeParams(x.TypeParams), w.coreOf(x.Type, !w.aliasEncodedByMkunion(x)))
}

func (w *tsWriter) renderUnion(x *UnionLike) {
	if len(x.Variant) == 0 {
		w.fail("%s.%s has no variants", x.PkgName, x.Name)
		return
	}
	first := x.PkgName + "." + Name(x.Variant[0])
	w.line(0, "/** %s is a union: one object per variant, on the wire as {\"$type\": %q, %q: {...}}. Go writes both keys. */", x.Name, first, first)
	w.line(0, "export type %s%s =", tsTypeName(x.Name), tsTypeParams(x.TypeParams))
	for i, v := range x.Variant {
		end := ""
		if i == len(x.Variant)-1 {
			end = ";"
		}
		w.line(1, "| { \"$type\": %q; %q: %s }%s", x.PkgName+"."+Name(v), x.PkgName+"."+Name(v), w.variantType(v), end)
	}
}

// variantType is a variant as the union names it, with its own type parameters.
func (w *tsWriter) variantType(v Shape) string {
	switch y := v.(type) {
	case *StructLike:
		return w.localName(y.PkgName, y.PkgImportName, y.Name) + tsTypeParams(y.TypeParams)
	case *AliasLike:
		return w.localName(y.PkgName, y.PkgImportName, y.Name) + tsTypeParams(y.TypeParams)
	}
	return w.typeOf(v, false)
}

// tsField is a struct field as TypeScript sees it.
type tsField struct {
	name     string // the json name, quoted when it is not an identifier
	typ      string
	optional bool
	desc     *string
}

func (w *tsWriter) fieldsOf(x *StructLike) []tsField {
	var fields []tsField
	native := !w.encodedByMkunion(x)
	for _, field := range x.Fields {
		if IsPhantomField(field) {
			continue
		}
		jsonName := TagGetValue(field.Tags, "json", field.Name)
		if jsonName == "-" {
			continue
		}
		if st, inline := field.Type.(*StructLike); inline && st.Name == "" {
			w.fail("%s.%s.%s is an inline struct; name the type", x.PkgName, x.Name, field.Name)
			continue
		}
		f := tsField{
			name:     tsPropertyName(jsonName),
			desc:     field.Desc,
			optional: TagHasOption(field.Tags, "json", "omitempty") || TagHasOption(field.Tags, "json", "omitzero"),
		}
		if f.optional {
			// Go leaves a nil pointer, union, slice or map out, so when the key is there the value is not null.
			f.typ = w.coreOf(field.Type, native)
		} else {
			f.typ = w.typeOf(field.Type, native)
		}
		fields = append(fields, f)
	}
	return fields
}

func (w *tsWriter) renderStruct(x *StructLike) {
	fields := w.fieldsOf(x)
	name := tsTypeName(x.Name) + tsTypeParams(x.TypeParams)
	if len(fields) == 0 {
		w.line(0, "/** %s has no fields; Go writes {}. */", x.Name)
		w.line(0, "export type %s = Record<string, never>;", name)
		return
	}
	w.line(0, "export type %s = {", name)
	for _, f := range fields {
		if f.desc != nil && *f.desc != "" {
			w.line(1, "/** %s */", *f.desc)
		}
		if f.optional {
			w.line(1, "%s?: %s;", f.name, f.typ)
		} else {
			w.line(1, "%s: %s;", f.name, f.typ)
		}
	}
	w.line(0, "};")
}

// tsOperation is one variant with f.Returns, as the send table needs it.
type tsOperation struct {
	union    *UnionLike
	typ      string // "api.Search"
	answer   string // "api.SearchResults"
	answerOf string // the answer union's TypeScript name, "" when the answers are not one union
}

// renderOperations writes the <Union>Answer and <Union>Payload tables for the
// variants with f.Returns, and <Union>UnexpectedAnswer when every answer is a
// variant of one union. It returns the operations send() must know.
func (w *tsWriter) renderOperations(x *UnionLike) []tsOperation {
	union := tsTypeName(x.Name)
	for _, suffix := range []string{"Answer", "Payload", "Operation", "UnexpectedAnswer"} {
		if _, taken := w.r.known[knownKey(x.PkgImportName, x.Name+suffix)]; taken {
			w.fail("%s.%s%s is a type and the name %s needs for its f.Returns tables; rename it", x.PkgName, x.Name, suffix, x.Name)
			return nil
		}
	}

	answerUnion, oneUnion := w.answerUnionOf(x)
	var ops []tsOperation

	w.line(0, "/** The answer each %s operation names with f.Returns[R]. */", union)
	w.line(0, "export type %sAnswer = {", union)
	for _, v := range x.Variant {
		st, ok := v.(*StructLike)
		if !ok {
			continue
		}
		answer, ok := returnsOfStruct(st)
		if !ok {
			continue
		}
		answerShape := unwrapPointer(answer)
		w.line(1, "%q: %s;", x.PkgName+"."+st.Name, w.typeOf(answerShape, false))
		op := tsOperation{union: x, typ: x.PkgName + "." + st.Name}
		if oneUnion {
			if ref, ok := answerShape.(*RefName); ok {
				op.answer = ref.PkgName + "." + ref.Name
				op.answerOf = answerUnion
			}
		}
		ops = append(ops, op)
	}
	w.line(0, "};")
	w.blank()
	w.line(0, "/** The payload each %s operation carries. */", union)
	w.line(0, "export type %sPayload = {", union)
	for _, v := range x.Variant {
		st, ok := v.(*StructLike)
		if !ok {
			continue
		}
		if _, ok := returnsOfStruct(st); !ok {
			continue
		}
		w.line(1, "%q: %s;", x.PkgName+"."+st.Name, w.variantType(st))
	}
	w.line(0, "};")
	w.blank()
	w.line(0, "export type %sOperation = keyof %sAnswer;", union, union)

	if !oneUnion {
		return nil
	}
	w.blank()
	w.line(0, "/** The %s carried a variant other than the one the operation names, such as an error variant. */", answerUnion)
	w.prelude["UnexpectedAnswer"] = true
	w.line(0, "export class %sUnexpectedAnswer extends mkunion.UnexpectedAnswer<%s> {}", union, answerUnion)
	return ops
}

// renderSend writes one send() with an overload per union that declares
// f.Returns, and the table the implementation reads. The transport is the
// caller's, exactly as in Swift: send does the typing, not the HTTP.
func (w *tsWriter) renderSend(ops []tsOperation) {
	w.line(0, "/** The \"$type\" of the answer each operation names, and the error for any other answer. */")
	w.line(0, "const operations: { [operation: string]: { answer: string; unexpected: new (answer: never, operation: string) => mkunion.UnexpectedAnswer } } = {")
	for _, op := range ops {
		w.line(1, "%q: { answer: %q, unexpected: %sUnexpectedAnswer },", op.typ, op.answer, tsTypeName(op.union.Name))
	}
	w.line(0, "};")
	w.blank()

	seen := map[*UnionLike]bool{}
	for _, op := range ops {
		if seen[op.union] {
			continue
		}
		seen[op.union] = true
		union := tsTypeName(op.union.Name)
		w.line(0, "/**")
		w.line(0, " * Sends one %s operation through a transport that carries %s and %s as JSON, and returns the", union, union, op.answerOf)
		w.line(0, " * typed answer. Any other variant is thrown as %sUnexpectedAnswer, with the whole %s in it.", union, op.answerOf)
		w.line(0, " */")
		w.line(0, "export function send<K extends %sOperation>(type: K, payload: %sPayload[K], via: (request: %s) => Promise<%s>): Promise<%sAnswer[K]>;", union, union, union, op.answerOf, union)
	}
	w.line(0, "export async function send(type: string, payload: unknown, via: (request: never) => Promise<unknown>): Promise<unknown> {")
	w.line(1, "const operation = operations[type];")
	w.line(1, "if (operation === undefined) {")
	w.line(2, "throw new globalThis.Error(`${type} is not an operation of this package`);")
	w.line(1, "}")
	w.line(1, "const transport = via as (request: unknown) => Promise<unknown>;")
	w.line(1, "const response = (await transport({ \"$type\": type, [type]: payload })) as { [key: string]: unknown } | null;")
	w.line(1, "if (response !== null && response[\"$type\"] === operation.answer) {")
	w.line(2, "return response[operation.answer];")
	w.line(1, "}")
	w.line(1, "throw new operation.unexpected(response as never, type);")
	w.line(0, "}")
}

// answerUnionOf finds the one union every f.Returns answer of x is a variant
// of, as its TypeScript type seen from this package. When no answer is a
// variant there is no union to send through and no send; when some are and
// one is not, that one is a modelling error (no value of the answer union can
// carry it), reported as such.
func (w *tsWriter) answerUnionOf(x *UnionLike) (string, bool) {
	type answer struct {
		op    string
		typ   string
		union *RefName
	}
	var answers []answer
	var found *RefName
	for _, v := range x.Variant {
		st, ok := v.(*StructLike)
		if !ok {
			continue
		}
		for _, field := range st.Fields {
			shape, ok := ReturnsOf(field)
			if !ok {
				continue
			}
			a := answer{op: x.PkgName + "." + st.Name, typ: ToGoTypeName(unwrapPointer(shape))}
			if ref, ok := unwrapPointer(shape).(*RefName); ok {
				if resolved, ok := w.r.resolve(ref); ok {
					a.union = RetrieveVariantTypeRef(resolved)
				}
			}
			if a.union != nil && found == nil {
				found = a.union
			}
			answers = append(answers, a)
		}
	}
	if found == nil {
		return "", false
	}
	ok := true
	for _, a := range answers {
		if a.union == nil || a.union.Name != found.Name || a.union.PkgImportName != found.PkgImportName {
			w.fail("%s answers with %s, which is not a variant of %s.%s like the other answers of %s.%s; no %s can carry it, so there is no send", a.op, a.typ, found.PkgName, found.Name, x.PkgName, x.Name, found.Name)
			ok = false
		}
	}
	if !ok {
		return "", false
	}
	return w.localName(found.PkgName, found.PkgImportName, found.Name), true
}

// encodedByMkunion reports whether the generated serde writes the struct: a
// union variant, a struct tagged serde:"json", or one that holds a union
// (fromfile.go, RetrieveStructsHoldingUnion). Every other struct is written
// by encoding/json.
func (w *tsWriter) encodedByMkunion(x *StructLike) bool {
	if RetrieveVariantTypeRef(x) != nil {
		return true
	}
	if _, tagged := x.Tags["serde"]; tagged {
		return TagGetValue(x.Tags, "serde", "") == "json"
	}
	return w.holdsUnion(x, map[string]bool{})
}

// aliasEncodedByMkunion is encodedByMkunion for a named list, map or other
// non-struct type.
func (w *tsWriter) aliasEncodedByMkunion(x *AliasLike) bool {
	if RetrieveVariantTypeRef(x) != nil {
		return true
	}
	if _, tagged := x.Tags["serde"]; tagged {
		return TagGetValue(x.Tags, "serde", "") == "json"
	}
	return w.holdsUnion(x.Type, map[string]bool{knownKey(x.PkgImportName, x.Name): true})
}

// holdsUnion follows the walk the serde generator makes: a union anywhere
// encoding/json would reach on its own, through pointers, lists, maps and
// non-variant aliases. A named struct or a variant stops the walk, since it
// has JSON methods of its own.
func (w *tsWriter) holdsUnion(x Shape, seen map[string]bool) bool {
	switch y := x.(type) {
	case *UnionLike:
		return true
	case *PointerLike:
		return w.holdsUnion(y.Type, seen)
	case *ListLike:
		return w.holdsUnion(y.Element, seen)
	case *MapLike:
		return w.holdsUnion(y.Key, seen) || w.holdsUnion(y.Val, seen)
	case *AliasLike:
		return w.holdsUnion(y.Type, seen)
	case *StructLike:
		for _, field := range y.Fields {
			if !IsPhantomField(field) && w.holdsUnion(field.Type, seen) {
				return true
			}
		}
		return false
	case *RefName:
		if !w.r.renderable(y) {
			return false
		}
		s, ok := w.r.resolve(y)
		if !ok {
			return false
		}
		switch z := s.(type) {
		case *UnionLike:
			return true
		case *AliasLike:
			key := knownKey(z.PkgImportName, z.Name)
			if seen[key] || RetrieveVariantTypeRef(z) != nil {
				return false
			}
			seen[key] = true
			return w.holdsUnion(z.Type, seen)
		}
	}
	return false
}

// nilable reports whether Go may write null for a value of this type: a nil
// pointer, union or []byte always; a nil slice or map only from encoding/json
// (native), since the generated serde writes [] and {}. A named type is
// written by its own encoder.
func (w *tsWriter) nilable(x Shape, native bool) bool {
	switch y := x.(type) {
	case *PointerLike, *UnionLike:
		return true
	case *MapLike:
		return native
	case *ListLike:
		if IsBinary(y) {
			return true
		}
		return native && y.ArrayLen == nil
	case *AliasLike:
		return w.nilable(y.Type, !w.aliasEncodedByMkunion(y))
	case *RefName:
		if !w.r.renderable(y) {
			return false
		}
		s, ok := w.r.resolve(y)
		if !ok {
			return false
		}
		switch z := s.(type) {
		case *UnionLike:
			return true
		case *AliasLike:
			return w.nilable(z.Type, !w.aliasEncodedByMkunion(z))
		}
	}
	return false
}

// typeOf is the TypeScript type of a shape as JSON.parse returns it, null
// included when Go may write one; native says the enclosing struct is
// written by encoding/json.
func (w *tsWriter) typeOf(x Shape, native bool) string {
	core := w.coreOf(x, native)
	if w.nilable(x, native) {
		return core + " | null"
	}
	return core
}

// coreOf is the type without the null a nil value would add.
func (w *tsWriter) coreOf(x Shape, native bool) string {
	switch y := x.(type) {
	case *Any:
		return "unknown"
	case *RefName:
		return w.refType(y, native)
	case *PointerLike:
		return w.coreOf(y.Type, native)
	case *AliasLike:
		return w.localName(y.PkgName, y.PkgImportName, y.Name)
	case *PrimitiveLike:
		return tsPrimitive(y)
	case *ListLike:
		if IsBinary(y) {
			w.prelude["Base64"] = true
			return "mkunion.Base64"
		}
		element := w.typeOf(y.Element, native)
		if strings.Contains(element, " | ") {
			element = "(" + element + ")"
		}
		return element + "[]"
	case *MapLike:
		return "Record<string, " + w.typeOf(y.Val, native) + ">"
	case *StructLike:
		return w.localName(y.PkgName, y.PkgImportName, y.Name)
	case *UnionLike:
		return w.localName(y.PkgName, y.PkgImportName, y.Name)
	}
	w.fail("no TypeScript type for %T", x)
	return "unknown"
}

func (w *tsWriter) refType(ref *RefName, native bool) string {
	switch ref.PkgImportName {
	case "":
		if prim := NameToPrimitiveShape(ref.Name); prim != nil {
			return w.coreOf(prim, native)
		}
		// A type parameter of the enclosing type.
		return tsTypeName(ref.Name)
	case "time":
		switch ref.Name {
		case "Time":
			w.prelude["RFC3339"] = true
			return "mkunion.RFC3339"
		case "Duration", "Month", "Weekday":
			return "number"
		}
		w.fail("time.%s has no TypeScript type", ref.Name)
		return "unknown"
	}
	name := w.localName(ref.PkgName, ref.PkgImportName, ref.Name)
	if len(ref.Indexed) > 0 {
		args := make([]string, 0, len(ref.Indexed))
		for _, arg := range ref.Indexed {
			args = append(args, w.typeOf(arg, native))
		}
		return name + "<" + strings.Join(args, ", ") + ">"
	}
	return name
}

// localName is a type as written from this file: bare in its own package,
// qualified by the package otherwise, which is then imported.
func (w *tsWriter) localName(pkgName, importName, name string) string {
	if importName == w.pkg.importName {
		return tsTypeName(name)
	}
	w.imports[pkgName] = importName
	return pkgName + "." + tsTypeName(name)
}

func tsPrimitive(x *PrimitiveLike) string {
	return MatchPrimitiveKindR1(x.Kind,
		func(*BooleanLike) string { return "boolean" },
		func(*StringLike) string { return "string" },
		func(*NumberLike) string { return "number" },
	)
}

func tsTypeParams(params []TypeParam) string {
	if len(params) == 0 {
		return ""
	}
	names := make([]string, 0, len(params))
	for _, p := range params {
		names = append(names, tsTypeName(p.Name))
	}
	return "<" + strings.Join(names, ", ") + ">"
}

// tsTypeName keeps a Go type name unless the generated code needs that global.
func tsTypeName(name string) string {
	switch name {
	case "Record", "Promise":
		return name + "_"
	}
	return name
}

var tsIdentifier = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)

// tsPropertyName quotes a json name that is not an identifier.
func tsPropertyName(jsonName string) string {
	if tsIdentifier.MatchString(jsonName) {
		return jsonName
	}
	return fmt.Sprintf("%q", jsonName)
}

const tsPrelude = `// Code generated by mkunion. DO NOT EDIT.
/* eslint-disable */

// What every file mkunion generated shares. TypeScript has no decoder of its
// own: JSON.parse returns the value and the generated types describe it, so
// nothing here runs while decoding. UnexpectedAnswer is the one piece of
// runtime, thrown by a generated send().

/**
 * An RFC 3339 timestamp as Go writes time.Time: "2026-11-05T12:00:00.123456789Z",
 * with up to nine fractional digits. new Date(s) reads it and keeps milliseconds.
 */
export type RFC3339 = string;

/** Go's zero time.Time: what Go writes for a date that was never set. */
export const zeroTime: RFC3339 = "0001-01-01T00:00:00Z";

/** Base64 (RFC 4648, padded) as Go writes []byte. */
export type Base64 = string;

/**
 * Thrown by a generated send() when the answer union carried a variant other
 * than the one the operation names, such as an error variant. The whole answer
 * is kept, so the caller can switch on its $type.
 */
export class UnexpectedAnswer<Answer = unknown> extends Error {
  readonly answer: Answer;
  readonly operation: string;

  constructor(answer: Answer, operation: string) {
    super(` + "`${operation}: unexpected answer ${typeOf(answer)}`" + `);
    this.name = "UnexpectedAnswer";
    this.answer = answer;
    this.operation = operation;
  }
}

/** The "$type" of a union value, for messages. */
function typeOf(value: unknown): string {
  if (typeof value === "object" && value !== null && "$type" in value && typeof value["$type"] === "string") {
    return value["$type"];
  }
  return "with no $type";
}
`
