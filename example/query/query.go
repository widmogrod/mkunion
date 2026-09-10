// Package query shows the typed handler on its own, with no effect system
// around it: a union of questions, each saying what it answers with, and a
// typed handler generated from that.
package query

import (
	"context"
	"strings"

	"github.com/widmogrod/mkunion/f"
)

// --8<-- [start:model]

// User is what the directory knows about a person.
type User struct{ ID, Name string }

// Query is every question the user directory can answer. A variant embeds
// f.Returns[R] to say what it answers with; that marker alone makes mkunion
// generate QueryHandler, one typed method per question. A variant without
// the marker, like DeleteUser, is handled with an error only.
//
//go:tag mkunion:"Query"
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
	// DeleteUser answers with nothing: it either worked or it did not.
	DeleteUser struct{ ID string }
)

// --8<-- [end:model]

// --8<-- [start:ask]

// Answers is a Query that answers with R. Ret comes from the f.Returns[R]
// marker and Perform is generated for every variant that has one.
type Answers[R any] interface {
	Query
	Ret() R
	Perform(ctx context.Context, h any) (any, error)
}

// Ask puts one query to a handler and returns the answer with the type the
// query declared. R is inferred from the query, so a caller never spells it:
// Ask(ctx, h, &CountUsers{}) is an int. The cast is safe: Perform hands the
// query to its own typed method.
func Ask[R any](ctx context.Context, h QueryHandler, q Answers[R]) (R, error) {
	answer, err := q.Perform(ctx, h)
	if err != nil {
		var zero R
		return zero, err
	}
	return answer.(R), nil
}

// --8<-- [end:ask]

// --8<-- [start:in-memory]

// InMemory answers every query from a slice.
type InMemory struct{ Users []User }

var _ QueryHandler = (*InMemory)(nil) // the compiler checks that every query is answered

func (m *InMemory) HandleGetUser(_ context.Context, q *GetUser) (*User, error) {
	for _, u := range m.Users {
		if u.ID == q.ID {
			return &u, nil
		}
	}
	return nil, nil
}

func (m *InMemory) HandleFindUsers(_ context.Context, q *FindUsers) ([]User, error) {
	var found []User
	for _, u := range m.Users {
		if strings.HasPrefix(u.Name, q.Prefix) {
			found = append(found, u)
		}
	}
	return found, nil
}

func (m *InMemory) HandleCountUsers(context.Context, *CountUsers) (int, error) {
	return len(m.Users), nil
}

func (m *InMemory) HandleDeleteUser(_ context.Context, q *DeleteUser) error {
	kept := m.Users[:0]
	for _, u := range m.Users {
		if u.ID != q.ID {
			kept = append(kept, u)
		}
	}
	m.Users = kept
	return nil
}

// --8<-- [end:in-memory]
