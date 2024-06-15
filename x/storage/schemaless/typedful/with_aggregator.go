package typedful

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

func NewTypedRepoWithAggregator[T, C any](
	store schemaless.Repository[schema.Schema],
	aggregator func() schemaless.Aggregator[T, C],
) *TypedRepoWithAggregator[T, C] {
	location, err := schema.NewTypedLocation[schemaless.Record[T]]()
	if err != nil {
		panic(fmt.Errorf("typedful.NewTypedRepoWithAggregator: %w", err))
	}

	return &TypedRepoWithAggregator[T, C]{
		loc:        location,
		store:      store,
		aggregator: aggregator,
	}
}

var _ schemaless.Repository[any] = &TypedRepoWithAggregator[any, any]{}

type TypedRepoWithAggregator[T any, C any] struct {
	loc        *schema.TypedLocation
	store      schemaless.Repository[schema.Schema]
	aggregator func() schemaless.Aggregator[T, C]
}

func (repo *TypedRepoWithAggregator[T, C]) Get(recordID string, recordType schemaless.RecordType) (schemaless.Record[T], error) {
	v, err := repo.store.Get(recordID, recordType)
	if err != nil {
		return schemaless.Record[T]{}, fmt.Errorf("store.TypedRepoWithAggregator.GetSchema store error ID=%s Type=%s. %w", recordID, recordType, err)
	}

	typed, err := schemaless.RecordAs[T](v)
	if err != nil {
		return schemaless.Record[T]{}, fmt.Errorf("store.TypedRepoWithAggregator.GetSchema type assertion error ID=%s Type=%s. %w", recordID, recordType, err)
	}

	return typed, nil
}

func (repo *TypedRepoWithAggregator[T, C]) UpdateRecords(s schemaless.UpdateRecords[schemaless.Record[T]]) (*schemaless.UpdateRecordsResult[schemaless.Record[T]], error) {
	schemas := schemaless.UpdateRecords[schemaless.Record[schema.Schema]]{
		UpdatingPolicy: s.UpdatingPolicy,
		Saving:         make(map[string]schemaless.Record[schema.Schema]),
		Deleting:       make(map[string]schemaless.Record[schema.Schema]),
	}

	// This is fix to in memory aggregator
	aggregate := repo.aggregator()

	for id, record := range s.Saving {
		err := aggregate.Append(record)
		if err != nil {
			return nil, fmt.Errorf("store.TypedRepoWithAggregator.UpdateRecords aggregator.Append %w", err)
		}

		schemed := schema.FromGo(record.Data)
		schemas.Saving[id] = schemaless.Record[schema.Schema]{
			ID:      record.ID,
			Type:    record.Type,
			Data:    schemed,
			Version: record.Version,
		}
	}

	// TODO: add deletion support in aggregate!
	for id, record := range s.Deleting {
		schemas.Deleting[id] = schemaless.Record[schema.Schema]{
			ID:      record.ID,
			Type:    record.Type,
			Data:    schema.FromGo(record.Data),
			Version: record.Version,
		}
	}

	for index, versionedData := range aggregate.GetVersionedIndices() {
		log.Printf("index %s %#v\n", index, versionedData)
		schemas.Saving["indices:"+versionedData.ID+":"+versionedData.Type] = versionedData
	}

	updated, err := repo.store.UpdateRecords(schemas)
	if err != nil {
		return nil, fmt.Errorf("store.TypedRepoWithAggregator.UpdateRecords schemas store err %w", err)
	}

	result := &schemaless.UpdateRecordsResult[schemaless.Record[T]]{
		Saved:   make(map[string]schemaless.Record[T]),
		Deleted: make(map[string]schemaless.Record[T]),
	}

	for id, record := range updated.Saved {
		typed, err := schemaless.RecordAs[T](record)
		if err != nil {
			return nil, fmt.Errorf("store.TypedRepoWithAggregator.UpdateRecords RecordAs error id=%s %w", id, err)
		}

		result.Saved[id] = typed
	}

	for id, record := range updated.Deleted {
		typed, err := schemaless.RecordAs[T](record)
		if err != nil {
			return nil, fmt.Errorf("store.TypedRepoWithAggregator.UpdateRecords RecordAs error id=%s %w", id, err)
		}

		result.Deleted[id] = typed
	}

	return result, nil
}

