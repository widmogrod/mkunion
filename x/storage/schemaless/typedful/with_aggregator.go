package typedful

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	. "github.com/widmogrod/mkunion/x/storage/schemaless"
	"log"
	"reflect"
)

func NewTypedRepoWithAggregator[T, C any](
	store Repository[schema.Schema],
	aggregator func() Aggregator[T, C],
) *TypedRepoWithAggregator[T, C] {
	return &TypedRepoWithAggregator[T, C]{
		store:      store,
		aggregator: aggregator,
	}
}

var _ Repository[any] = &TypedRepoWithAggregator[any, any]{}

type TypedRepoWithAggregator[T any, C any] struct {
	store      Repository[schema.Schema]
	aggregator func() Aggregator[T, C]
}

func (repo *TypedRepoWithAggregator[T, C]) shapeOf() shape.Shape {
	v := reflect.TypeOf(new(Record[T])).Elem()
	original := shape.MkRefNameFromReflect(v)

	s, found := shape.LookupShape(original)
	if !found {
		panic(fmt.Errorf("typedful.TypedRepoWithAggregator: shape.RefName not found %s", v.String()))
	}

	s = shape.IndexWith(s, original.Indexed)

	return s
}

func (repo *TypedRepoWithAggregator[T, C]) Get(recordID string, recordType RecordType) (Record[T], error) {
	v, err := repo.store.Get(recordID, recordType)
	if err != nil {
		return Record[T]{}, fmt.Errorf("store.TypedRepoWithAggregator.GetSchema store error ID=%s Type=%s. %w", recordID, recordType, err)
	}

	typed, err := RecordAs[T](v)
	if err != nil {
		return Record[T]{}, fmt.Errorf("store.TypedRepoWithAggregator.GetSchema type assertion error ID=%s Type=%s. %w", recordID, recordType, err)
	}

	return typed, nil
}

