package state

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"testing"
	"time"
)

// --8<-- [start:example-store-state]
// Example_storeStateInDatabase is an example how to store state in database
func Example_storeStateInDatabase() {
	now := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	// example state
	state := &OrderCompleted{
		Order: Order{
			ID:          "123",
			OrderAttr:   OrderAttr{Price: 100, Quantity: 3},
			DeliveredAt: &now,
		},
	}

	// let's use in memory storage for storing State union
	storage := schemaless.NewInMemoryRepository[State]()

	// let's save it to storage
	_, err := storage.UpdateRecords(schemaless.Save(schemaless.Record[State]{
		ID:   state.Order.ID,
		Type: "orders",
		Data: state,
	}))

	records, err := storage.FindingRecords(schemaless.FindingRecords[schemaless.Record[State]]{
		RecordType: "orders",
	})

	fmt.Println(err)
	fmt.Printf("%+#v\n", *records.Items[0].Data.(*OrderCompleted))
	//Output: <nil>
	//state.OrderCompleted{Order:state.Order{ID:"123", OrderAttr:state.OrderAttr{Price:100, Quantity:3}, WorkerID:"", StockRemovedAt:<nil>, PaymentChargedAt:<nil>, DeliveredAt:time.Date(2021, time.January, 1, 0, 0, 0, 0, time.UTC), CancelledAt:<nil>, CancelledReason:""}}
}

// --8<-- [end:example-store-state]

func TestPersistMachine(t *testing.T) {
	orderId := "123"
	recordType := "orders"

	// let's use in memory storage for storing State union
	storage := schemaless.NewInMemoryRepository[State]()

	// before we will save our state to storage, let's check if orderId it's not there already
	records, err := storage.FindingRecords(schemaless.FindingRecords[schemaless.Record[State]]{
		RecordType: recordType,
		Where: predicate.MustWhere("ID = :id", predicate.ParamBinds{
			":id": schema.MkString(orderId),
		}, nil),
	})

	assert.NoError(t, err)
	assert.Len(t, records.Items, 0)

	// let's simulate order processing
	now := time.Now()
	dep := &DependencyMock{
		TimeNowFunc: func() *time.Time {
			return &now
		},
	}

	order := OrderAttr{
		Price:    100,
		Quantity: 3,
	}

	m := NewMachine(dep, nil)
	err = m.Handle(nil, &CreateOrderCMD{OrderID: "123", Attr: order})
	assert.NoError(t, err)

	err = m.Handle(nil, &MarkAsProcessingCMD{OrderID: "123", WorkerID: "worker-1"})
	assert.NoError(t, err)

	err = m.Handle(nil, &MarkOrderCompleteCMD{
		OrderID:  "123",
		WorkerID: "worker-2",
	})
	assert.NoError(t, err)

	state := m.State()
	assert.Equal(t, &OrderCompleted{
		Order: Order{
			ID:               "123",
			OrderAttr:        order,
			WorkerID:         "worker-1",
			DeliveredAt:      &now,
			StockRemovedAt:   &now,
			PaymentChargedAt: &now,
		},
	}, state)

	res, err := shared.JSONMarshal[State](state)
	assert.NoError(t, err)
	t.Log(string(res))

	schemed := schema.FromGo[State](state)
	t.Logf("%+v", schemed)

	// we have correct state, let's save it to storage
	_, err = storage.UpdateRecords(schemaless.Save(schemaless.Record[State]{
		ID:   "123",
		Data: state,
		Type: recordType,
	}))
	assert.NoError(t, err)

	// let's check if we can load state from storage
	records, err = storage.FindingRecords(schemaless.FindingRecords[schemaless.Record[State]]{
		RecordType: recordType,
		Where: predicate.MustWhere("ID = :id", predicate.ParamBinds{
			":id": schema.MkString(orderId),
		}, nil),
	})
	assert.NoError(t, err)
	if assert.Len(t, records.Items, 1) {
		if diff := cmp.Diff(state, records.Items[0].Data); diff != "" {
			assert.Fail(t, "unexpected state (-want +got):\n%s", diff)
		}
	}
}
