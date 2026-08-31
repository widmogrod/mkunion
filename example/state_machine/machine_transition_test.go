package state_machine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testItems() []OrderItem {
	return []OrderItem{{SKU: "tea", Quantity: 1, Price: 2.5}}
}

func apply(t *testing.T, state State, cmd Command) (State, error) {
	t.Helper()
	return Transition(context.Background(), Dependencies{}, cmd, state)
}

func pendingOrder() *OrderPending {
	return &OrderPending{OrderID: "o-1", Items: testItems()}
}

func processingOrder() *OrderProcessing {
	return &OrderProcessing{OrderID: "o-1", Items: testItems(), WorkerID: "w-1"}
}

func TestTransitionCreateOrder(t *testing.T) {
	t.Run("creates from the blank initial state", func(t *testing.T) {
		got, err := apply(t, &OrderPending{}, &CreateOrderCMD{OrderID: "o-1", Items: testItems()})
		require.NoError(t, err)
		assert.Equal(t, pendingOrder(), got)
	})

	t.Run("missing order id is rejected", func(t *testing.T) {
		_, err := apply(t, &OrderPending{}, &CreateOrderCMD{Items: testItems()})
		assert.ErrorContains(t, err, "order ID is required")
	})

	t.Run("empty items are rejected", func(t *testing.T) {
		_, err := apply(t, &OrderPending{}, &CreateOrderCMD{OrderID: "o-1"})
		assert.ErrorContains(t, err, "at least one item")
	})

	t.Run("cannot create twice", func(t *testing.T) {
		_, err := apply(t, pendingOrder(), &CreateOrderCMD{OrderID: "o-2", Items: testItems()})
		assert.ErrorContains(t, err, "cannot create order")
	})

	t.Run("cannot create over a processing order", func(t *testing.T) {
		_, err := apply(t, processingOrder(), &CreateOrderCMD{OrderID: "o-2", Items: testItems()})
		assert.ErrorContains(t, err, "cannot create order")
	})
}

func TestTransitionProcessingAndCompletion(t *testing.T) {
	t.Run("pending starts processing", func(t *testing.T) {
		got, err := apply(t, pendingOrder(), &StartProcessingCMD{WorkerID: "w-1"})
		require.NoError(t, err)
		assert.Equal(t, processingOrder(), got)
	})

	t.Run("only pending can start processing", func(t *testing.T) {
		_, err := apply(t, processingOrder(), &StartProcessingCMD{WorkerID: "w-2"})
		assert.ErrorContains(t, err, "only start processing from pending")
	})

	t.Run("processing completes with a total", func(t *testing.T) {
		got, err := apply(t, processingOrder(), &CompleteOrderCMD{TotalAmount: 42})
		require.NoError(t, err)
		assert.Equal(t, &OrderCompleted{
			OrderID: "o-1", Items: testItems(), TotalAmount: 42,
		}, got)
	})

	t.Run("only processing can complete", func(t *testing.T) {
		_, err := apply(t, pendingOrder(), &CompleteOrderCMD{TotalAmount: 42})
		assert.ErrorContains(t, err, "only complete from processing")
	})
}

func TestTransitionCancellation(t *testing.T) {
	t.Run("pending cancels", func(t *testing.T) {
		got, err := apply(t, pendingOrder(), &CancelOrderCMD{Reason: "changed my mind"})
		require.NoError(t, err)
		assert.Equal(t, &OrderCancelled{OrderID: "o-1", Reason: "changed my mind"}, got)
	})

	t.Run("processing cancels", func(t *testing.T) {
		got, err := apply(t, processingOrder(), &CancelOrderCMD{Reason: "out of stock"})
		require.NoError(t, err)
		assert.Equal(t, &OrderCancelled{OrderID: "o-1", Reason: "out of stock"}, got)
	})

	t.Run("completed orders cannot cancel", func(t *testing.T) {
		_, err := apply(t, &OrderCompleted{OrderID: "o-1"}, &CancelOrderCMD{Reason: "too late"})
		assert.ErrorContains(t, err, "cannot cancel completed")
	})

	t.Run("cancelling twice is rejected", func(t *testing.T) {
		_, err := apply(t, &OrderCancelled{OrderID: "o-1"}, &CancelOrderCMD{Reason: "again"})
		assert.ErrorContains(t, err, "already cancelled")
	})
}

func TestNewMachineDefaults(t *testing.T) {
	m := NewMachine(Dependencies{}, nil)
	assert.Equal(t, &OrderPending{}, m.State(), "nil state falls back to the initial state")

	require.NoError(t, m.Handle(context.Background(), &CreateOrderCMD{OrderID: "o-1", Items: testItems()}))
	assert.Equal(t, pendingOrder(), m.State())
}