func (repo *TypedRepoWithAggregator[T, C]) FindingRecords(query schemaless.FindingRecords[schemaless.Record[T]]) (schemaless.PageResult[schemaless.Record[T]], error) {
	// Typed version of FindingRecords should work with different form of where and sort fields
	// Typed version suggest that data stored in storage is typed,
	// but in fact it stored as schema.Schema
	// For example Record[User]
	// should be accessed as
	//		Data.Name, Data.Age
	// wheere internal representation Record[schema.Schen] is access it
	//		Data["schema.Map"].Name, Data["schema.Map"].Age
	// This means, that we need add between data path and

	if query.Where != nil {
		query.Where.Predicate = repo.wrapPredicate(query.Where.Predicate)
	}

	// do the same for sort fields
	for i, sort := range query.Sort {
		var err error
		query.Sort[i].Field, err = repo.loc.WrapLocationStr(sort.Field)
		if err != nil {
			return schemaless.PageResult[schemaless.Record[T]]{}, fmt.Errorf("store.TypedRepoWithAggregator.FindingRecords wrapLocation in sort; %w", err)
		}
	}

	found, err := repo.store.FindingRecords(schemaless.FindingRecords[schemaless.Record[schema.Schema]]{
		RecordType: query.RecordType,
		Where:      query.Where,
		Sort:       query.Sort,
		Limit:      query.Limit,
		After:      query.After,
		Before:     query.Before,
	})
	if err != nil {
		return schemaless.PageResult[schemaless.Record[T]]{}, fmt.Errorf("store.TypedRepoWithAggregator.FindingRecords store error %w", err)
	}

	result := schemaless.PageResult[schemaless.Record[T]]{
		Items: nil,
		Next:  nil,
	}

	if found.HasNext() {
		result.Next = &schemaless.FindingRecords[schemaless.Record[T]]{
			RecordType: query.RecordType,
			Where:      query.Where,
			Sort:       query.Sort,
			Limit:      query.Limit,
			After:      found.Next.After,
			Before:     nil,
		}
	}

	if found.HasPrev() {
		result.Prev = &schemaless.FindingRecords[schemaless.Record[T]]{
			RecordType: query.RecordType,
			Where:      query.Where,
			Sort:       query.Sort,
			Limit:      query.Limit,
			After:      nil,
			Before:     found.Prev.Before,
		}
	}

	for _, item := range found.Items {
		typed, err := schemaless.RecordAs[T](item)
		if err != nil {
			return schemaless.PageResult[schemaless.Record[T]]{}, fmt.Errorf("store.TypedRepoWithAggregator.FindingRecords RecordAs error id=%s %w", item.ID, err)
		}

		result.Items = append(result.Items, typed)
	}

	return result, nil
}

// ReindexAll is used to reindex all records with a provided aggregator definition
// Example: when aggregator is created, it's empty, so it needs to be filled with all records
// Example: when aggregator definition is changed, it needs to be reindexed
// Example: when aggregator is corrupted, it needs to be reindexed
//
// How it works?
// 1. It's called by the user
// 2. It's called by the system when it detects that aggregator is corrupted
// 3. It's called by the system when it detects that aggregator definition is changed
//
// How it's implemented?
//  1. Create index from snapshot of all records. Because it's snapshot, changes are not applied.
//  2. In parallel process stream of changes from give point of time.
//  3. KayedAggregate must be idempotent, so same won't be indexed twice.
//  4. When aggregator detects same record with new Version, it retracts old Version and accumulates new Version.
//  5. When it's done, it's ready to be used
//  6. When indices are set up as synchronous, then every change is indexed immediately.
//     But, because synchronous index is from point of time, it needs to trigger reindex.
//     Which imply that aggregator myst know when index was created, so it can know when to stop rebuilding process.
//     This implies control plane. Versions of records should follow monotonically increasing order, that way it will be easier to detect when index is up to date.
func (repo *TypedRepoWithAggregator[T, C]) ReindexAll() {
	panic("not implemented")
}

func (repo *TypedRepoWithAggregator[T, C]) wrapPredicate(p predicate.Predicate) predicate.Predicate {
	return WrapPredicate(p, repo.loc)
}

func WrapPredicate(p predicate.Predicate, loc *schema.TypedLocation) predicate.Predicate {
	return predicate.MatchPredicateR1(
		p,
		func(x *predicate.And) predicate.Predicate {
			r := &predicate.And{}
			for _, p := range x.L {
				r.L = append(r.L, WrapPredicate(p, loc))
			}
			return r
		},
		func(x *predicate.Or) predicate.Predicate {
			r := &predicate.Or{}
			for _, p := range x.L {
				r.L = append(r.L, WrapPredicate(p, loc))
			}
			return r
		},
		func(x *predicate.Not) predicate.Predicate {
			r := &predicate.Not{}
			r.P = WrapPredicate(x.P, loc)
			return r
		},
		func(x *predicate.Compare) predicate.Predicate {
			locw, err := loc.WrapLocationStr(x.Location)
			if err != nil {
				panic(fmt.Errorf("wrapPredicate: %w", err))
			}

			return &predicate.Compare{
				Location:  locw,
				Operation: x.Operation,
				BindValue: predicate.MatchBindableR1(
					x.BindValue,
					func(x *predicate.BindValue) predicate.Bindable {
						return x
					},
					func(x *predicate.Literal) predicate.Bindable {
						return x
					},
					func(x *predicate.Locatable) predicate.Bindable {
						locw, err := loc.WrapLocationStr(x.Location)
						if err != nil {
							panic(fmt.Errorf("wrapPredicate: %w", err))
						}

						return &predicate.Locatable{
							Location: locw,
						}
					},
				),
			}
		},
	)
}
