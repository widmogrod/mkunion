package machine

import (
	"context"
	"fmt"
)

// PersistentMachine provides persistent state machine functionality
// It combines state storage with command processing and automatic persistence
type PersistentMachine[D, C, S any] struct {
	store      StateStore[S]
	router     CommandRouter[C]
	identifier StateIdentifier[S]
	provider   MachineProvider[D, C, S]
}

// NewPersistentMachine creates a new persistent state machine
func NewPersistentMachine[D, C, S any](
	store StateStore[S],
	router CommandRouter[C],
	identifier StateIdentifier[S],
	provider MachineProvider[D, C, S],
) *PersistentMachine[D, C, S] {
	return &PersistentMachine[D, C, S]{
		store:      store,
		router:     router,
		identifier: identifier,
		provider:   provider,
	}
}

// CreateOrUpdate processes a command, loading or creating state as needed
// It handles the full lifecycle: query extraction, state loading, command processing, and persistence
func (pm *PersistentMachine[D, C, S]) CreateOrUpdate(ctx context.Context, cmd C) (S, error) {
	var state S
	var version uint16
	var recordID string
	
	// Extract query from command to determine if we should load existing state
	query, shouldLoad := pm.router.ExtractQuery(cmd)
	
	if shouldLoad {
		// Load existing state based on query
		records, err := pm.store.Query(ctx, query, 1)
		if err != nil {
			return state, fmt.Errorf("failed to query state: %w", err)
		}
		
		if len(records) == 0 {
			return state, fmt.Errorf("expected at least one record")
		}
		
		record := records[0]
		state = record.State
		version = record.Version
		recordID = record.ID
	}
	
	// Create machine with loaded or zero state
	machine := pm.provider.NewMachine(state)
	
	// Process command
	if err := machine.Handle(ctx, cmd); err != nil {
		return state, fmt.Errorf("failed to handle command: %w", err)
	}
	
	// Get new state after processing
	newState := machine.State()
	
	// Extract ID from state if creating new record
	if !shouldLoad {
		id, ok := pm.identifier.ExtractID(newState)
		if !ok {
			return state, fmt.Errorf("expected ID in state")
		}
		recordID = id
	}
	
	if recordID == "" {
		return state, fmt.Errorf("expected recordID in state")
	}
	
	// Save state with optimistic locking
	_, err := pm.store.Save(ctx, recordID, newState, version)
	if err != nil {
		return state, fmt.Errorf("failed to save state: %w", err)
	}
	
	return newState, nil
}

// StateByID retrieves a state by its ID
func (pm *PersistentMachine[D, C, S]) StateByID(ctx context.Context, id string) (S, error) {
	state, _, err := pm.store.Load(ctx, id)
	if err != nil {
		return state, fmt.Errorf("PersistentMachine.StateByID(%s): %w", id, err)
	}
	return state, nil
}

// Builder provides a fluent API for constructing PersistentMachine instances
type PersistentMachineBuilder[D, C, S any] struct {
	store      StateStore[S]
	router     CommandRouter[C]
	identifier StateIdentifier[S]
	provider   MachineProvider[D, C, S]
}

// NewPersistentMachineBuilder creates a new builder
func NewPersistentMachineBuilder[D, C, S any]() *PersistentMachineBuilder[D, C, S] {
	return &PersistentMachineBuilder[D, C, S]{}
}

// WithStore sets the state store
func (b *PersistentMachineBuilder[D, C, S]) WithStore(store StateStore[S]) *PersistentMachineBuilder[D, C, S] {
	b.store = store
	return b
}

// WithRouter sets the command router
func (b *PersistentMachineBuilder[D, C, S]) WithRouter(router CommandRouter[C]) *PersistentMachineBuilder[D, C, S] {
	b.router = router
	return b
}

// WithRouterFunc sets the command router using a function
func (b *PersistentMachineBuilder[D, C, S]) WithRouterFunc(f func(cmd C) (query interface{}, shouldLoad bool)) *PersistentMachineBuilder[D, C, S] {
	b.router = CommandRouterFunc[C](f)
	return b
}

// WithIdentifier sets the state identifier
func (b *PersistentMachineBuilder[D, C, S]) WithIdentifier(identifier StateIdentifier[S]) *PersistentMachineBuilder[D, C, S] {
	b.identifier = identifier
	return b
}

// WithIdentifierFunc sets the state identifier using a function
func (b *PersistentMachineBuilder[D, C, S]) WithIdentifierFunc(f func(state S) (id string, ok bool)) *PersistentMachineBuilder[D, C, S] {
	b.identifier = StateIdentifierFunc[S](f)
	return b
}

// WithProvider sets the machine provider
func (b *PersistentMachineBuilder[D, C, S]) WithProvider(provider MachineProvider[D, C, S]) *PersistentMachineBuilder[D, C, S] {
	b.provider = provider
	return b
}

// WithProviderFunc sets the machine provider using a function
func (b *PersistentMachineBuilder[D, C, S]) WithProviderFunc(f func(state S) *Machine[D, C, S]) *PersistentMachineBuilder[D, C, S] {
	b.provider = MachineProviderFunc[D, C, S](f)
	return b
}

// Build creates the PersistentMachine, validating all required components
func (b *PersistentMachineBuilder[D, C, S]) Build() (*PersistentMachine[D, C, S], error) {
	if b.store == nil {
		return nil, fmt.Errorf("store is required")
	}
	if b.router == nil {
		return nil, fmt.Errorf("router is required")
	}
	if b.identifier == nil {
		return nil, fmt.Errorf("identifier is required")
	}
	if b.provider == nil {
		return nil, fmt.Errorf("provider is required")
	}
	
	return NewPersistentMachine(b.store, b.router, b.identifier, b.provider), nil
}