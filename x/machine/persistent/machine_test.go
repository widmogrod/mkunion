package persistent

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
	"github.com/widmogrod/mkunion/x/schema"
)

// Test types and transition function
type TestDependencies struct {
	Counter int
}

func testTransitionFunc(ctx context.Context, deps TestDependencies, cmd TestCommand, state TestState) (TestState, error) {
	return MatchTestCommandR2(cmd,
		func(c *CreateTestCommand) (TestState, error) {
			return &TestActiveState{ID: c.ID, Name: c.Name}, nil
		},
		func(c *UpdateTestCommand) (TestState, error) {
			// Update existing state
			return MatchTestStateR2(state,
				func(s *TestInitialState) (TestState, error) {
					return state, nil // Can't update initial state
				},
				func(s *TestActiveState) (TestState, error) {
					return &TestActiveState{ID: s.ID, Name: c.NewName}, nil
				},
				func(s *TestDeletedState) (TestState, error) {
					return state, nil // Can't update deleted state
				},
			)
		},
		func(c *DeleteTestCommand) (TestState, error) {
			return &TestDeletedState{ID: c.ID}, nil
		},
	)
}

func TestPersistentMachine_CreateNew(t *testing.T) {
	repo := setupTestRepo(t)
	deps := TestDependencies{Counter: 0}
	
	machine := New(
		"test",
		repo,
		deps,
		testTransitionFunc,
		Configure[TestCommand, TestState]().
			CommandThatCreatesState(&CreateTestCommand{}).
			StateIDFrom(ExtractStateIDFromLocation[TestState]("ID")),
	)
	
	ctx := context.Background()
	
	// Test creating new state
	cmd := &CreateTestCommand{ID: "123", Name: "Test Item"}
	state, err := machine.CreateOrUpdate(ctx, cmd)
	
	assert.NoError(t, err)
	assert.IsType(t, &TestActiveState{}, state)
	assert.Equal(t, "123", state.(*TestActiveState).ID)
	assert.Equal(t, "Test Item", state.(*TestActiveState).Name)
	
	// Verify it was saved
	saved, err := machine.StateByID("123")
	assert.NoError(t, err)
	assert.Equal(t, state, saved)
}

func TestPersistentMachine_UpdateExisting(t *testing.T) {
	repo := setupTestRepo(t)
	deps := TestDependencies{Counter: 0}
	
	machine := New(
		"test",
		repo,
		deps,
		testTransitionFunc,
		Configure[TestCommand, TestState]().
			CommandThatCreatesState(&CreateTestCommand{}).
			CommandThatUpdatesState(&UpdateTestCommand{}, func(cmd TestCommand) *predicate.WherePredicates {
				update := cmd.(*UpdateTestCommand)
				return predicate.MustWhere(
					`ID = :id`,
					predicate.ParamBinds{":id": schema.MkString(update.ID)},
					nil,
				)
			}).
			StateIDFrom(ExtractStateIDFromLocation[TestState]("ID")),
	)
	
	ctx := context.Background()
	
	// First create a state
	createCmd := &CreateTestCommand{ID: "123", Name: "Original"}
	_, err := machine.CreateOrUpdate(ctx, createCmd)
	assert.NoError(t, err)
	
	// Update it
	updateCmd := &UpdateTestCommand{ID: "123", NewName: "Updated"}
	updated, err := machine.CreateOrUpdate(ctx, updateCmd)
	
	assert.NoError(t, err)
	assert.IsType(t, &TestActiveState{}, updated)
	assert.Equal(t, "123", updated.(*TestActiveState).ID)
	assert.Equal(t, "Updated", updated.(*TestActiveState).Name)
	
	// Verify version was incremented
	record, err := repo.Get("123", "test")
	assert.NoError(t, err)
	assert.Equal(t, uint16(2), record.Version) // Created with version 1, updated to version 2
}

func TestPersistentMachine_RequiredConfig(t *testing.T) {
	repo := setupTestRepo(t)
	deps := TestDependencies{Counter: 0}
	
	// Should panic without StateIDFrom
	assert.Panics(t, func() {
		New(
			"test",
			repo,
			deps,
			testTransitionFunc,
			Configure[TestCommand, TestState](),
		)
	})
}

func TestPersistentMachine_Handle(t *testing.T) {
	repo := setupTestRepo(t)
	deps := TestDependencies{Counter: 0}
	
	machine := New(
		"test",
		repo,
		deps,
		testTransitionFunc,
		Configure[TestCommand, TestState]().
			CommandThatCreatesState(&CreateTestCommand{}).
			StateIDFrom(ExtractStateIDFromLocation[TestState]("ID")),
	)
	
	ctx := context.Background()
	
	// Test Handle method (for machine.Machine compatibility)
	cmd := &CreateTestCommand{ID: "456", Name: "Handle Test"}
	err := machine.Handle(ctx, cmd)
	
	assert.NoError(t, err)
	
	// Verify it was created
	state, err := machine.StateByID("456")
	assert.NoError(t, err)
	assert.IsType(t, &TestActiveState{}, state)
}

func TestPersistentMachine_DefaultBehavior(t *testing.T) {
	repo := setupTestRepo(t)
	deps := TestDependencies{Counter: 0}
	
	// Without any command configuration, all commands should create new states
	machine := New(
		"test",
		repo,
		deps,
		testTransitionFunc,
		Configure[TestCommand, TestState]().
			StateIDFrom(ExtractStateIDFromLocation[TestState]("ID")),
	)
	
	ctx := context.Background()
	
	// All commands should create new states
	cmd := &CreateTestCommand{ID: "789", Name: "Default"}
	state, err := machine.CreateOrUpdate(ctx, cmd)
	
	assert.NoError(t, err)
	assert.IsType(t, &TestActiveState{}, state)
}

func TestPersistentMachine_MissingStateForUpdate(t *testing.T) {
	repo := setupTestRepo(t)
	deps := TestDependencies{Counter: 0}
	
	machine := New(
		"test",
		repo,
		deps,
		testTransitionFunc,
		Configure[TestCommand, TestState]().
			CommandThatUpdatesState(&UpdateTestCommand{}, func(cmd TestCommand) *predicate.WherePredicates {
				update := cmd.(*UpdateTestCommand)
				return predicate.MustWhere(
					`ID = :id`,
					predicate.ParamBinds{":id": schema.MkString(update.ID)},
					nil,
				)
			}).
			StateIDFrom(ExtractStateIDFromLocation[TestState]("ID")),
	)
	
	ctx := context.Background()
	
	// Try to update non-existent state
	cmd := &UpdateTestCommand{ID: "nonexistent", NewName: "Should Fail"}
	_, err := machine.CreateOrUpdate(ctx, cmd)
	
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no existing state found")
}

// Helper to setup test repository
func setupTestRepo(t *testing.T) *typedful.TypedRepoWithAggregator[TestState, any] {
	// Create in-memory repository
	baseRepo := schemaless.NewInMemoryRepository[schema.Schema]()
	// Use noop aggregator for tests
	aggregator := func() schemaless.Aggregator[TestState, any] {
		return schemaless.NewNoopAggregator[TestState, any]()
	}
	return typedful.NewTypedRepoWithAggregator[TestState, any](baseRepo, aggregator)
}