package testasset

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shared"
)

// swiftCase is one value pushed through the Swift the exporter wrote for
// type_api.go: Go encodes want (or hands over wire as written), the driver
// under swift/driver decodes and re-encodes it with the generated types, and
// Go decodes the result and compares it to want. With throws set, Swift must
// refuse the input with that MkunionJSON.Error case instead.
type swiftCase struct {
	name   string
	kind   string // query, answer or status: the union Swift decodes into
	wire   string // hand-written JSON for Swift; empty means Go encodes want
	want   any    // the Go value Swift's output must decode to; nil is a nil union
	throws string // the MkunionJSON.Error case Swift must throw
}

func swiftCases() []swiftCase {
	// Milliseconds only: the Swift side keeps three fractional digits.
	updated := time.Date(2026, 11, 5, 21, 36, 16, 123000000, time.UTC)
	cover := "front.jpg"
	label := Label("x")
	document := func(status Status) *Document {
		return &Document{
			Name:    "One Piece",
			Cover:   &cover,
			Pages:   map[string]int{"001.jpg": 1, "002.jpg": 2},
			Raw:     []byte{0, 1, 2, 255},
			Updated: updated,
			Status:  status,
			Secret:  "", // json:"-": never on the wire
		}
	}
	return []swiftCase{
		// Every variant of every union, as Go writes it.
		{name: "Ping", kind: "query", want: &Ping{}},
		{name: "Find", kind: "query", want: &Find{Text: "naruto", Limit: 5, Tags: []string{"shonen", "ongoing"}}},
		{name: "Open", kind: "query", want: &Open{URL: "https://example.test/one-piece"}},
		{name: "nil Query", kind: "query", want: nil},
		{name: "Pong", kind: "answer", want: &Pong{Version: "1.2.3"}},
		{name: "Found", kind: "answer", want: &Found{Hits: []Hit{{Name: "a", Score: 1.5}, {Name: "b", Score: 0.25}}, Took: 12.5}},
		{name: "Document with Draft", kind: "answer", want: document(&Draft{})},
		{name: "Document with Published", kind: "answer", want: document(&Published{At: updated})},
		{name: "Document with Label", kind: "answer", want: document(&label)},
		{name: "Document zero", kind: "answer", want: &Document{}},
		{name: "Failure", kind: "answer", want: &Failure{Code: "E_NOPE", Message: "not here"}},
		{name: "nil Answer", kind: "answer", want: nil},
		{name: "Draft", kind: "status", want: &Draft{}},
		{name: "Published", kind: "status", want: &Published{At: updated}},
		{name: "Label", kind: "status", want: &label},
		{name: "nil Status", kind: "status", want: nil},
		// Lenient decoding: documents Go accepts, written by hand.
		{name: "no payload", kind: "query", wire: `{"$type":"testasset.Ping"}`, want: &Ping{}},
		{name: "null payload", kind: "query", wire: `{"$type":"testasset.Find","testasset.Find":null}`, want: &Find{}},
		{name: "key without $type", kind: "query", wire: `{"testasset.Open":{"url":"u"}}`, want: &Open{URL: "u"}},
		{name: "null status", kind: "answer", wire: `{"$type":"testasset.Document","testasset.Document":{"name":"x","status":null,"updated":"0001-01-01T00:00:00Z"}}`, want: &Document{Name: "x"}},
		{
			name: "zeroDate for a missing time", kind: "answer",
			wire: `{"$type":"testasset.Document","testasset.Document":{"name":"x"}}`,
			want: &Document{Name: "x"},
		},
		// Errors: Swift must throw the named MkunionJSON.Error case.
		{name: "empty object", kind: "query", wire: `{}`, throws: "noType"},
		{name: "unknown $type", kind: "query", wire: `{"$type":"testasset.Nope"}`, throws: "unknownType"},
		{name: "empty object answer", kind: "answer", wire: `{}`, throws: "noType"},
	}
}

// sameValue compares what came back with what went in. Go drops empty slices,
// maps and nil pointers on the way out (omitempty) and Swift writes them back
// as [], {} and "", so nil and empty are the same; times are the same instant.
var sameValue = cmp.Options{
	cmpopts.EquateEmpty(),
	cmp.Comparer(func(a, b time.Time) bool { return a.Equal(b) }),
}

