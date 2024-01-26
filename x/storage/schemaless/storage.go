package schemaless

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
)

//go:generate go run ../../../cmd/mkunion/main.go

type RecordType = string
type Repository[T any] interface {
	Get(recordID string, recordType RecordType) (Record[T], error)
	UpdateRecords(command UpdateRecords[Record[T]]) (*UpdateRecordsResult[Record[T]], error)
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
//
//go:tag serde:"json"
type Record[A any] struct {
	ID      string
	Type    string
	Data    A
	Version uint16
}

type UpdatingPolicy uint

const (
	PolicyIfServerNotChanged UpdatingPolicy = iota
	PolicyOverwriteServerChanges
)

type UpdateRecords[T any] struct {
	UpdatingPolicy UpdatingPolicy
	Saving         map[string]T
	Deleting       map[string]T
}

type UpdateRecordsResult[T any] struct {
	Saved   map[string]T
	Deleted map[string]T
}

type FindingRecords[T any] struct {
	RecordType string
	Where      *predicate.WherePredicates
	Sort       []SortField
	Limit      uint8
	After      *Cursor
	Before     *Cursor
}

func (s *UpdateRecords[T]) IsEmpty() bool {
	return len(s.Saving) == 0 && len(s.Deleting) == 0
}

//go:tag serde:"json"
type SortField struct {
	Field      string
	Descending bool
}

type Cursor = string

//go:tag serde:"json"
type PageResult[A any] struct {
	Items []A
	Next  *FindingRecords[A]
	Prev  *FindingRecords[A]
}

func (a *PageResult[A]) HasNext() bool {
	return a.Next != nil
}
func (a *PageResult[A]) HasPrev() bool {
	return a.Prev != nil
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
	//panic("not implemented")
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
