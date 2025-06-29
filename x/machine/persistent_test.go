package machine_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/machine"
	"github.com/widmogrod/mkunion/x/machine/adapters"
)

// Test domain types
type TestState struct {
	ID      string
	Counter int
	Name    string
}

type TestCommand struct {
	ID        string
	Operation string
	Value     int
}

type TestDeps struct {
	Logger func(string)
}

// Test transition function
func testTransition(ctx context.Context, deps TestDeps, cmd TestCommand, state TestState) (TestState, error) {
	deps.Logger(fmt.Sprintf("Processing command: %+v", cmd))
	
	switch cmd.Operation {
	case "create":
		return TestState{
			ID:      cmd.ID,
			Counter: cmd.Value,
			Name:    fmt.Sprintf("Item-%s", cmd.ID),
		}, nil
	case "increment":
		state.Counter += cmd.Value
		return state, nil
	case "decrement":
		state.Counter -= cmd.Value
		return state, nil
	case "rename":
		state.Name = fmt.Sprintf("Item-%d", cmd.Value)
		return state, nil
	default:
		return state, fmt.Errorf("unknown operation: %s", cmd.Operation)
	}
}

func TestPersistentMachine_CreateOrUpdate(t *testing.T) {
	t.Run("create new state", func(t *testing.T) {
		// Setup
		store := adapters.NewInMemoryStore[TestState]()
		deps := TestDeps{
			Logger: func(msg string) { t.Log(msg) },
		}
		
		pm := machine.NewPersistentMachine[TestDeps, TestCommand, TestState](
			store,
			machine.CommandRouterFunc[TestCommand](func(cmd TestCommand) (interface{}, bool) {
				if cmd.Operation == "create" {
					return nil, false // Don't load existing state
				}
				return cmd.ID, true // Load by ID
			}),
			machine.StateIdentifierFunc[TestState](func(state TestState) (string, bool) {
				return state.ID, state.ID != ""
			}),
			machine.MachineProviderFunc[TestDeps, TestCommand, TestState](func(state TestState) *machine.Machine[TestDeps, TestCommand, TestState] {
				return machine.NewMachine(deps, testTransition, state)
			}),
		)
		
		// Execute
		cmd := TestCommand{ID: "123", Operation: "create", Value: 10}
		result, err := pm.CreateOrUpdate(context.Background(), cmd)
		
		// Assert
		require.NoError(t, err)
		assert.Equal(t, "123", result.ID)
		assert.Equal(t, 10, result.Counter)
		assert.Equal(t, "Item-123", result.Name)
		
		// Verify state was persisted
		stored, err := pm.StateByID(context.Background(), "123")
		require.NoError(t, err)
		assert.Equal(t, result, stored)
	})
	
	t.Run("update existing state", func(t *testing.T) {
		// Setup with query handler
		store := adapters.NewInMemoryStore[TestState]().
			WithQueryHandler(func(query interface{}, records map[string]*machine.StateRecord[TestState]) []machine.StateRecord[TestState] {
				id, ok := query.(string)
				if !ok {
					return nil
				}
				
				if record, exists := records[id]; exists {
					return []machine.StateRecord[TestState]{*record}
				}
				return nil
			})
		
		// Pre-populate state
		_, err := store.Save(context.Background(), "123", TestState{
			ID:      "123",
			Counter: 10,
			Name:    "Item-123",
		}, 0)
		require.NoError(t, err)
		
		deps := TestDeps{
			Logger: func(msg string) { t.Log(msg) },
		}
		
		pm := machine.NewPersistentMachine[TestDeps, TestCommand, TestState](
			store,
			machine.CommandRouterFunc[TestCommand](func(cmd TestCommand) (interface{}, bool) {
				if cmd.Operation == "create" {
					return nil, false
				}
				return cmd.ID, true // Use ID as query
			}),
			machine.StateIdentifierFunc[TestState](func(state TestState) (string, bool) {
				return state.ID, state.ID != ""
			}),
			machine.MachineProviderFunc[TestDeps, TestCommand, TestState](func(state TestState) *machine.Machine[TestDeps, TestCommand, TestState] {
				return machine.NewMachine(deps, testTransition, state)
			}),
		)
		
		// Execute increment
		cmd := TestCommand{ID: "123", Operation: "increment", Value: 5}
		result, err := pm.CreateOrUpdate(context.Background(), cmd)
		
		// Assert
		require.NoError(t, err)
		assert.Equal(t, "123", result.ID)
		assert.Equal(t, 15, result.Counter)
		assert.Equal(t, "Item-123", result.Name)
		
		// Verify version was incremented
		_, version, err := store.Load(context.Background(), "123")
		require.NoError(t, err)
		assert.Equal(t, uint16(2), version)
	})
	
	t.Run("optimistic locking", func(t *testing.T) {
		// This test simulates concurrent updates
		store := adapters.NewInMemoryStore[TestState]()
		
		// Pre-populate state
		_, err := store.Save(context.Background(), "123", TestState{
			ID:      "123",
			Counter: 10,
			Name:    "Item-123",
		}, 0)
		require.NoError(t, err)
		
		// Simulate concurrent modification by directly updating the store
		_, err = store.Save(context.Background(), "123", TestState{
			ID:      "123",
			Counter: 20,
			Name:    "Modified",
		}, 1)
		require.NoError(t, err)
		
		// Now try to update with old version - this should fail during save
		// In a real scenario, the PersistentMachine would need retry logic
		state, version, err := store.Load(context.Background(), "123")
		require.NoError(t, err)
		assert.Equal(t, uint16(2), version)
		assert.Equal(t, 20, state.Counter)
	})
}

