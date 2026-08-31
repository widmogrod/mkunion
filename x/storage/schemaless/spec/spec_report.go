package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
)

// Canonical backend names: they name the report files and the columns of the
// generated capability matrix, in this order.
const (
	BackendInMemory   = "in-memory"
	BackendDynamoDB   = "dynamodb"
	BackendOpenSearch = "opensearch"
)

var backendOrder = []string{BackendInMemory, BackendDynamoDB, BackendOpenSearch}

var backendDisplayName = map[string]string{
	BackendInMemory:   "In-Memory",
	BackendDynamoDB:   "DynamoDB",
	BackendOpenSearch: "OpenSearch",
}

// Canonical append-log implementation names; their reports live in
// report/appendlog-<name>.json so they never collide with repository backends.
const (
	AppendLogInMemory = "in-memory"
	AppendLogTypedful = "typedful"
)

var appendLogOrder = []string{AppendLogInMemory, AppendLogTypedful}

var appendLogDisplayName = map[string]string{
	AppendLogInMemory: "In-Memory",
	AppendLogTypedful: "Typedful",
}

const appendLogReportPrefix = "appendlog-"

const (
	suiteRepository = "repository"
	suiteComplex    = "complex queries"
	suiteAppendLog  = "append log"
)

var suiteOrder = map[string]int{
	suiteRepository: 1,
	suiteComplex:    2,
	suiteAppendLog:  3,
}

const (
	statusSupported  = "supported"
	statusDowngraded = "downgraded"
	statusFailing    = "failing"
)

var capabilityDescription = map[string]string{
	"SortByDataField":           "Sort results by any record field (e.g. `Data.Name`)",
	"BackwardPagination":        "Page backward with `Before` cursors and `Prev` links",
	"AtomicBatch":               "All-or-nothing `UpdateRecords` batches",
	"MonotonicOverwriteVersion": "Versions keep increasing under `PolicyOverwriteServerChanges`",

	"Filtering":    "`Subscribe` honours a where-predicate filter",
	"OffsetResume": "`Subscribe` resumes from a given change offset",
	"Replay":       "A late subscriber receives every past change",
	"MergeAppend":  "`Append` merges another log's changes",
}

// BehaviourReport is one spec subtest's outcome on one backend.
type BehaviourReport struct {
	Suite      string
	Order      int
	Name       string
	Status     string
	Capability string `json:",omitempty"`
}

// BackendReport is what a test run of one backend writes to
// report/<backend>.json. Capabilities is a name→enabled map so repository
// backends and append-log implementations can share the report machinery.
type BackendReport struct {
	Backend      string
	Capabilities map[string]bool
	Behaviours   []BehaviourReport
}

var collector = struct {
	sync.Mutex
	reports map[string]*BackendReport
}{reports: map[string]*BackendReport{}}

// runner routes the suite's subtests so every outcome is recorded for the
// capability report.
type runner struct {
	t       *testing.T
	backend string
	suite   string
	order   int
}

// capsToMap flattens a capabilities struct (bool fields only) into the
// name→enabled map stored in reports.
func capsToMap(caps any) map[string]bool {
	result := map[string]bool{}
	v := reflect.ValueOf(caps)
	for i := 0; i < v.NumField(); i++ {
		result[v.Type().Field(i).Name] = v.Field(i).Bool()
	}
	return result
}

func newRunner(t *testing.T, backend, suite string, caps any) *runner {
	collector.Lock()
	report, ok := collector.reports[backend]
	if !ok {
		report = &BackendReport{Backend: backend}
		collector.reports[backend] = report
	}
	report.Capabilities = capsToMap(caps)
	collector.Unlock()

	t.Cleanup(func() {
		if err := writeBackendReport(backend); err != nil {
			t.Logf("spec: could not write capability report: %s", err)
		}
	})

	return &runner{t: t, backend: backend, suite: suite}
}

func (r *runner) record(entry BehaviourReport) {
	collector.Lock()
	defer collector.Unlock()
	report := collector.reports[r.backend]
	for i, existing := range report.Behaviours {
		if existing.Suite == entry.Suite && existing.Name == entry.Name {
			report.Behaviours[i] = entry
			return
		}
	}
	report.Behaviours = append(report.Behaviours, entry)
}

// run registers a behaviour every backend must support.
func (r *runner) run(name string, fn func(t *testing.T)) {
	r.order++
	entry := BehaviourReport{Suite: r.suite, Order: r.order, Name: name}
	r.t.Run(name, func(t *testing.T) {
		t.Cleanup(func() {
			entry.Status = statusSupported
			if t.Failed() {
				entry.Status = statusFailing
			}
			r.record(entry)
		})
		fn(t)
	})
}

