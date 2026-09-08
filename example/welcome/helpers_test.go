package welcome

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"testing/fstest"
	"time"
)

// Shared fixtures for all parts. Every test states the data it expects in
// full, so the shape of a trace, a tape or a mailbox is visible in the test.

// noon is the fixed time every fake clock returns.
var noon = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// to is the address Notify mails to; greeting is the message it sends.
const to = "ada@example.com"

const greeting = "Hello Ada, it is 12:00PM"

// notifyOps is the trace Notify("name.txt", to) leaves, in order.
var notifyOps = []MyEff{
	&ReadFile{Path: "name.txt"},
	&Now{},
	&Send{To: to, Msg: greeting},
	&Log{Msg: "sent receipt-1"},
}

// notifyTape is the tape Record writes for that run: every operation with the
// answer the world gave.
var notifyTape = []Step[MyEff]{
	{Op: &ReadFile{Path: "name.txt"}, Answer: []byte("Ada\n")},
	{Op: &Now{}, Answer: noon},
	{Op: &Send{To: to, Msg: greeting}, Answer: "receipt-1"},
	{Op: &Log{Msg: "sent receipt-1"}, Answer: Unit{}},
}

// newWorld is a small real world: one file, a fixed clock, a seeded die, and a
// mail server. Live talks to it; the tests inspect it.
func newWorld() (*Live, *bytes.Buffer, *Mailbox) {
	var out bytes.Buffer
	mail := &Mailbox{}
	bank := &Bank{Budget: 15, Quota: 2, ResetAt: noon.Add(time.Hour)}
	return &Live{
		Out:  &out,
		FS:   fstest.MapFS{"name.txt": {Data: []byte("Ada\n")}},
		Rand: rand.New(rand.NewPCG(1, 2)),
		Now:  func() time.Time { return noon },
		Mail: mail.Send,
		Bank: bank.Charge,
	}, &out, mail
}

// journal records every attempt the wrapped handler sees, with its outcome.
// It is the "what really happened" view the tests assert on.
func journal[Op any](lines *[]string) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		return func(ctx context.Context, op Op) (any, error) {
			answer, err := h(ctx, op)
			outcome := "ok"
			if err != nil {
				outcome = "err: " + err.Error()
			}
			*lines = append(*lines, fmt.Sprintf("%T %s", op, outcome))
			return answer, err
		}
	}
}

// flakyAt fails the calls listed in errs (1-based call number) before performing them.
func flakyAt[Op any](errs map[int]error) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		calls := 0
		return func(ctx context.Context, op Op) (any, error) {
			calls++
			if err, ok := errs[calls]; ok {
				return nil, err
			}
			return h(ctx, op)
		}
	}
}

func opsOf(tape []Step[MyEff]) []MyEff {
	ops := make([]MyEff, 0, len(tape))
	for _, step := range tape {
		ops = append(ops, step.Op)
	}
	return ops
}
