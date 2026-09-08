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

// Query shows the `handler` union option: every variant embeds f.Returns[R]
// to declare its answer type, and mkunion generates QueryHandler, QueryOf[R],
// QueryHandlerFunc and QueryDefaults from it.
//
//go:tag mkunion:"Query,handler"
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
)
