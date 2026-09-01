package effect

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"time"
)

// --8<-- [start:live]

// Live performs operations against the real world.
type Live struct {
	Out  io.Writer
	FS   fs.FS
	Rand *rand.Rand
	Now  func() time.Time
}

var _ EffectHandler = (*Live)(nil)

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

// --8<-- [end:live]

// --8<-- [start:fake]

// Fake answers from fixed data and remembers what was logged.
// Tests use it to run programs with no clock, file system or randomness.
type Fake struct {
	Clock time.Time
	Files map[string]string
	Rolls []int
	Logs  []string
}

var _ EffectHandler = (*Fake)(nil)

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

// --8<-- [end:fake]
