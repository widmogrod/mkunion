package schemaless

import (
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
)

func NewKeyedAggregate[T, R any](
	recordTypeName string,
	supportedRecordTypes []string,
	groupByFunc func(data T) (string, R),
	combineByFunc func(a, b R) (R, error),
	storage Repository[schema.Schema],
) *KayedAggregate[T, R] {
	return &KayedAggregate[T, R]{
		aggregateRecordTypeName: recordTypeName,
		supportedRecordTypes:    supportedRecordTypes,

		dataByKey:    make(map[string]Record[R]),
		groupByKey:   groupByFunc,
		combineByKey: combineByFunc,
		storage:      storage,
	}
}

var _ Aggregator[any, any] = (*KayedAggregate[any, any])(nil)

type KayedAggregate[T, R any] struct {
	supportedRecordTypes    []string
	aggregateRecordTypeName string

	groupByKey     func(data T) (string, R)
	combineByKey   func(a, b R) (R, error)
	unCombineByKey func(from, b R) (R, error)

	dataByKey map[string]Record[R]

	storage Repository[schema.Schema]
}

func (t *KayedAggregate[T, R]) Append(data Record[T]) error {
	if !t.supportedType(data.Type) {
		return nil
	}

	index, result := t.groupByKey(data.Data)
	if _, ok := t.dataByKey[index]; !ok {
		initial, err := t.loadIndex(index)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return err
			}

			t.dataByKey[index] = Record[R]{
				ID:      index,
				Type:    t.aggregateRecordTypeName,
				Data:    result,
				Version: 0,
			}
			return nil
		}

		t.dataByKey[index] = initial
	}

	result, err := t.combineByKey(t.dataByKey[index].Data, result)
	if err != nil {
		return err
	}

	existing := t.dataByKey[index]
	existing.Data = result
	t.dataByKey[index] = existing

	return nil
}

func (t *KayedAggregate[T, R]) Delete(data Record[T]) error {
	// is supported type?
	if !t.supportedType(data.Type) {
		return nil
	}

	index, result := t.groupByKey(data.Data)
	if _, ok := t.dataByKey[index]; !ok {
		initial, err := t.loadIndex(index)
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return err
			}

			t.dataByKey[index] = Record[R]{
				ID:      index,
				Type:    t.aggregateRecordTypeName,
				Data:    result,
				Version: 0,
			}
			return nil
		}

		t.dataByKey[index] = initial
	}

	result, err := t.unCombineByKey(t.dataByKey[index].Data, result)
	if err != nil {
		return err
	}

	existing := t.dataByKey[index]
	existing.Data = result
	t.dataByKey[index] = existing

	return nil
}

func (t *KayedAggregate[T, R]) GetVersionedIndices() map[string]Record[schema.Schema] {
	var result = make(map[string]Record[schema.Schema])
	for k, v := range t.dataByKey {
		schemed := schema.FromGo(v.Data)
		result[k] = Record[schema.Schema]{
			ID:      v.ID,
			Type:    t.aggregateRecordTypeName,
			Data:    schemed,
			Version: v.Version,
		}
	}

	return result
}

func (t *KayedAggregate[T, R]) GetIndexByKey(key string) R {
	return t.dataByKey[key].Data
}

func (t *KayedAggregate[T, R]) loadIndex(index string) (Record[R], error) {
	var r Record[R]
	// load index state from storage
	// if index is found, then concat with unversionedData
	// otherwise just use unversionedData.
	initial, err := t.storage.Get(index, t.aggregateRecordTypeName)
	if err != nil {
		return r, fmt.Errorf("store.RepositoryWithAggregator.UpdateRecords index(1)=%s %w", index, err)
	}

	indexValue, err := RecordAs[R](initial)
	if err != nil {
		return r, fmt.Errorf("store.RepositoryWithAggregator.UpdateRecords index(2)=%s %w", index, err)
	}

	return indexValue, nil
}

func (t *KayedAggregate[T, R]) supportedType(recordType string) bool {
	for _, v := range t.supportedRecordTypes {
		if v == recordType {
			return true
		}
	}

	return false
}
