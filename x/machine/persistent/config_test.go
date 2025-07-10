package persistent

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/schema"
)

//go:tag mkunion:"TestCommand"
type (
	CreateTestCommand struct {
		ID   string
		Name string
	}
	UpdateTestCommand struct {
		ID      string
		NewName string
	}
	DeleteTestCommand struct {
		ID string
	}
)

//go:tag mkunion:"TestState"
type (
	TestInitialState struct{}
	TestActiveState struct {
		ID   string
		Name string
	}
	TestDeletedState struct {
		ID string
	}
)

func TestConfig_CommandClassification(t *testing.T) {
	t.Run("should classify create commands", func(t *testing.T) {
		config := Configure[TestCommand, TestState]().
			CommandThatCreatesState(&CreateTestCommand{})
		
		assert.True(t, config.IsCreateCommand(&CreateTestCommand{}))
		assert.False(t, config.IsCreateCommand(&UpdateTestCommand{}))
	})
	
	t.Run("should classify update commands", func(t *testing.T) {
		config := Configure[TestCommand, TestState]().
			CommandThatUpdatesState(&UpdateTestCommand{}, func(cmd TestCommand) *predicate.WherePredicates {
				update := cmd.(*UpdateTestCommand)
				return predicate.MustWhere(
					"ID = :id",
					predicate.ParamBinds{":id": schema.MkString(update.ID)},
					nil,
				)
			})
		
		assert.False(t, config.IsCreateCommand(&UpdateTestCommand{}))
		assert.True(t, config.IsUpdateCommand(&UpdateTestCommand{}))
		
		// Should be able to get query function
		queryFunc := config.GetUpdateQuery(&UpdateTestCommand{ID: "123"})
		assert.NotNil(t, queryFunc)
	})
	
	t.Run("should support multiple commands of same type", func(t *testing.T) {
		config := Configure[TestCommand, TestState]().
			CommandThatCreatesState(
				&CreateTestCommand{},
				&DeleteTestCommand{},
			)
		
		assert.True(t, config.IsCreateCommand(&CreateTestCommand{}))
		assert.True(t, config.IsCreateCommand(&DeleteTestCommand{}))
	})
}

func TestConfig_StateIDExtraction(t *testing.T) {
	t.Run("should require state ID function", func(t *testing.T) {
		config := Configure[TestCommand, TestState]()
		assert.Nil(t, config.stateIDFunc)
	})
	
	t.Run("should set custom state ID function", func(t *testing.T) {
		config := Configure[TestCommand, TestState]().
			StateIDFrom(func(s TestState) (string, bool) {
				return MatchTestStateR2(s,
					func(x *TestInitialState) (string, bool) { return "", false },
					func(x *TestActiveState) (string, bool) { return x.ID, true },
					func(x *TestDeletedState) (string, bool) { return x.ID, true },
				)
			})
		
		assert.NotNil(t, config.stateIDFunc)
		
		// Test extraction
		id, ok := config.stateIDFunc(&TestActiveState{ID: "123", Name: "Test"})
		assert.True(t, ok)
		assert.Equal(t, "123", id)
		
		id, ok = config.stateIDFunc(&TestInitialState{})
		assert.False(t, ok)
		assert.Equal(t, "", id)
	})
}

func TestExtractStateIDFromLocation(t *testing.T) {
	t.Run("should extract ID from simple location", func(t *testing.T) {
		extractor := ExtractStateIDFromLocation[TestState]("ID")
		
		id, ok := extractor(&TestActiveState{ID: "123", Name: "Test"})
		assert.True(t, ok)
		assert.Equal(t, "123", id)
	})
	
	t.Run("should handle missing field", func(t *testing.T) {
		extractor := ExtractStateIDFromLocation[TestState]("NonExistent")
		
		id, ok := extractor(&TestActiveState{ID: "123", Name: "Test"})
		assert.False(t, ok)
		assert.Equal(t, "", id)
	})
	
}

func TestExtractStateIDUsingShape(t *testing.T) {
	t.Run("should find common ID field", func(t *testing.T) {
		extractor := ExtractStateIDUsingShape[TestState]()
		
		id, ok := extractor(&TestActiveState{ID: "123", Name: "Test"})
		assert.True(t, ok)
		assert.Equal(t, "123", id)
	})
	
	t.Run("should handle state without ID field", func(t *testing.T) {
		extractor := ExtractStateIDUsingShape[TestState]()
		
		id, ok := extractor(&TestInitialState{})
		assert.False(t, ok)
		assert.Equal(t, "", id)
	})
}

func TestConfig_Validation(t *testing.T) {
	t.Run("should validate required state ID function", func(t *testing.T) {
		config := Configure[TestCommand, TestState]()
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "StateIDFrom must be configured")
	})
	
	t.Run("should pass validation with complete config", func(t *testing.T) {
		config := Configure[TestCommand, TestState]().
			StateIDFrom(ExtractStateIDUsingShape[TestState]())
		
		err := config.Validate()
		assert.NoError(t, err)
	})
}

func TestConfig_FluentAPI(t *testing.T) {
	t.Run("should support fluent chaining", func(t *testing.T) {
		config := Configure[TestCommand, TestState]().
			CommandThatCreatesState(&CreateTestCommand{}).
			CommandThatUpdatesState(&UpdateTestCommand{}, func(cmd TestCommand) *predicate.WherePredicates {
				update := cmd.(*UpdateTestCommand)
				return predicate.MustWhere("ID = :id", predicate.ParamBinds{":id": schema.MkString(update.ID)}, nil)
			}).
			CommandThatCreatesState(&DeleteTestCommand{}).
			StateIDFrom(ExtractStateIDFromLocation[TestState]("ID"))
		
		assert.True(t, config.IsCreateCommand(&CreateTestCommand{}))
		assert.True(t, config.IsUpdateCommand(&UpdateTestCommand{}))
		assert.True(t, config.IsCreateCommand(&DeleteTestCommand{}))
		assert.NotNil(t, config.stateIDFunc)
	})
}