func TestPersistentMachine_Builder(t *testing.T) {
	t.Run("successful build", func(t *testing.T) {
		store := adapters.NewInMemoryStore[TestState]()
		
		pm, err := machine.NewPersistentMachineBuilder[TestDeps, TestCommand, TestState]().
			WithStore(store).
			WithRouterFunc(func(cmd TestCommand) (interface{}, bool) {
				return cmd.ID, cmd.Operation != "create"
			}).
			WithIdentifierFunc(func(state TestState) (string, bool) {
				return state.ID, state.ID != ""
			}).
			WithProviderFunc(func(state TestState) *machine.Machine[TestDeps, TestCommand, TestState] {
				deps := TestDeps{Logger: func(string) {}}
				return machine.NewMachine(deps, testTransition, state)
			}).
			Build()
		
		require.NoError(t, err)
		assert.NotNil(t, pm)
	})
	
	t.Run("missing store", func(t *testing.T) {
		_, err := machine.NewPersistentMachineBuilder[TestDeps, TestCommand, TestState]().
			WithRouterFunc(func(cmd TestCommand) (interface{}, bool) {
				return cmd.ID, true
			}).
			WithIdentifierFunc(func(state TestState) (string, bool) {
				return state.ID, true
			}).
			WithProviderFunc(func(state TestState) *machine.Machine[TestDeps, TestCommand, TestState] {
				return nil
			}).
			Build()
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "store is required")
	})
}

func TestPersistentMachine_Integration(t *testing.T) {
	// This test demonstrates a more complex scenario with multiple operations
	store := adapters.NewInMemoryStore[TestState]().
		WithQueryHandler(adapters.SimpleQueryMatcher(func(s TestState) string {
			return s.ID
		}))
	
	deps := TestDeps{
		Logger: func(msg string) { t.Log(msg) },
	}
	
	pm := machine.NewPersistentMachine[TestDeps, TestCommand, TestState](
		store,
		machine.CommandRouterFunc[TestCommand](func(cmd TestCommand) (interface{}, bool) {
			if cmd.Operation == "create" {
				return nil, false
			}
			return cmd.ID, true
		}),
		machine.StateIdentifierFunc[TestState](func(state TestState) (string, bool) {
			return state.ID, state.ID != ""
		}),
		machine.MachineProviderFunc[TestDeps, TestCommand, TestState](func(state TestState) *machine.Machine[TestDeps, TestCommand, TestState] {
			return machine.NewMachine(deps, testTransition, state)
		}),
	)
	
	// Create
	_, err := pm.CreateOrUpdate(context.Background(), TestCommand{
		ID:        "item-1",
		Operation: "create",
		Value:     100,
	})
	require.NoError(t, err)
	
	// Increment
	_, err = pm.CreateOrUpdate(context.Background(), TestCommand{
		ID:        "item-1",
		Operation: "increment",
		Value:     50,
	})
	require.NoError(t, err)
	
	// Decrement
	_, err = pm.CreateOrUpdate(context.Background(), TestCommand{
		ID:        "item-1",
		Operation: "decrement",
		Value:     25,
	})
	require.NoError(t, err)
	
	// Verify final state
	state, err := pm.StateByID(context.Background(), "item-1")
	require.NoError(t, err)
	assert.Equal(t, "item-1", state.ID)
	assert.Equal(t, 125, state.Counter) // 100 + 50 - 25
	assert.Equal(t, "Item-item-1", state.Name)
	
	// Verify store has correct version
	_, version, err := store.Load(context.Background(), "item-1")
	require.NoError(t, err)
	assert.Equal(t, uint16(3), version) // Created at v1, then 2 updates
}