// TestSwiftRoundTrip proves the Swift in swift/ and the Go serde agree on the
// wire format. It needs swiftc, so CI on Linux and -short skip it.
func TestSwiftRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles Swift; skipped with -short")
	}
	if _, err := exec.LookPath("swiftc"); err != nil {
		t.Skip("swiftc not found")
	}

	cases := swiftCases()
	var in driverBatch
	for _, c := range cases {
		in.Items = append(in.Items, driverItem{Kind: c.kind, Value: c.input(t)})
	}
	out := runSwiftDriver(t, buildSwiftDriver(t), in)
	require.Len(t, out.Items, len(cases), "the driver must answer every item")

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			item := out.Items[i]
			if c.throws != "" {
				assert.Equal(t, c.throws, item.Error, "swift wrote: %s", item.Value)
				return
			}
			require.Empty(t, item.Error, "swift refused the value")
			got := decodeUnion(t, c.kind, item.Value)
			if diff := cmp.Diff(c.want, got, sameValue); diff != "" {
				t.Errorf("Go -> Swift -> Go changed the value (-want +got):\n%s\nswift wrote: %s", diff, item.Value)
			}
		})
	}
}

// driverBatch is the document the driver reads and writes: one item per case.
type driverBatch struct {
	Items []driverItem `json:"items"`
}

type driverItem struct {
	Kind  string          `json:"kind"`
	Value json.RawMessage `json:"value"`
	Error string          `json:"error,omitempty"`
}

// input is what Swift gets: the hand-written wire, or want as Go encodes it.
func (c swiftCase) input(t *testing.T) json.RawMessage {
	if c.wire != "" {
		return json.RawMessage(c.wire)
	}
	var data []byte
	var err error
	switch c.kind {
	case "query":
		v, _ := c.want.(Query)
		data, err = shared.JSONMarshal[Query](v)
	case "answer":
		v, _ := c.want.(Answer)
		data, err = shared.JSONMarshal[Answer](v)
	case "status":
		v, _ := c.want.(Status)
		data, err = shared.JSONMarshal[Status](v)
	default:
		t.Fatalf("unknown kind %q", c.kind)
	}
	require.NoError(t, err, "encoding %s %#v", c.kind, c.want)
	return data
}

// decodeUnion reads what Swift wrote with the Go serde for the union of kind.
func decodeUnion(t *testing.T, kind string, raw json.RawMessage) any {
	var got any
	var err error
	switch kind {
	case "query":
		got, err = shared.JSONUnmarshal[Query](raw)
	case "answer":
		got, err = shared.JSONUnmarshal[Answer](raw)
	case "status":
		got, err = shared.JSONUnmarshal[Status](raw)
	default:
		t.Fatalf("unknown kind %q", kind)
	}
	require.NoError(t, err, "Go cannot read what Swift wrote: %s", raw)
	return got
}

// buildSwiftDriver copies the golden Swift and the driver into a temporary
// directory, compiles them there and returns the binary's path.
func buildSwiftDriver(t *testing.T) string {
	dir := t.TempDir()
	var sources []string
	for _, name := range []string{"MkunionJSON.swift", "Testasset.swift", "driver/main.swift"} {
		data, err := os.ReadFile(filepath.Join("swift", name))
		require.NoError(t, err)
		dst := filepath.Join(dir, filepath.Base(name))
		require.NoError(t, os.WriteFile(dst, data, 0o644))
		sources = append(sources, dst)
	}
	binary := filepath.Join(dir, "driver")
	args := append([]string{"-swift-version", "6", "-o", binary}, sources...)
	output, err := exec.Command("swiftc", args...).CombinedOutput()
	require.NoError(t, err, "swiftc %v:\n%s", args, output)
	return binary
}

// runSwiftDriver feeds the batch to the driver on stdin and reads its answer.
func runSwiftDriver(t *testing.T, binary string, in driverBatch) driverBatch {
	input, err := json.Marshal(in)
	require.NoError(t, err)
	cmd := exec.Command(binary)
	cmd.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	require.NoError(t, err, "driver failed: %s\ninput: %s", stderr.String(), input)
	var out driverBatch
	require.NoError(t, json.Unmarshal(output, &out), "driver wrote: %s", output)
	return out
}
