package effect

import (
	"context"
	"fmt"
)

// Middleware wraps a Handler and gets every operation of every program, in
// order, as data. One function covers all operations. With plain dependency
// injection this would be one wrapper per interface, per method.

// --8<-- [start:middleware]

// Retry performs op again on error, up to attempts times in total.
// It stops early when the context is done.
func Retry[Op any](h Handler[Op], attempts int) Handler[Op] {
	return func(ctx context.Context, op Op) (any, error) {
		var answer any
		var err error
		for i := 0; i < attempts; i++ {
			if answer, err = h(ctx, op); err == nil {
				return answer, nil
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
		}
		return nil, fmt.Errorf("effect: %T failed after %d attempts: %w", op, attempts, err)
	}
}

// Count records how many times each operation type was performed.
func Count[Op any](h Handler[Op], counts map[string]int) Handler[Op] {
	return func(ctx context.Context, op Op) (any, error) {
		counts[fmt.Sprintf("%T", op)]++
		return h(ctx, op)
	}
}

// FailEvery makes every nth operation fail with err. Fault injection in ten lines:
// deterministic, in-process, no mocks per test.
func FailEvery[Op any](h Handler[Op], n int, err error) Handler[Op] {
	calls := 0
	return func(ctx context.Context, op Op) (any, error) {
		calls++
		if calls%n == 0 {
			return nil, err
		}
		return h(ctx, op)
	}
}

// --8<-- [end:middleware]
