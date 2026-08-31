package spec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderToString(section sectionSpec) string {
	var b strings.Builder
	renderSection(&b, section)
	return b.String()
}

func twoBackendSection() sectionSpec {
	return sectionSpec{
		title:           "Repositories",
		behavioursTitle: "Repository behaviours",
		prose:           "What each backend supports.",
		capabilities:    []string{"AtomicBatch", "NotDescribedAnywhere"},
		displayName:     func(backend string) string { return strings.ToUpper(backend) },
		reports: []*BackendReport{
			{
				Backend:      "memory",
				Capabilities: map[string]bool{"AtomicBatch": true},
				Behaviours: []BehaviourReport{
					{Suite: "crud", Order: 1, Name: "saves records", Status: statusSupported},
					{Suite: "crud", Order: 2, Name: "atomic batches", Status: statusFailing},
				},
			},
			{
				Backend:      "dynamo",
				Capabilities: map[string]bool{"AtomicBatch": false},
				Behaviours: []BehaviourReport{
					{Suite: "crud", Order: 1, Name: "saves records", Status: statusSupported},
					{Suite: "crud", Order: 2, Name: "atomic batches", Status: statusDowngraded, Capability: "AtomicBatch"},
					// this behaviour exists only on dynamo; memory shows "—"
					{Suite: "crud", Order: 3, Name: "dynamo only", Status: statusSupported},
				},
			},
		},
	}
}

func TestRenderSection(t *testing.T) {
	out := renderToString(twoBackendSection())

	t.Run("titles and prose are rendered", func(t *testing.T) {
		assert.Contains(t, out, "#### Repositories")
		assert.Contains(t, out, "#### Repository behaviours")
		assert.Contains(t, out, "What each backend supports.")
	})

	t.Run("backend names go through displayName", func(t *testing.T) {
		assert.Contains(t, out, "MEMORY")
		assert.Contains(t, out, "DYNAMO")
	})

	t.Run("capability matrix marks support", func(t *testing.T) {
		require.Contains(t, out, "**AtomicBatch**")
		row := lineContaining(t, out, "**AtomicBatch**")
		assert.Contains(t, row, "✅")
		assert.Contains(t, row, "⛔")
		// the described capability uses its description text
		assert.Contains(t, row, "All-or-nothing")
	})

	t.Run("capability without a description falls back to its name", func(t *testing.T) {
		row := lineContaining(t, out, "**NotDescribedAnywhere**")
		assert.Contains(t, row, "— NotDescribedAnywhere |")
	})

	t.Run("behaviour rows carry per-backend status", func(t *testing.T) {
		row := lineContaining(t, out, "atomic batches")
		assert.Contains(t, row, "❌", "memory fails this behaviour")
		assert.Contains(t, row, "⛔ `AtomicBatch`", "dynamo downgraded it")
	})

	t.Run("behaviour missing on a backend renders an em dash", func(t *testing.T) {
		row := lineContaining(t, out, "dynamo only")
		assert.Contains(t, row, "—")
		assert.Contains(t, row, "✅")
	})

	t.Run("duplicate behaviours across backends render once", func(t *testing.T) {
		assert.Equal(t, 1, strings.Count(out, "saves records |"),
			"row must appear once even though both backends report it")
	})
}

func TestRenderSectionNoReports(t *testing.T) {
	out := renderToString(sectionSpec{
		title:           "Empty",
		behavioursTitle: "Empty behaviours",
		prose:           "Nothing ran.",
		capabilities:    []string{"AtomicBatch"},
		displayName:     func(backend string) string { return backend },
	})

	assert.Contains(t, out, "#### Empty")
	assert.Contains(t, out, "| Capability |")
	// with no reports the capability rows have no status cells
	row := lineContaining(t, out, "**AtomicBatch**")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(row), "|"))
}

func lineContaining(t *testing.T, text, needle string) string {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	t.Fatalf("no line contains %q", needle)
	return ""
}

