package state

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/machine"
	"testing"
	"time"
)

// --8<-- [start:moq-init]
func TestSuite(t *testing.T) {
	now := time.Now()
	var di Dependency = &DependencyMock{
		TimeNowFunc: func() *time.Time {
			return &now
		},
	}
	// --8<-- [end:moq-init]

	order := OrderAttr{
		Price:    100,
		Quantity: 3,
	}

	suite := machine.NewTestSuite(di, NewMachine)
	suite.Case(t, "happy path of order state transition",
		func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
			c.
				GivenCommand(&CreateOrderCMD{OrderID: "123", Attr: order}).
				ThenState(t, &OrderPending{
					Order: Order{
						ID:        "123",
						OrderAttr: order,
					},
				}).
				ForkCase(t, "start processing order", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
					c.
						GivenCommand(&MarkAsProcessingCMD{
							OrderID:  "123",
							WorkerID: "worker-1",
						}).
						ThenState(t, &OrderProcessing{
							Order: Order{
								ID:        "123",
								OrderAttr: order,
								WorkerID:  "worker-1",
							},
						}).
						ForkCase(t, "mark order as completed", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
							c.
								GivenCommand(&MarkOrderCompleteCMD{
									OrderID:  "123",
									WorkerID: "worker-2",
								}).
								ThenState(t, &OrderCompleted{
									Order: Order{
										ID:               "123",
										OrderAttr:        order,
										WorkerID:         "worker-1",
										DeliveredAt:      &now,
										StockRemovedAt:   &now,
										PaymentChargedAt: &now,
									},
								})
						}).
						ForkCase(t, "mark order cannot be by the same worker", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
							c.
								GivenCommand(&MarkOrderCompleteCMD{
									OrderID:  "123",
									WorkerID: "worker-1",
								}).
								ThenStateAndError(
									t,
									&OrderProcessing{
										Order: Order{
											ID:        "123",
											OrderAttr: order,
											WorkerID:  "worker-1",
										},
									},
									ErrWorkerSelfApprove,
								)
						}).
						ForkCase(t, "cancel order", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
							c.
								GivenCommand(&CancelOrderCMD{
									OrderID: "123",
									Reason:  "out of stock",
								}).
								ThenState(t, &OrderCancelled{
									Order: Order{
										ID:              "123",
										OrderAttr:       order,
										WorkerID:        "worker-1",
										CancelledAt:     &now,
										CancelledReason: "out of stock",
									},
								})
						}).
						ForkCase(t, "try complete order but removing products from stock fails", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
							c.
								GivenCommand(&MarkOrderCompleteCMD{
									OrderID:  "123",
									WorkerID: "worker-2",
								}).
								BeforeCommand(func(t testing.TB, di Dependency) {
									di.(*DependencyMock).ResetCalls()
									di.(*DependencyMock).WarehouseRemoveStockFunc = func(ctx context.Context, quantity int) error {
										return fmt.Errorf("warehouse api unreachable")
									}
								}).
								AfterCommand(func(t testing.TB, di Dependency) {
									dep := di.(*DependencyMock)
									dep.WarehouseRemoveStockFunc = nil
									if assert.Len(t, dep.WarehouseRemoveStockCalls(), 1) {
										assert.Equal(t, order.Quantity, dep.WarehouseRemoveStockCalls()[0].Quantity)
									}

									assert.Len(t, dep.PaymentChargeCalls(), 0)
								}).
								ThenState(t, &OrderError{
									Retried:     0,
									RetriedAt:   nil,
									ProblemCode: ProblemWarehouseAPIUnreachable,
									ProblemCommand: &MarkOrderCompleteCMD{
										OrderID:  "123",
										WorkerID: "worker-2",
									},
									ProblemState: &OrderProcessing{
										Order: Order{
											ID:        "123",
											OrderAttr: order,
											WorkerID:  "worker-1",
										},
									},
								}).
								// --8<-- [start:moq-usage]
								ForkCase(t, "successfully recover", func(t *testing.T, c *machine.Case[Dependency, Command, State]) {
									c.
										GivenCommand(&TryRecoverErrorCMD{OrderID: "123"}).
										BeforeCommand(func(t testing.TB, di Dependency) {
											di.(*DependencyMock).ResetCalls()
										}).
										AfterCommand(func(t testing.TB, di Dependency) {
											dep := di.(*DependencyMock)
											if assert.Len(t, dep.WarehouseRemoveStockCalls(), 1) {
												assert.Equal(t, order.Quantity, dep.WarehouseRemoveStockCalls()[0].Quantity)
											}
											if assert.Len(t, dep.PaymentChargeCalls(), 1) {
												assert.Equal(t, order.Price, dep.PaymentChargeCalls()[0].Price)
											}
										}).
										ThenState(t, &OrderCompleted{
											Order: Order{
												ID:               "123",
												OrderAttr:        order,
												WorkerID:         "worker-1",
												DeliveredAt:      &now,
												StockRemovedAt:   &now,
												PaymentChargedAt: &now,
											},
										})
									// --8<-- [end:moq-usage]
								})
						})
				})
		},
	)

	if suite.AssertSelfDocumentStateDiagram(t, "machine_test.go") {
		suite.SelfDocumentStateDiagram(t, "machine_test.go")
	}
}
