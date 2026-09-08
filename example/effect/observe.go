package effect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"
)

// Observability over typed operations. A trace of union values can be diffed,
// checked against a policy, and turned into spans, all in one place.

// --8<-- [start:diff]

// DiffTraces compares two traces and reports the difference, one line per
// operation: "  " unchanged, "+ " only in got, "- " only in want.
// It is a plain longest-common-subsequence diff over DeepEqual.
func DiffTraces[Op any](want, got []Op) []string {
	n, m := len(want), len(got)
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if reflect.DeepEqual(want[i], got[j]) {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var lines []string
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && reflect.DeepEqual(want[i], got[j]):
			lines = append(lines, "  "+describe(want[i]))
			i, j = i+1, j+1
		case j < m && (i == n || lcs[i][j+1] >= lcs[i+1][j]):
			lines = append(lines, "+ "+describe(got[j]))
			j++
		default:
			lines = append(lines, "- "+describe(want[i]))
			i++
		}
	}
	return lines
}

// --8<-- [end:diff]

// --8<-- [start:guard]

// ErrDenied marks an operation refused by Guard.
var ErrDenied = errors.New("effect: denied by policy")

// Guard refuses operations before they are performed. allow is usually an
// exhaustive match over the union: authorization or dry-run as one function.
func Guard[Op any](allow func(Op) error) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		return func(ctx context.Context, op Op) (any, error) {
			if err := allow(op); err != nil {
				return nil, fmt.Errorf("%w: %w", ErrDenied, err)
			}
			return h(ctx, op)
		}
	}
}

// --8<-- [end:guard]

// --8<-- [start:spans]

// Span is what a tracing backend such as OpenTelemetry would receive.
type Span struct {
	Name  string
	Attrs string
	Start time.Time
	End   time.Time
	Err   string
}

// Spans emits one span per operation, for every operation, from one place.
func Spans[Op any](now func() time.Time, sink *[]Span) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		return func(ctx context.Context, op Op) (any, error) {
			span := Span{Name: fmt.Sprintf("%T", op), Attrs: attrs(op), Start: now()}
			answer, err := h(ctx, op)
			span.End = now()
			if err != nil {
				span.Err = err.Error()
			}
			*sink = append(*sink, span)
			return answer, err
		}
	}
}

// --8<-- [end:spans]

// attrs renders an operation's fields as JSON, through the serde mkunion
// generated for it. JSON shows the data and skips the f.Returns phantom,
// which fmt's %+v would print as `Returns:{}`.
func attrs(op any) string {
	if m, ok := op.(json.Marshaler); ok {
		if data, err := m.MarshalJSON(); err == nil {
			return string(data)
		}
	}
	return fmt.Sprintf("%+v", op)
}

// describe is attrs with the operation's type in front: *effect.Log{"Msg":"hi"}.
func describe(op any) string {
	return fmt.Sprintf("%T%s", op, attrs(op))
}
