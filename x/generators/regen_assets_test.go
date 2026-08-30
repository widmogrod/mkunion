//go:build regen_assets

package generators

import (
	"os"
	"testing"

	"github.com/widmogrod/mkunion/x/shape"
)

func TestRegenAssets(t *testing.T) {
	tree, err := shape.InferFromFile("testutils/tree.go")
	if err != nil {
		t.Fatal(err)
	}
	generic, err := shape.InferFromFile("testutils/generic.go")
	if err != nil {
		t.Fatal(err)
	}

	write := func(name string, data []byte) {
		if err := os.WriteFile(name, data, 0644); err != nil {
			t.Fatal(err)
		}
	}

	r1, err := NewSerdeJSONUnion(tree.RetrieveUnion("Tree")).Generate()
	if err != nil {
		t.Fatal(err)
	}
	write("serde_json_union_test.go.asset", r1)

	r2, err := NewSerdeJSONUnion(tree.RetrieveUnion("Forest")).Generate()
	if err != nil {
		t.Fatal(err)
	}
	write("serde_json_union_alias_test.go.asset", r2)

	r3, err := NewSerdeJSONUnion(generic.RetrieveUnion("Record")).Generate()
	if err != nil {
		t.Fatal(err)
	}
	write("serde_json_union_generic_test.go.asset", r3)
}
