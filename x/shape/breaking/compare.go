package breaking

import (
	"fmt"
	"sort"

	"github.com/widmogrod/mkunion/x/shape"
)

// Compare lists what changed from the old snapshot to the new one. loc
// anchors findings in the current source and may be nil.
func Compare(old, current *Snapshot, loc *Locator) []Finding {
	c := &comparer{old: newIndex(old), new: newIndex(current), loc: loc, renames: map[string]string{}}
	c.comparePackage()
	c.findRenames()
	c.compareTopLevel()
	sort.SliceStable(c.findings, func(i, j int) bool {
		if c.findings[i].File != c.findings[j].File {
			return c.findings[i].File < c.findings[j].File
		}
		return c.findings[i].Line < c.findings[j].Line
	})
	return c.findings
}

type comparer struct {
	old, new *index
	loc      *Locator
	renames  map[string]string // old Go type name → new, when a removed and an added type have the same body
	findings []Finding
}

func (c *comparer) add(s Severity, file string, line int, subject, format string, args ...any) {
	c.findings = append(c.findings, Finding{Severity: s, File: file, Line: line, Subject: subject, Message: fmt.Sprintf(format, args...)})
}

func (c *comparer) renamed(oldName string) string {
	if n, ok := c.renames[oldName]; ok {
		return n
	}
	return oldName
}

// directionOf prefers the current snapshot and falls back to the old one
// for what was removed.
func (c *comparer) directionOf(name string) Direction {
	if d := c.new.directionOf(c.renamed(name)); d != 0 {
		return d
	}
	return c.old.directionOf(name)
}

// comparePackage: $type is "pkg.Variant", so a package rename changes
// every union on the wire at once.
func (c *comparer) comparePackage() {
	if c.old.pkg == "" || c.new.pkg == "" || c.old.pkg == c.new.pkg {
		return
	}
	file, _ := c.loc.Type("")
	c.add(Breaking, file, 0, c.new.pkg, "package renamed %s → %s: every $type changes (%s.X → %s.X); old clients and the server reject each other's unions", c.old.pkg, c.new.pkg, c.old.pkg, c.new.pkg)
}

// findRenames pairs removed types with added types of the same body, at
// the top level and among the variants of each union.
func (c *comparer) findRenames() {
	c.matchRenames(c.old.shapes, c.new.shapes)
	for name, x := range c.old.shapes {
		ou, ok := x.(*shape.UnionLike)
		if !ok {
			continue
		}
		nu, ok := c.new.shapes[c.renamed(name)].(*shape.UnionLike)
		if !ok {
			continue
		}
		c.matchRenames(byName(ou.Variant), byName(nu.Variant))
	}
}

func byName(xs []shape.Shape) map[string]shape.Shape {
	m := map[string]shape.Shape{}
	for _, x := range xs {
		m[shape.Name(x)] = x
	}
	return m
}

func (c *comparer) matchRenames(old, current map[string]shape.Shape) {
	taken := map[string]bool{}
	for _, name := range sortedKeys(old) {
		if _, stillThere := current[name]; stillThere {
			continue
		}
		var candidates []string
		for _, added := range sortedKeys(current) {
			if _, existed := old[added]; existed || taken[added] {
				continue
			}
			if c.sameBody(old[name], current[added]) {
				candidates = append(candidates, added)
			}
		}
		if len(candidates) == 1 {
			c.renames[name] = candidates[0]
			taken[candidates[0]] = true
		}
	}
}

// sameBody is true when two named shapes differ only by name.
func (c *comparer) sameBody(a, b shape.Shape) bool {
	switch x := a.(type) {
	case *shape.StructLike:
		y, ok := b.(*shape.StructLike)
		return ok && c.fieldsEqual(x.Fields, y.Fields)
	case *shape.AliasLike:
		y, ok := b.(*shape.AliasLike)
		if !ok {
			return false
		}
		eq, _ := c.typeEqual(x.Type, y.Type)
		return eq
	case *shape.UnionLike:
		y, ok := b.(*shape.UnionLike)
		return ok && sameKeys(byName(x.Variant), byName(y.Variant))
	}
	return false
}

