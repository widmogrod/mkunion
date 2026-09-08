package effect

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

// A recording is the list of operations a run performed, with the answers.
// Because operations and answers are data, a recording can be saved, replayed
// without the real world, or used to resume a program that stopped half way.

// --8<-- [start:recording]

// Step is one performed operation and what the handler answered.
type Step[Op any] struct {
	Op     Op
	Answer any
	Err    error
}

// Record appends every step a handler performs to tape.
func Record[Op any](tape *[]Step[Op]) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		return func(ctx context.Context, op Op) (any, error) {
			answer, err := h(ctx, op)
			*tape = append(*tape, Step[Op]{Op: op, Answer: answer, Err: err})
			return answer, err
		}
	}
}

// ErrTapeEnded is returned by Replay when the tape is exhausted and there is no rest handler.
var ErrTapeEnded = errors.New("effect: replay tape ended")

// Replay answers from tape, in order, and checks that the program asks for the
// same operations it asked for when the tape was recorded. After the tape ends,
// operations go to rest. A nil rest makes the end of the tape an error.
//
// Replay with rest == nil is a golden test with no clock, files or network.
// Replay with rest == live resumes a program that stopped after the last step,
// which is how durable workflow engines survive a crash.
func Replay[Op any](tape []Step[Op], rest Handler[Op]) Handler[Op] {
	pos := 0
	return func(ctx context.Context, op Op) (any, error) {
		if pos >= len(tape) {
			if rest == nil {
				return nil, fmt.Errorf("%w: unexpected %T", ErrTapeEnded, op)
			}
			return rest(ctx, op)
		}
		step := tape[pos]
		pos++
		if !reflect.DeepEqual(step.Op, op) {
			return nil, fmt.Errorf("effect: replay mismatch at step %d: recorded %#v, program asked %#v", pos, step.Op, op)
		}
		return step.Answer, step.Err
	}
}

// --8<-- [end:recording]
