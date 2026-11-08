package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initAPI(t *testing.T, dir string, extra ...string) error {
	t.Helper()
	args := append([]string{"init", "api", "--module", "example.com/todo", "--resource", "task", "--dir", dir, "--mkunion-replace", "/tmp/mkunion"}, extra...)
	return runCLI(t, args...)
}

func TestInitAPIWritesTheModule(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "todo")
	require.NoError(t, initAPI(t, dir))

	for _, f := range []string{
		"go.mod", "AGENTS.md", "Makefile",
		"api/server.go", "api/server_test.go", "cmd/server/main.go",
		"api/resource/task/query.go", "api/resource/task/command.go", "api/resource/task/answer.go",
		"api/resource/task/handler.go", "api/resource/task/store.go", "api/resource/task/endpoint.go", "api/resource/task/handler_test.go",
	} {
		assert.FileExists(t, filepath.Join(dir, f))
	}

	goMod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(goMod), "module example.com/todo\n")
	assert.Contains(t, string(goMod), "replace github.com/widmogrod/mkunion => /tmp/mkunion\n")
	assert.Contains(t, string(goMod), "tool github.com/widmogrod/mkunion/cmd/mkunion\n")

	query, err := os.ReadFile(filepath.Join(dir, "api", "resource", "task", "query.go"))
	require.NoError(t, err)
	assert.Contains(t, string(query), `//go:tag mkunion:"TaskQuery,answers=TaskAnswer"`)
	assert.Contains(t, string(query), "f.Returns[*Task]")

	server, err := os.ReadFile(filepath.Join(dir, "api", "server.go"))
	require.NoError(t, err)
	assert.Contains(t, string(server), `"example.com/todo/api/resource/task"`)
	assert.Contains(t, string(server), `mount(s, "task/query", task.QueryEndpoint(deps.Task))`)

	agents, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	require.NoError(t, err)
	assert.Contains(t, string(agents), "go generate ./... && go test ./... -race")
	assert.Contains(t, string(agents), "missing method HandleX")
}

func TestInitAPIRefusesANonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "keep.txt"), []byte("x"), 0o644))
	err := initAPI(t, dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not empty")
}

func TestInitAPIRefusesABadResourceName(t *testing.T) {
	err := runCLI(t, "init", "api", "--module", "example.com/todo", "--resource", "Task", "--dir", filepath.Join(t.TempDir(), "x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `resource name "Task"`)
}

func TestInitResourceAddsAPackage(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "todo")
	require.NoError(t, initAPI(t, dir))

	require.NoError(t, runCLI(t, "init", "resource", "--name", "project", "--dir", dir))
	handler, err := os.ReadFile(filepath.Join(dir, "api", "resource", "project", "handler.go"))
	require.NoError(t, err)
	assert.Contains(t, string(handler), "package project")
	assert.Contains(t, string(handler), "func (h *Handler) HandleGetProject(ctx context.Context, op *GetProject) (*Project, error)")

	t.Run("refuses to overwrite", func(t *testing.T) {
		err := runCLI(t, "init", "resource", "--name", "project", "--dir", dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exists")
	})

	t.Run("needs a go.mod", func(t *testing.T) {
		err := runCLI(t, "init", "resource", "--name", "note", "--dir", t.TempDir())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "go.mod")
	})
}

// TestInitAPIEndToEnd runs the generated module's own loop. It needs the Go
// proxy and this checkout, so it runs only with MKUNION_INIT_E2E=1.
func TestInitAPIEndToEnd(t *testing.T) {
	if os.Getenv("MKUNION_INIT_E2E") == "" {
		t.Skip("set MKUNION_INIT_E2E=1 to run go mod tidy, go generate and go test in a generated module")
	}
	checkout, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err)
	dir := filepath.Join(t.TempDir(), "todo")
	require.NoError(t, runCLI(t, "init", "api", "--module", "example.com/todo", "--resource", "task", "--dir", dir, "--mkunion-replace", checkout))

	for _, args := range [][]string{
		{"mod", "tidy"},
		{"generate", "./..."},
		{"vet", "./..."},
		{"test", "./...", "-race"},
	} {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "go %s:\n%s", strings.Join(args, " "), out)
	}
}
