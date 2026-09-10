package effect

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"
)

// Every function here returns a Middleware (see eff.go): it wraps a Handler
// once and sees every operation of every program, in order, as data. With
// plain dependency injection each of these would be one wrapper per
// interface, per method.

// --8<-- [start:retry]

// RetryPolicy says how often an operation may be attempted and how long to wait.
type RetryPolicy struct {
	// Attempts is the total number of tries. 1 means: never retry.
	Attempts int
	// Backoff returns the wait before retry n (1 for the first retry). Nil means no wait.
	Backoff func(retry int) time.Duration
}

// RetryBudget caps the retries of a whole run, across all operations.
type RetryBudget struct{ Left int }

// ErrRetryBudget is returned when a retry was allowed by policy but the run's budget is gone.
var ErrRetryBudget = errors.New("effect: retry budget exhausted")

// ExponentialBackoff doubles base on every retry: base, 2*base, 4*base, ...
func ExponentialBackoff(base time.Duration) func(int) time.Duration {
	return func(retry int) time.Duration { return base << (retry - 1) }
}

// RetryWith retries according to a policy chosen per operation.
//
// policy is usually an exhaustive match over the operation union, so the
// compiler asks "may this be retried?" for every new operation. budget may be
// nil for no cap. sleep may be nil for no waiting; tests pass a fake.
func RetryWith[Op any](policy func(Op) RetryPolicy, budget *RetryBudget, sleep func(time.Duration)) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		return func(ctx context.Context, op Op) (any, error) {
			p := policy(op)
			if p.Attempts < 1 {
				p.Attempts = 1
			}
			var err error
			for attempt := 1; ; attempt++ {
				var answer any
				if answer, err = h(ctx, op); err == nil {
					return answer, nil
				}
				if attempt >= p.Attempts || ctx.Err() != nil {
					break
				}
				if budget != nil {
					if budget.Left <= 0 {
						return nil, fmt.Errorf("%w: %T: %w", ErrRetryBudget, op, err)
					}
					budget.Left--
				}
				if p.Backoff != nil && sleep != nil {
					sleep(p.Backoff(attempt))
				}
			}
			if p.Attempts == 1 {
				return nil, err // not retried: the error passes through untouched
			}
			return nil, fmt.Errorf("effect: %T failed after %d attempts: %w", op, p.Attempts, err)
		}
	}
}

// Retry is RetryWith with the same number of attempts for every operation,
// no backoff and no budget.
func Retry[Op any](attempts int) Middleware[Op] {
	return RetryWith(func(Op) RetryPolicy { return RetryPolicy{Attempts: attempts} }, nil, nil)
}

// --8<-- [end:retry]

// --8<-- [start:step-keys]

type stepKeyCtx struct{}

// StepKeys gives every operation of a run a stable key: prefix/1, prefix/2, ...
// Handlers read it with StepKey and use it as an idempotency key.
//
// Put StepKeys outermost. Then all retries of one step share a key, and on
// resume the replayed steps still count, so step 3 is "prefix/3" both times.
func StepKeys[Op any](prefix string) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		n := 0
		return func(ctx context.Context, op Op) (any, error) {
			n++
			return h(context.WithValue(ctx, stepKeyCtx{}, fmt.Sprintf("%s/%d", prefix, n)), op)
		}
	}
}

// StepKey returns the key StepKeys attached, or "" when there is none.
func StepKey(ctx context.Context) string {
	key, _ := ctx.Value(stepKeyCtx{}).(string)
	return key
}

// --8<-- [end:step-keys]

// --8<-- [start:faults]

// FailEvery makes every nth operation fail with err, before it is performed.
func FailEvery[Op any](n int, err error) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		calls := 0
		return func(ctx context.Context, op Op) (any, error) {
			calls++
			if calls%n == 0 {
				return nil, err
			}
			return h(ctx, op)
		}
	}
}

// LoseAnswerAt performs the nth operation and then reports err anyway.
// This is the nasty fault: the side effect happened, but nobody heard back.
func LoseAnswerAt[Op any](n int, err error) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		calls := 0
		return func(ctx context.Context, op Op) (any, error) {
			calls++
			answer, herr := h(ctx, op)
			if calls == n {
				return nil, err
			}
			return answer, herr
		}
	}
}

// CrashAfter performs the first n operations and then fails every later one,
// as if the process had died. Used to search every crash point of a program.
func CrashAfter[Op any](n int, err error) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		calls := 0
		return func(ctx context.Context, op Op) (any, error) {
			calls++
			if calls > n {
				return nil, err
			}
			return h(ctx, op)
		}
	}
}

// ChaosConfig drives Chaos. Rates are probabilities per operation in [0, 1].
type ChaosConfig struct {
	Seed           uint64
	FailRate       float64 // refuse before performing
	LoseAnswerRate float64 // perform, then report an error
}

// ErrChaos marks a fault injected by Chaos.
var ErrChaos = errors.New("chaos")

// Chaos injects faults at random, from a seed, so every run can be replayed.
func Chaos[Op any](cfg ChaosConfig) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		r := rand.New(rand.NewPCG(cfg.Seed, 0))
		return func(ctx context.Context, op Op) (any, error) {
			if r.Float64() < cfg.FailRate {
				return nil, fmt.Errorf("%w: refused %T", ErrChaos, op)
			}
			answer, err := h(ctx, op)
			if err == nil && r.Float64() < cfg.LoseAnswerRate {
				return nil, fmt.Errorf("%w: answer to %T lost", ErrChaos, op)
			}
			return answer, err
		}
	}
}

// --8<-- [end:faults]
