package breaking

import "fmt"

// Severity says what a change does to clients that were built against the
// snapshot: nothing, something to look at, or a decode failure.
type Severity int

const (
	Compatible Severity = iota
	Warn
	Breaking
)

func (s Severity) String() string {
	switch s {
	case Breaking:
		return "breaking"
	case Warn:
		return "warn"
	default:
		return "compatible"
	}
}

// Finding is one change, anchored to the current source when the type or
// field still exists there; a removed thing points at its container.
type Finding struct {
	Severity Severity
	File     string
	Line     int
	Subject  string // the type, union or field, as "Search.Query"
	Message  string
}

// String prints a finding as `api.go:45: breaking: Request: variant ListJobs removed (...)`.
func (f Finding) String() string {
	pos := f.File
	if f.Line > 0 {
		pos = fmt.Sprintf("%s:%d", f.File, f.Line)
	}
	if pos != "" {
		pos += ": "
	}
	return fmt.Sprintf("%s%s: %s: %s", pos, f.Severity, f.Subject, f.Message)
}

// CountBreaking is how many findings fail the check.
func CountBreaking(findings []Finding) int {
	n := 0
	for _, f := range findings {
		if f.Severity == Breaking {
			n++
		}
	}
	return n
}