func sameKeys(a, b map[string]shape.Shape) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if _, ok := b[k]; !ok {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]shape.Shape) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (c *comparer) compareTopLevel() {
	targets := map[string]bool{}
	for _, name := range sortedKeys(c.old.shapes) {
		o := c.old.shapes[name]
		newName := c.renamed(name)
		n, ok := c.new.shapes[newName]
		if !ok {
			c.removedType(name, o)
			continue
		}
		targets[newName] = true
		if newName != name {
			file, line := c.loc.Type(newName)
			c.add(Compatible, file, line, newName, "renamed from %s in Go; a type name is not on the wire unless it is a union variant", name)
		}
		c.compareNamed(newName, o, n)
	}
	for _, name := range sortedKeys(c.new.shapes) {
		if _, existed := c.old.shapes[name]; existed || targets[name] {
			continue
		}
		file, line := c.loc.Type(name)
		c.add(Compatible, file, line, name, "added (%s)", c.new.directionOf(name))
	}
}

func (c *comparer) removedType(name string, o shape.Shape) {
	file, line := c.loc.Type(name)
	if _, isUnion := o.(*shape.UnionLike); isUnion {
		c.add(Breaking, file, line, name, "union removed; every message that carried it changes shape")
		return
	}
	c.add(Compatible, file, line, name, "removed; a field that used it reports its own type change")
}

func (c *comparer) compareNamed(name string, o, n shape.Shape) {
	switch x := o.(type) {
	case *shape.UnionLike:
		if y, ok := n.(*shape.UnionLike); ok {
			c.compareUnion(x, y)
			return
		}
	case *shape.StructLike:
		if y, ok := n.(*shape.StructLike); ok {
			c.compareStruct(name, c.directionOf(name), x, y)
			return
		}
	case *shape.AliasLike:
		if y, ok := n.(*shape.AliasLike); ok {
			c.compareAlias(name, x, y)
			return
		}
	}
	file, line := c.loc.Type(name)
	c.add(Breaking, file, line, name, "changed kind from %s to %s", kindOf(o), kindOf(n))
}

func kindOf(x shape.Shape) string {
	switch x.(type) {
	case *shape.UnionLike:
		return "union"
	case *shape.StructLike:
		return "struct"
	case *shape.AliasLike:
		return "alias"
	}
	return "other"
}

func (c *comparer) compareAlias(name string, o, n *shape.AliasLike) {
	eq, widthOnly := c.typeEqual(o.Type, n.Type)
	if eq {
		return
	}
	file, line := c.loc.Type(name)
	if widthOnly {
		c.add(Warn, file, line, name, "number width changed %s → %s: same JSON number, Swift and TypeScript types change", c.describe(o.Type), c.describe(n.Type))
		return
	}
	c.add(Breaking, file, line, name, "underlying type changed %s → %s", c.describe(o.Type), c.describe(n.Type))
}

func (c *comparer) compareUnion(o, n *shape.UnionLike) {
	dir := c.directionOf(n.Name)
	oldV, newV := byName(o.Variant), byName(n.Variant)
	matched := map[string]bool{}
	for _, name := range sortedKeys(oldV) {
		newName := c.renamed(name)
		nv, ok := newV[newName]
		if !ok {
			c.removedVariant(n, dir, name)
			continue
		}
		matched[newName] = true
		if newName != name {
			file, line := c.loc.Type(newName)
			c.add(Breaking, file, line, n.Name, "variant %s renamed to %s: $type %s.%s becomes %s.%s; old clients %s", name, newName, o.PkgName, name, n.PkgName, newName, unknownTypeEffect(dir))
		}
		c.compareVariant(n.Name, dir, oldV[name], nv)
	}
	for _, name := range sortedKeys(newV) {
		if _, existed := oldV[name]; existed || matched[name] {
			continue
		}
		c.addedVariant(n, dir, name)
	}
}

// unknownTypeEffect is what an old party does with a $type it does not know.
func unknownTypeEffect(dir Direction) string {
	switch {
	case dir.request() && dir.response():
		return "sending it get ErrUnknownType from the server and decoding it throw unknownType (Swift)"
	case dir.request():
		return "sending it get ErrUnknownType from the server (RequestFromJSON)"
	case dir.response():
		return "decoding it throw MkunionJSON.Error.unknownType (Swift) or see an object without a known key (TypeScript)"
	}
	return "on either side reject it as an unknown $type"
}

