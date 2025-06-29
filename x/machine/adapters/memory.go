package adapters

import (
	"context"
	"fmt"
	"sync"

	"github.com/widmogrod/mkunion/x/machine"
)

// InMemoryStore provides an in-memory implementation of StateStore for testing
type InMemoryStore[S any] struct {
	mu      sync.RWMutex
	records map[string]*machine.StateRecord[S]
	
	// Optional query handler for custom query logic
	queryHandler func(query interface{}, records map[string]*machine.StateRecord[S]) []machine.StateRecord[S]
}

// NewInMemoryStore creates a new in-memory state store
func NewInMemoryStore[S any]() *InMemoryStore[S] {
	return &InMemoryStore[S]{
		records: make(map[string]*machine.StateRecord[S]),
	}
}

// WithQueryHandler sets a custom query handler
func (s *InMemoryStore[S]) WithQueryHandler(handler func(query interface{}, records map[string]*machine.StateRecord[S]) []machine.StateRecord[S]) *InMemoryStore[S] {
	s.queryHandler = handler
	return s
}

// Load retrieves a state by its ID
func (s *InMemoryStore[S]) Load(ctx context.Context, id string) (S, uint16, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	record, exists := s.records[id]
	if !exists {
		var zero S
		return zero, 0, fmt.Errorf("record not found: %s", id)
	}
	
	return record.State, record.Version, nil
}

// Save persists a state with optimistic locking
func (s *InMemoryStore[S]) Save(ctx context.Context, id string, state S, version uint16) (uint16, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	existing, exists := s.records[id]
	
	// Check version for optimistic locking
	if exists && existing.Version != version {
		return 0, fmt.Errorf("version mismatch: expected %d, got %d", existing.Version, version)
	}
	
	// Calculate new version
	newVersion := version + 1
	if !exists && version == 0 {
		newVersion = 1
	}
	
	// Save record
	s.records[id] = &machine.StateRecord[S]{
		ID:      id,
		State:   state,
		Version: newVersion,
	}
	
	return newVersion, nil
}

// Query retrieves states based on custom query logic
func (s *InMemoryStore[S]) Query(ctx context.Context, query interface{}, limit int) ([]machine.StateRecord[S], error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// If no query handler, return error
	if s.queryHandler == nil {
		return nil, fmt.Errorf("no query handler configured for query type %T", query)
	}
	
	// Use custom query handler
	results := s.queryHandler(query, s.records)
	
	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	
	return results, nil
}

// SimpleQueryMatcher creates a basic query handler that matches by a single field
func SimpleQueryMatcher[S any](fieldExtractor func(S) string) func(query interface{}, records map[string]*machine.StateRecord[S]) []machine.StateRecord[S] {
	return func(query interface{}, records map[string]*machine.StateRecord[S]) []machine.StateRecord[S] {
		// Expect query to be a string for simple matching
		searchValue, ok := query.(string)
		if !ok {
			return nil
		}
		
		var results []machine.StateRecord[S]
		for _, record := range records {
			if fieldExtractor(record.State) == searchValue {
				results = append(results, *record)
			}
		}
		
		return results
	}
}

// MapQueryMatcher creates a query handler that matches based on a map of field values
func MapQueryMatcher[S any](fieldExtractors map[string]func(S) string) func(query interface{}, records map[string]*machine.StateRecord[S]) []machine.StateRecord[S] {
	return func(query interface{}, records map[string]*machine.StateRecord[S]) []machine.StateRecord[S] {
		// Expect query to be a map[string]string
		searchCriteria, ok := query.(map[string]string)
		if !ok {
			return nil
		}
		
		var results []machine.StateRecord[S]
		for _, record := range records {
			match := true
			for field, expectedValue := range searchCriteria {
				extractor, exists := fieldExtractors[field]
				if !exists {
					match = false
					break
				}
				
				if extractor(record.State) != expectedValue {
					match = false
					break
				}
			}
			
			if match {
				results = append(results, *record)
			}
		}
		
		return results
	}
}

// Clear removes all records from the store (useful for testing)
func (s *InMemoryStore[S]) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = make(map[string]*machine.StateRecord[S])
}

// Size returns the number of records in the store
func (s *InMemoryStore[S]) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// GetAll returns all records (useful for testing and debugging)
func (s *InMemoryStore[S]) GetAll() []machine.StateRecord[S] {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	results := make([]machine.StateRecord[S], 0, len(s.records))
	for _, record := range s.records {
		results = append(results, *record)
	}
	return results
}