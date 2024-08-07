package state

import "time"

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
		OrderID OrderID
	}
	// TryRecoverErrorCMD is a special command that can be used to recover from error state
	// you can have different "self-healing" rules based on the error code or even return to previous healthy state
	TryRecoverErrorCMD struct {
		OrderID OrderID
	}
)

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
	// treating error as state is a good practice in state machine, it allow you to centralise the error handling
	OrderError struct {
		// error information
		Retried   int
		RetriedAt *time.Time

		ProblemCode ProblemCode

		ProblemCommand Command
		ProblemState   State
	}
)

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
