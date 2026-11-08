package breaking

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readAll(t *testing.T, path string) (string, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data), err
}

func TestSnapshotUnmarshalRejectsGarbage(t *testing.T) {
	s := &Snapshot{}
	assert.Error(t, s.UnmarshalJSON([]byte(`{"shapes": [{"$type": "shape.Nope", "shape.Nope": {}}]}`)))
	assert.Error(t, s.UnmarshalJSON([]byte(`not json`)))

	empty := &Snapshot{}
	data, err := empty.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, `{"shapes":[]}`, string(data))
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestWriteFileFailsOnMissingDir(t *testing.T) {
	assert.Error(t, WriteFile(filepath.Join(t.TempDir(), "no", "dir", "shapes.json"), &Snapshot{}))
}
