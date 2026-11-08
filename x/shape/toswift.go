package shape

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"unicode"

	log "github.com/sirupsen/logrus"
)

// SwiftRenderer writes Swift for the shapes it is given: one file per Go
// package, named after the package (api → Api.swift), that holds a namespace
// enum of the same name with one Codable type per shape, plus
// MkunionJSON.swift with the coders every file needs.
//
// The JSON is the one the generated Go serde reads and writes
// (docs/examples/json.md): a union is {"$type": "api.X", "api.X": {...}},
// $type picks the variant and a missing or null payload is its zero value;
// struct fields use their json tag names; a missing or null field is the Go
// zero value; a nil pointer or nil union is null, so those fields are
// Optional; time.Time is Date, []byte is Data, any is JSONValue. Elements of
// a list or map are not Optional: a null element is a decoding error.
//
// A variant that embeds f.Returns[R] gets a conformance to the union's
// <Union>Operation protocol with R as its Answer; when every answer is a
// variant of one union, or a union whose variants alias variants of that one
// union (type GetSeriesSeries = Series), a generic send(_:via:) is emitted
// that posts the request union through a transport and unwraps the typed
// answer: a <Answer>Answer protocol with init?(_:) from the answer union,
// which every answer variant and every such sub-union conforms to.
//
// Generic types are not supported yet: a shape with type parameters is an
// error. Map keys other than string are written as String, which is what Go
// puts on the wire.
type SwiftRenderer struct {
	packages map[string]*swiftPackage // by package import name
	known    map[string]Shape         // by "importName.Name", every shape seen, variants included
	added    map[string]bool
	errs     []string
}

type swiftPackage struct {
	name       string
	importName string
	unions     []*UnionLike
	structs    []*StructLike
	aliases    []*AliasLike
}

func NewSwiftRenderer() *SwiftRenderer {
	return &SwiftRenderer{
		packages: map[string]*swiftPackage{},
		known:    map[string]Shape{},
		added:    map[string]bool{},
	}
}

// AddShape queues a named shape and everything it refers to. Union variants
// are rendered with their union, so adding a variant adds the union.
func (r *SwiftRenderer) AddShape(x Shape) {
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
		log.Infof("toswift: AddShape: %T is not a named type, skipped", x)
		return
	}

	r.FollowRef(x)
}

// addUnionOfVariant adds the union a variant belongs to and reports whether
// the shape was a variant.
func (r *SwiftRenderer) addUnionOfVariant(x Shape) bool {
	ref := RetrieveVariantTypeRef(x)
	if ref == nil {
		return false
	}
	if union, ok := r.resolve(ref); ok {
		r.AddShape(union)
	} else {
		r.errs = append(r.errs, fmt.Sprintf("toswift: %s is a variant of %s.%s, which was not found", Name(x), ref.PkgName, ref.Name))
	}
	return true
}

// FollowRef adds the shapes x refers to. Builtins, time.Time and the
// f.Returns marker are not shapes to render.
func (r *SwiftRenderer) FollowRef(x Shape) {
	for _, ref := range ExtractRefs(x) {
		if !r.renderable(ref) {
			continue
		}
		if s, ok := r.resolve(ref); ok {
			r.AddShape(s)
		} else {
			log.Debugf("toswift: FollowRef: %s not found", ToGoTypeName(ref))
		}
	}
}

func (r *SwiftRenderer) renderable(ref *RefName) bool {
	switch ref.PkgImportName {
	case "", "time", returnsPkgImportName:
		return false
	}
	return true
}

func (r *SwiftRenderer) pkg(name, importName string) *swiftPackage {
	if p, ok := r.packages[importName]; ok {
		return p
	}
	p := &swiftPackage{name: name, importName: importName}
	r.packages[importName] = p
	return p
}

func (r *SwiftRenderer) know(x Shape) {
	r.known[knownKey(ToGoPkgImportName(x), Name(x))] = x
}

func knownKey(importName, name string) string {
	return importName + "." + name
}

// resolve finds the shape a reference points to: among the shapes already
// seen, else on disk.
func (r *SwiftRenderer) resolve(ref *RefName) (Shape, bool) {
	if s, ok := r.known[knownKey(ref.PkgImportName, ref.Name)]; ok {
		return s, true
	}
	s, ok := LookupShapeOnDisk(ref)
	if ok {
		r.know(s)
	}
	return s, ok
}

