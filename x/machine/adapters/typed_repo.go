package adapters

import (
	"context"
	"fmt"

	"github.com/widmogrod/mkunion/x/machine"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
)

// TypedRepoAdapter adapts typedful.TypedRepoWithAggregator to machine.StateStore interface
type TypedRepoAdapter[S any] struct {
	repo       *typedful.TypedRepoWithAggregator[S, any]
	recordType string
}

// NewTypedRepoAdapter creates a new adapter for TypedRepoWithAggregator
func NewTypedRepoAdapter[S any](
	repo *typedful.TypedRepoWithAggregator[S, any],
	recordType string,
) *TypedRepoAdapter[S] {
	return &TypedRepoAdapter[S]{
		repo:       repo,
		recordType: recordType,
	}
}

// Load retrieves a state by its ID
func (a *TypedRepoAdapter[S]) Load(ctx context.Context, id string) (S, uint16, error) {
	record, err := a.repo.Get(id, a.recordType)
	if err != nil {
		var zero S
		return zero, 0, fmt.Errorf("failed to load state %s: %w", id, err)
	}
	return record.Data, record.Version, nil
}

// Save persists a state with optimistic locking
func (a *TypedRepoAdapter[S]) Save(ctx context.Context, id string, state S, version uint16) (uint16, error) {
	result, err := a.repo.UpdateRecords(schemaless.Save(schemaless.Record[S]{
		ID:      id,
		Type:    a.recordType,
		Data:    state,
		Version: version,
	}))
	
	if err != nil {
		return 0, fmt.Errorf("failed to save state %s: %w", id, err)
	}
	
	// Extract new version from result
	if saved, ok := result.Saved[id]; ok {
		return saved.Version, nil
	}
	
	// If not in saved, it might be a new record with version 1
	return version + 1, nil
}

// Query retrieves states based on predicate queries
func (a *TypedRepoAdapter[S]) Query(ctx context.Context, query interface{}, limit int) ([]machine.StateRecord[S], error) {
	// Type assert to predicate.WherePredicates
	where, ok := query.(*predicate.WherePredicates)
	if !ok {
		return nil, fmt.Errorf("expected *predicate.WherePredicates, got %T", query)
	}
	
	records, err := a.repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[S]]{
		RecordType: a.recordType,
		Where:      where,
		Limit:      uint8(limit),
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to query states: %w", err)
	}
	
	// Convert to StateRecord format
	result := make([]machine.StateRecord[S], len(records.Items))
	for i, record := range records.Items {
		result[i] = machine.StateRecord[S]{
			ID:      record.ID,
			State:   record.Data,
			Version: record.Version,
		}
	}
	
	return result, nil
}

// PredicateRouter helps create CommandRouter implementations that use predicate queries
type PredicateRouter[C any] struct {
	extractWhere func(C) (*predicate.WherePredicates, bool)
}

// NewPredicateRouter creates a router that extracts predicate queries from commands
func NewPredicateRouter[C any](extractWhere func(C) (*predicate.WherePredicates, bool)) *PredicateRouter[C] {
	return &PredicateRouter[C]{
		extractWhere: extractWhere,
	}
}

// ExtractQuery implements CommandRouter interface
func (r *PredicateRouter[C]) ExtractQuery(cmd C) (interface{}, bool) {
	return r.extractWhere(cmd)
}