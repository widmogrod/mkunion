package schemaless

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
)

//go:generate go run ../../../cmd/mkunion/main.go serde

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
//
//go:tag serde:"json"
type Record[A any] struct {
	ID      string
	Type    string
	Data    A
	Version uint16
}

//	func (r *Record[A]) MarshalJSON() ([]byte, error) {
//		result := make(map[string]json.RawMessage)
//
//		field_ID, err := json.Marshal(r.ID)
//		if err != nil {
//			return nil, err
//		}
//		result["ID"] = field_ID
//
//		field_Type, err := json.Marshal(r.Type)
//		if err != nil {
//			return nil, err
//		}
//		result["Type"] = field_Type
//
//		field_Data, err := shared.JSONMarshal[A](r.Data)
//		if err != nil {
//			return nil, err
//		}
//		result["Data"] = field_Data
//
//		field_Version, err := json.Marshal(r.Version)
//		if err != nil {
//			return nil, err
//		}
//		result["Version"] = field_Version
//
//		return json.Marshal(result)
//	}
//
//	func (r *Record[A]) UnmarshalJSON(bytes []byte) error {
//		return shared.JSONParseObject(bytes, func(key string, bytes []byte) error {
//			switch key {
//			case "ID":
//				return json.Unmarshal(bytes, &r.ID)
//			case "Type":
//				return json.Unmarshal(bytes, &r.Type)
//			case "Data":
//				return shared.JSONUnmarshal[A](bytes, &r.Data)
//			case "Version":
//				return json.Unmarshal(bytes, &r.Version)
//			}
//
//			return fmt.Errorf("schemaless.Record[A].UnmarshalJSON: unknown key: %s", key)
//		})
//	}
//
// var (
//
//	_ json.Unmarshaler = (*Record[any])(nil)
//	_ json.Marshaler   = (*Record[any])(nil)
//
// )
type UpdatingPolicy uint

var WithOnlyRecordSchemaOptions = schema.WithExtraRules()

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
	FindingRecords[T any] struct {
		RecordType string
		Where      *predicate.WherePredicates
		Sort       []SortField
		Limit      uint8
		After      *Cursor
		//Before *Cursor
	}
)

func (s UpdateRecords[T]) IsEmpty() bool {
	return len(s.Saving) == 0 && len(s.Deleting) == 0
}

type (
	SortField struct {
		Field      string
		Descending bool
	}

	Cursor = string

	//go:tag serde:"json"
	PageResult[A any] struct {
		Items []A
		Next  *FindingRecords[A]
	}
)

//func (a *PageResult[A]) MarshalJSON() ([]byte, error) {
//	result := map[string]json.RawMessage{}
//
//	var field_Items []json.RawMessage
//	for _, item := range a.Items {
//		bytes, err := shared.JSONMarshal[A](item)
//		if err != nil {
//			return nil, err
//		}
//
//		field_Items = append(field_Items, bytes)
//	}
//
//	filed_ItemsS, err := json.Marshal(field_Items)
//	if err != nil {
//		return nil, err
//	}
//
//	result["Items"] = filed_ItemsS
//
//	if a.Next != nil {
//		bytes, err := shared.JSONMarshal[*FindingRecords[A]](a.Next)
//		if err != nil {
//			return nil, err
//		}
//
//		result["Next"] = bytes
//	}
//
//	return json.Marshal(result)
//}
//
//func (a *PageResult[A]) UnmarshalJSON(bytes []byte) error {
//	return shared.JSONParseObject(bytes, func(key string, bytes []byte) error {
//		switch key {
//		case "Items":
//			var inter []json.RawMessage
//			err := json.Unmarshal(bytes, &inter)
//			if err != nil {
//				return err
//			}
//
//			var items []A
//			for _, raw := range inter {
//				var item *A = new(A)
//				err := shared.JSONUnmarshal[A](raw, item)
//				if err != nil {
//					return err
//				}
//
//				items = append(items, *item)
//			}
//
//			a.Items = items
//			return nil
//
//		case "Next":
//			var next *FindingRecords[A] = new(FindingRecords[A])
//			err := shared.JSONUnmarshal[*FindingRecords[A]](bytes, next)
//			if err != nil {
//				return err
//			}
//
//			a.Next = next
//			return nil
//		}
//
//		return fmt.Errorf("schemaless.PageResult[A].UnmarshalJSON: unknown key: %s", key)
//	})
//}
//
//var (
//	_ json.Unmarshaler = (*PageResult[any])(nil)
//	_ json.Marshaler   = (*PageResult[any])(nil)
//)

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
