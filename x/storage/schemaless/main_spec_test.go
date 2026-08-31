package schemaless_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/widmogrod/mkunion/x/storage/schemaless/spec"
)

// TestMain regenerates the storage capability documentation after a green
// run: spec/report/*.json (written by the spec suites while they run) are
// rendered into spec/report/capabilities.md and stamped into
// x/storage/README.md. With no new reports this reproduces the committed
// files bit for bit.
func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 {
		if err := spec.GenerateDocs(); err != nil {
			fmt.Fprintf(os.Stderr, "spec: could not generate capability docs: %s\n", err)
			code = 1
		}
	}
	os.Exit(code)
}
