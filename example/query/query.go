// Package query shows the `handler` union option on its own, with no effect
// system around it: a union of questions, each saying what it answers with,
// and a typed handler generated from that.
package query

import (
	"context"
	"strings"

	"github.com/widmogrod/mkunion/f"
)

// --8<-- [start:model]

// User is what the directory knows about a person.
type User struct{ ID, Name string }

// Query is every question the user directory can answer. Each variant embeds
// f.Returns[R] to say what it answers with. The `handler` option turns that
// into QueryHandler, one typed method per question.
//
//go:tag mkunion:"Query,handler"
type (
	// GetUser answers with the user, or nil when there is none.
	GetUser struct {
		f.Returns[*User]
		ID string
	}
	// FindUsers answers with every user whose name starts with Prefix.
	FindUsers struct {
		f.Returns[[]User]
		Prefix string
	}
	// CountUsers answers with how many users there are.
	CountUsers struct{ f.Returns[int] }
)

// --8<-- [end:model]

// --8<-- [start:ask]

// Ask puts one query to a handler and returns the answer with the type the
// query declared. R is inferred from the query, so a caller never spells it:
// Ask(ctx, h, &CountUsers{}) is an int.
func Ask[R any](ctx context.Context, h QueryHandler, q QueryOf[R]) (R, error) {
	return q.HandleQuery(ctx, h)
}

// --8<-- [end:ask]

// --8<-- [start:in-memory]

// InMemory answers every query from a slice.
type InMemory struct{ Users []User }

var _ QueryHandler = InMemory{} // the compiler checks that every query is answered

func (m InMemory) HandleGetUser(_ context.Context, q *GetUser) (*User, error) {
	for _, u := range m.Users {
		if u.ID == q.ID {
			return &u, nil
		}
	}
	return nil, nil
}

func (m InMemory) HandleFindUsers(_ context.Context, q *FindUsers) ([]User, error) {
	var found []User
	for _, u := range m.Users {
		if strings.HasPrefix(u.Name, q.Prefix) {
			found = append(found, u)
		}
	}
	return found, nil
}

func (m InMemory) HandleCountUsers(context.Context, *CountUsers) (int, error) {
	return len(m.Users), nil
}

// --8<-- [end:in-memory]
