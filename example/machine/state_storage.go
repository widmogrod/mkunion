package machine

import (
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
)

// --8<-- [start:example]
// ExampleStateStorage demonstrates basic state storage using typed repository
func ExampleStateStorage() error {
	// Initialize typed repository for state storage
	store := schemaless.NewInMemoryRepository[schema.Schema]()
	repo := typedful.NewTypedRepository[State](store)

	// Save state to database
	_, err := repo.UpdateRecords(schemaless.Save(schemaless.Record[State]{
		ID:   "order-123",
		Type: "orders",
		Data: &OrderPending{
			OrderID: "order-123",
			Items: []OrderItem{
				{SKU: "WIDGET-1", Quantity: 2, Price: 29.99},
			},
		},
	}))
	return err
}

// --8<-- [end:example]