// Render returns the Swift files by name.
func (r *SwiftRenderer) Render() (map[string]string, error) {
	files := map[string]string{}
	if len(r.packages) == 0 {
		return files, nil
	}

	importNames := make([]string, 0, len(r.packages))
	for importName := range r.packages {
		importNames = append(importNames, importName)
	}
	sort.Strings(importNames)

	fileOf := map[string]string{}
	for _, importName := range importNames {
		pkg := r.packages[importName]
		fileName := swiftPkgName(pkg.name) + ".swift"
		if other, taken := fileOf[fileName]; taken {
			r.errs = append(r.errs, fmt.Sprintf("toswift: packages %s and %s both render as %s", other, importName, fileName))
			continue
		}
		fileOf[fileName] = importName

		w := &swiftWriter{r: r, pkg: pkg, declared: map[string]bool{}, conformed: map[string]bool{}}
		files[fileName] = w.renderPackage()
	}
	files["MkunionJSON.swift"] = swiftPrelude

	if len(r.errs) > 0 {
		return files, fmt.Errorf("%s", strings.Join(r.errs, "\n"))
	}
	return files, nil
}

// WriteToDir renders and writes the files into dir, creating it when needed.
func (r *SwiftRenderer) WriteToDir(dir string) error {
	files, err := r.Render()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("toswift: WriteToDir: %w", err)
	}
	for name, content := range files {
		if err := os.WriteFile(path.Join(dir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("toswift: WriteToDir %s: %w", name, err)
		}
	}
	return nil
}

// swiftWriter renders one package.
type swiftWriter struct {
	r   *SwiftRenderer
	pkg *swiftPackage
	b   strings.Builder
	// declared and conformed keep the <Answer>Answer protocol and its
	// conformances to one copy when several request unions share an answer union.
	declared  map[string]bool
	conformed map[string]bool
}

func (w *swiftWriter) line(indent int, format string, args ...any) {
	w.b.WriteString(strings.Repeat("    ", indent))
	fmt.Fprintf(&w.b, format, args...)
	w.b.WriteString("\n")
}

func (w *swiftWriter) blank() {
	w.b.WriteString("\n")
}

func (w *swiftWriter) fail(format string, args ...any) {
	w.r.errs = append(w.r.errs, "toswift: "+fmt.Sprintf(format, args...))
}

func (w *swiftWriter) renderPackage() string {
	ns := swiftPkgName(w.pkg.name)
	sort.Slice(w.pkg.unions, func(i, j int) bool { return w.pkg.unions[i].Name < w.pkg.unions[j].Name })
	sort.Slice(w.pkg.structs, func(i, j int) bool { return w.pkg.structs[i].Name < w.pkg.structs[j].Name })
	sort.Slice(w.pkg.aliases, func(i, j int) bool { return w.pkg.aliases[i].Name < w.pkg.aliases[j].Name })

	w.line(0, "// Code generated by mkunion. DO NOT EDIT.")
	w.blank()
	w.line(0, "import Foundation")
	w.blank()
	w.line(0, "/// Go package %s (%s).", w.pkg.name, w.pkg.importName)
	w.line(0, "public enum %s {", ns)

	first := true
	section := func() {
		if !first {
			w.blank()
		}
		first = false
	}

	for _, union := range w.pkg.unions {
		for _, v := range union.Variant {
			if alias, ok := v.(*AliasLike); ok {
				section()
				w.renderAlias(alias)
			}
		}
		section()
		w.renderUnion(union)
	}
	for _, union := range w.pkg.unions {
		for _, v := range union.Variant {
			if st, ok := v.(*StructLike); ok {
				section()
				w.renderStruct(st, union)
			}
		}
	}
	for _, st := range w.pkg.structs {
		section()
		w.renderStruct(st, nil)
	}
	for _, alias := range w.pkg.aliases {
		section()
		w.renderAlias(alias)
	}
	for _, union := range w.pkg.unions {
		if UnionDeclaresReturns(union) {
			section()
			w.renderOperationProtocol(union)
		}
	}
	w.line(0, "}")

	for _, union := range w.pkg.unions {
		if UnionDeclaresReturns(union) {
			w.renderOperationConformances(union)
		}
	}

	return w.b.String()
}

func (w *swiftWriter) renderAlias(x *AliasLike) {
	if len(x.TypeParams) > 0 {
		w.fail("%s.%s has type parameters; generic types are not supported yet", x.PkgName, x.Name)
		return
	}
	w.line(1, "public typealias %s = %s", swiftTypeName(x.Name), w.typeOf(x.Type))
}

