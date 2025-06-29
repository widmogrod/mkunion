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

// Test domain for suite integration
type SuiteState struct {
	ID     string
	Status string
	Count  int
}

// SuiteCommand represents test commands
type SuiteCommand interface {
	isSuiteCommand()
}

type InitCmd struct {
	ID string
}

func (*InitCmd) isSuiteCommand() {}

type ProcessCmd struct {
	ID    string
	Value int
}

func (*ProcessCmd) isSuiteCommand() {}

type CompleteCmd struct {
	ID string
}

func (*CompleteCmd) isSuiteCommand() {}

type SuiteDeps struct {
	MaxCount int
}

func suiteTransition(ctx context.Context, deps SuiteDeps, cmd SuiteCommand, state SuiteState) (SuiteState, error) {
	switch c := cmd.(type) {
	case *InitCmd:
		return SuiteState{
			ID:     c.ID,
			Status: "initialized",
			Count:  0,
		}, nil
	case *ProcessCmd:
		if state.Status == "completed" {
			return state, fmt.Errorf("cannot process completed state")
		}
		
		newCount := state.Count + c.Value
		if newCount > deps.MaxCount {
			return state, fmt.Errorf("count %d exceeds max %d", newCount, deps.MaxCount)
		}
		
		state.Count = newCount
		state.Status = "processing"
		return state, nil
	case *CompleteCmd:
		if state.Status != "processing" {
			return state, fmt.Errorf("can only complete from processing state")
		}
		state.Status = "completed"
		return state, nil
	default:
		return state, fmt.Errorf("unknown command type: %T", cmd)
	}
}

func TestPersistentMachine_WithTestSuite(t *testing.T) {
	// Create in-memory store with query handler
	store := adapters.NewInMemoryStore[SuiteState]().
		WithQueryHandler(func(query interface{}, records map[string]*machine.StateRecord[SuiteState]) []machine.StateRecord[SuiteState] {
			id, ok := query.(string)
			if !ok {
				return nil
			}
			
			if record, exists := records[id]; exists {
				return []machine.StateRecord[SuiteState]{*record}
			}
			return nil
		})
	
	deps := SuiteDeps{MaxCount: 100}
	
	// Create persistent machine
	pm := machine.NewPersistentMachine[SuiteDeps, SuiteCommand, SuiteState](
		store,
		machine.CommandRouterFunc[SuiteCommand](func(cmd SuiteCommand) (interface{}, bool) {
			switch c := cmd.(type) {
			case *InitCmd:
				return nil, false // Create new
			case *ProcessCmd:
				return c.ID, true // Load existing
			case *CompleteCmd:
				return c.ID, true // Load existing
			default:
				return nil, false
			}
		}),
		machine.StateIdentifierFunc[SuiteState](func(state SuiteState) (string, bool) {
			return state.ID, state.ID != ""
		}),
		machine.MachineProviderFunc[SuiteDeps, SuiteCommand, SuiteState](func(state SuiteState) *machine.Machine[SuiteDeps, SuiteCommand, SuiteState] {
			return machine.NewMachine(deps, suiteTransition, state)
		}),
	)
	
	// Test sequence
	t.Run("full lifecycle", func(t *testing.T) {
		// Initialize
		state, err := pm.CreateOrUpdate(context.Background(), &InitCmd{ID: "test-123"})
		require.NoError(t, err)
		assert.Equal(t, "test-123", state.ID)
		assert.Equal(t, "initialized", state.Status)
		assert.Equal(t, 0, state.Count)
		
		// Process multiple times
		state, err = pm.CreateOrUpdate(context.Background(), &ProcessCmd{ID: "test-123", Value: 10})
		require.NoError(t, err)
		assert.Equal(t, "processing", state.Status)
		assert.Equal(t, 10, state.Count)
		
		state, err = pm.CreateOrUpdate(context.Background(), &ProcessCmd{ID: "test-123", Value: 20})
		require.NoError(t, err)
		assert.Equal(t, 30, state.Count)
		
		// Complete
		state, err = pm.CreateOrUpdate(context.Background(), &CompleteCmd{ID: "test-123"})
		require.NoError(t, err)
		assert.Equal(t, "completed", state.Status)
		
		// Verify cannot process after completion
		_, err = pm.CreateOrUpdate(context.Background(), &ProcessCmd{ID: "test-123", Value: 10})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot process completed state")
	})
}

