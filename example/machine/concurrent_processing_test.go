package machine

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
)

func TestExampleStateStorage(t *testing.T) {
	err := ExampleStateStorage()
	assert.NoError(t, err)
}

func TestProcessOrderCommandWithConcurrency(t *testing.T) {
	store := schemaless.NewInMemoryRepository[schema.Schema]()
	ctx := context.Background()
	orderID := "order-123"

	// Create initial order
	cmd := &CreateOrderCMD{
		OrderID: orderID,
		Items: []OrderItem{
			{SKU: "WIDGET-1", Quantity: 2, Price: 29.99},
		},
	}

	err := ProcessOrderCommandWithConcurrency(ctx, store, orderID, cmd)
	assert.NoError(t, err)

	// Verify state was saved
	repo := typedful.NewTypedRepository[State](store)
	record, err := repo.Get(orderID, "orders")
	assert.NoError(t, err)
	assert.IsType(t, &OrderPending{}, record.Data)
}

// --8<-- [start:concurrent-test]
func TestConcurrentStateUpdates(t *testing.T) {
	store := schemaless.NewInMemoryRepository[schema.Schema]()
	repo := typedful.NewTypedRepository[State](store)

	// Create initial state
	orderID := "order-123"
	_, err := repo.UpdateRecords(schemaless.Save(schemaless.Record[State]{
		ID:   orderID,
		Type: "orders",
		Data: &OrderPending{
			OrderID: orderID,
			Items: []OrderItem{
				{SKU: "WIDGET-1", Quantity: 2, Price: 29.99},
			},
		},
	}))
	assert.NoError(t, err)

	// Simulate concurrent updates
	var wg sync.WaitGroup
	results := make(chan error, 2)

	// Process 1: Try to confirm order
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := ProcessOrderCommandWithConcurrency(context.Background(), store, orderID, &ConfirmOrderCMD{})
		results <- err
	}()

	// Process 2: Try to cancel order
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := ProcessOrderCommandWithConcurrency(context.Background(), store, orderID, &CancelOrderCMD{Reason: "out of stock"})
		results <- err
	}()

	wg.Wait()
	close(results)

	// One should succeed, one should either retry and fail due to invalid transition
	// or succeed if it got there first
	var successes, failures int
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	// At least one should succeed
	assert.GreaterOrEqual(t, successes, 1, "At least one operation should succeed")
	assert.LessOrEqual(t, successes, 2, "At most both operations could succeed if transitions allow")
}

// --8<-- [end:concurrent-test]

func TestProcessBulkOrdersWithConcurrency(t *testing.T) {
	store := schemaless.NewInMemoryRepository[schema.Schema]()
	repo := typedful.NewTypedRepository[State](store)
	ctx := context.Background()

	// Create multiple orders
	orderIDs := []string{"order-1", "order-2", "order-3"}
	for _, id := range orderIDs {
		_, err := repo.UpdateRecords(schemaless.Save(schemaless.Record[State]{
			ID:   id,
			Type: "orders",
			Data: &OrderPending{
				OrderID: id,
				Items: []OrderItem{
					{SKU: "WIDGET-1", Quantity: 1, Price: 19.99},
				},
			},
		}))
		assert.NoError(t, err)
	}

	// Bulk update - confirm all orders
	updates := make(map[string]Command)
	for _, id := range orderIDs {
		updates[id] = &ConfirmOrderCMD{}
	}

	err := ProcessBulkOrdersWithConcurrency(ctx, store, updates)
	assert.NoError(t, err)

	// Verify all orders are now processing
	for _, id := range orderIDs {
		record, err := repo.Get(id, "orders")
		assert.NoError(t, err)
		assert.IsType(t, &OrderProcessing{}, record.Data)
	}
}

func TestVersionConflictRetry(t *testing.T) {
	store := schemaless.NewInMemoryRepository[schema.Schema]()
	repo := typedful.NewTypedRepository[State](store)
	ctx := context.Background()
	orderID := "order-123"

	// Create initial order
	_, err := repo.UpdateRecords(schemaless.Save(schemaless.Record[State]{
		ID:   orderID,
		Type: "orders",
		Data: &OrderPending{
			OrderID: orderID,
			Items: []OrderItem{
				{SKU: "WIDGET-1", Quantity: 1, Price: 19.99},
			},
		},
	}))
	assert.NoError(t, err)

	// Test retry logic
	attempt := 0
	err = retryWithBackoff(ctx, 3, func() error {
		attempt++
		if attempt < 2 {
			return schemaless.ErrVersionConflict
		}
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, attempt)

	// Test max retries exceeded
	err = retryWithBackoff(ctx, 2, func() error {
		return schemaless.ErrVersionConflict
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max retries exceeded")

	// Test non-retryable error
	nonRetryableErr := errors.New("some other error")
	err = retryWithBackoff(ctx, 3, func() error {
		return nonRetryableErr
	})
	assert.Equal(t, nonRetryableErr, err)
}