func (w *swiftWriter) renderUnion(x *UnionLike) {
	if len(x.TypeParams) > 0 {
		w.fail("%s.%s has type parameters; generic types are not supported yet", x.PkgName, x.Name)
		return
	}
	if len(x.Variant) == 0 {
		w.fail("%s.%s has no variants", x.PkgName, x.Name)
		return
	}
	name := swiftTypeName(x.Name)
	jsonName := func(v Shape) string { return x.PkgName + "." + Name(v) }

	w.line(1, "/// %s is a union: one case per variant, on the wire as {\"$type\": \"%s.%s\", \"%s.%s\": {...}}.", name, x.PkgName, Name(x.Variant[0]), x.PkgName, Name(x.Variant[0]))
	w.line(1, "public enum %s: Codable, Hashable, Sendable {", name)
	for _, v := range x.Variant {
		w.line(2, "case %s(%s)", swiftCaseName(Name(v)), w.typeOf(v))
	}
	w.blank()
	w.line(2, "private enum CodingKeys: String, CodingKey {")
	w.line(3, "case type = \"$type\"")
	for _, v := range x.Variant {
		w.line(3, "case %s = %q", swiftCaseName(Name(v)), jsonName(v))
	}
	w.line(2, "}")
	w.blank()
	w.line(2, "public init(from decoder: any Decoder) throws {")
	w.line(3, "let c = try decoder.container(keyedBy: CodingKeys.self)")
	w.line(3, "// $type picks the variant; a missing or null payload is its zero value.")
	w.line(3, "let type = try c.decodeIfPresent(String.self, forKey: .type) ?? \"\"")
	w.line(3, "switch type {")
	for _, v := range x.Variant {
		c := swiftCaseName(Name(v))
		w.line(3, "case %q: self = .%s(try c.decodeIfPresent(%s.self, forKey: .%s) ?? %s); return", jsonName(v), c, w.typeOf(v), c, w.zeroOf(v))
	}
	w.line(3, "default: break")
	w.line(3, "}")
	w.line(3, "// No known $type: the first key that names a variant, as the Go decoder does.")
	for _, v := range x.Variant {
		c := swiftCaseName(Name(v))
		w.line(3, "if c.contains(.%s) { self = .%s(try c.decodeIfPresent(%s.self, forKey: .%s) ?? %s); return }", c, c, w.typeOf(v), c, w.zeroOf(v))
	}
	w.line(3, "if type.isEmpty { throw MkunionJSON.Error.noType(union: %q) }", x.PkgName+"."+x.Name)
	w.line(3, "throw MkunionJSON.Error.unknownType(type, union: %q)", x.PkgName+"."+x.Name)
	w.line(2, "}")
	w.blank()
	w.line(2, "public func encode(to encoder: any Encoder) throws {")
	w.line(3, "var c = encoder.container(keyedBy: CodingKeys.self)")
	w.line(3, "switch self {")
	for _, v := range x.Variant {
		c := swiftCaseName(Name(v))
		w.line(3, "case .%s(let value):", c)
		w.line(4, "try c.encode(%q, forKey: .type)", jsonName(v))
		w.line(4, "try c.encode(value, forKey: .%s)", c)
	}
	w.line(3, "}")
	w.line(2, "}")
	w.blank()
	w.line(2, "/// The variant's value when it is a T, else nil.")
	w.line(2, "public func payload<T>(as type: T.Type = T.self) -> T? {")
	w.line(3, "switch self {")
	for _, v := range x.Variant {
		w.line(3, "case .%s(let value): return value as? T", swiftCaseName(Name(v)))
	}
	w.line(3, "}")
	w.line(2, "}")
	w.line(1, "}")
}

// swiftField is a struct field as Swift sees it.
type swiftField struct {
	name     string // Swift property name, escaped when it is a keyword
	label    string // the same, for an init argument label
	jsonName string
	typ      string
	zero     string // the value for a missing key; "" when the type is Optional
	desc     *string
}

func (w *swiftWriter) fieldsOf(x *StructLike) []swiftField {
	var fields []swiftField
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
		f := swiftField{
			name:     swiftIdentifier(lowerCamel(field.Name)),
			label:    swiftIdentifier(lowerCamel(field.Name)),
			jsonName: jsonName,
			typ:      w.typeOf(field.Type),
			desc:     field.Desc,
		}
		if w.nilable(field.Type) {
			if !strings.HasSuffix(f.typ, "?") {
				f.typ += "?"
			}
		} else {
			f.zero = w.zeroOf(field.Type)
		}
		fields = append(fields, f)
	}
	return fields
}

