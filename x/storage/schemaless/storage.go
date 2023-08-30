package schemaless

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
)

type RecordType = string
type Repository[T any] interface {
	Get(recordID string, recordType RecordType) (Record[T], error)
	UpdateRecords(command UpdateRecords[Record[T]]) error
	FindingRecords(query FindingRecords[Record[T]]) (PageResult[Record[T]], error)
}

var (
	ErrNotFound        = fmt.Errorf("not found")
	ErrEmptyCommand    = fmt.Errorf("empty command")
	ErrInvalidType     = fmt.Errorf("invalid type")
	ErrVersionConflict = fmt.Errorf("version conflict")
	ErrInternalError   = fmt.Errorf("internal error")
)

// Record could have two types (to think about it more):
// data records, which is current implementation
// index records, which is future implementation
//   - when two replicas have same aggregator rules, then during replication of logs, index can be reused
type Record[A any] struct {
	ID      string
	Type    string
	Data    A
	Version uint16
}

var (
	_ schema.Unmarshaler = &Record[schema.Schema]{}
	_ schema.Marshaler   = &Record[schema.Schema]{}

	WithOnlyRecordSchemaOptions = schema.WithOnlyTheseRules(
		schema.WhenPath(nil, schema.UseStruct(&Record[schema.Schema]{})),
	)
)

func (record *Record[A]) MarshalSchema() (*schema.Map, error) {
	return schema.MkMap(
		schema.MkField("ID", schema.MkString(record.ID)),
		schema.MkField("Type", schema.MkString(record.Type)),
		schema.MkField("Data", schema.FromGo(record.Data)),
		schema.MkField("Version", schema.MkInt(int(record.Version))),
	), nil
}

func (record *Record[A]) UnmarshalSchema(x *schema.Map) error {
	var ok bool
	for _, field := range x.Field {
		switch field.Name {
		case "ID":
			record.ID, ok = schema.As[string](field.Value)
			if !ok {
				return fmt.Errorf("schemaless.Record[A].UnmarshalSchema expected string(ID), got %T", field.Value)
			}

		case "Type":
			record.Type, ok = schema.As[string](field.Value)
			if !ok {
				return fmt.Errorf("schemaless.Record[A].UnmarshalSchema expected string(Type), got %T", field.Value)
			}

		case "Data":
			var err error
			record.Data, err = schema.ToGoG[A](field.Value)
			if err != nil {
				return fmt.Errorf("schemaless.Record[A].UnmarshalSchema expected A(Data), got %T; %w", field.Value, err)
			}
		case "Version":
			record.Version, ok = schema.As[uint16](field.Value)
			if !ok {
				return fmt.Errorf("schemaless.Record[A].UnmarshalSchema expected uint16(Version), got %T", field.Value)
			}

		default:
			return fmt.Errorf("schemaless.Record[A].UnmarshalSchema unexpected field %q", field.Name)
		}
	}

	return nil
}

type UpdatingPolicy uint

const (
	PolicyIfServerNotChanged UpdatingPolicy = iota
	PolicyOverwriteServerChanges
)

type (
	UpdateRecords[T any] struct {
		UpdatingPolicy UpdatingPolicy
		Saving         map[string]T
		Deleting       map[string]T
	}
)

func (s UpdateRecords[T]) IsEmpty() bool {
	return len(s.Saving) == 0 && len(s.Deleting) == 0
}

type (
	FindingRecords[T any] struct {
		RecordType string
		Where      *predicate.WherePredicates
		Sort       []SortField
		Limit      uint8
		After      *Cursor
		//Before *Cursor
	}

	SortField struct {
		Field      string
		Descending bool
	}

	Cursor = string

	PageResult[A any] struct {
		Items []A
		Next  *FindingRecords[A]
	}
)

func (a PageResult[A]) HasNext() bool {
	return a.Next != nil
}

type Storage[T any] interface {
	GetAs(id string, x *T) error
}

func Save[T any](xs ...Record[T]) UpdateRecords[Record[T]] {
	m := make(map[string]Record[T])
	for _, x := range xs {
		m[x.ID+":"+x.Type] = x
	}

	return UpdateRecords[Record[T]]{
		Saving: m,
	}
}

func Delete[T any](xs ...Record[T]) UpdateRecords[Record[T]] {
	m := make(map[string]Record[T])
	for _, x := range xs {
		m[x.ID+":"+x.Type] = x
	}

	return UpdateRecords[Record[T]]{
		Deleting: m,
	}
}

func SaveAndDelete(saving, deleting UpdateRecords[Record[schema.Schema]]) UpdateRecords[Record[schema.Schema]] {
	return UpdateRecords[Record[schema.Schema]]{
		Saving:   saving.Saving,
		Deleting: deleting.Deleting,
	}
}

func RecordAs[A any](record Record[schema.Schema]) (Record[A], error) {
	typed, err := schema.ToGoG[A](record.Data)
	if err != nil {
		return Record[A]{}, err
	}

	return Record[A]{
		ID:      record.ID,
		Type:    record.Type,
		Data:    typed,
		Version: record.Version,
	}, nil
}
