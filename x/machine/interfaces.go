package machine

import (
	"context"
)

// StateStore provides an abstraction for storing and retrieving states
// It is generic over the state type S and supports any storage backend
type StateStore[S any] interface {
	// Load retrieves a state by its ID
	// Returns the state and version, or an error if not found
	Load(ctx context.Context, id string) (state S, version uint16, err error)
	
	// Save persists a state with the given ID and version
	// For new states, use version 0. For updates, provide the current version
	// Returns the new version after successful save
	Save(ctx context.Context, id string, state S, version uint16) (newVersion uint16, err error)
	
	// Query retrieves states based on a query interface
	// The query parameter is backend-specific (e.g., predicate.WherePredicates)
	Query(ctx context.Context, query interface{}, limit int) ([]StateRecord[S], error)
}

// StateRecord represents a stored state with metadata
type StateRecord[S any] struct {
	ID      string
	State   S
	Version uint16
}

// CommandRouter extracts query information from commands
// This is used to load the appropriate state before processing a command
type CommandRouter[C any] interface {
	// ExtractQuery returns a query object and a boolean indicating if the command
	// should load an existing state (true) or create a new one (false)
	ExtractQuery(cmd C) (query interface{}, shouldLoad bool)
}

// StateIdentifier extracts the ID from a state for persistence
type StateIdentifier[S any] interface {
	// ExtractID returns the ID from the state and a boolean indicating success
	ExtractID(state S) (id string, ok bool)
}

// MachineProvider creates state machines with dependency injection
type MachineProvider[D, C, S any] interface {
	// NewMachine creates a new machine instance with the given state
	NewMachine(state S) *Machine[D, C, S]
}

// MachineProviderFunc is a function adapter for MachineProvider
type MachineProviderFunc[D, C, S any] func(state S) *Machine[D, C, S]

func (f MachineProviderFunc[D, C, S]) NewMachine(state S) *Machine[D, C, S] {
	return f(state)
}

// CommandRouterFunc is a function adapter for CommandRouter
type CommandRouterFunc[C any] func(cmd C) (query interface{}, shouldLoad bool)

func (f CommandRouterFunc[C]) ExtractQuery(cmd C) (query interface{}, shouldLoad bool) {
	return f(cmd)
}

// StateIdentifierFunc is a function adapter for StateIdentifier
type StateIdentifierFunc[S any] func(state S) (id string, ok bool)

func (f StateIdentifierFunc[S]) ExtractID(state S) (id string, ok bool) {
	return f(state)
}