// renderStruct writes a struct; union is set when the struct is a variant,
// so a variant with f.Returns can name its Answer and wrap itself as the union.
func (w *swiftWriter) renderStruct(x *StructLike, union *UnionLike) {
	if len(x.TypeParams) > 0 {
		w.fail("%s.%s has type parameters; generic types are not supported yet", x.PkgName, x.Name)
		return
	}
	name := swiftTypeName(x.Name)
	fields := w.fieldsOf(x)

	w.line(1, "public struct %s: Codable, Hashable, Sendable {", name)
	if answer, ok := returnsOfStruct(x); ok && union != nil {
		unionName := swiftTypeName(union.Name)
		w.line(2, "public typealias Answer = %s", w.typeOf(unwrapPointer(answer)))
		w.line(2, "public var as%s: %s { .%s(self) }", unionName, unionName, swiftCaseName(x.Name))
		w.blank()
	}

	if len(fields) == 0 {
		w.line(2, "public init() {}")
		w.line(1, "}")
		return
	}

	for _, f := range fields {
		if f.desc != nil && *f.desc != "" {
			w.line(2, "/// %s", *f.desc)
		}
		w.line(2, "public var %s: %s", f.name, f.typ)
	}
	w.blank()

	params := make([]string, 0, len(fields))
	for _, f := range fields {
		def := f.zero
		if def == "" {
			def = "nil"
		}
		params = append(params, fmt.Sprintf("%s: %s = %s", f.label, f.typ, def))
	}
	w.line(2, "public init(%s) {", strings.Join(params, ", "))
	for _, f := range fields {
		w.line(3, "self.%s = %s", f.name, f.name)
	}
	w.line(2, "}")
	w.blank()

	w.line(2, "private enum CodingKeys: String, CodingKey {")
	for _, f := range fields {
		w.line(3, "case %s = %q", f.name, f.jsonName)
	}
	w.line(2, "}")
	w.blank()

	w.line(2, "public init(from decoder: any Decoder) throws {")
	w.line(3, "let c = try decoder.container(keyedBy: CodingKeys.self)")
	w.line(3, "// A missing or null key is the Go zero value.")
	for _, f := range fields {
		if f.zero == "" {
			w.line(3, "%s = try c.decodeIfPresent(%s.self, forKey: .%s)", f.name, strings.TrimSuffix(f.typ, "?"), f.name)
		} else {
			w.line(3, "%s = try c.decodeIfPresent(%s.self, forKey: .%s) ?? %s", f.name, f.typ, f.name, f.zero)
		}
	}
	w.line(2, "}")
	w.line(1, "}")
}

// operationName is the protocol a union's variants with f.Returns conform to.
func operationName(union *UnionLike) string {
	return swiftTypeName(union.Name) + "Operation"
}

func (w *swiftWriter) renderOperationProtocol(x *UnionLike) {
	union := swiftTypeName(x.Name)
	as := "as" + union
	plan, ok := w.answerPlan(x)
	w.line(1, "/// A %s variant that names its answer, from f.Returns[R] in Go.", union)
	w.line(1, "public protocol %s: Codable, Hashable, Sendable {", operationName(x))
	if ok {
		w.line(2, "associatedtype Answer: Codable & %s", plan.protocol)
	} else {
		w.line(2, "associatedtype Answer: Codable")
	}
	w.line(2, "/// The operation as its union, so it carries its $type on the wire.")
	w.line(2, "var %s: %s { get }", as, union)
	w.line(1, "}")

	if !ok {
		return
	}
	answerUnion := plan.name
	if !w.declared[plan.protocol] {
		w.declared[plan.protocol] = true
		w.blank()
		w.line(1, "/// What a %s unwraps to: one of its variants, or a union of some of them.", answerUnion)
		w.line(1, "public protocol %s {", plan.protocol)
		w.line(2, "init?(_ answer: %s)", answerUnion)
		w.line(1, "}")
	}
	w.blank()
	w.line(1, "/// The %s carried a variant other than the one the operation names, such as an error variant.", answerUnion)
	w.line(1, "public struct %sUnexpectedAnswer: Swift.Error, Hashable, Sendable {", union)
	w.line(2, "public let answer: %s", answerUnion)
	w.line(2, "public init(answer: %s) { self.answer = answer }", answerUnion)
	w.line(1, "}")
	w.blank()
	w.line(1, "/// Sends one operation through a transport that carries %s and %s as JSON, and returns", union, answerUnion)
	w.line(1, "/// the typed answer. Any other variant is thrown as %sUnexpectedAnswer.", union)
	w.line(1, "public static func send<Op: %s>(_ op: Op, via transport: (%s) async throws -> %s) async throws -> Op.Answer {", operationName(x), union, answerUnion)
	w.line(2, "let response = try await transport(op.%s)", as)
	w.line(2, "if let answer = Op.Answer(response) { return answer }")
	w.line(2, "throw %sUnexpectedAnswer(answer: response)", union)
	w.line(1, "}")
}