func (c *comparer) removedVariant(u *shape.UnionLike, dir Direction, name string) {
	file, line := c.loc.Type(u.Name)
	switch {
	case dir.request():
		c.add(Breaking, file, line, u.Name, "variant %s removed; clients sending it get ErrUnknownType from the server (unknown_request)", name)
	case dir.response():
		c.add(Warn, file, line, u.Name, "variant %s removed from a response union; old clients keep a case the server never sends, regenerated clients lose it (source change, not wire)", name)
	default:
		c.add(Breaking, file, line, u.Name, "variant %s removed; a party still sending it gets an unknown $type error", name)
	}
}

func (c *comparer) addedVariant(u *shape.UnionLike, dir Direction, name string) {
	file, line := c.loc.Type(name)
	switch {
	case dir.response():
		c.add(Breaking, file, line, u.Name, "variant %s added to a response union; old clients that receive it throw unknownType (Swift) or fall through to an object with no known key (TypeScript)", name)
	case dir.request():
		c.add(Compatible, file, line, u.Name, "variant %s added: a new operation old clients never send", name)
	default:
		c.add(Warn, file, line, u.Name, "variant %s added; the union is not reached from any f.Returns union, so whether old clients decode it is unknown", name)
	}
}

func (c *comparer) compareVariant(union string, dir Direction, o, n shape.Shape) {
	name := shape.Name(n)
	switch x := o.(type) {
	case *shape.StructLike:
		if y, ok := n.(*shape.StructLike); ok {
			c.compareStruct(name, dir, x, y)
			c.compareReturns(name, x, y)
			return
		}
	case *shape.AliasLike:
		if y, ok := n.(*shape.AliasLike); ok {
			c.compareAlias(name, x, y)
			return
		}
	}
	file, line := c.loc.Type(name)
	c.add(Breaking, file, line, union, "variant %s changed kind from %s to %s", name, kindOf(o), kindOf(n))
}

// compareReturns: f.Returns[R] names the answer every client's send(op)
// decodes, so a different R fails every call of the operation.
func (c *comparer) compareReturns(name string, o, n *shape.StructLike) {
	oa, hadAnswer := returnsOf(o)
	na, hasAnswer := returnsOf(n)
	file, line := c.loc.Field(name, "Returns")
	switch {
	case hadAnswer && hasAnswer:
		if eq, _ := c.typeEqual(oa, na); !eq {
			c.add(Breaking, file, line, name, "answer type changed f.Returns[%s] → f.Returns[%s]; every client's send(%s) decodes the old answer and fails", c.describe(oa), c.describe(na), name)
		}
	case hadAnswer:
		c.add(Warn, file, line, name, "f.Returns[%s] removed; the variant no longer names an answer, generated clients lose send(%s) (source change)", c.describe(oa), name)
	case hasAnswer:
		c.add(Compatible, file, line, name, "now declares its answer f.Returns[%s]; generated clients gain send(%s)", c.describe(na), name)
	}
}

func (c *comparer) compareStruct(path string, dir Direction, o, n *shape.StructLike) {
	oldFields, newFields := wireFields(o), wireFields(n)
	newByName, newByJSON := map[string]*shape.FieldLike{}, map[string]*shape.FieldLike{}
	for _, f := range newFields {
		newByName[f.Name] = f
		newByJSON[jsonName(f)] = f
	}
	oldByName := map[string]*shape.FieldLike{}
	for _, f := range oldFields {
		oldByName[f.Name] = f
	}
	matched := map[string]bool{}
	for _, f := range oldFields {
		g, ok := newByName[f.Name]
		if !ok {
			g, ok = newByJSON[jsonName(f)]
			if ok && oldByName[g.Name] == nil {
				file, line := c.loc.Field(n.Name, g.Name)
				c.add(Compatible, file, line, path+"."+g.Name, "renamed from %s in Go; wire name %q unchanged", f.Name, jsonName(g))
			} else {
				c.removedField(path, dir, n.Name, f)
				continue
			}
		}
		matched[g.Name] = true
		c.compareField(path, dir, n.Name, f, g)
	}
	for _, g := range newFields {
		if !matched[g.Name] {
			c.addedField(path, dir, n.Name, g)
		}
	}
}

