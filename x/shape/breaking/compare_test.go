package breaking

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shape"
)

// Tiny shapes, the way InferFromFile builds them for package api.

func str() shape.Shape { return &shape.PrimitiveLike{Kind: &shape.StringLike{}} }
func integer() shape.Shape {
	return &shape.PrimitiveLike{Kind: &shape.NumberLike{Kind: &shape.Int{}}}
}
func int64s() shape.Shape {
	return &shape.PrimitiveLike{Kind: &shape.NumberLike{Kind: &shape.Int64{}}}
}
func float() shape.Shape {
	return &shape.PrimitiveLike{Kind: &shape.NumberLike{Kind: &shape.Float64{}}}
}
func ptr(x shape.Shape) shape.Shape  { return &shape.PointerLike{Type: x} }
func list(x shape.Shape) shape.Shape { return &shape.ListLike{Element: x} }
func ref(name string) shape.Shape {
	return &shape.RefName{Name: name, PkgName: "api", PkgImportName: "example.com/api"}
}

func field(name, tag string, t shape.Shape) *shape.FieldLike {
	return &shape.FieldLike{Name: name, Type: t, Tags: shape.ExtractTags(tag)}
}

func returns(answer shape.Shape) *shape.FieldLike {
	return &shape.FieldLike{Name: "Returns", Type: &shape.RefName{
		Name: "Returns", PkgName: "f", PkgImportName: "github.com/widmogrod/mkunion/f", Indexed: []shape.Shape{answer},
	}}
}

func structOf(name string, fields ...*shape.FieldLike) *shape.StructLike {
	return &shape.StructLike{Name: name, PkgName: "api", PkgImportName: "example.com/api", Fields: fields}
}

func alias(name string, t shape.Shape) *shape.AliasLike {
	return &shape.AliasLike{Name: name, PkgName: "api", PkgImportName: "example.com/api", Type: t}
}

// union tags its variants with mkunion:"Name" as injectUnionNameTags does.
func union(name string, options []string, variants ...shape.Shape) *shape.UnionLike {
	for _, v := range variants {
		if st, ok := v.(*shape.StructLike); ok {
			st.Tags = map[string]shape.Tag{shape.TagUnionName: {Value: name}}
		}
	}
	return &shape.UnionLike{
		Name: name, PkgName: "api", PkgImportName: "example.com/api", Variant: variants,
		Tags: map[string]shape.Tag{shape.TagUnionName: {Value: name, Options: options}},
	}
}

// api is Request{Get{f.Returns[*Answer], ID}} answered by Response{Answer{...}}, plus extra types.
func api(get *shape.StructLike, answer *shape.StructLike, extra ...shape.Shape) *Snapshot {
	shapes := []shape.Shape{
		union("Request", []string{"answers=Response"}, get),
		union("Response", nil, answer),
	}
	return &Snapshot{Shapes: append(shapes, extra...)}
}

func get(fields ...*shape.FieldLike) *shape.StructLike {
	return structOf("Get", append([]*shape.FieldLike{returns(ptr(ref("Answer")))}, fields...)...)
}

func answer(fields ...*shape.FieldLike) *shape.StructLike { return structOf("Answer", fields...) }

type want struct {
	severity Severity
	subject  string
	message  string // a substring
}