// renderAnswerConformances declares, at file scope, how each answer is taken
// out of the answer union: a variant by its case, a sub-union by the cases
// of the variants it aliases.
func (w *swiftWriter) renderAnswerConformances(x *UnionLike) {
	plan, ok := w.answerPlan(x)
	if !ok {
		return
	}
	ns := swiftPkgName(w.pkg.name)
	for _, v := range plan.union.Variant {
		st, ok := v.(*StructLike)
		if !ok || w.conformed[st.Name] {
			continue
		}
		w.conformed[st.Name] = true
		w.blank()
		w.line(0, "extension %s.%s: %s.%s {", ns, swiftTypeName(st.Name), ns, plan.protocol)
		w.line(1, "public init?(_ answer: %s.%s) {", ns, swiftTypeName(plan.union.Name))
		w.line(2, "guard case .%s(let value) = answer else { return nil }", swiftCaseName(st.Name))
		w.line(2, "self = value")
		w.line(1, "}")
		w.line(0, "}")
	}
	for _, sub := range plan.subs {
		if w.conformed[sub.Name] {
			continue
		}
		w.conformed[sub.Name] = true
		w.blank()
		w.line(0, "extension %s.%s: %s.%s {", ns, swiftTypeName(sub.Name), ns, plan.protocol)
		w.line(1, "public init?(_ answer: %s.%s) {", ns, swiftTypeName(plan.union.Name))
		w.line(2, "switch answer {")
		for _, v := range sub.Variant {
			leaf, _ := w.leafOf(v)
			w.line(2, "case .%s(let value): self = .%s(value)", swiftCaseName(leaf.Name), swiftCaseName(Name(v)))
		}
		w.line(2, "default: return nil")
		w.line(2, "}")
		w.line(1, "}")
		w.line(0, "}")
	}
}

// renderOperationConformances declares, at file scope, that each variant with
// f.Returns is an operation of its union. The members live in the struct.
func (w *swiftWriter) renderOperationConformances(x *UnionLike) {
	ns := swiftPkgName(w.pkg.name)
	for _, v := range x.Variant {
		st, ok := v.(*StructLike)
		if !ok {
			continue
		}
		if _, ok := returnsOfStruct(st); !ok {
			continue
		}
		w.blank()
		w.line(0, "extension %s.%s: %s.%s {}", ns, swiftTypeName(st.Name), ns, operationName(x))
	}
	w.renderAnswerConformances(x)
}

// returnsOfStruct is the answer type a variant declares with f.Returns, if any.
func returnsOfStruct(x *StructLike) (Shape, bool) {
	for _, field := range x.Fields {
		if answer, ok := ReturnsOf(field); ok {
			return answer, true
		}
	}
	return nil, false
}

// swiftAnswerPlan is the answer union of a request union and how each answer
// is taken out of it.
type swiftAnswerPlan struct {
	union    *UnionLike   // the answer union
	name     string       // its Swift type seen from this package
	protocol string       // <Union>Answer: the protocol every answer conforms to
	subs     []*UnionLike // answers that are unions of aliased answer-union variants
}

// answerPlan finds the one union every f.Returns answer of x is a variant
// of, directly or through a union whose variants alias its variants.
func (w *swiftWriter) answerPlan(x *UnionLike) (*swiftAnswerPlan, bool) {
	var found *RefName
	var subs []*UnionLike
	seen := map[string]bool{}
	for _, v := range x.Variant {
		st, ok := v.(*StructLike)
		if !ok {
			continue
		}
		for _, field := range st.Fields {
			answer, ok := ReturnsOf(field)
			if !ok {
				continue
			}
			ref, ok := unwrapPointer(answer).(*RefName)
			if !ok {
				return nil, false
			}
			shape, ok := w.r.resolve(ref)
			if !ok {
				return nil, false
			}
			leaves := []Shape{shape}
			if sub, isUnion := shape.(*UnionLike); isUnion {
				leaves = sub.Variant
				if !seen[sub.Name] {
					seen[sub.Name] = true
					subs = append(subs, sub)
				}
			}
			for _, leaf := range leaves {
				st, ok := w.leafOf(leaf)
				if !ok {
					return nil, false
				}
				unionRef := RetrieveVariantTypeRef(st)
				if unionRef == nil {
					return nil, false
				}
				if found != nil && (found.Name != unionRef.Name || found.PkgImportName != unionRef.PkgImportName) {
					return nil, false
				}
				found = unionRef
			}
		}
	}
	if found == nil {
		return nil, false
	}
	resolved, ok := w.r.resolve(found)
	if !ok {
		return nil, false
	}
	union, ok := resolved.(*UnionLike)
	if !ok {
		return nil, false
	}
	return &swiftAnswerPlan{
		union:    union,
		name:     w.localName(found.PkgName, found.PkgImportName, found.Name),
		protocol: swiftTypeName(found.Name) + "Answer",
		subs:     subs,
	}, true
}

