package breaking

import (
	"strings"

	"github.com/widmogrod/mkunion/x/shape"
)

// Direction is which way a type travels: clients send requests and the
// server decodes them with the generated Go serde; the server sends
// responses and clients decode them with the exported Swift or TypeScript.
type Direction uint8

const (
	dirRequest  Direction = 1
	dirResponse Direction = 2
)

func (d Direction) request() bool  { return d&dirRequest != 0 }
func (d Direction) response() bool { return d&dirResponse != 0 }
func (d Direction) unknown() bool  { return d == 0 }

func (d Direction) String() string {
	switch d {
	case dirRequest:
		return "request"
	case dirResponse:
		return "response"
	case dirRequest | dirResponse:
		return "request and response"
	}
	return "unknown direction"
}

// index is the named shapes of one snapshot by Go name, with the direction
// each one is reached from.
type index struct {
	shapes   map[string]shape.Shape
	variants map[string]*shape.UnionLike // variant name → its union
	dir      map[string]Direction
	pkg      string // the one package of the snapshot; "" when there are several
}

func newIndex(s *Snapshot) *index {
	ix := &index{shapes: map[string]shape.Shape{}, variants: map[string]*shape.UnionLike{}, dir: map[string]Direction{}}
	pkgs := map[string]bool{}
	for _, x := range s.Shapes {
		ix.shapes[shape.Name(x)] = x
		pkgs[shape.PkgName(x)] = true
		if u, ok := x.(*shape.UnionLike); ok {
			for _, v := range u.Variant {
				ix.variants[shape.Name(v)] = u
			}
		}
	}
	if len(pkgs) == 1 {
		for p := range pkgs {
			ix.pkg = p
		}
	}
	ix.seedDirections()
	return ix
}

// seedDirections marks every union whose variants embed f.Returns as a
// request union, the union it answers with as a response union, and walks
// from both into everything they reference.
func (ix *index) seedDirections() {
	for name, x := range ix.shapes {
		u, ok := x.(*shape.UnionLike)
		if !ok || !shape.UnionDeclaresReturns(u) {
			continue
		}
		ix.mark(name, dirRequest)
		for _, opt := range u.Tags[shape.TagUnionName].Options {
			if answers, ok := strings.CutPrefix(opt, "answers="); ok {
				ix.mark(answers, dirResponse)
			}
		}
		for _, answer := range answersOf(u) {
			ix.mark(shape.Name(unwrapPointer(answer)), dirResponse)
		}
	}
}

// answersOf is every f.Returns[R] answer declared by the union's variants.
func answersOf(u *shape.UnionLike) []shape.Shape {
	var result []shape.Shape
	for _, v := range u.Variant {
		st, ok := v.(*shape.StructLike)
		if !ok {
			continue
		}
		if answer, ok := returnsOf(st); ok {
			result = append(result, answer)
		}
	}
	return result
}

func returnsOf(st *shape.StructLike) (shape.Shape, bool) {
	for _, f := range st.Fields {
		if answer, ok := shape.ReturnsOf(f); ok {
			return answer, true
		}
	}
	return nil, false
}

// mark gives a type a direction and propagates it to what the type refers
// to. A variant name marks its union.
func (ix *index) mark(name string, d Direction) {
	if u, isVariant := ix.variants[name]; isVariant {
		name = u.Name
	}
	x, ok := ix.shapes[name]
	if !ok || ix.dir[name]&d != 0 {
		return
	}
	ix.dir[name] |= d
	for _, ref := range dataRefs(x) {
		ix.mark(ref.Name, d)
	}
}

// dataRefs is every type a shape carries on the wire, so without the
// f.Returns phantom: the answer type is not part of the request.
func dataRefs(x shape.Shape) []*shape.RefName {
	switch y := x.(type) {
	case *shape.StructLike:
		var refs []*shape.RefName
		for _, f := range y.Fields {
			if shape.IsPhantomField(f) {
				continue
			}
			refs = append(refs, shape.ExtractRefs(f.Type)...)
		}
		return refs
	case *shape.UnionLike:
		var refs []*shape.RefName
		for _, v := range y.Variant {
			refs = append(refs, dataRefs(v)...)
		}
		return refs
	case *shape.AliasLike:
		return shape.ExtractRefs(y.Type)
	}
	return shape.ExtractRefs(x)
}

// directionOf is the direction of a type or variant, else unknown.
func (ix *index) directionOf(name string) Direction {
	if u, isVariant := ix.variants[name]; isVariant {
		name = u.Name
	}
	return ix.dir[name]
}

func unwrapPointer(x shape.Shape) shape.Shape {
	for {
		p, ok := x.(*shape.PointerLike)
		if !ok {
			return x
		}
		x = p.Type
	}
}
