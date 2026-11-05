package testutils

import "github.com/widmogrod/mkunion/f"

// WithReturns embeds the phantom f.Returns marker. Serde must not emit it,
// and unmarshalling must not look for it.
//
//go:tag serde:"json"
type WithReturns struct {
	f.Returns[int]
	Name string
}