// leafOf is the struct a variant stands for: itself, or for a type alias the
// struct it aliases.
func (w *swiftWriter) leafOf(v Shape) (*StructLike, bool) {
	switch y := v.(type) {
	case *StructLike:
		return y, true
	case *AliasLike:
		ref, ok := y.Type.(*RefName)
		if !ok || !y.IsAlias {
			return nil, false
		}
		resolved, ok := w.r.resolve(ref)
		if !ok {
			return nil, false
		}
		st, ok := resolved.(*StructLike)
		return st, ok
	}
	return nil, false
}

func unwrapPointer(x Shape) Shape {
	for {
		p, ok := x.(*PointerLike)
		if !ok {
			return x
		}
		x = p.Type
	}
}

// nilable reports whether a field of this type is Optional in Swift: a
// pointer, or a union, since Go encodes a nil of either as null.
func (w *swiftWriter) nilable(x Shape) bool {
	switch y := x.(type) {
	case *PointerLike:
		return true
	case *UnionLike:
		return true
	case *RefName:
		return w.isUnionRef(y)
	}
	return false
}

func (w *swiftWriter) isUnionRef(ref *RefName) bool {
	if !w.r.renderable(ref) {
		return false
	}
	s, ok := w.r.resolve(ref)
	if !ok {
		return false
	}
	return IsUnion(s)
}

// typeOf is the Swift type of a shape, as written inside this package's namespace.
func (w *swiftWriter) typeOf(x Shape) string {
	switch y := x.(type) {
	case *Any:
		return "JSONValue"
	case *RefName:
		return w.refType(y)
	case *PointerLike:
		inner := w.typeOf(y.Type)
		if strings.HasSuffix(inner, "?") {
			return inner
		}
		return inner + "?"
	case *AliasLike:
		return w.localName(y.PkgName, y.PkgImportName, y.Name)
	case *PrimitiveLike:
		return swiftPrimitive(y)
	case *ListLike:
		if IsBinary(y) {
			return "Data"
		}
		return "[" + w.typeOf(y.Element) + "]"
	case *MapLike:
		return "[String: " + w.typeOf(y.Val) + "]"
	case *StructLike:
		return w.localName(y.PkgName, y.PkgImportName, y.Name)
	case *UnionLike:
		return w.localName(y.PkgName, y.PkgImportName, y.Name)
	}
	w.fail("no Swift type for %T", x)
	return "JSONValue"
}

func (w *swiftWriter) refType(ref *RefName) string {
	if len(ref.Indexed) > 0 {
		w.fail("%s.%s is instantiated with type arguments; generic types are not supported yet", ref.PkgName, ref.Name)
		return "JSONValue"
	}
	switch ref.PkgImportName {
	case "":
		if prim := NameToPrimitiveShape(ref.Name); prim != nil {
			return w.typeOf(prim)
		}
		w.fail("%s is a builtin with no Swift type", ref.Name)
		return "JSONValue"
	case "time":
		switch ref.Name {
		case "Time":
			return "Date"
		case "Duration":
			return "Int"
		}
	}
	return w.localName(ref.PkgName, ref.PkgImportName, ref.Name)
}

func (w *swiftWriter) localName(pkgName, importName, name string) string {
	if importName == w.pkg.importName {
		return swiftTypeName(name)
	}
	return swiftPkgName(pkgName) + "." + swiftTypeName(name)
}

