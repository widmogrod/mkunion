// Package breaking compares the shapes of an API against a saved snapshot
// and lists the changes that break clients on the wire, the way
// `buf breaking` does for protobuf. It reads the shape model mkunion
// already infers and thinks in terms of what the generated Go serde and
// the Swift and TypeScript exporters do with the JSON.
package breaking

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/widmogrod/mkunion/x/shape"
)

// Snapshot is every named shape of an API: unions with their variants,
// plain structs and aliases, sorted by package and name.
//
//go:tag serde:"-"
type Snapshot struct {
	Shapes []shape.Shape
}

// Take infers the shapes of the given Go files.
func Take(files []string) (*Snapshot, error) {
	var all []shape.Shape
	seen := map[string]bool{}
	for _, file := range files {
		inferred, err := shape.InferFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("breaking: %s: %w", file, err)
		}
		for _, x := range inferred.RetrieveShapes() {
			k := shapeKey(x)
			if seen[k] {
				continue
			}
			seen[k] = true
			all = append(all, x)
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return shapeKey(all[i]) < shapeKey(all[j]) })
	return &Snapshot{Shapes: all}, nil
}

func shapeKey(x shape.Shape) string { return shape.PkgName(x) + "." + shape.Name(x) }

type snapshotJSON struct {
	Shapes []json.RawMessage `json:"shapes"`
}

// MarshalJSON writes {"shapes": [...]} with each shape in mkunion's own
// Shape JSON, keys sorted so two snapshots of the same code are identical.
func (s *Snapshot) MarshalJSON() ([]byte, error) {
	out := snapshotJSON{Shapes: []json.RawMessage{}}
	for _, x := range s.Shapes {
		data, err := shape.ShapeToJSON(x)
		if err != nil {
			return nil, fmt.Errorf("breaking: %s: %w", shapeKey(x), err)
		}
		canonical, err := sortKeys(data)
		if err != nil {
			return nil, fmt.Errorf("breaking: %s: %w", shapeKey(x), err)
		}
		out.Shapes = append(out.Shapes, canonical)
	}
	return json.Marshal(out)
}

// sortKeys re-encodes JSON through encoding/json, which sorts object keys.
func sortKeys(data []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

// UnmarshalJSON reads what MarshalJSON wrote.
func (s *Snapshot) UnmarshalJSON(data []byte) error {
	var in snapshotJSON
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	s.Shapes = nil
	for i, raw := range in.Shapes {
		x, err := shape.ShapeFromJSON(raw)
		if err != nil {
			return fmt.Errorf("breaking: shapes[%d]: %w", i, err)
		}
		s.Shapes = append(s.Shapes, x)
	}
	return nil
}

// WriteFile saves a snapshot, indented so it diffs well in git.
func WriteFile(path string, s *Snapshot) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// ReadFile loads a snapshot written by WriteFile.
func ReadFile(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("breaking: %w", err)
	}
	s := &Snapshot{}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("breaking: %s: %w", path, err)
	}
	return s, nil
}
