package welcome

import (
	"context"
	"fmt"
	"github.com/widmogrod/mkunion/x/effect"
	"io"
	"io/fs"
	"math/rand/v2"
	"time"

	"github.com/widmogrod/mkunion/f"
)

// --8<-- [start:live]

// Live performs operations against the real world.
type Live struct {
	Out  io.Writer
	FS   fs.FS
	Rand *rand.Rand
	Now  func() time.Time
	Mail func(key, to, msg string) (string, error)
	Bank func(amount int) (f.Result[Receipt, ChargeError], error)
}

var _ MyEffHandler = (*Live)(nil)

func (l *Live) HandleLog(_ context.Context, op *Log) (Unit, error) {
	_, err := fmt.Fprintln(l.Out, op.Msg)
	return Unit{}, err
}

func (l *Live) HandleNow(context.Context, *Now) (time.Time, error) {
	return l.Now(), nil
}

func (l *Live) HandleReadFile(_ context.Context, op *ReadFile) ([]byte, error) {
	return fs.ReadFile(l.FS, op.Path)
}

func (l *Live) HandleRandom(_ context.Context, op *Random) (int, error) {
	return l.Rand.IntN(op.Max), nil
}

// HandleSend passes the step key along, so a mail server can drop duplicates (part 4).
func (l *Live) HandleSend(ctx context.Context, op *Send) (string, error) {
	return l.Mail(effect.StepKey(ctx), op.To, op.Msg)
}

// HandleCharge asks the bank. A refusal comes back as a value; only a broken
// line to the bank is an error.
func (l *Live) HandleCharge(_ context.Context, op *Charge) (f.Result[Receipt, ChargeError], error) {
	return l.Bank(op.Amount)
}

// --8<-- [end:live]

// --8<-- [start:fake]

// Fake answers from fixed data and remembers what was logged and sent.
// Tests use it to run programs with no clock, file system, randomness or mail.
type Fake struct {
	Clock   time.Time
	Files   map[string]string
	Rolls   []int
	Budget  int
	Logs    []string
	Sent    []string
	Charged []int
}

var _ MyEffHandler = (*Fake)(nil)

func (f *Fake) HandleLog(_ context.Context, op *Log) (Unit, error) {
	f.Logs = append(f.Logs, op.Msg)
	return Unit{}, nil
}

func (f *Fake) HandleNow(context.Context, *Now) (time.Time, error) {
	return f.Clock, nil
}

func (f *Fake) HandleReadFile(_ context.Context, op *ReadFile) ([]byte, error) {
	content, ok := f.Files[op.Path]
	if !ok {
		return nil, fmt.Errorf("fake: no file %q", op.Path)
	}
	return []byte(content), nil
}

// HandleRandom hands out Rolls in order, then repeats the last one forever.
func (f *Fake) HandleRandom(_ context.Context, op *Random) (int, error) {
	if len(f.Rolls) == 0 {
		return 0, fmt.Errorf("fake: no rolls configured")
	}
	got := f.Rolls[0]
	if len(f.Rolls) > 1 {
		f.Rolls = f.Rolls[1:]
	}
	return got % op.Max, nil
}

func (f *Fake) HandleSend(_ context.Context, op *Send) (string, error) {
	f.Sent = append(f.Sent, op.To+": "+op.Msg)
	return fmt.Sprintf("receipt-%d", len(f.Sent)), nil
}

// HandleCharge spends Budget. Past it, the answer is OutOfBudget, not an error.
func (fk *Fake) HandleCharge(_ context.Context, op *Charge) (f.Result[Receipt, ChargeError], error) {
	if op.Amount > fk.Budget {
		return f.MkErr[Receipt, ChargeError](&OutOfBudget{Missing: op.Amount - fk.Budget}), nil
	}
	fk.Budget -= op.Amount
	fk.Charged = append(fk.Charged, op.Amount)
	return f.MkOk[ChargeError](Receipt{ID: fmt.Sprintf("charge-%d", len(fk.Charged))}), nil
}

// --8<-- [end:fake]

// --8<-- [start:defaults]

// Defaults is an MyEffHandler with harmless answers. Embed it in a test
// handler and override only the methods the test cares about. The compiler
// still checks that the embedding type is a complete MyEffHandler.
//
// Nothing here is generated: every answer is written down, so a reader can
// see what a test gets when it does not say. A read fails loudly: a test
// must not depend on a file it never declared.
type Defaults struct{}

var _ MyEffHandler = Defaults{}

func (Defaults) HandleLog(context.Context, *Log) (Unit, error)      { return Unit{}, nil }
func (Defaults) HandleNow(context.Context, *Now) (time.Time, error) { return time.Time{}, nil }
func (Defaults) HandleReadFile(_ context.Context, op *ReadFile) ([]byte, error) {
	return nil, fmt.Errorf("defaults: no file %q", op.Path)
}
func (Defaults) HandleRandom(context.Context, *Random) (int, error) { return 0, nil }
func (Defaults) HandleSend(context.Context, *Send) (string, error)  { return "receipt-0", nil }
func (Defaults) HandleCharge(context.Context, *Charge) (f.Result[Receipt, ChargeError], error) {
	return f.MkOk[ChargeError](Receipt{ID: "charge-0"}), nil
}

// --8<-- [end:defaults]

// --8<-- [start:mailbox]

// Mailbox is a mail server. It remembers the receipt for every idempotency key,
// so a Send that is repeated with the same key is delivered once.
type Mailbox struct {
	Sent     []Mail
	receipts map[string]string
}

// Mail is one delivered message.
type Mail struct{ Key, To, Msg string }

// Send delivers, unless key was seen before; then it returns the old receipt.
func (m *Mailbox) Send(key, to, msg string) (string, error) {
	if receipt, ok := m.receipts[key]; ok && key != "" {
		return receipt, nil
	}
	m.Sent = append(m.Sent, Mail{Key: key, To: to, Msg: msg})
	receipt := fmt.Sprintf("receipt-%d", len(m.Sent))
	if key != "" {
		if m.receipts == nil {
			m.receipts = map[string]string{}
		}
		m.receipts[key] = receipt
	}
	return receipt, nil
}

// --8<-- [end:mailbox]

// --8<-- [start:bank]

// Bank is an account with a budget and a daily quota of charges.
type Bank struct {
	Budget  int
	Quota   int
	ResetAt time.Time
	Charged []int
}

// Charge takes amount from the account. The two refusals are answers: the
// bank worked, and said no. Nothing here returns a Go error, because nothing
// here can break; a real client would return one for a timeout.
func (b *Bank) Charge(amount int) (f.Result[Receipt, ChargeError], error) {
	if len(b.Charged) >= b.Quota {
		return f.MkErr[Receipt, ChargeError](&QuotaExceeded{ResetAt: b.ResetAt}), nil
	}
	if amount > b.Budget {
		return f.MkErr[Receipt, ChargeError](&OutOfBudget{Missing: amount - b.Budget}), nil
	}
	b.Budget -= amount
	b.Charged = append(b.Charged, amount)
	return f.MkOk[ChargeError](Receipt{ID: fmt.Sprintf("charge-%d", len(b.Charged))}), nil
}

// --8<-- [end:bank]