// zeroOf is the Swift value a missing or null JSON value stands for: the Go
// zero value of the shape.
func (w *swiftWriter) zeroOf(x Shape) string {
	switch y := x.(type) {
	case *Any:
		return ".null"
	case *RefName:
		return w.refZero(y)
	case *PointerLike:
		return "nil"
	case *AliasLike:
		return w.zeroOf(y.Type)
	case *PrimitiveLike:
		return MatchPrimitiveKindR1(y.Kind,
			func(*BooleanLike) string { return "false" },
			func(*StringLike) string { return `""` },
			func(*NumberLike) string { return "0" },
		)
	case *ListLike:
		if IsBinary(y) {
			return "Data()"
		}
		return "[]"
	case *MapLike:
		return "[:]"
	case *StructLike:
		return w.localName(y.PkgName, y.PkgImportName, y.Name) + "()"
	case *UnionLike:
		return "nil"
	}
	return ".null"
}

func (w *swiftWriter) refZero(ref *RefName) string {
	if len(ref.Indexed) > 0 {
		return ".null"
	}
	switch ref.PkgImportName {
	case "":
		if prim := NameToPrimitiveShape(ref.Name); prim != nil {
			return w.zeroOf(prim)
		}
		return ".null"
	case "time":
		if ref.Name == "Time" {
			return "MkunionJSON.zeroDate"
		}
		return "0"
	}
	s, ok := w.r.resolve(ref)
	if !ok {
		return w.localName(ref.PkgName, ref.PkgImportName, ref.Name) + "()"
	}
	switch y := s.(type) {
	case *UnionLike:
		return "nil"
	case *AliasLike:
		return w.zeroOf(y.Type)
	}
	return w.localName(ref.PkgName, ref.PkgImportName, ref.Name) + "()"
}

func swiftPrimitive(x *PrimitiveLike) string {
	return MatchPrimitiveKindR1(x.Kind,
		func(*BooleanLike) string { return "Bool" },
		func(*StringLike) string { return "String" },
		func(n *NumberLike) string {
			return MatchNumberKindR1(n.Kind,
				func(*UInt) string { return "UInt" },
				func(*UInt8) string { return "UInt8" },
				func(*UInt16) string { return "UInt16" },
				func(*UInt32) string { return "UInt32" },
				func(*UInt64) string { return "UInt64" },
				func(*Int) string { return "Int" },
				func(*Int8) string { return "Int8" },
				func(*Int16) string { return "Int16" },
				func(*Int32) string { return "Int32" },
				func(*Int64) string { return "Int64" },
				func(*Float32) string { return "Float" },
				func(*Float64) string { return "Double" },
			)
		},
	)
}

