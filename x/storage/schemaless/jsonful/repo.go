// Package jsonful is a Repository implementation that stores records in the
// plain mkunion JSON encoding instead of the schema.Schema tree.
//
// Records are encoded once, at write time, with shared.JSONMarshal. Queries
// are evaluated against the decoded JSON document with shape-driven location
// resolution, so:
//
//	Data.Age = :n                     works directly on struct fields
//	Data["testutil.Branch"].Name = :x names a union variant explicitly
//	Data.Name = :x                    expands over every variant with Name
//	Data["$type"] = :t                matches the union discriminator
//
// No reflection runs per record at query time, and a location that does not
// exist in the type is an error instead of an empty result.
package jsonful

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

func NewInMemoryRepository[T any]() (*InMemoryRepository[T], error) {
	dataShape, found := shape.LookupShapeReflectAndIndex[T]()
	if !found {
		return nil, fmt.Errorf("jsonful.NewInMemoryRepository: %w", shape.ErrShapeNotFound)
	}

	evaluator := predicate.NewJSONEvaluator(recordShape(dataShape))
	return &InMemoryRepository[T]{
		store:     make(map[string]entry[T]),
		evaluator: evaluator,
		appendLog: &AppendLog[T]{
			inner:     schemaless.NewAppendLog[T](dataShape),
			evaluator: evaluator,
		},
	}, nil
}

var _ schemaless.Repository[any] = (*InMemoryRepository[any])(nil)

type InMemoryRepository[T any] struct {
	store     map[string]entry[T]
	evaluator *predicate.JSONEvaluator
	appendLog *AppendLog[T]
	mux       sync.RWMutex
}

// entry keeps the typed record next to its decoded JSON document, built once
// at write time, so queries never marshal or reflect.
type entry[T any] struct {
	record schemaless.Record[T]
	doc    any
}

func (s *InMemoryRepository[T]) Get(recordID string, recordType schemaless.RecordType) (schemaless.Record[T], error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	e, ok := s.store[storeKey(recordID, recordType)]
	if !ok {
		return schemaless.Record[T]{}, schemaless.ErrNotFound
	}
	return e.record, nil
}

