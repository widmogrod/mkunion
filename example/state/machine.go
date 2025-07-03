package state

// --8<-- [start:imports]
import (
	"context"
	"fmt"
	"github.com/widmogrod/mkunion/x/machine"
	"time"
)

// --8<-- [end:imports]

// --8<-- [start:new-machine]
func NewMachine(di Dependency, init State) *machine.Machine[Dependency, Command, State] {
	return machine.NewMachine(di, Transition, init)
}

// --8<-- [end:new-machine]

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

	ErrWorkerSelfApprove = fmt.Errorf("cannot self approve order; %w", ErrValidationFailed)
)

// --8<-- [start:dependency]

//go:generate moq -with-resets -stub -out machine_mock.go . Dependency
type Dependency interface {
	TimeNow() *time.Time
	WarehouseRemoveStock(ctx context.Context, quantity Quantity) error
	PaymentCharge(ctx context.Context, price Price) error
}

// --8<-- [end:dependency]

// --8<-- [start:transition]
// --8<-- [start:transition-fragment]

func Transition(ctx context.Context, di Dependency, cmd Command, state State) (State, error) {
	return MatchCommandR2(
		cmd,
		// --8<-- [start:create-order]
		func(x *CreateOrderCMD) (State, error) {
			// 1. Structural validation as simple checks and explicit error type
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
		// --8<-- [end:transition-fragment]
		// --8<-- [end:create-order]
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
		// --8<-- [end:transition-partial]
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
		// --8<-- [start:advanced-handling]
		func(x *MarkOrderCompleteCMD) (State, error) {
			//  1. Structural validation of commands (you could use go-validate library):
			//
			//     if err := di.Validator().Struct(x); err != nil {
			//        return nil, fmt.Errorf("validation failed: %w. %s", err, ErrValidationFailed)
			//     }
			//
			//    or do it manually like in this example:
			if x.OrderID == "" {
				return nil, ErrOrderIDRequired
			}
			if x.WorkerID == "" {
				return nil, ErrWorkerIDRequired
			}

			// 2. Ensure valid transitions
			s, ok := state.(*OrderProcessing)
			if !ok {
				return nil, ErrCannotCompleteNonProcessingOrder
			}

			// 3. Business rule validation:
			//    Worker cannot approve it's own order
			if s.Order.WorkerID == x.WorkerID {
				return nil, ErrWorkerSelfApprove
			}

			// 4. External validation or mutations:
			if s.Order.StockRemovedAt == nil {
				// We need to remove stock first
				// We can retry this operation (assuming warehouse is idempotent, see TryRecoverErrorCMD)
				// OrderID could be used to deduplicate operation
				// it's not required in this example
				err := di.WarehouseRemoveStock(ctx, s.Order.OrderAttr.Quantity)
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
				// We need to charge payment first
				// We can retry this operation (assuming payment gateway is idempotent, see TryRecoverErrorCMD))
				// OrderID could be used to deduplicate operation
				// it's not required in this example
				err := di.PaymentCharge(ctx, s.Order.OrderAttr.Price)
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
		},
		// --8<-- [end:advanced-handling]
		// --8<-- [start:error-recovery]
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
		// --8<-- [end:error-recovery]
	)
}

// --8<-- [end:transition]