func (c *comparer) removedField(path string, dir Direction, typ string, f *shape.FieldLike) {
	file, line := c.loc.Type(typ)
	subject := path + "." + f.Name
	switch {
	case dir.response():
		c.add(Breaking, file, line, subject, "field %q removed; clients read it (Swift decodes the zero value, TypeScript sees undefined)", jsonName(f))
	case dir.request():
		c.add(Compatible, file, line, subject, "field %q removed; old clients still send it and the server ignores unknown keys", jsonName(f))
	default:
		c.add(Warn, file, line, subject, "field %q removed (%s: the type is not reached from a union)", jsonName(f), dir)
	}
}

func (c *comparer) addedField(path string, dir Direction, typ string, g *shape.FieldLike) {
	file, line := c.loc.Field(typ, g.Name)
	subject := path + "." + g.Name
	switch {
	case dir.request() && !omitempty(g):
		c.add(Warn, file, line, subject, "field %q added without omitempty; old clients never send it and the server reads the zero value", jsonName(g))
	case dir.request():
		c.add(Compatible, file, line, subject, "field %q added with omitempty; old clients omit it, as new ones do for the zero value", jsonName(g))
	case dir.response():
		c.add(Compatible, file, line, subject, "field %q added; old clients ignore unknown keys", jsonName(g))
	default:
		c.add(Warn, file, line, subject, "field %q added (%s)", jsonName(g), dir)
	}
}

func (c *comparer) compareField(path string, dir Direction, typ string, f, g *shape.FieldLike) {
	file, line := c.loc.Field(typ, g.Name)
	subject := path + "." + g.Name
	if jsonName(f) != jsonName(g) {
		c.add(Breaking, file, line, subject, "json name changed %q → %q; old messages carry %q and the reader sees the zero value", jsonName(f), jsonName(g), jsonName(f))
	}
	c.compareNullability(subject, dir, file, line, shape.IsPointer(f.Type), shape.IsPointer(g.Type))
	ot, nt := unwrapPointer(f.Type), unwrapPointer(g.Type)
	if eq, widthOnly := c.typeEqual(ot, nt); !eq {
		if widthOnly {
			c.add(Warn, file, line, subject, "number width changed %s → %s: same JSON number, Swift and TypeScript types change", c.describe(ot), c.describe(nt))
		} else {
			c.add(Breaking, file, line, subject, "type changed %s → %s", c.describe(f.Type), c.describe(g.Type))
		}
	}
	if omitempty(f) != omitempty(g) {
		if omitempty(g) {
			c.add(Compatible, file, line, subject, "omitempty added; a zero value is now omitted and every reader treats a missing key as the zero value")
		} else {
			c.add(Compatible, file, line, subject, "omitempty removed; the zero value is now always sent")
		}
	}
}

// compareNullability: a pointer is Optional in Swift and T | null in
// TypeScript, so adding one to a response changes the generated client.
func (c *comparer) compareNullability(subject string, dir Direction, file string, line int, wasPointer, isPointer bool) {
	switch {
	case wasPointer == isPointer:
	case isPointer && dir.response():
		c.add(Breaking, file, line, subject, "became a pointer; Optional in Swift, nullable in TypeScript, and the server may now send null")
	case isPointer && dir.request():
		c.add(Compatible, file, line, subject, "became a pointer; the server now accepts null, old clients always send a value")
	case isPointer:
		c.add(Warn, file, line, subject, "became a pointer (%s)", dir)
	case dir.request():
		c.add(Warn, file, line, subject, "no longer a pointer; an old client that sends null makes the server read the zero value")
	case dir.response():
		c.add(Compatible, file, line, subject, "no longer a pointer; old clients keep an Optional that is always set")
	default:
		c.add(Warn, file, line, subject, "no longer a pointer (%s)", dir)
	}
}