// TestPersistentMachine_TestSuiteCompatibility demonstrates using TestSuite with persistent storage
func TestPersistentMachine_TestSuiteCompatibility(t *testing.T) {
	// Create shared store for persistence
	store := adapters.NewInMemoryStore[SuiteState]().
		WithQueryHandler(func(query interface{}, records map[string]*machine.StateRecord[SuiteState]) []machine.StateRecord[SuiteState] {
			id, ok := query.(string)
			if !ok {
				return nil
			}
			
			if record, exists := records[id]; exists {
				return []machine.StateRecord[SuiteState]{*record}
			}
			return nil
		})
	
	deps := SuiteDeps{MaxCount: 50}
	
	// Factory function that creates machines with persistence awareness
	mkMachine := func(dep SuiteDeps, init SuiteState) *machine.Machine[SuiteDeps, SuiteCommand, SuiteState] {
		// If init state has an ID, try to load from store
		if init.ID != "" {
			if loaded, _, err := store.Load(context.Background(), init.ID); err == nil {
				init = loaded
			}
		}
		
		// Create a wrapper transition that persists after each command
		persistingTransition := func(ctx context.Context, d SuiteDeps, cmd SuiteCommand, state SuiteState) (SuiteState, error) {
			newState, err := suiteTransition(ctx, d, cmd, state)
			if err != nil {
				return newState, err
			}
			
			// Persist state if it has an ID
			if newState.ID != "" {
				version := uint16(0)
				if state.ID == newState.ID {
					// Get current version for update
					_, v, _ := store.Load(ctx, newState.ID)
					version = v
				}
				_, saveErr := store.Save(ctx, newState.ID, newState, version)
				if saveErr != nil {
					return state, fmt.Errorf("failed to persist state: %w", saveErr)
				}
			}
			
			return newState, nil
		}
		
		return machine.NewMachine(dep, persistingTransition, init)
	}
	
	// Create test suite
	suite := machine.NewTestSuite(deps, mkMachine)
	
	suite.Case(t, "init and process", func(t *testing.T, c *machine.Case[SuiteDeps, SuiteCommand, SuiteState]) {
		c.GivenCommand(&InitCmd{ID: "suite-test-1"}).
			ThenState(t, SuiteState{
				ID:     "suite-test-1",
				Status: "initialized",
				Count:  0,
			})
		
		// The state was already persisted by the previous command, so we need to
		// explicitly set it as the initial state for the next command
		c.ForkCase(t, "process after init", func(t *testing.T, c *machine.Case[SuiteDeps, SuiteCommand, SuiteState]) {
			c.GivenCommand(&ProcessCmd{ID: "suite-test-1", Value: 25}).
				ThenState(t, SuiteState{
					ID:     "suite-test-1",
					Status: "processing",
					Count:  25,
				})
		})
	})
	
	suite.Case(t, "exceed max count", func(t *testing.T, c *machine.Case[SuiteDeps, SuiteCommand, SuiteState]) {
		c.GivenCommand(&InitCmd{ID: "suite-test-2"}).
			ThenState(t, SuiteState{
				ID:     "suite-test-2",
				Status: "initialized",
				Count:  0,
			})
		
		c.GivenCommand(&ProcessCmd{ID: "suite-test-2", Value: 60}).
			ThenStateAndError(t, 
				SuiteState{
					ID:     "suite-test-2",
					Status: "initialized",
					Count:  0,
				},
				fmt.Errorf("count 60 exceeds max 50"),
			)
	})
	
	// Verify persistence worked
	stored, _, err := store.Load(context.Background(), "suite-test-1")
	require.NoError(t, err)
	assert.Equal(t, "processing", stored.Status)
	assert.Equal(t, 25, stored.Count)
}