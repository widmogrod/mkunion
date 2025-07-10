package persistent

import (
	"context"
	"fmt"
	"reflect"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
)

// TransitionFunc is the state machine transition function signature
type TransitionFunc[D, C, S any] func(ctx context.Context, deps D, cmd C, state S) (S, error)

type PersistentMachine[D, C, S any] struct {
	repo       *typedful.TypedRepoWithAggregator[S, any]
	recordType string
	deps       D
	transition TransitionFunc[D, C, S]
	config     *Config[C, S]
}

func New[D, C, S any](
	recordType string,
	repo *typedful.TypedRepoWithAggregator[S, any],
	deps D,
	transition TransitionFunc[D, C, S],
	config *Config[C, S],
) *PersistentMachine[D, C, S] {
	// Validate required configuration
	if config == nil {
		panic("persistent.New: config is required")
	}
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("persistent.New: %v", err))
	}
	
	return &PersistentMachine[D, C, S]{
		repo:       repo,
		recordType: recordType,
		deps:       deps,
		transition: transition,
		config:     config,
	}
}

func (pm *PersistentMachine[D, C, S]) CreateOrUpdate(ctx context.Context, cmd C) (S, error) {
	var zero S
	cmdType := reflect.TypeOf(cmd)
	
	// Check if this is an update command
	if queryFunc, isUpdate := pm.config.updateCommands[cmdType]; isUpdate {
		// Query for existing state
		where := queryFunc(cmd)
		records, err := pm.repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[S]]{
			RecordType: pm.recordType,
			Where:      where,
			Limit:      1,
		})
		if err != nil {
			return zero, fmt.Errorf("failed to find records: %w", err)
		}
		
		if len(records.Items) == 0 {
			return zero, fmt.Errorf("no existing state found for update command %T", cmd)
		}
		
		record := records.Items[0]
		state := record.Data
		version := record.Version
		recordID := record.ID
		
		// Apply command to existing state using the transition function
		newState, err := pm.transition(ctx, pm.deps, cmd, state)
		if err != nil {
			return zero, fmt.Errorf("failed to handle command: %w", err)
		}
		
		// Save with incremented version
		_, err = pm.repo.UpdateRecords(schemaless.Save(schemaless.Record[S]{
			ID:      recordID,
			Type:    pm.recordType,
			Data:    newState,
			Version: version,
		}))
		
		if err != nil {
			return zero, fmt.Errorf("failed to update state: %w", err)
		}
		
		return newState, nil
		
	} else if pm.config.createCommands[cmdType] || 
	          (len(pm.config.createCommands) == 0 && len(pm.config.updateCommands) == 0) {
		// Create new state (default behavior if no commands configured)
		newState, err := pm.transition(ctx, pm.deps, cmd, zero)
		if err != nil {
			return zero, fmt.Errorf("failed to handle command: %w", err)
		}
		
		// Extract ID from state
		recordID, ok := pm.config.stateIDFunc(newState)
		if !ok || recordID == "" {
			return zero, fmt.Errorf("failed to extract ID from state %T", newState)
		}
		
		// Save with version 0
		_, err = pm.repo.UpdateRecords(schemaless.Save(schemaless.Record[S]{
			ID:      recordID,
			Type:    pm.recordType,
			Data:    newState,
			Version: 0,
		}))
		
		if err != nil {
			return zero, fmt.Errorf("failed to create state: %w", err)
		}
		
		return newState, nil
		
	} else {
		return zero, fmt.Errorf("command %T not configured as create or update", cmd)
	}
}

func (pm *PersistentMachine[D, C, S]) StateByID(id string) (S, error) {
	var zero S
	record, err := pm.repo.Get(id, pm.recordType)
	if err != nil {
		return zero, fmt.Errorf("failed to get state by ID %s: %w", id, err)
	}
	return record.Data, nil
}

// Handle provides compatibility with machine.Machine interface
func (pm *PersistentMachine[D, C, S]) Handle(ctx context.Context, cmd C) error {
	_, err := pm.CreateOrUpdate(ctx, cmd)
	return err
}

// State returns the zero value - for compatibility only
// Real state should be retrieved via StateByID or returned from CreateOrUpdate
func (pm *PersistentMachine[D, C, S]) State() S {
	var zero S
	return zero
}