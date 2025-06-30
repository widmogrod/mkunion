package state_machine

import (
	"context"
	"fmt"

	"github.com/widmogrod/mkunion/x/machine"
)

func NewMachine(deps Dependencies, state State) *machine.Machine[Dependencies, Command, State] {
	if state == nil {
		state = &OrderPending{} // Default initial state
	}
	return machine.NewMachine(deps, Transition, state)
}

func Transition(ctx context.Context, deps Dependencies, cmd Command, state State) (State, error) {
	return MatchCommandR2(cmd,
		func(c *CreateOrderCMD) (State, error) {
			// Validate command
			if c.OrderID == "" {
				return nil, fmt.Errorf("order ID is required")
			}
			if len(c.Items) == 0 {
				return nil, fmt.Errorf("order must have at least one item")
			}

			// Can only create if no state exists
			if _, ok := state.(*OrderPending); ok && state.(*OrderPending).OrderID == "" {
				return &OrderPending{
					OrderID: c.OrderID,
					Items:   c.Items,
				}, nil
			}
			return nil, fmt.Errorf("cannot create order in current state")
		},
		func(c *StartProcessingCMD) (State, error) {
			switch s := state.(type) {
			case *OrderPending:
				return &OrderProcessing{
					OrderID:  s.OrderID,
					Items:    s.Items,
					WorkerID: c.WorkerID,
				}, nil
			default:
				return nil, fmt.Errorf("can only start processing from pending state")
			}
		},
		func(c *CompleteOrderCMD) (State, error) {
			switch s := state.(type) {
			case *OrderProcessing:
				return &OrderCompleted{
					OrderID:     s.OrderID,
					Items:       s.Items,
					TotalAmount: c.TotalAmount,
				}, nil
			default:
				return nil, fmt.Errorf("can only complete from processing state")
			}
		},
		func(c *CancelOrderCMD) (State, error) {
			switch s := state.(type) {
			case *OrderPending:
				return &OrderCancelled{
					OrderID: s.OrderID,
					Reason:  c.Reason,
				}, nil
			case *OrderProcessing:
				return &OrderCancelled{
					OrderID: s.OrderID,
					Reason:  c.Reason,
				}, nil
			case *OrderCompleted:
				return nil, fmt.Errorf("cannot cancel completed order")
			case *OrderCancelled:
				return nil, fmt.Errorf("order already cancelled")
			default:
				return nil, fmt.Errorf("invalid state for cancellation")
			}
		},
		func(c *ConfirmOrderCMD) (State, error) {
			switch s := state.(type) {
			case *OrderPending:
				// Simple confirmation that moves to processing
				return &OrderProcessing{
					OrderID:  s.OrderID,
					Items:    s.Items,
					WorkerID: "system",
				}, nil
			default:
				return nil, fmt.Errorf("can only confirm pending orders")
			}
		},
	)
}
