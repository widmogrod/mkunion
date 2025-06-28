package machine

//go:tag mkunion:"State,no-type-registry"
type (
	OrderPending struct {
		OrderID string
		Items   []OrderItem
	}
	OrderProcessing struct {
		OrderID  string
		Items    []OrderItem
		WorkerID string
	}
	OrderCompleted struct {
		OrderID     string
		Items       []OrderItem
		TotalAmount float64
	}
	OrderCancelled struct {
		OrderID string
		Reason  string
	}
)

//go:tag mkunion:"Command,no-type-registry"
type (
	CreateOrderCMD struct {
		OrderID string
		Items   []OrderItem
	}
	StartProcessingCMD struct {
		WorkerID string
	}
	CompleteOrderCMD struct {
		TotalAmount float64
	}
	CancelOrderCMD struct {
		Reason string
	}
	ConfirmOrderCMD struct{}
)

type OrderItem struct {
	SKU      string
	Quantity int
	Price    float64
}

type Dependencies struct {
	// Add dependencies like database, logger, etc. as needed
}
