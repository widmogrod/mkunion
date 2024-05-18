package state

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/machine"
	"testing"
	"time"
)

func TestSuite(t *testing.T) {
	now := time.Now()
	var di Dependency = &DependencyMock{
		TimeNowFunc: func() *time.Time {
			return &now
		},
	}

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
									OrderID: "123",
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
									OrderID: "123",
								}).
								BeforeCommand(func(t testing.TB, di Dependency) {
									di.(*DependencyMock).ResetCalls()
									di.(*DependencyMock).WarehouseRemoveStockFunc = func(quantity int) error {
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
									Retried:        0,
									RetriedAt:      nil,
									ProblemCode:    ProblemWarehouseAPIUnreachable,
									ProblemCommand: &MarkOrderCompleteCMD{OrderID: "123"},
									ProblemState: &OrderProcessing{
										Order: Order{
											ID:        "123",
											OrderAttr: order,
											WorkerID:  "worker-1",
										},
									},
								}).
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
								})
						})
				})
		},
	)

	if suite.AssertSelfDocumentStateDiagram(t, "machine_test.go") {
		suite.SelfDocumentStateDiagram(t, "machine_test.go")
	}
}

func TestStateTransition_UsingTableTests(t *testing.T) {
	now := time.Now()
	dep := &DependencyMock{
		TimeNowFunc: func() *time.Time {
			return &now
		},
	}

	order := OrderAttr{
		Price:    100,
		Quantity: 3,
	}

	steps := []machine.Step[Dependency, Command, State]{
		{
			Name:          "create order without order ID is not allowed",
			GivenCommand:  &CreateOrderCMD{OrderID: ""},
			ExpectedState: nil,
			ExpectedErr:   ErrOrderIDRequired,
		},
		{
			Name:         "create order with valid data",
			GivenCommand: &CreateOrderCMD{OrderID: "123", Attr: order},
			ExpectedState: &OrderPending{
				Order: Order{ID: "123", OrderAttr: order},
			},
		},
		{
			Name:          "double order creation is not allowed",
			GivenCommand:  &CreateOrderCMD{OrderID: "123", Attr: order},
			ExpectedState: &OrderPending{Order: Order{ID: "123", OrderAttr: order}},
			ExpectedErr:   ErrOrderAlreadyExist,
		},
		{
			Name:          "mark order as processing without order ID must return validation error and not change state",
			GivenCommand:  &MarkAsProcessingCMD{},
			ExpectedState: &OrderPending{Order: Order{ID: "123", OrderAttr: order}},
			ExpectedErr:   ErrOrderIDRequired,
		},
		{
			Name:          "mark order as processing without worker ID must return validation error and not change state",
			GivenCommand:  &MarkAsProcessingCMD{OrderID: "123"},
			ExpectedState: &OrderPending{Order: Order{ID: "123", OrderAttr: order}},
			ExpectedErr:   ErrWorkerIDRequired,
		},
		{
			Name:          "mark order as with not matching order ID must return validation error and not change state",
			GivenCommand:  &MarkAsProcessingCMD{OrderID: "xxx", WorkerID: "worker-1"},
			ExpectedState: &OrderPending{Order: Order{ID: "123", OrderAttr: order}},
			ExpectedErr:   ErrOrderIDMismatch,
		},
		{
			Name:         "mark order as processing with valid data",
			GivenCommand: &MarkAsProcessingCMD{OrderID: "123", WorkerID: "worker-1"},
			ExpectedState: &OrderProcessing{
				Order: Order{
					ID:        "123",
					OrderAttr: order,
					WorkerID:  "worker-1",
				},
			},
		},
		{
			Name:          "complete order without order ID must return validation error and not change state",
			GivenCommand:  &MarkOrderCompleteCMD{},
			ExpectedState: &OrderProcessing{Order: Order{ID: "123", OrderAttr: order, WorkerID: "worker-1"}},
			ExpectedErr:   ErrOrderIDRequired,
		},
		{
			Name:         "complete order but removing products from stock fails",
			GivenCommand: &MarkOrderCompleteCMD{OrderID: "123"},
			BeforeCommand: func(t testing.TB, di Dependency) {
				di.(*DependencyMock).ResetCalls()
				di.(*DependencyMock).WarehouseRemoveStockFunc = func(quantity int) error {
					return fmt.Errorf("warehouse api unreachable")
				}
			},
			AfterCommand: func(t testing.TB, di Dependency) {
				dep := di.(*DependencyMock)
				dep.WarehouseRemoveStockFunc = nil
				if assert.Len(t, dep.WarehouseRemoveStockCalls(), 1) {
					assert.Equal(t, order.Quantity, dep.WarehouseRemoveStockCalls()[0].Quantity)
				}

				assert.Len(t, dep.PaymentChargeCalls(), 0)
			},
			ExpectedState: &OrderError{
				Retried:        0,
				RetriedAt:      nil,
				ProblemCode:    ProblemWarehouseAPIUnreachable,
				ProblemCommand: &MarkOrderCompleteCMD{OrderID: "123"},
				ProblemState:   &OrderProcessing{Order: Order{ID: "123", OrderAttr: order, WorkerID: "worker-1"}},
			},
		},
		{
			Name:         "attempt and fail recover error",
			GivenCommand: &TryRecoverErrorCMD{OrderID: "123"},
			BeforeCommand: func(t testing.TB, di Dependency) {
				di.(*DependencyMock).ResetCalls()
				di.(*DependencyMock).WarehouseRemoveStockFunc = func(quantity int) error {
					return fmt.Errorf("warehouse api unreachable")
				}
			},
			AfterCommand: func(t testing.TB, di Dependency) {
				dep := di.(*DependencyMock)
				dep.WarehouseRemoveStockFunc = nil
				if assert.Len(t, dep.WarehouseRemoveStockCalls(), 1) {
					assert.Equal(t, order.Quantity, dep.WarehouseRemoveStockCalls()[0].Quantity)
				}

				assert.Len(t, dep.PaymentChargeCalls(), 0)
			},
			ExpectedState: &OrderError{
				Retried:        1,
				RetriedAt:      &now,
				ProblemCode:    ProblemWarehouseAPIUnreachable,
				ProblemCommand: &MarkOrderCompleteCMD{OrderID: "123"},
				ProblemState:   &OrderProcessing{Order: Order{ID: "123", OrderAttr: order, WorkerID: "worker-1"}},
			},
		},
		{
			Name:         "successful recover from warehouse api unreachable error, and complete order",
			GivenCommand: &TryRecoverErrorCMD{OrderID: "123"},
			BeforeCommand: func(t testing.TB, di Dependency) {
				di.(*DependencyMock).ResetCalls()
			},
			AfterCommand: func(t testing.TB, di Dependency) {
				dep := di.(*DependencyMock)
				if assert.Len(t, dep.WarehouseRemoveStockCalls(), 1) {
					assert.Equal(t, order.Quantity, dep.WarehouseRemoveStockCalls()[0].Quantity)
				}
				if assert.Len(t, dep.PaymentChargeCalls(), 1) {
					assert.Equal(t, order.Price, dep.PaymentChargeCalls()[0].Price)
				}
			},
			ExpectedState: &OrderCompleted{
				Order: Order{
					ID:               "123",
					OrderAttr:        order,
					WorkerID:         "worker-1",
					DeliveredAt:      &now,
					StockRemovedAt:   &now,
					PaymentChargedAt: &now,
				},
			},
		},
	}

	AssertScenario[Dependency](t, dep, NewMachine, steps)
}

func AssertScenario[D, C, S any](
	t *testing.T,
	dep D,
	newMachine func(dep D, init S) *machine.Machine[D, C, S],
	steps []machine.Step[D, C, S],
) {
	var prev S
	for _, step := range steps {
		t.Run(step.Name, func(t *testing.T) {
			if any(step.InitState) != nil {
				prev = step.InitState
			}

			m := newMachine(dep, prev)
			if step.BeforeCommand != nil {
				step.BeforeCommand(t, m.Dep())
			}

			err := m.Handle(context.TODO(), step.GivenCommand)

			if step.AfterCommand != nil {
				step.AfterCommand(t, m.Dep())
			}

			assert.ErrorIs(t, step.ExpectedErr, err, step.Name)
			if diff := cmp.Diff(step.ExpectedState, m.State()); diff != "" {
				assert.Fail(t, "unexpected state (-want +got):\n%s", diff)
			}

			prev = m.State()

			//infer.Record(step.GivenCommand, m.State(), step.ExpectedState, err)
		})
	}
}