// swiftPkgName is the namespace for a Go package: api → Api, my_app → MyApp.
func swiftPkgName(pkgName string) string {
	var b strings.Builder
	upper := true
	for _, r := range pkgName {
		switch {
		case r == '_' || r == '-' || r == '.':
			upper = true
		case upper:
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// swiftTypeName keeps a Go type name unless Swift cannot nest a type of that name.
func swiftTypeName(name string) string {
	switch name {
	case "Type", "Protocol", "Any", "Self":
		return name + "_"
	}
	return name
}

// swiftCaseName is the enum case for a variant: GetInfo → getInfo, URL → url.
func swiftCaseName(name string) string {
	return swiftIdentifier(lowerCamel(name))
}

// lowerCamel lowers the leading word of a Go identifier: ID → id,
// HTTPServer → httpServer, ChaptersDone → chaptersDone, K → k.
func lowerCamel(name string) string {
	runes := []rune(name)
	n := 0
	for n < len(runes) && unicode.IsUpper(runes[n]) {
		n++
	}
	switch {
	case n == 0:
		return name
	case n == len(runes), n == 1:
		for i := 0; i < n; i++ {
			runes[i] = unicode.ToLower(runes[i])
		}
	default:
		for i := 0; i < n-1; i++ {
			runes[i] = unicode.ToLower(runes[i])
		}
	}
	return string(runes)
}

var swiftKeywords = map[string]bool{
	"Any": true, "Self": true, "Type": true, "any": true, "as": true, "associatedtype": true, "break": true,
	"case": true, "catch": true, "class": true, "continue": true, "default": true, "defer": true, "deinit": true,
	"do": true, "else": true, "enum": true, "extension": true, "fallthrough": true, "false": true, "fileprivate": true,
	"for": true, "func": true, "guard": true, "if": true, "import": true, "in": true, "init": true, "inout": true,
	"internal": true, "is": true, "let": true, "nil": true, "operator": true, "private": true,
	"protocol": true, "public": true, "repeat": true, "rethrows": true, "return": true, "self": true, "static": true,
	"struct": true, "subscript": true, "super": true, "switch": true, "throw": true, "throws": true, "true": true,
	"try": true, "typealias": true, "var": true, "where": true, "while": true,
}

// swiftIdentifier escapes a name Swift reserves.
func swiftIdentifier(name string) string {
	if swiftKeywords[name] {
		return "`" + name + "`"
	}
	return name
}

const swiftPrelude = `// Code generated by mkunion. DO NOT EDIT.

import Foundation

/// What every type mkunion generated shares: the errors a union decoder
/// throws, Go's zero time, and coders that read and write dates the way Go does.
public enum MkunionJSON {
    public enum Error: Swift.Error, Hashable, Sendable, CustomStringConvertible {
        /// The object had no "$type" and no key that names a variant.
        case noType(union: String)
        /// "$type" named a variant the union does not have, and no key named one either.
        case unknownType(String, union: String)

        public var description: String {
            switch self {
            case .noType(let union): return "\(union): no $type field"
            case .unknownType(let type, let union): return "\(union): unknown type: \(type)"
            }
        }
    }

    /// Go's zero time.Time, 0001-01-01T00:00:00Z: what a missing date decodes to.
    public static let zeroDate = Date(timeIntervalSinceReferenceDate: -63_113_904_000)

    /// A decoder for JSON written by Go: RFC 3339 dates with up to nine
    /// fractional digits, base64 Data.
    public static func decoder() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { source in
            let text = try source.singleValueContainer().decode(String.self)
            guard let date = parseRFC3339(text) else {
                throw DecodingError.dataCorrupted(.init(codingPath: source.codingPath, debugDescription: "not an RFC 3339 date: \(text)"))
            }
            return date
        }
        return decoder
    }

    /// An encoder for JSON read by Go: RFC 3339 dates with fractional seconds.
    public static func encoder() -> JSONEncoder {
        let encoder = JSONEncoder()
        encoder.dateEncodingStrategy = .custom { date, target in
            var c = target.singleValueContainer()
            try c.encode(formatRFC3339(date))
        }
        return encoder
    }

    /// Reads an RFC 3339 date. ISO8601DateFormatter takes at most three
    /// fractional digits and Go writes up to nine, so the fraction is cut to
    /// three first. Go's zero time is read by hand: the formatter's calendar
    /// turns Julian before 1582 and would land two days late.
    public static func parseRFC3339(_ text: String) -> Date? {
        if text.hasPrefix(zeroDateText) {
            return zeroDate
        }
        var normalised = text
        if let dot = text.firstIndex(of: "."),
           let end = text[dot...].firstIndex(where: { $0 == "Z" || $0 == "+" || $0 == "-" }) {
            let fraction = String(text[text.index(after: dot)..<end].prefix(3))
            let padded = fraction.padding(toLength: 3, withPad: "0", startingAt: 0)
            normalised = String(text[..<dot]) + "." + padded + String(text[end...])
        }
        if normalised.contains(".") {
            return fractional().date(from: normalised)
        }
        return whole().date(from: normalised)
    }

    public static func formatRFC3339(_ date: Date) -> String {
        if date == zeroDate {
            return zeroDateText + "Z"
        }
        return fractional().string(from: date)
    }

    private static let zeroDateText = "0001-01-01T00:00:00"

    private static func fractional() -> ISO8601DateFormatter {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }

    private static func whole() -> ISO8601DateFormatter {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }
}

/// A JSON value of no fixed shape: what Go's any becomes.
public enum JSONValue: Codable, Hashable, Sendable {
    case null
    case bool(Bool)
    case number(Double)
    case string(String)
    case array([JSONValue])
    case object([String: JSONValue])

    public init(from decoder: any Decoder) throws {
        let c = try decoder.singleValueContainer()
        if c.decodeNil() {
            self = .null
        } else if let value = try? c.decode(Bool.self) {
            self = .bool(value)
        } else if let value = try? c.decode(Double.self) {
            self = .number(value)
        } else if let value = try? c.decode(String.self) {
            self = .string(value)
        } else if let value = try? c.decode([JSONValue].self) {
            self = .array(value)
        } else if let value = try? c.decode([String: JSONValue].self) {
            self = .object(value)
        } else {
            throw DecodingError.dataCorrupted(.init(codingPath: decoder.codingPath, debugDescription: "not a JSON value"))
        }
    }

    public func encode(to encoder: any Encoder) throws {
        var c = encoder.singleValueContainer()
        switch self {
        case .null: try c.encodeNil()
        case .bool(let value): try c.encode(value)
        case .number(let value): try c.encode(value)
        case .string(let value): try c.encode(value)
        case .array(let value): try c.encode(value)
        case .object(let value): try c.encode(value)
        }
    }
}
`