func TestCompareRules(t *testing.T) {
	cases := []struct {
		name     string
		old, new *Snapshot
		want     []want
	}{
		{
			name: "no change",
			old:  api(get(field("ID", `json:"id"`, str())), answer(field("Name", `json:"name"`, str()))),
			new:  api(get(field("ID", `json:"id"`, str())), answer(field("Name", `json:"name"`, str()))),
		},
		{
			name: "request variant removed",
			old:  &Snapshot{Shapes: []shape.Shape{union("Request", nil, get(), structOf("Ping", returns(ptr(ref("Answer"))))), union("Response", nil, answer())}},
			new:  &Snapshot{Shapes: []shape.Shape{union("Request", nil, get()), union("Response", nil, answer())}},
			want: []want{{Breaking, "Request", "variant Ping removed"}},
		},
		{
			name: "request variant added",
			old:  &Snapshot{Shapes: []shape.Shape{union("Request", nil, get()), union("Response", nil, answer())}},
			new:  &Snapshot{Shapes: []shape.Shape{union("Request", nil, get(), structOf("Ping", returns(ptr(ref("Answer"))))), union("Response", nil, answer())}},
			want: []want{{Compatible, "Request", "variant Ping added"}},
		},
		{
			name: "response variant added",
			old:  api(get(), answer()),
			new:  &Snapshot{Shapes: []shape.Shape{union("Request", []string{"answers=Response"}, get()), union("Response", nil, answer(), structOf("Error", field("Code", `json:"code"`, str())))}},
			want: []want{{Breaking, "Response", "variant Error added to a response union"}},
		},
		{
			name: "response variant removed",
			old:  &Snapshot{Shapes: []shape.Shape{union("Request", []string{"answers=Response"}, get()), union("Response", nil, answer(), structOf("Error", field("Code", `json:"code"`, str())))}},
			new:  api(get(), answer()),
			want: []want{{Warn, "Response", "variant Error removed from a response union"}},
		},
		{
			name: "variant added to a union of unknown direction",
			old:  &Snapshot{Shapes: []shape.Shape{union("State", nil, structOf("Queued"))}},
			new:  &Snapshot{Shapes: []shape.Shape{union("State", nil, structOf("Queued"), structOf("Done"))}},
			want: []want{{Warn, "State", "variant Done added"}},
		},
		{
			name: "variant removed from a union of unknown direction",
			old:  &Snapshot{Shapes: []shape.Shape{union("State", nil, structOf("Queued"), structOf("Done"))}},
			new:  &Snapshot{Shapes: []shape.Shape{union("State", nil, structOf("Queued"))}},
			want: []want{{Breaking, "State", "variant Done removed"}},
		},
		{
			name: "nested union reached from a response is a response union",
			old:  api(get(), answer(field("State", `json:"state"`, ref("State"))), union("State", nil, structOf("Queued"))),
			new:  api(get(), answer(field("State", `json:"state"`, ref("State"))), union("State", nil, structOf("Queued"), structOf("Done"))),
			want: []want{{Breaking, "State", "variant Done added to a response union"}},
		},
		{
			name: "variant renamed changes $type",
			old:  api(get(), answer(field("Name", `json:"name"`, str()))),
			new:  api(structOf("Get", returns(ptr(ref("Reply")))), structOf("Reply", field("Name", `json:"name"`, str()))),
			want: []want{{Breaking, "Response", "variant Answer renamed to Reply: $type api.Answer becomes api.Reply"}},
		},
		{
			name: "response field removed",
			old:  api(get(), answer(field("Name", `json:"name"`, str()), field("Age", `json:"age"`, integer()))),
			new:  api(get(), answer(field("Name", `json:"name"`, str()))),
			want: []want{{Breaking, "Answer.Age", `field "age" removed; clients read it`}},
		},
		{
			name: "request field removed",
			old:  api(get(field("ID", `json:"id"`, str()), field("Hint", `json:"hint"`, str())), answer()),
			new:  api(get(field("ID", `json:"id"`, str())), answer()),
			want: []want{{Compatible, "Get.Hint", `field "hint" removed; old clients still send it and the server ignores unknown keys`}},
		},
		{
			name: "request field added without omitempty",
			old:  api(get(field("ID", `json:"id"`, str())), answer()),
			new:  api(get(field("ID", `json:"id"`, str()), field("Limit", `json:"limit"`, integer())), answer()),
			want: []want{{Warn, "Get.Limit", `field "limit" added without omitempty`}},
		},
		{
			name: "request field added with omitempty",
			old:  api(get(field("ID", `json:"id"`, str())), answer()),
			new:  api(get(field("ID", `json:"id"`, str()), field("Limit", `json:"limit,omitempty"`, integer())), answer()),
			want: []want{{Compatible, "Get.Limit", `field "limit" added with omitempty`}},
		},
		{
			name: "response field added",
			old:  api(get(), answer(field("Name", `json:"name"`, str()))),
			new:  api(get(), answer(field("Name", `json:"name"`, str()), field("Age", `json:"age"`, integer()))),
			want: []want{{Compatible, "Answer.Age", `field "age" added; old clients ignore unknown keys`}},
		},
		{
			name: "json name changed",
			old:  api(get(field("Query", `json:"query"`, str())), answer()),
			new:  api(get(field("Query", `json:"q"`, str())), answer()),
			want: []want{{Breaking, "Get.Query", `json name changed "query" → "q"`}},
		},
		{
			name: "Go field renamed, wire name kept",
			old:  api(get(field("Query", `json:"q"`, str())), answer()),
			new:  api(get(field("Q", `json:"q"`, str())), answer()),
			want: []want{{Compatible, "Get.Q", `renamed from Query in Go; wire name "q" unchanged`}},
		},
		{
			name: "type changed string to int",
			old:  api(get(field("Limit", `json:"limit"`, str())), answer()),
			new:  api(get(field("Limit", `json:"limit"`, integer())), answer()),
			want: []want{{Breaking, "Get.Limit", "type changed string → int"}},
		},
		{
			name: "type changed T to []T",
			old:  api(get(), answer(field("Tag", `json:"tag"`, str()))),
			new:  api(get(), answer(field("Tag", `json:"tag"`, list(str())))),
			want: []want{{Breaking, "Answer.Tag", "type changed string → []string"}},
		},
		{
			name: "int to float is not a width change",
			old:  api(get(), answer(field("N", `json:"n"`, integer()))),
			new:  api(get(), answer(field("N", `json:"n"`, float()))),
			want: []want{{Breaking, "Answer.N", "type changed int → float64"}},
		},
		{
			name: "integer width changed",
			old:  api(get(), answer(field("N", `json:"n"`, integer()))),
			new:  api(get(), answer(field("N", `json:"n"`, int64s()))),
			want: []want{{Warn, "Answer.N", "number width changed int → int64"}},
		},
		{
			name: "response field became a pointer",
			old:  api(get(), answer(field("Items", `json:"items"`, list(ref("Item")))), structOf("Item")),
			new:  api(get(), answer(field("Items", `json:"items"`, ptr(list(ref("Item"))))), structOf("Item")),
			want: []want{{Breaking, "Answer.Items", "became a pointer; Optional in Swift"}},
		},
		{
			name: "response field no longer a pointer",
			old:  api(get(), answer(field("Name", `json:"name"`, ptr(str())))),
			new:  api(get(), answer(field("Name", `json:"name"`, str()))),
			want: []want{{Compatible, "Answer.Name", "no longer a pointer; old clients keep an Optional"}},
		},
		{
			name: "request field became a pointer",
			old:  api(get(field("ID", `json:"id"`, str())), answer()),
			new:  api(get(field("ID", `json:"id"`, ptr(str()))), answer()),
			want: []want{{Compatible, "Get.ID", "became a pointer; the server now accepts null"}},
		},
		{
			name: "request field no longer a pointer",
			old:  api(get(field("ID", `json:"id"`, ptr(str()))), answer()),
			new:  api(get(field("ID", `json:"id"`, str())), answer()),
			want: []want{{Warn, "Get.ID", "no longer a pointer; an old client that sends null"}},
		},
		{
			name: "omitempty added and removed",
			old:  api(get(field("A", `json:"a"`, str()), field("B", `json:"b,omitempty"`, str())), answer()),
			new:  api(get(field("A", `json:"a,omitempty"`, str()), field("B", `json:"b"`, str())), answer()),
			want: []want{{Compatible, "Get.A", "omitempty added"}, {Compatible, "Get.B", "omitempty removed"}},
		},
		{
			name: "f.Returns answer changed",
			old:  api(get(), answer(), structOf("Other", field("X", `json:"x"`, str()))),
			new:  api(structOf("Get", returns(ptr(ref("Other")))), answer(), structOf("Other", field("X", `json:"x"`, str()))),
			want: []want{{Breaking, "Get", "answer type changed f.Returns[*Answer] → f.Returns[*Other]"}},
		},
		{
			name: "f.Returns removed and added",
			old:  &Snapshot{Shapes: []shape.Shape{union("Request", nil, get(), structOf("Ping")), union("Response", nil, answer())}},
			new:  &Snapshot{Shapes: []shape.Shape{union("Request", nil, structOf("Get"), structOf("Ping", returns(ptr(ref("Answer"))))), union("Response", nil, answer())}},
			want: []want{{Warn, "Get", "f.Returns[*Answer] removed"}, {Compatible, "Ping", "now declares its answer f.Returns[*Answer]"}},
		},
		{
			name: "package renamed",
			old:  api(get(), answer()),
			new: func() *Snapshot {
				s := api(get(), answer())
				for _, x := range s.Shapes {
					u := x.(*shape.UnionLike)
					u.PkgName = "api2"
					for _, v := range u.Variant {
						v.(*shape.StructLike).PkgName = "api2"
					}
				}
				return s
			}(),
			want: []want{{Breaking, "api2", "package renamed api → api2: every $type changes"}},
		},
		{
			name: "plain struct renamed is not on the wire",
			old:  api(get(), answer(field("Items", `json:"items"`, list(ref("Item")))), structOf("Item", field("N", `json:"n"`, integer()))),
			new:  api(get(), answer(field("Items", `json:"items"`, list(ref("Row")))), structOf("Row", field("N", `json:"n"`, integer()))),
			want: []want{{Compatible, "Row", "renamed from Item in Go; a type name is not on the wire"}},
		},
		{
			name: "plain struct removed and added",
			old:  api(get(), answer(), structOf("Old", field("N", `json:"n"`, integer()))),
			new:  api(get(), answer(), structOf("New", field("S", `json:"s"`, str()))),
			want: []want{{Compatible, "Old", "removed"}, {Compatible, "New", "added"}},
		},
		{
			name: "alias underlying type changed",
			old:  api(get(), answer(field("Kind", `json:"kind"`, ref("Kind"))), alias("Kind", str())),
			new:  api(get(), answer(field("Kind", `json:"kind"`, ref("Kind"))), alias("Kind", integer())),
			want: []want{{Breaking, "Kind", "underlying type changed string → int"}},
		},
		{
			name: "union removed",
			old:  api(get(), answer(), union("State", nil, structOf("Queued"))),
			new:  api(get(), answer()),
			want: []want{{Breaking, "State", "union removed"}},
		},
		{
			name: "kind changed",
			old:  api(get(), answer(), structOf("Kind", field("N", `json:"n"`, integer()))),
			new:  api(get(), answer(), alias("Kind", str())),
			want: []want{{Breaking, "Kind", "changed kind from struct to alias"}},
		},
		{
			name: "field removed from a type of unknown direction",
			old:  &Snapshot{Shapes: []shape.Shape{structOf("Loose", field("A", `json:"a"`, str()), field("B", `json:"b"`, str()))}},
			new:  &Snapshot{Shapes: []shape.Shape{structOf("Loose", field("A", `json:"a"`, str()))}},
			want: []want{{Warn, "Loose.B", `field "b" removed (unknown direction`}},
		},
		{
			name: "phantom and json:- fields are not on the wire",
			old:  api(get(field("Skip", `json:"-"`, str())), answer()),
			new:  api(get(), answer()),
		},
		{
			name: "map, union, alias, any and indexed types compare structurally",
			old: api(get(), answer(
				field("M", `json:"m"`, &shape.MapLike{Key: str(), Val: integer()}),
				field("U", `json:"u"`, &shape.UnionLike{Name: "State", PkgName: "api"}),
				field("A", `json:"a"`, &shape.AliasLike{Name: "Kind", PkgName: "api", Type: str()}),
				field("Any", `json:"any"`, &shape.Any{}),
				field("G", `json:"g"`, &shape.RefName{Name: "Box", PkgName: "api", Indexed: []shape.Shape{str()}}),
				field("In", `json:"in"`, &shape.StructLike{Fields: []*shape.FieldLike{field("X", `json:"x"`, str())}}),
				field("Arr", `json:"arr"`, &shape.ListLike{Element: str(), ArrayLen: shape.Ptr(2)}),
			)),
			new: api(get(), answer(
				field("M", `json:"m"`, &shape.MapLike{Key: str(), Val: integer()}),
				field("U", `json:"u"`, &shape.UnionLike{Name: "State", PkgName: "api"}),
				field("A", `json:"a"`, &shape.AliasLike{Name: "Kind", PkgName: "api", Type: str()}),
				field("Any", `json:"any"`, &shape.Any{}),
				field("G", `json:"g"`, &shape.RefName{Name: "Box", PkgName: "api", Indexed: []shape.Shape{str()}}),
				field("In", `json:"in"`, &shape.StructLike{Fields: []*shape.FieldLike{field("X", `json:"x"`, str())}}),
				field("Arr", `json:"arr"`, &shape.ListLike{Element: str()}),
			)),
		},
		{
			name: "map value, indexed type, inline struct and bool changes are type changes",
			old: api(get(), answer(
				field("M", `json:"m"`, &shape.MapLike{Key: str(), Val: integer()}),
				field("G", `json:"g"`, &shape.RefName{Name: "Box", PkgName: "api", Indexed: []shape.Shape{str()}}),
				field("In", `json:"in"`, &shape.StructLike{Fields: []*shape.FieldLike{field("X", `json:"x"`, str())}}),
				field("B", `json:"b"`, &shape.PrimitiveLike{Kind: &shape.BooleanLike{}}),
				field("U", `json:"u"`, &shape.UnionLike{Name: "State", PkgName: "api"}),
				field("Any", `json:"any"`, &shape.Any{}),
			)),
			new: api(get(), answer(
				field("M", `json:"m"`, &shape.MapLike{Key: str(), Val: str()}),
				field("G", `json:"g"`, &shape.RefName{Name: "Box", PkgName: "api", Indexed: []shape.Shape{integer()}}),
				field("In", `json:"in"`, &shape.StructLike{Fields: []*shape.FieldLike{field("X", `json:"y"`, str())}}),
				field("B", `json:"b"`, str()),
				field("U", `json:"u"`, &shape.UnionLike{Name: "Other", PkgName: "api"}),
				field("Any", `json:"any"`, str()),
			)),
			want: []want{
				{Breaking, "Answer.M", "type changed map[string]int → map[string]string"},
				{Breaking, "Answer.G", "type changed"},
				{Breaking, "Answer.In", "type changed"},
				{Breaking, "Answer.B", "type changed bool → string"},
				{Breaking, "Answer.U", "type changed"},
				{Breaking, "Answer.Any", "type changed"},
			},
		},
		{
			name: "union renamed is not on the wire",
			old:  api(get(), answer(field("S", `json:"s"`, ref("State"))), union("State", nil, structOf("Queued"), structOf("Done"))),
			new:  api(get(), answer(field("S", `json:"s"`, ref("Phase"))), union("Phase", nil, structOf("Queued"), structOf("Done"))),
			want: []want{{Compatible, "Phase", "renamed from State in Go"}},
		},
		{
			name: "variant renamed in a request union",
			old:  &Snapshot{Shapes: []shape.Shape{union("Request", nil, get(), structOf("Ping", returns(ptr(ref("Answer"))))), union("Response", nil, answer())}},
			new:  &Snapshot{Shapes: []shape.Shape{union("Request", nil, get(), structOf("Pong", returns(ptr(ref("Answer"))))), union("Response", nil, answer())}},
			want: []want{{Breaking, "Request", "variant Ping renamed to Pong: $type api.Ping becomes api.Pong; old clients sending it get ErrUnknownType"}},
		},
		{
			name: "variant renamed in a union of unknown direction",
			old:  &Snapshot{Shapes: []shape.Shape{union("State", nil, structOf("Queued"))}},
			new:  &Snapshot{Shapes: []shape.Shape{union("State", nil, structOf("Waiting"))}},
			want: []want{{Breaking, "State", "old clients on either side reject it as an unknown $type"}},
		},
		{
			name: "variant renamed in a union used both ways",
			old:  api(get(field("S", `json:"s"`, ref("State"))), answer(field("S", `json:"s"`, ref("State"))), union("State", nil, structOf("Queued"))),
			new:  api(get(field("S", `json:"s"`, ref("State"))), answer(field("S", `json:"s"`, ref("State"))), union("State", nil, structOf("Waiting"))),
			want: []want{{Breaking, "State", "get ErrUnknownType from the server and decoding it throw unknownType"}},
		},
		{
			name: "variant changed kind, alias variants compared",
			old:  &Snapshot{Shapes: []shape.Shape{union("V", nil, structOf("A"), alias("B", str()), alias("C", str()))}},
			new:  &Snapshot{Shapes: []shape.Shape{union("V", nil, alias("A", str()), alias("B", integer()), alias("C", str()))}},
			want: []want{{Breaking, "V", "variant A changed kind from struct to alias"}, {Breaking, "B", "underlying type changed string → int"}},
		},
		{
			name: "alias width change warns",
			old:  &Snapshot{Shapes: []shape.Shape{alias("N", integer())}},
			new:  &Snapshot{Shapes: []shape.Shape{alias("N", int64s())}},
			want: []want{{Warn, "N", "number width changed int → int64"}},
		},
		{
			name: "pointer changes on a type of unknown direction warn",
			old:  &Snapshot{Shapes: []shape.Shape{structOf("Loose", field("A", `json:"a"`, str()), field("B", `json:"b"`, ptr(str())))}},
			new:  &Snapshot{Shapes: []shape.Shape{structOf("Loose", field("A", `json:"a"`, ptr(str())), field("B", `json:"b"`, str()), field("C", `json:"c"`, str()))}},
			want: []want{{Warn, "Loose.A", "became a pointer (unknown direction)"}, {Warn, "Loose.B", "no longer a pointer (unknown direction)"}, {Warn, "Loose.C", `field "c" added (unknown direction)`}},
		},
		{
			name: "union with a different variant set is not a rename",
			old:  api(get(), answer(), union("State", nil, structOf("Queued"))),
			new:  api(get(), answer(), union("Phase", nil, structOf("Queued"), structOf("Done"))),
			want: []want{{Breaking, "State", "union removed"}, {Compatible, "Phase", "added"}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := Compare(tc.old, tc.new, nil)
			var got []string
			for _, f := range findings {
				got = append(got, f.String())
			}
			require.Len(t, findings, len(tc.want), "findings:\n%s", strings.Join(got, "\n"))
			for _, w := range tc.want {
				assert.True(t, hasFinding(findings, w), "want %s: %s: %q in:\n%s", w.severity, w.subject, w.message, strings.Join(got, "\n"))
			}
		})
	}
}

func hasFinding(findings []Finding, w want) bool {
	for _, f := range findings {
		if f.Severity == w.severity && f.Subject == w.subject && strings.Contains(f.Message, w.message) {
			return true
		}
	}
	return false
}

func TestFindingString(t *testing.T) {
	assert.Equal(t, "api.go:45: breaking: Request: variant ListJobs removed", Finding{Breaking, "api.go", 45, "Request", "variant ListJobs removed"}.String())
	assert.Equal(t, "api.go: warn: Request: x", Finding{Warn, "api.go", 0, "Request", "x"}.String())
	assert.Equal(t, "compatible: Request: x", Finding{Compatible, "", 0, "Request", "x"}.String())
	assert.Equal(t, 1, CountBreaking([]Finding{{Severity: Breaking}, {Severity: Warn}}))
}

func TestDirectionString(t *testing.T) {
	assert.Equal(t, "request", dirRequest.String())
	assert.Equal(t, "response", dirResponse.String())
	assert.Equal(t, "request and response", (dirRequest | dirResponse).String())
	assert.Equal(t, "unknown direction", Direction(0).String())
	assert.True(t, Direction(0).unknown())
}
