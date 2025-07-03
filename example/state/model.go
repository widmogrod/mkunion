package state

import "time"

// --8<-- [start:commands]
//
//go:tag mkunion:"Command"
type (
	CreateOrderCMD struct {
		OrderID OrderID
		Attr    OrderAttr
	}
	MarkAsProcessingCMD struct {
		OrderID  OrderID
		WorkerID WorkerID
	}
	CancelOrderCMD struct {
		OrderID OrderID
		Reason  string
	}
	MarkOrderCompleteCMD struct {
		OrderID  OrderID
		WorkerID WorkerID
	}
	// TryRecoverErrorCMD is a special command that can be used to recover from error state
	// you can have different "self-healing" rules based on the error code or even return to previous healthy state
	TryRecoverErrorCMD struct {
		OrderID OrderID
	}
)

// --8<-- [end:commands]

// --8<-- [start:states]
//
//go:tag mkunion:"State"
type (
	OrderPending struct {
		Order Order
	}
	OrderProcessing struct {
		Order Order
	}
	OrderCompleted struct {
		Order Order
	}
	OrderCancelled struct {
		Order Order
	}
	// OrderError is a special state that represent an error
	// during order processing, you can have different "self-healing jobs" based on the error code
	// like retrying the order, cancel the order, etc.
	//
	// This pattern enables:
	// 1. Perfect reproduction of the failure
	// 2. Automatic retry with the same command
	// 3. Debugging with full context
	// 4. Recovery to previous valid state
	OrderError struct {
		// error information
		Retried   int
		RetriedAt *time.Time

		ProblemCode ProblemCode

		ProblemCommand Command
		ProblemState   State
	}
)

// --8<-- [end:states]

// --8<-- [start:value-objects]
type (
	// OrderID Price, Quantity are placeholders for value objects, to ensure better data semantic and type safety
	OrderID  = string
	Price    = float64
	Quantity = int

	OrderAttr struct {
		// placeholder for order attributes
		// like customer name, address, etc.
		// like product name, price, etc.
		// for simplicity we only have Price and Quantity
		Price    Price
		Quantity Quantity
	}

	// WorkerID represent human that process the order
	WorkerID = string

	// Order everything we know about order
	Order struct {
		ID               OrderID
		OrderAttr        OrderAttr
		WorkerID         WorkerID
		StockRemovedAt   *time.Time
		PaymentChargedAt *time.Time
		DeliveredAt      *time.Time
		CancelledAt      *time.Time
		CancelledReason  string
	}
)

type ProblemCode int

const (
	ProblemWarehouseAPIUnreachable ProblemCode = iota
	ProblemPaymentAPIUnreachable
)

// --8<-- [end:value-objects]
