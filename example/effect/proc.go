package effect

import (
	"errors"
	"fmt"
	"iter"
)

// This file adds direct style on top of the same Eff union.
//
// Proc takes a plain Go function. Inside it, DoAs asks for an operation and
// returns the answer as a normal value. No continuations, no nesting. The body
// runs as a coroutine (iter.Pull): every DoAs pauses the body, hands the
// operation to Run through an ordinary Bind, and resumes with the answer.
//
// The program is still a value. Nothing runs before Run, and handlers, Trace,
// Then and Map work unchanged, because Proc produces the same Bind chain that
// Perform and Then produce, one step at a time.

// --8<-- [start:proc]

// Env is the handle a direct-style body uses to perform operations.
type Env[Op any] struct {
	yield  func(Op) bool
	answer any
	err    error
}

// ErrStopped is the error a body sees when Run stops before the body is done.
var ErrStopped = errors.New("effect: program stopped")

// abort unwinds a body. It is a panic value, recovered inside the coroutine.
type abort struct{ err error }

// procState is shared between the coroutine and the Bind chain that drives it.
type procState[Op, A any] struct {
	env   *Env[Op]
	value A
	err   error
}

// Proc turns a direct-style body into a program.
//
// The body returns its value or an error. A DoAs on a failed operation unwinds
// the body with that error; use AttemptAs to handle the error in place.
func Proc[Op, A any](body func(e *Env[Op]) (A, error)) Eff[Op, A] {
	return &Suspend[Op, A]{Resume: func() Eff[Op, A] {
		st := &procState[Op, A]{}
		next, stop := iter.Pull(func(yield func(Op) bool) {
			st.env = &Env[Op]{yield: yield}
			defer func() {
				if r := recover(); r != nil {
					a, ok := r.(abort)
					if !ok {
						panic(r)
					}
					st.err = a.err
				}
			}()
			st.value, st.err = body(st.env)
		})
		return stepProc(st, next, stop)
	}}
}

// stepProc resumes the body until its next operation, or until it is done.
func stepProc[Op, A any](st *procState[Op, A], next func() (Op, bool), stop func()) Eff[Op, A] {
	op, ok := next()
	if !ok {
		stop()
		if st.err != nil {
			return Throw[Op, A](st.err)
		}
		return Return[Op](st.value)
	}
	return &Bind[Op, A]{
		Op: op,
		Cont: func(answer any, err error) Eff[Op, A] {
			st.env.answer, st.env.err = answer, err
			return stepProc(st, next, stop)
		},
	}
}

// AttemptAs performs op and returns the handler's answer or error.
func AttemptAs[Op, R any](e *Env[Op], op Op) (R, error) {
	var zero R
	if !e.yield(op) {
		return zero, ErrStopped
	}
	if e.err != nil {
		return zero, e.err
	}
	value, ok := e.answer.(R)
	if !ok {
		return zero, fmt.Errorf("effect: handler answered %T to %T, want %T", e.answer, op, zero)
	}
	return value, nil
}

// DoAs performs op and returns its answer. On error it unwinds the body, and
// the program fails with that error. Prefer the typed Fx.Do in program.go.
func DoAs[Op, R any](e *Env[Op], op Op) R {
	value, err := AttemptAs[Op, R](e, op)
	if err != nil {
		panic(abort{err: err})
	}
	return value
}

// --8<-- [end:proc]
