package schemaless

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"sort"
	"sync"
)

func NewInMemoryRepository[A any]() *InMemoryRepository[A] {
	shapeDef, found := shape.LookupShapeReflectAndIndex[A]()
	if !found {
		panic(fmt.Errorf("store.InMemoryRepository: shape not found %w", shape.ErrShapeNotFound))
	}

	return &InMemoryRepository[A]{
		store:     make(map[string]Record[A]),
		appendLog: NewAppendLog[A](shapeDef),
		shapeDef:  shapeDef,
	}
}

var _ Repository[any] = (*InMemoryRepository[any])(nil)

type InMemoryRepository[A any] struct {
	store     map[string]Record[A]
	appendLog *AppendLog[A]
	mux       sync.RWMutex
	shapeDef  shape.Shape
}

func (s *InMemoryRepository[A]) Get(recordID, recordType string) (Record[A], error) {
	result, err := s.FindingRecords(FindingRecords[Record[A]]{
		RecordType: recordType,
		Where: predicate.MustWhere(
			"ID = :id",
			predicate.ParamBinds{
				":id": schema.MkString(recordID),
			},
			nil,
		),
		Limit: 1,
	})
	if err != nil {
		return Record[A]{}, err
	}

	if len(result.Items) == 0 {
		return Record[A]{}, ErrNotFound
	}

	return result.Items[0], nil
}

func (s *InMemoryRepository[A]) UpdateRecords(x UpdateRecords[Record[A]]) (*UpdateRecordsResult[Record[A]], error) {
	if x.IsEmpty() {
		return nil, fmt.Errorf("store.InMemoryRepository.UpdateRecords: empty command %w", ErrEmptyCommand)
	}

	s.mux.Lock()
	defer s.mux.Unlock()

	result := &UpdateRecordsResult[Record[A]]{
		Saved:   make(map[string]Record[A]),
		Deleted: make(map[string]Record[A]),
	}
	newLog := NewAppendLog[A](s.shapeDef)

	for key, record := range x.Saving {
		stored, ok := s.store[s.toKey(record)]
		if !ok {
			// new record, should have version 1
			// and since few lines below we increment version
			// we need to set it to 0
			record.Version = 0
			continue
		}

		storedVersion := stored.Version

		if x.UpdatingPolicy == PolicyIfServerNotChanged {
			if storedVersion != record.Version {
				return nil, fmt.Errorf("store.InMemoryRepository.UpdateRecords ID=%s Type=%s %d != %d %w",
					record.ID, record.Type, storedVersion, record.Version, ErrVersionConflict)
			}
		} else if x.UpdatingPolicy == PolicyOverwriteServerChanges {
			record.Version = storedVersion
			x.Saving[key] = record
		}
	}

	for _, record := range x.Saving {
		var err error
		var before *Record[A]
		if b, ok := s.store[s.toKey(record)]; ok {
			before = &b
		}

		record.Version += 1
		s.store[s.toKey(record)] = record

		if before == nil {
			err = newLog.Change(nil, &record)
		} else {
			err = newLog.Change(before, &record)
		}
		if err != nil {
			panic(fmt.Errorf("store.InMemoryRepository.UpdateRecords: append log failed (1) %s %w", err, ErrInternalError))
		}

		result.Saved[s.toKey(record)] = record
	}

	for _, record := range x.Deleting {
		if before, ok := s.store[s.toKey(record)]; ok {
			err := newLog.Delete(before)
			if err != nil {
				panic(fmt.Errorf("store.InMemoryRepository.UpdateRecords: append log failed (2) %s %w", err, ErrInternalError))
			}
		}

		result.Deleted[s.toKey(record)] = record
		delete(s.store, s.toKey(record))
	}

	s.appendLog.Append(newLog)

	return result, nil
}

func (s *InMemoryRepository[A]) toKey(record Record[A]) string {
	return record.ID + record.Type
}

func (s *InMemoryRepository[A]) FindingRecords(query FindingRecords[Record[A]]) (PageResult[Record[A]], error) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	records := make([]Record[A], 0)
	for _, v := range s.store {
		records = append(records, v)
	}

	if query.RecordType != "" {
		newRecords := make([]Record[A], 0)
		for _, record := range records {
			if predicate.EvaluateEqual[Record[A]](record, "Type", query.RecordType) {
				newRecords = append(newRecords, record)
			}
		}
		records = newRecords
	}

	if query.Where != nil {
		newRecords := make([]Record[A], 0)
		for _, record := range records {
			if predicate.EvaluateSchema(query.Where.Predicate, schema.FromGo[Record[A]](record), query.Where.Params) {
				newRecords = append(newRecords, record)
			}
		}
		records = newRecords
	}

	if len(query.Sort) > 0 {
		records = sortRecords(records, query.Sort)
	}

	// Use limit to reduce number of records
	var next, prev *FindingRecords[Record[A]]

	// given list
	//	1,2,3,4,5,6,7
	// find limit=4
	// 	1,2,3,4					{next: 4, prev: nil}
	// find limit=4 after=4
	// 	5,6,7					{next: nil, prev: 4}
	// find limit=4 before=4
	// 	1,2,3					{next: 4, prev: nil}

	if query.After != nil {
		found := false
		newRecords := make([]Record[A], 0)
		positionID := *query.After
		for _, record := range records {
			if record.ID == positionID {
				found = true
				continue // we're interested in records after this one
			}
			if found {
				newRecords = append(newRecords, record)
			}
		}
		records = newRecords

		prev = &FindingRecords[Record[A]]{
			Where:  query.Where,
			Sort:   query.Sort,
			Limit:  query.Limit,
			Before: &positionID,
		}

		if query.Limit > 0 {
			if len(records) > int(query.Limit) {
				records = records[:query.Limit]
				next = &FindingRecords[Record[A]]{
					Where: query.Where,
					Sort:  query.Sort,
					Limit: query.Limit,
					After: &records[len(records)-1].ID,
				}
			}
		}

	} else if query.Before != nil {
		newRecords := make([]Record[A], 0)
		positionID := *query.Before
		for _, record := range records {
			newRecords = append(newRecords, record)
			if record.ID == positionID {
				break
			}
		}
		records = newRecords

		next = &FindingRecords[Record[A]]{
			Where: query.Where,
			Sort:  query.Sort,
			Limit: query.Limit,
			After: &positionID,
		}

		if query.Limit > 0 {
			if len(records) > int(query.Limit) {
				records = records[len(records)-int(query.Limit):]

				prev = &FindingRecords[Record[A]]{
					Where:  query.Where,
					Sort:   query.Sort,
					Limit:  query.Limit,
					Before: &records[0].ID,
				}
			}
		}
	} else {
		if query.Limit > 0 {
			if len(records) > int(query.Limit) {
				records = records[:query.Limit]
				next = &FindingRecords[Record[A]]{
					Where: query.Where,
					Sort:  query.Sort,
					Limit: query.Limit,
					After: &records[len(records)-1].ID,
				}
			}
		}
	}

	result := PageResult[Record[A]]{
		Items: records,
		Next:  next,
		Prev:  prev,
	}

	return result, nil
}

func (s *InMemoryRepository[A]) AppendLog() *AppendLog[A] {
	return s.appendLog
}

func sortRecords[A any](records []Record[A], sortFields []SortField) []Record[A] {
	sort.Slice(records, func(i, j int) bool {
		for _, sortField := range sortFields {
			// TODO: fix this it's inefficient, we should propage shape information
			fieldA, _, _ := schema.Get[Record[A]](records[i], sortField.Field)
			fieldB, _, _ := schema.Get[Record[A]](records[j], sortField.Field)
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