// runGated registers a behaviour behind a capability: when the backend
// declared the downgrade, the subtest is skipped and reported as such.
func (r *runner) runGated(enabled bool, capability, name string, fn func(t *testing.T)) {
	r.order++
	entry := BehaviourReport{Suite: r.suite, Order: r.order, Name: name, Capability: capability}
	r.t.Run(name, func(t *testing.T) {
		t.Cleanup(func() {
			if t.Failed() {
				entry.Status = statusFailing
			}
			r.record(entry)
		})
		if !enabled {
			entry.Status = statusDowngraded
			t.Skipf("capability downgrade: this storage does not support %s (%s)", name, capability)
		}
		entry.Status = statusSupported
		fn(t)
	})
}

func specDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("spec: cannot locate the spec package directory")
	}
	return filepath.Dir(file)
}

func reportDir() string {
	return filepath.Join(specDir(), "report")
}

func reportPath(backend string) string {
	return filepath.Join(reportDir(), backend+".json")
}

// writeBackendReport persists the collected outcomes of one backend, merged
// with the suites already on disk (a filtered `go test -run` must not erase
// the rows of suites it did not run).
func writeBackendReport(backend string) error {
	collector.Lock()
	collected := collector.reports[backend]
	fresh := &BackendReport{
		Backend:      collected.Backend,
		Capabilities: collected.Capabilities,
		Behaviours:   append([]BehaviourReport{}, collected.Behaviours...),
	}
	collector.Unlock()

	suitesCollected := map[string]bool{}
	for _, entry := range fresh.Behaviours {
		suitesCollected[entry.Suite] = true
	}

	if previous, err := readBackendReport(reportPath(backend)); err == nil {
		for _, entry := range previous.Behaviours {
			if !suitesCollected[entry.Suite] {
				fresh.Behaviours = append(fresh.Behaviours, entry)
			}
		}
	}

	sortBehaviours(fresh.Behaviours)

	data, err := json.MarshalIndent(fresh, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if err := os.MkdirAll(reportDir(), 0o755); err != nil {
		return err
	}
	return os.WriteFile(reportPath(backend), data, 0o644)
}

func readBackendReport(path string) (*BackendReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	report := &BackendReport{}
	if err := json.Unmarshal(data, report); err != nil {
		return nil, err
	}
	return report, nil
}

func sortBehaviours(behaviours []BehaviourReport) {
	sort.SliceStable(behaviours, func(i, j int) bool {
		if suiteOrder[behaviours[i].Suite] != suiteOrder[behaviours[j].Suite] {
			return suiteOrder[behaviours[i].Suite] < suiteOrder[behaviours[j].Suite]
		}
		return behaviours[i].Order < behaviours[j].Order
	})
}

// capabilityNames lists a capabilities struct's fields in declaration order,
// so the generated matrix follows the struct.
func capabilityNames(capsType any) []string {
	var names []string
	tp := reflect.TypeOf(capsType)
	for i := 0; i < tp.NumField(); i++ {
		names = append(names, tp.Field(i).Name)
	}
	return names
}

// GenerateDocs renders report/*.json into report/capabilities.md and stamps
// the same content into x/storage/README.md between the marker comments.
// Call it after a successful test run (see TestMain in the schemaless
// package); with no new reports it reproduces the committed files bit for
// bit, so it is safe to run anywhere.
func GenerateDocs() error {
	readReports := func(keys []string, prefix string) ([]*BackendReport, error) {
		var reports []*BackendReport
		for _, backend := range keys {
			report, err := readBackendReport(reportPath(prefix + backend))
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			reports = append(reports, report)
		}
		return reports, nil
	}

	repoReports, err := readReports(backendOrder, "")
	if err != nil {
		return err
	}
	appendLogReports, err := readReports(appendLogOrder, appendLogReportPrefix)
	if err != nil {
		return err
	}
	if len(repoReports) == 0 && len(appendLogReports) == 0 {
		return nil
	}

	b := &strings.Builder{}
	fmt.Fprintf(b, "<!-- Code generated by the x/storage/schemaless/spec test suite. DO NOT EDIT. -->\n")

	if len(repoReports) > 0 {
		renderSection(b, sectionSpec{
			title:           "Repository capability matrix",
			behavioursTitle: "Repository verified behaviours",
			prose: "Every `Repository` backend passes the same behavioural spec\n" +
				"(`spec.RunRepositorySpec`), modulo the capabilities it explicitly\n" +
				"downgrades. The in-memory repository defines the full contract.",
			capabilities: capabilityNames(Capabilities{}),
			displayName: func(backend string) string {
				return backendDisplayName[backend]
			},
			reports: repoReports,
		})
	}

	if len(appendLogReports) > 0 {
		renderSection(b, sectionSpec{
			title:           "Append log capability matrix",
			behavioursTitle: "Append log verified behaviours",
			prose: "Every `AppendLoger` implementation passes the same behavioural spec\n" +
				"(`spec.RunAppendLogSpec`), modulo the capabilities it explicitly downgrades.\n" +
				"The in-memory append log defines the full contract.",
			capabilities: capabilityNames(AppendLogCapabilities{}),
			displayName: func(backend string) string {
				return appendLogDisplayName[strings.TrimPrefix(backend, appendLogReportPrefix)]
			},
			reports: appendLogReports,
		})
	}

	content := b.String()

	markdownPath := filepath.Join(reportDir(), "capabilities.md")
	if err := os.WriteFile(markdownPath, []byte(content), 0o644); err != nil {
		return err
	}

	readmePath := filepath.Join(specDir(), "..", "..", "README.md")
	return stampBetweenMarkers(readmePath, content)
}

type sectionSpec struct {
	title           string
	behavioursTitle string
	prose           string
	capabilities    []string
	displayName     func(backend string) string
	reports         []*BackendReport
}

func renderSection(b *strings.Builder, section sectionSpec) {
	reports := section.reports

	fmt.Fprintf(b, "\n#### %s\n\n", section.title)
	fmt.Fprintf(b, "%s\n\n", section.prose)

	fmt.Fprintf(b, "| Capability |")
	for _, report := range reports {
		fmt.Fprintf(b, " %s |", section.displayName(report.Backend))
	}
	fmt.Fprintf(b, "\n|---|")
	for range reports {
		fmt.Fprintf(b, ":---:|")
	}
	fmt.Fprintf(b, "\n")
	for _, name := range section.capabilities {
		description := capabilityDescription[name]
		if description == "" {
			description = name
		}
		fmt.Fprintf(b, "| **%s** — %s |", name, description)
		for _, report := range reports {
			if report.Capabilities[name] {
				fmt.Fprintf(b, " ✅ |")
			} else {
				fmt.Fprintf(b, " ⛔ |")
			}
		}
		fmt.Fprintf(b, "\n")
	}

	fmt.Fprintf(b, "\n#### %s\n\n", section.behavioursTitle)
	fmt.Fprintf(b, "Each row is a spec subtest; ✅ verified, ⛔ skipped by a declared capability\n")
	fmt.Fprintf(b, "downgrade, ❌ failing when the report was generated.\n\n")

	fmt.Fprintf(b, "| Suite | Behaviour |")
	for _, report := range reports {
		fmt.Fprintf(b, " %s |", section.displayName(report.Backend))
	}
	fmt.Fprintf(b, "\n|---|---|")
	for range reports {
		fmt.Fprintf(b, ":---:|")
	}
	fmt.Fprintf(b, "\n")

	// rows follow the first backend that has each behaviour; all backends run
	// the same suites, so this is the registration order
	type key struct{ suite, name string }
	seen := map[key]bool{}
	var rows []BehaviourReport
	for _, report := range reports {
		for _, entry := range report.Behaviours {
			k := key{entry.Suite, entry.Name}
			if !seen[k] {
				seen[k] = true
				rows = append(rows, entry)
			}
		}
	}
	sortBehaviours(rows)

	for _, row := range rows {
		fmt.Fprintf(b, "| %s | %s |", row.Suite, row.Name)
		for _, report := range reports {
			cell := "—"
			for _, entry := range report.Behaviours {
				if entry.Suite == row.Suite && entry.Name == row.Name {
					switch entry.Status {
					case statusSupported:
						cell = "✅"
					case statusDowngraded:
						cell = fmt.Sprintf("⛔ `%s`", entry.Capability)
					case statusFailing:
						cell = "❌"
					}
					break
				}
			}
			fmt.Fprintf(b, " %s |", cell)
		}
		fmt.Fprintf(b, "\n")
	}
}

const (
	markerBegin = "<!-- BEGIN mkunion:storage-capabilities -->"
	markerEnd   = "<!-- END mkunion:storage-capabilities -->"
)

func stampBetweenMarkers(path, content string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(data)

	section := markerBegin + "\n" + content + markerEnd

	begin := strings.Index(text, markerBegin)
	end := strings.Index(text, markerEnd)
	if begin >= 0 && end > begin {
		text = text[:begin] + section + text[end+len(markerEnd):]
	} else {
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		text += "\n## Storage backend capabilities\n\n" + section + "\n"
	}

	return os.WriteFile(path, []byte(text), 0o644)
}
