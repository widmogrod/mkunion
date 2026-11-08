package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

const vehicleSource = `package vehicle

//go:tag mkunion:"Vehicle"
type (
	Car   struct{ Wheels int }
	Plane struct{ Engines int }
)
`

func writeVehicleModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/vehicle\n\ngo 1.21\n"), 0644))
	sourcePath := filepath.Join(dir, "vehicle.go")
	require.NoError(t, os.WriteFile(sourcePath, []byte(vehicleSource), 0644))
	return sourcePath
}

func runCLI(t *testing.T, args ...string) error {
	t.Helper()
	app := newApp(context.Background())
	return app.RunContext(context.Background(), append([]string{"mkunion"}, args...))
}

func generatedFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_gen.go") {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestWatchGenerateOnly(t *testing.T) {
	sourcePath := writeVehicleModule(t)
	dir := filepath.Dir(sourcePath)

	err := runCLI(t, "watch", "-g", "-G", dir)
	require.NoError(t, err)

	names := generatedFiles(t, dir)
	assert.Contains(t, names, "vehicle_union_gen.go")
	assert.Contains(t, names, "vehicle_shape_gen.go")
	assert.Contains(t, names, "types_reg_gen.go")
}

func TestWatchGenerateOnlyBadPath(t *testing.T) {
	err := runCLI(t, "watch", "-g", "-G", filepath.Join(t.TempDir(), "missing", "..."))
	assert.Error(t, err)
}

func TestCleanRemovesGeneratedFiles(t *testing.T) {
	sourcePath := writeVehicleModule(t)
	dir := filepath.Dir(sourcePath)
	require.NoError(t, runCLI(t, "watch", "-g", "-G", dir))
	require.NotEmpty(t, generatedFiles(t, dir))

	t.Run("dry run previews without removing", func(t *testing.T) {
		require.NoError(t, runCLI(t, "clean", "-n", dir))
		assert.NotEmpty(t, generatedFiles(t, dir), "dry run must not delete")
	})

	t.Run("clean removes generated files but keeps sources", func(t *testing.T) {
		require.NoError(t, runCLI(t, "clean", dir))
		assert.Empty(t, generatedFiles(t, dir))
		_, err := os.Stat(sourcePath)
		assert.NoError(t, err, "hand-written sources must survive")
	})

	t.Run("cleaning an already clean directory reports nothing to do", func(t *testing.T) {
		require.NoError(t, runCLI(t, "clean", dir))
	})
}

func TestGenerateForFile(t *testing.T) {
	sourcePath := writeVehicleModule(t)
	dir := filepath.Dir(sourcePath)

	require.NoError(t, runCLI(t, "-i", sourcePath))
	assert.Contains(t, generatedFiles(t, dir), "vehicle_union_gen.go")
}

func TestShapeExport(t *testing.T) {
	sourcePath := writeVehicleModule(t)
	outDir := filepath.Join(t.TempDir(), "ts")

	require.NoError(t, runCLI(t, "shape-export", "-i", sourcePath, "-o", outDir))

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	assert.NotEmpty(t, entries, "typescript files must be exported")
}

func TestShapeExportSwift(t *testing.T) {
	sourcePath := writeVehicleModule(t)
	outDir := filepath.Join(t.TempDir(), "swift")

	// the file comes as a positional argument, not only through -i
	require.NoError(t, runCLI(t, "shape-export", "--language", "swift", "-o", outDir, sourcePath))

	content, err := os.ReadFile(filepath.Join(outDir, "Vehicle.swift"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "public enum Vehicle {")
	assert.Contains(t, string(content), "public enum Vehicle: Codable, Hashable, Sendable {")
	assert.Contains(t, string(content), "case car(Car)")
	assert.FileExists(t, filepath.Join(outDir, "MkunionJSON.swift"))
}

func TestShapeExportUnknownLanguage(t *testing.T) {
	sourcePath := writeVehicleModule(t)

	err := runCLI(t, "shape-export", "--language", "rust", "-o", t.TempDir(), sourcePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown language "rust"`)
}

const apiV1Source = `package api

import "github.com/widmogrod/mkunion/f"

//go:tag mkunion:"Request,answers=Response"
type (
	Search struct {
		f.Returns[*Results]
		Query string ` + "`json:\"query\"`" + `
	}
)

//go:tag mkunion:"Response"
type (
	Results struct {
		Names []string ` + "`json:\"names\"`" + `
	}
)
`

// runCLICapture runs the app without letting cli.Exit end the test process.
func runCLICapture(t *testing.T, args ...string) (string, error) {
	t.Helper()
	app := newApp(context.Background())
	var out strings.Builder
	app.Writer = &out
	app.ExitErrHandler = func(*cli.Context, error) {}
	err := app.RunContext(context.Background(), append([]string{"mkunion"}, args...))
	return out.String(), err
}

func TestShapeSnapshotAndBreaking(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "api.go")
	require.NoError(t, os.WriteFile(source, []byte(apiV1Source), 0644))
	snapshot := filepath.Join(dir, "shapes.json")

	out, err := runCLICapture(t, "shape-snapshot", "-o", snapshot, source)
	require.NoError(t, err)
	assert.Equal(t, snapshot+"\n", out)
	assert.FileExists(t, snapshot)

	t.Run("unchanged API passes", func(t *testing.T) {
		out, err := runCLICapture(t, "breaking", "--against", snapshot, source)
		require.NoError(t, err)
		assert.Empty(t, out)
	})

	t.Run("renamed json field fails with exit code 1", func(t *testing.T) {
		changed := strings.Replace(apiV1Source, `json:"query"`, `json:"q"`, 1)
		require.NoError(t, os.WriteFile(source, []byte(changed), 0644))

		out, err := runCLICapture(t, "breaking", "--against", snapshot, source)
		var exit cli.ExitCoder
		require.ErrorAs(t, err, &exit)
		assert.Equal(t, 1, exit.ExitCode())
		assert.Contains(t, out, source+`:9: breaking: Search.Query: json name changed "query" → "q"`)
	})

	t.Run("missing snapshot is an error", func(t *testing.T) {
		_, err := runCLICapture(t, "breaking", "--against", filepath.Join(dir, "nope.json"), source)
		assert.Error(t, err)
	})
}