func TestStampBetweenMarkers(t *testing.T) {
	t.Run("replaces the section between markers", func(t *testing.T) {
		path := writeTempFile(t, "# Readme\n\n"+markerBegin+"\nold\n"+markerEnd+"\ntail\n")
		require.NoError(t, stampBetweenMarkers(path, "new content\n"))

		got := readFile(t, path)
		assert.Contains(t, got, "new content")
		assert.NotContains(t, got, "old")
		assert.Contains(t, got, "tail", "content after the marker must survive")
		assert.Contains(t, got, "# Readme", "content before the marker must survive")
	})

	t.Run("appends a fresh section when markers are absent", func(t *testing.T) {
		path := writeTempFile(t, "# Readme without markers")
		require.NoError(t, stampBetweenMarkers(path, "fresh\n"))

		got := readFile(t, path)
		assert.Contains(t, got, "## Storage backend capabilities")
		assert.Contains(t, got, markerBegin)
		assert.Contains(t, got, "fresh")
		assert.Contains(t, got, markerEnd)
	})

	t.Run("stamping twice is idempotent", func(t *testing.T) {
		path := writeTempFile(t, "# Readme")
		require.NoError(t, stampBetweenMarkers(path, "v1\n"))
		require.NoError(t, stampBetweenMarkers(path, "v2\n"))

		got := readFile(t, path)
		assert.Contains(t, got, "v2")
		assert.NotContains(t, got, "v1")
		assert.Equal(t, 1, strings.Count(got, markerBegin))
	})

	t.Run("missing file errors", func(t *testing.T) {
		assert.Error(t, stampBetweenMarkers("/nonexistent/readme.md", "x"))
	})
}

// The docstring promises: with no new reports GenerateDocs reproduces the
// committed files bit for bit. Hold it to that.
func TestGenerateDocsIsByteStable(t *testing.T) {
	capabilitiesPath := filepath.Join(reportDir(), "capabilities.md")
	readmePath := filepath.Join(specDir(), "..", "..", "README.md")

	beforeCapabilities := readFile(t, capabilitiesPath)
	beforeReadme := readFile(t, readmePath)

	require.NoError(t, GenerateDocs())

	assert.Equal(t, beforeCapabilities, readFile(t, capabilitiesPath))
	assert.Equal(t, beforeReadme, readFile(t, readmePath))
}

func TestWriteBackendReportMergesFilteredRuns(t *testing.T) {
	const backend = "unittest-fake-backend"
	path := reportPath(backend)
	t.Cleanup(func() { _ = os.Remove(path) })

	setCollected := func(report *BackendReport) {
		collector.Lock()
		collector.reports[backend] = report
		collector.Unlock()
	}
	t.Cleanup(func() {
		collector.Lock()
		delete(collector.reports, backend)
		collector.Unlock()
	})

	// full run: two suites
	setCollected(&BackendReport{
		Backend:      backend,
		Capabilities: map[string]bool{"AtomicBatch": true},
		Behaviours: []BehaviourReport{
			{Suite: "crud", Order: 1, Name: "saves", Status: statusSupported},
			{Suite: "stream", Order: 1, Name: "replays", Status: statusSupported},
		},
	})
	require.NoError(t, writeBackendReport(backend))

	// filtered re-run: only the crud suite ran, with a new outcome
	setCollected(&BackendReport{
		Backend:      backend,
		Capabilities: map[string]bool{"AtomicBatch": true},
		Behaviours: []BehaviourReport{
			{Suite: "crud", Order: 1, Name: "saves", Status: statusFailing},
		},
	})
	require.NoError(t, writeBackendReport(backend))

	merged, err := readBackendReport(path)
	require.NoError(t, err)

	byName := map[string]string{}
	for _, entry := range merged.Behaviours {
		byName[entry.Suite+"/"+entry.Name] = entry.Status
	}
	assert.Equal(t, statusFailing, byName["crud/saves"], "re-run suite takes the fresh outcome")
	assert.Equal(t, statusSupported, byName["stream/replays"], "suites that did not run keep their rows")
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "readme.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