func (repo *TypedRepoWithAggregator[T, C]) UpdateRecords(s UpdateRecords[Record[T]]) error {
	schemas := UpdateRecords[Record[schema.Schema]]{
		UpdatingPolicy: s.UpdatingPolicy,
		Saving:         make(map[string]Record[schema.Schema]),
		Deleting:       make(map[string]Record[schema.Schema]),
	}

	// This is fix to in memory aggregator
	aggregate := repo.aggregator()

	for id, record := range s.Saving {
		err := aggregate.Append(record)
		if err != nil {
			return fmt.Errorf("store.TypedRepoWithAggregator.UpdateRecords aggregator.Append %w", err)
		}

		schemed := schema.FromGo(record.Data)
		schemas.Saving[id] = Record[schema.Schema]{
			ID:      record.ID,
			Type:    record.Type,
			Data:    schemed,
			Version: record.Version,
		}
	}

	// TODO: add deletion support in aggregate!
	for id, record := range s.Deleting {
		schemas.Deleting[id] = Record[schema.Schema]{
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

	err := repo.store.UpdateRecords(schemas)
	if err != nil {
		return fmt.Errorf("store.TypedRepoWithAggregator.UpdateRecords schemas store err %w", err)
	}

	return nil
}

func (repo *TypedRepoWithAggregator[T, C]) FindingRecords(query FindingRecords[Record[T]]) (PageResult[Record[T]], error) {
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
		query.Where.Predicate = repo.wrapLocationInShemaMap(query.Where.Predicate)
	}

	// do the same for sort fields
	for i, sort := range query.Sort {
		query.Sort[i].Field = repo.wrapLocation(sort.Field)
	}

	found, err := repo.store.FindingRecords(FindingRecords[Record[schema.Schema]]{
		RecordType: query.RecordType,
		Where:      query.Where,
		Sort:       query.Sort,
		Limit:      query.Limit,
		After:      query.After,
	})
	if err != nil {
		return PageResult[Record[T]]{}, fmt.Errorf("store.TypedRepoWithAggregator.FindingRecords store error %w", err)
	}

	result := PageResult[Record[T]]{
		Items: nil,
		Next:  nil,
	}

	if found.HasNext() {
		result.Next = &FindingRecords[Record[T]]{
			Where: query.Where,
			Sort:  query.Sort,
			Limit: query.Limit,
			After: found.Next.After,
		}
	}

	for _, item := range found.Items {
		typed, err := RecordAs[T](item)
		if err != nil {
			return PageResult[Record[T]]{}, fmt.Errorf("store.TypedRepoWithAggregator.FindingRecords RecordAs error id=%s %w", item.ID, err)
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

func (repo *TypedRepoWithAggregator[T, C]) wrapLocation(field string) string {
	loc, err := schema.ParseLocation(field)
	if err != nil {
		panic(fmt.Errorf("store.TypedRepoWithAggregator.wrapLocation schema.ParseLocation %w", err))
	}

	s := repo.shapeOf()

	loc = repo.wrapLocationShapeAware(loc, s)

	return schema.LocationToStr(loc)
}

func (repo *TypedRepoWithAggregator[T, C]) wrapLocationInShemaMap(p predicate.Predicate) predicate.Predicate {
	return predicate.MustMatchPredicate(
		p,
		func(x *predicate.And) predicate.Predicate {
			r := &predicate.And{}
			for _, p := range x.L {
				r.L = append(r.L, repo.wrapLocationInShemaMap(p))
			}
			return r
		},
		func(x *predicate.Or) predicate.Predicate {
			r := &predicate.Or{}
			for _, p := range x.L {
				r.L = append(r.L, repo.wrapLocationInShemaMap(p))
			}
			return r
		},
		func(x *predicate.Not) predicate.Predicate {
			r := &predicate.Not{}
			r.P = repo.wrapLocationInShemaMap(x.P)
			return r
		},
		func(x *predicate.Compare) predicate.Predicate {
			return &predicate.Compare{
				Location:  repo.wrapLocation(x.Location),
				Operation: x.Operation,
				BindValue: predicate.MustMatchBindable(
					x.BindValue,
					func(x *predicate.BindValue) predicate.Bindable {
						return x
					},
					func(x *predicate.Literal) predicate.Bindable {
						return x
					},
					func(x *predicate.Locatable) predicate.Bindable {
						return &predicate.Locatable{
							Location: repo.wrapLocation(x.Location),
						}
					},
				),
			}
		},
	)
}

func (repo *TypedRepoWithAggregator[T, C]) wrapLocationShapeAware(loc []schema.Location, s shape.Shape) []schema.Location {
	if len(loc) == 0 {
		return loc
	}

	return schema.MustMatchLocation(
		loc[0],
		func(x *schema.LocationField) []schema.Location {
			return shape.MustMatchShape(
				s,
				func(y *shape.Any) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.RefName) []schema.Location {
					s, ok := shape.LookupShape(y)
					if !ok {
						panic(fmt.Errorf("wrapLocationShapeAware: shape.RefName not found %s", y.Name))
					}

					return repo.wrapLocationShapeAware(loc, s)
				},
				func(y *shape.AliasLike) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.BooleanLike) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.StringLike) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.NumberLike) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.ListLike) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.MapLike) []schema.Location {
					panic("not implemented")
				},
				func(y *shape.StructLike) []schema.Location {
					for _, field := range y.Fields {
						if field.Name == x.Name {
							result := repo.wrapLocationShapeAware(loc[1:], field.Type)
							return append(
								append([]schema.Location{x}, shapeToSchemaName(field.Type)...),
								result...,
							)
						}
					}

					panic(fmt.Errorf("wrapLocationShapeAware: field %s not found in struct %s", x.Name, y.Name))
				},
				func(y *shape.UnionLike) []schema.Location {
					for _, variant := range y.Variant {
						if shape.ToGoTypeName(variant) == x.Name {
							result := repo.wrapLocationShapeAware(loc[1:], variant)
							return append(
								append([]schema.Location{x}, shapeToSchemaName(variant)...),
								result...,
							)
						}
					}

					panic("not implemented")
				},
			)
		},
		func(x *schema.LocationIndex) []schema.Location {
			panic("not implemented")
		},
		func(x *schema.LocationAnything) []schema.Location {
			panic("not implemented")
		},
	)
}

func shapeToSchemaName(x shape.Shape) []schema.Location {
	return shape.MustMatchShape(
		x,
		func(x *shape.Any) []schema.Location {
			panic("not implemented")
		},
		func(x *shape.RefName) []schema.Location {
			s, found := shape.LookupShape(x)
			if !found {
				panic(fmt.Errorf("shapeToSchemaName: shape.RefName not found %s", x.Name))
			}
			return shapeToSchemaName(s)
		},
		func(x *shape.AliasLike) []schema.Location {
			panic("not implemented")
		},
		func(x *shape.BooleanLike) []schema.Location {
			return []schema.Location{
				&schema.LocationField{
					Name: "schema.Boolean",
				},
			}
		},
		func(x *shape.StringLike) []schema.Location {
			return []schema.Location{
				&schema.LocationField{
					Name: "schema.String",
				},
			}
		},
		func(x *shape.NumberLike) []schema.Location {
			return []schema.Location{
				&schema.LocationField{
					Name: "schema.Number",
				},
			}
		},
		func(x *shape.ListLike) []schema.Location {
			panic("not implemented")
		},
		func(x *shape.MapLike) []schema.Location {
			panic("not implemented")
		},
		func(x *shape.StructLike) []schema.Location {
			return []schema.Location{
				&schema.LocationField{
					Name: "schema.Map",
				},
			}
		},
		func(x *shape.UnionLike) []schema.Location {
			return []schema.Location{
				&schema.LocationField{
					Name: "schema.Map",
				},
			}
		},
	)
}
