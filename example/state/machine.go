package state

import (
	"context"
	"fmt"
	"github.com/widmogrod/mkunion/x/machine"
	"time"
)

func NewMachine(di Dependency, init State) *machine.Machine[Dependency, Command, State] {
	return machine.NewMachine(di, Transition, init)
}

var (
	ErrInvalidTransition                = fmt.Errorf("invalid transition")
	ErrOrderAlreadyExist                = fmt.Errorf("cannot attemp order creation, order exists: %w", ErrInvalidTransition)
	ErrCannotCancelNonProcessingOrder   = fmt.Errorf("cannot cancel order, order must be processing to cancel it; %w", ErrInvalidTransition)
	ErrCannotCompleteNonProcessingOrder = fmt.Errorf("cannot mark order as complete, order is not being process; %w", ErrInvalidTransition)
	ErrCannotRecoverNonErrorState       = fmt.Errorf("cannot recover from non error state; %w", ErrInvalidTransition)
)

var (
	ErrValidationFailed = fmt.Errorf("validation failed")

	ErrOrderIDRequired = fmt.Errorf("order ID is required; %w", ErrValidationFailed)
	ErrOrderIDMismatch = fmt.Errorf("order ID mismatch; %w", ErrValidationFailed)

	ErrWorkerIDRequired = fmt.Errorf("worker ID required; %w", ErrValidationFailed)
)

// go:generate moq -with-resets -stub -out machine_mock.go . Dependency
type Dependency interface {
	TimeNow() *time.Time
	WarehouseRemoveStock(quantity Quantity) error
	PaymentCharge(price Price) error
}

func Transition(ctx context.Context, di Dependency, cmd Command, state State) (State, error) {
	return MatchCommandR2(
		cmd,
		func(x *CreateOrderCMD) (State, error) {
			if x.OrderID == "" {
				return nil, ErrOrderIDRequired
			}

			switch state.(type) {
			case nil:
				o := Order{
					ID:        x.OrderID,
					OrderAttr: x.Attr,
				}
				return &OrderPending{
					Order: o,
				}, nil
			}

			return nil, ErrOrderAlreadyExist
		},
		func(x *MarkAsProcessingCMD) (State, error) {
			if x.OrderID == "" {
				return nil, ErrOrderIDRequired
			}
			if x.WorkerID == "" {
				return nil, ErrWorkerIDRequired
			}

			switch s := state.(type) {
			case *OrderPending:
				if s.Order.ID != x.OrderID {
					return nil, ErrOrderIDMismatch
				}

				o := s.Order
				o.WorkerID = x.WorkerID

				return &OrderProcessing{
					Order: o,
				}, nil
			}

			return nil, ErrInvalidTransition

		},
		func(x *CancelOrderCMD) (State, error) {
			if x.OrderID == "" {
				return nil, ErrOrderIDRequired
			}

			switch s := state.(type) {
			case *OrderProcessing:
				o := s.Order
				o.CancelledAt = di.TimeNow()
				o.CancelledReason = x.Reason

				return &OrderCancelled{
					Order: o,
				}, nil
			}

			return nil, ErrCannotCancelNonProcessingOrder
		},
		func(x *MarkOrderCompleteCMD) (State, error) {
			if x.OrderID == "" {
				return nil, ErrOrderIDRequired
			}

			switch s := state.(type) {
			case *OrderProcessing:
				if s.Order.StockRemovedAt == nil {
					// we need to remove stock first
					// we can retry this operation (if warehouse is idempotent)
					// OrderID could be used to deduplicate operation
					// it's not required in this example
					err := di.WarehouseRemoveStock(s.Order.OrderAttr.Quantity)
					if err != nil {
						return &OrderError{
							ProblemCode:    ProblemWarehouseAPIUnreachable,
							ProblemCommand: x,
							ProblemState:   s,
						}, nil
					}

					s.Order.StockRemovedAt = di.TimeNow()
				}

				if s.Order.PaymentChargedAt == nil {
					// we need to charge payment first
					// we can retry this operation (if payment gateway is idempotent)
					// OrderID could be used to deduplicate operation
					// it's not required in this example
					err := di.PaymentCharge(s.Order.OrderAttr.Price)
					if err != nil {
						return &OrderError{
							ProblemCode:    ProblemPaymentAPIUnreachable,
							ProblemCommand: x,
							ProblemState:   s,
						}, nil
					}

					s.Order.PaymentChargedAt = di.TimeNow()
				}

				s.Order.DeliveredAt = di.TimeNow()

				return &OrderCompleted{
					Order: s.Order,
				}, nil
			}

			return nil, ErrCannotCompleteNonProcessingOrder
		},
		func(x *TryRecoverErrorCMD) (State, error) {
			if x.OrderID == "" {
				return nil, ErrOrderIDRequired
			}

			switch s := state.(type) {
			case *OrderError:
				s.Retried += 1
				s.RetriedAt = di.TimeNow()

				switch s.ProblemCode {
				case ProblemWarehouseAPIUnreachable,
					ProblemPaymentAPIUnreachable:
					// we can retry this operation
					newState, err := Transition(ctx, di, s.ProblemCommand, s.ProblemState)
					if err != nil {
						return s, err
					}

					// make sure that error retries are preserved
					if es, ok := newState.(*OrderError); ok {
						es.Retried = s.Retried
						es.RetriedAt = s.RetriedAt
						return es, nil
					}

					return newState, nil

				default:
					// we don't know what to do, return to previous state
					return s, nil
				}
			}

			return nil, ErrCannotRecoverNonErrorState
		},
	)
}
