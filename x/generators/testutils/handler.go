package testutils

import (
	"time"

	"github.com/widmogrod/mkunion/f"
)

// User is what GetUser answers with.
type User struct {
	ID   string
	Name string
}

// Query shows the typed handler: a variant embeds f.Returns[R] to declare its
// answer type, and that marker alone makes mkunion generate one handler
// interface per variant, QueryHandler for the whole union, HandleQuery with
// one typed arm per variant, and Perform on every variant that has a marker.
// Touch has no marker: it is handled with an error only, and has no Perform.
//
//go:tag mkunion:"Query"
type (
	GetUser struct {
		f.Returns[*User]
		ID string
	}
	Count    struct{ f.Returns[int] }
	LastSeen struct {
		f.Returns[time.Time]
		ID string
	}
	Touch struct{ ID string }
)