func (s *InMemoryRepository[T]) UpdateRecords(x schemaless.UpdateRecords[schemaless.Record[T]]) (*schemaless.UpdateRecordsResult[schemaless.Record[T]], error) {
	if x.IsEmpty() {
		return nil, fmt.Errorf("jsonful.InMemoryRepository.UpdateRecords: %w", schemaless.ErrEmptyCommand)
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	// Validate the whole command before mutating anything, so a version
	// conflict leaves the store untouched.
	saving := make(map[string]entry[T], len(x.Saving))
	for _, record := range x.Saving {
		key := storeKey(record.ID, record.Type)
		stored, exists := s.store[key]
		if exists {
			switch x.UpdatingPolicy {
			case schemaless.PolicyIfServerNotChanged:
				if stored.record.Version != record.Version {
					return nil, fmt.Errorf("jsonful.InMemoryRepository.UpdateRecords ID=%s Type=%s %d != %d %w",
						record.ID, record.Type, stored.record.Version, record.Version, schemaless.ErrVersionConflict)
				}
			case schemaless.PolicyOverwriteServerChanges:
				record.Version = stored.record.Version
			}
		} else if record.Version != 0 {
			// A record that does not exist yet must start at version zero;
			// accepting any number would let a client invent history.
			return nil, fmt.Errorf("jsonful.InMemoryRepository.UpdateRecords: new record ID=%s Type=%s must have Version 0, got %d %w",
				record.ID, record.Type, record.Version, schemaless.ErrVersionConflict)
		}

		record.Version += 1
		doc, err := recordDocument(record)
		if err != nil {
			return nil, fmt.Errorf("jsonful.InMemoryRepository.UpdateRecords ID=%s Type=%s: %w", record.ID, record.Type, err)
		}
		saving[key] = entry[T]{record: record, doc: doc}
	}

	result := &schemaless.UpdateRecordsResult[schemaless.Record[T]]{
		Saved:   make(map[string]schemaless.Record[T]),
		Deleted: make(map[string]schemaless.Record[T]),
	}
	newLog := schemaless.NewAppendLog[T](nil)

	for key, e := range saving {
		var before *schemaless.Record[T]
		if b, ok := s.store[key]; ok {
			before = &b.record
		}
		s.store[key] = e

		if err := newLog.Change(before, &e.record); err != nil {
			return nil, fmt.Errorf("jsonful.InMemoryRepository.UpdateRecords: append log %s %w", err, schemaless.ErrInternalError)
		}
		result.Saved[key] = e.record
	}

	for _, record := range x.Deleting {
		key := storeKey(record.ID, record.Type)
		if before, ok := s.store[key]; ok {
			if err := newLog.Delete(before.record); err != nil {
				return nil, fmt.Errorf("jsonful.InMemoryRepository.UpdateRecords: append log %s %w", err, schemaless.ErrInternalError)
			}
			delete(s.store, key)
			result.Deleted[key] = before.record
		}
	}

	s.appendLog.Append(newLog)

	return result, nil
}

func (s *InMemoryRepository[T]) FindingRecords(query schemaless.FindingRecords[schemaless.Record[T]]) (schemaless.PageResult[schemaless.Record[T]], error) {
	s.mux.RLock()
	entries := make([]entry[T], 0, len(s.store))
	for _, e := range s.store {
		if query.RecordType != "" && e.record.Type != query.RecordType {
			continue
		}
		entries = append(entries, e)
	}
	s.mux.RUnlock()

	if query.Where != nil {
		filtered := entries[:0]
		for _, e := range entries {
			ok, err := s.evaluator.Evaluate(query.Where.Predicate, e.doc, query.Where.Params)
			if err != nil {
				return schemaless.PageResult[schemaless.Record[T]]{}, fmt.Errorf("jsonful.InMemoryRepository.FindingRecords: %w", err)
			}
			if ok {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if err := s.sortEntries(entries, query.Sort); err != nil {
		return schemaless.PageResult[schemaless.Record[T]]{}, fmt.Errorf("jsonful.InMemoryRepository.FindingRecords: %w", err)
	}

	if query.After != nil {
		position := -1
		for i, e := range entries {
			if e.record.ID == *query.After {
				position = i
				break
			}
		}
		entries = entries[position+1:]
	} else if query.Before != nil {
		position := len(entries)
		for i, e := range entries {
			if e.record.ID == *query.Before {
				position = i
				break
			}
		}
		entries = entries[:position]
		if query.Limit > 0 && len(entries) > int(query.Limit) {
			entries = entries[len(entries)-int(query.Limit):]
		}
	}

	var next, prev *schemaless.FindingRecords[schemaless.Record[T]]
	hasMore := false
	if query.Limit > 0 && len(entries) > int(query.Limit) && query.Before == nil {
		entries = entries[:query.Limit]
		hasMore = true
	}

	if hasMore {
		cursor := entries[len(entries)-1].record.ID
		next = &schemaless.FindingRecords[schemaless.Record[T]]{
			RecordType: query.RecordType,
			Where:      query.Where,
			Sort:       query.Sort,
			Limit:      query.Limit,
			After:      &cursor,
		}
	}
	if (query.After != nil || query.Before != nil) && len(entries) > 0 {
		cursor := entries[0].record.ID
		prev = &schemaless.FindingRecords[schemaless.Record[T]]{
			RecordType: query.RecordType,
			Where:      query.Where,
			Sort:       query.Sort,
			Limit:      query.Limit,
			Before:     &cursor,
		}
	}

	items := make([]schemaless.Record[T], 0, len(entries))
	for _, e := range entries {
		items = append(items, e.record)
	}

	return schemaless.PageResult[schemaless.Record[T]]{
		Items: items,
		Next:  next,
		Prev:  prev,
	}, nil
}

func (s *InMemoryRepository[T]) AppendLog() *AppendLog[T] {
	return s.appendLog
}

// sortEntries orders by the requested fields, then by ID as a tiebreak, so
// result and cursor order stay deterministic even without an explicit sort.
func (s *InMemoryRepository[T]) sortEntries(entries []entry[T], fields []schemaless.SortField) error {
	type compiledSort struct {
		paths      []predicate.JSONPath
		descending bool
	}
	compiled := make([]compiledSort, 0, len(fields))
	for _, field := range fields {
		paths, err := s.evaluator.ResolvePaths(field.Field)
		if err != nil {
			return err
		}
		compiled = append(compiled, compiledSort{paths: paths, descending: field.Descending})
	}

	sort.Slice(entries, func(i, j int) bool {
		for _, c := range compiled {
			a := firstJSONValue(entries[i].doc, c.paths)
			b := firstJSONValue(entries[j].doc, c.paths)
			cmp, ok := predicate.CompareJSONValues(a, b)
			if !ok {
				continue
			}
			if c.descending {
				cmp = -cmp
			}
			if cmp != 0 {
				return cmp < 0
			}
		}
		return entries[i].record.ID < entries[j].record.ID
	})
	return nil
}

func firstJSONValue(doc any, paths []predicate.JSONPath) any {
	for _, path := range paths {
		if values := predicate.LookupJSONPath(doc, path); len(values) > 0 {
			return values[0]
		}
	}
	return nil
}

func storeKey(recordID string, recordType string) string {
	return recordID + ":" + recordType
}

// recordDocument encodes the record's data once and decodes it back into
// plain JSON values; this is the only serialization work a record ever costs.
func recordDocument[T any](record schemaless.Record[T]) (any, error) {
	raw, err := shared.JSONMarshal(record.Data)
	if err != nil {
		return nil, err
	}

	var data any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, err
		}
	}

	return map[string]any{
		"ID":      record.ID,
		"Type":    record.Type,
		"Version": float64(record.Version),
		"Data":    data,
	}, nil
}

// recordShape describes schemaless.Record[T] without needing the generic
// instantiation in the type registry.
func recordShape(dataShape shape.Shape) shape.Shape {
	return &shape.StructLike{
		Name:          "Record",
		PkgName:       "schemaless",
		PkgImportName: "github.com/widmogrod/mkunion/x/storage/schemaless",
		Fields: []*shape.FieldLike{
			{Name: "ID", Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}}},
			{Name: "Type", Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}}},
			{Name: "Data", Type: dataShape},
			{Name: "Version", Type: &shape.PrimitiveLike{Kind: &shape.NumberLike{Kind: &shape.UInt16{}}}},
		},
	}
}
