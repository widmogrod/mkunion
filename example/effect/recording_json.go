package effect

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/widmogrod/mkunion/f"
)

// A tape as JSON. The operations use the JSON that mkunion generates for the
// MyEff union, so a tape can be stored, shipped to another machine, or read by
// TypeScript (see `mkunion shape-export`). Answers are decoded into the type each
// operation declares in Result.

// --8<-- [start:tape-json]

type jsonStep struct {
	Op     json.RawMessage `json:"op"`
	Answer json.RawMessage `json:"answer,omitempty"`
	Err    string          `json:"err,omitempty"`
}

// TapeToJSON encodes a recording.
func TapeToJSON(tape []Step[MyEff]) ([]byte, error) {
	steps := make([]jsonStep, 0, len(tape))
	for _, step := range tape {
		op, err := MyEffToJSON(step.Op)
		if err != nil {
			return nil, err
		}
		js := jsonStep{Op: op}
		if step.Err != nil {
			js.Err = step.Err.Error()
		} else if js.Answer, err = answerToJSON(step.Op, step.Answer); err != nil {
			return nil, err
		}
		steps = append(steps, js)
	}
	return json.Marshal(steps)
}

// TapeFromJSON decodes a recording. Each answer gets the type its operation declared.
func TapeFromJSON(data []byte) ([]Step[MyEff], error) {
	var steps []jsonStep
	if err := json.Unmarshal(data, &steps); err != nil {
		return nil, err
	}
	tape := make([]Step[MyEff], 0, len(steps))
	for i, js := range steps {
		op, err := MyEffFromJSON(js.Op)
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", i, err)
		}
		step := Step[MyEff]{Op: op}
		if js.Err != "" {
			step.Err = errors.New(js.Err)
		} else if step.Answer, err = answerFromJSON(op, js.Answer); err != nil {
			return nil, fmt.Errorf("step %d: %w", i, err)
		}
		tape = append(tape, step)
	}
	return tape, nil
}

// answerToJSON encodes an answer. Plain values use encoding/json. An answer
// that is a union needs the JSON mkunion generates for it, so the variant
// survives the round trip. Exhaustive, so a new operation cannot be forgotten.
func answerToJSON(op MyEff, answer any) (json.RawMessage, error) {
	return MatchMyEffR2(op,
		func(*Log) (json.RawMessage, error) { return json.Marshal(answer) },
		func(*Now) (json.RawMessage, error) { return json.Marshal(answer) },
		func(*ReadFile) (json.RawMessage, error) { return json.Marshal(answer) },
		func(*Random) (json.RawMessage, error) { return json.Marshal(answer) },
		func(*Send) (json.RawMessage, error) { return json.Marshal(answer) },
		func(*Charge) (json.RawMessage, error) {
			return f.ResultToJSON(answer.(f.Result[Receipt, ChargeError]))
		},
	)
}

// answerFromJSON decodes raw into the answer type op declares.
func answerFromJSON(op MyEff, raw json.RawMessage) (any, error) {
	return MatchMyEffR2(op,
		func(*Log) (any, error) { return decodeAnswer[Unit](raw) },
		func(*Now) (any, error) { return decodeAnswer[time.Time](raw) },
		func(*ReadFile) (any, error) { return decodeAnswer[[]byte](raw) },
		func(*Random) (any, error) { return decodeAnswer[int](raw) },
		func(*Send) (any, error) { return decodeAnswer[string](raw) },
		func(*Charge) (any, error) { return f.ResultFromJSON[Receipt, ChargeError](raw) },
	)
}

func decodeAnswer[R any](raw json.RawMessage) (any, error) {
	var value R
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return value, nil
}

// --8<-- [end:tape-json]
