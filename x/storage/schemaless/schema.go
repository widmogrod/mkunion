package schemaless

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"sort"
	"sync"
)

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		store:     make(map[string]schema.Schema),
		appendLog: NewAppendLog[schema.Schema](),
	}
}

var _ Repository[schema.Schema] = &InMemoryRepository{}

type InMemoryRepository struct {
	store     map[string]schema.Schema
	appendLog *AppendLog[schema.Schema]
	mux       sync.RWMutex
}

func (s *InMemoryRepository) Get(recordID, recordType string) (Record[schema.Schema], error) {
	result, err := s.FindingRecords(FindingRecords[Record[schema.Schema]]{
		RecordType: recordType,
		Where: predicate.MustWhere("ID = :id", predicate.ParamBinds{
			":id": schema.MkString(recordID),
		}),
		Limit: 1,
	})
	if err != nil {
		return Record[schema.Schema]{}, err
	}

	if len(result.Items) == 0 {
		return Record[schema.Schema]{}, ErrNotFound
	}

	return result.Items[0], nil
}

func (s *InMemoryRepository) UpdateRecords(x UpdateRecords[Record[schema.Schema]]) error {
	if x.IsEmpty() {
		return fmt.Errorf("store.InMemoryRepository.UpdateRecords: empty command %w", ErrEmptyCommand)
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	newLog := NewAppendLog[schema.Schema]()

	for _, record := range x.Saving {
		stored, ok := s.store[record.ID+record.Type]
		if !ok {
			// new record, should have version 1
			// and since few lines below we increment version
			// we need to set it to 0
			record.Version = 0
			continue
		}

		storedVersion := schema.AsDefault[uint16](schema.Get(stored, "Version"), 0)

		if x.UpdatingPolicy == PolicyIfServerNotChanged {
			if storedVersion != record.Version {
				return fmt.Errorf("store.InMemoryRepository.UpdateRecords ID=%s Type=%s %d != %d %w",
					record.ID, record.Type, storedVersion, record.Version, ErrVersionConflict)
			}
		} else if x.UpdatingPolicy == PolicyOverwriteServerChanges {
			record.Version = storedVersion
		}
	}

	for _, record := range x.Saving {
		var err error
		var before *Record[schema.Schema]
		if _, ok := s.store[s.toKey(record)]; ok {
			before, err = schema.ToGoG[*Record[schema.Schema]](s.store[s.toKey(record)], WithOnlyRecordSchemaOptions)
			if err != nil {
				panic(fmt.Errorf("store.InMemoryRepository.UpdateRecords: to typed failed %s %w", err, ErrInternalError))
			}
		}

		record.Version += 1
		s.store[s.toKey(record)] = schema.FromGo(record)

		if before == nil {
			err = newLog.Change(Record[schema.Schema]{}, record)
		} else {
			err = newLog.Change(*before, record)
		}
		if err != nil {
			panic(fmt.Errorf("store.InMemoryRepository.UpdateRecords: append log failed (1) %s %w", err, ErrInternalError))
		}
	}

	for _, record := range x.Deleting {
		if _, ok := s.store[s.toKey(record)]; ok {
			before, err := schema.ToGoG[*Record[schema.Schema]](s.store[s.toKey(record)], WithOnlyRecordSchemaOptions)
			if err != nil {
				panic(fmt.Errorf("store.InMemoryRepository.UpdateRecords: to typed failed %s %w", err, ErrInternalError))
			}
			err = newLog.Delete(*before)
			if err != nil {
				panic(fmt.Errorf("store.InMemoryRepository.UpdateRecords: append log failed (2) %s %w", err, ErrInternalError))
			}
		}

		delete(s.store, s.toKey(record))
	}

	s.appendLog.Append(newLog)

	return nil
}

func (s *InMemoryRepository) toKey(record Record[schema.Schema]) string {
	return record.ID + record.Type
}

func (s *InMemoryRepository) FindingRecords(query FindingRecords[Record[schema.Schema]]) (PageResult[Record[schema.Schema]], error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	records := make([]schema.Schema, 0)
	for _, v := range s.store {
		records = append(records, v)
	}

	if query.RecordType != "" {
		newRecords := make([]schema.Schema, 0)
		for _, record := range records {
			if predicate.EvaluateEqual(record, "Type", query.RecordType) {
				newRecords = append(newRecords, record)
			}
		}
		records = newRecords
	}

	if query.Where != nil {
		newRecords := make([]schema.Schema, 0)
		for _, record := range records {
			if predicate.Evaluate(query.Where.Predicate, record, query.Where.Params) {
				newRecords = append(newRecords, record)
			}
		}
		records = newRecords
	}

	if len(query.Sort) > 0 {
		records = sortRecords(records, query.Sort)
	}

	if query.After != nil {
		found := false
		newRecords := make([]schema.Schema, 0)
		for _, record := range records {
			if predicate.EvaluateEqual(record, "ID", *query.After) {
				found = true
				continue // we're interested in records after this one
			}
			if found {
				newRecords = append(newRecords, record)
			}
		}
		records = newRecords
	}

	typedRecords := make([]Record[schema.Schema], 0)
	for _, record := range records {
		typed, err := schema.ToGoG[*Record[schema.Schema]](record, WithOnlyRecordSchemaOptions)
		if err != nil {
			return PageResult[Record[schema.Schema]]{}, err
		}
		typedRecords = append(typedRecords, *typed)
	}

	// Use limit to reduce number of records
	var next *FindingRecords[Record[schema.Schema]]
	if query.Limit > 0 {
		if len(typedRecords) > int(query.Limit) {
			typedRecords = typedRecords[:query.Limit]

			next = &FindingRecords[Record[schema.Schema]]{
				Where: query.Where,
				Sort:  query.Sort,
				Limit: query.Limit,
				After: &typedRecords[len(typedRecords)-1].ID,
			}
		}
	}

	result := PageResult[Record[schema.Schema]]{
		Items: typedRecords,
		Next:  next,
	}

	return result, nil
}

func (s *InMemoryRepository) AppendLog() *AppendLog[schema.Schema] {
	return s.appendLog
}

func sortRecords(records []schema.Schema, sortFields []SortField) []schema.Schema {
	sort.Slice(records, func(i, j int) bool {
		for _, sortField := range sortFields {
			fieldA := schema.Get(records[i], sortField.Field)
			fieldB := schema.Get(records[j], sortField.Field)
			cmp := schema.Compare(fieldA, fieldB)
			if sortField.Descending {
				cmp = -cmp
			}
			if cmp != 0 {
				return cmp < 0
			}
		}
		return false
	})
	return records
}
