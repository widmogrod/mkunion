package effect

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// A tape as JSON. The operations use the JSON that mkunion generates for the
// Effect union, so a tape can be stored, shipped to another machine, or read by
// TypeScript (see `mkunion shape-export`). Answers are decoded into the type each
// operation declares in Result.

// --8<-- [start:tape-json]

type jsonStep struct {
	Op     json.RawMessage `json:"op"`
	Answer json.RawMessage `json:"answer,omitempty"`
	Err    string          `json:"err,omitempty"`
}

// TapeToJSON encodes a recording.
func TapeToJSON(tape []Step[Effect]) ([]byte, error) {
	steps := make([]jsonStep, 0, len(tape))
	for _, step := range tape {
		op, err := EffectToJSON(step.Op)
		if err != nil {
			return nil, err
		}
		js := jsonStep{Op: op}
		if step.Err != nil {
			js.Err = step.Err.Error()
		} else if js.Answer, err = json.Marshal(step.Answer); err != nil {
			return nil, err
		}
		steps = append(steps, js)
	}
	return json.Marshal(steps)
}

// TapeFromJSON decodes a recording. Each answer gets the type its operation declared.
func TapeFromJSON(data []byte) ([]Step[Effect], error) {
	var steps []jsonStep
	if err := json.Unmarshal(data, &steps); err != nil {
		return nil, err
	}
	tape := make([]Step[Effect], 0, len(steps))
	for i, js := range steps {
		op, err := EffectFromJSON(js.Op)
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", i, err)
		}
		step := Step[Effect]{Op: op}
		if js.Err != "" {
			step.Err = errors.New(js.Err)
		} else if step.Answer, err = answerFromJSON(op, js.Answer); err != nil {
			return nil, fmt.Errorf("step %d: %w", i, err)
		}
		tape = append(tape, step)
	}
	return tape, nil
}

// answerFromJSON decodes raw into the answer type op declares. Exhaustive, so a
// new operation cannot be forgotten here.
func answerFromJSON(op Effect, raw json.RawMessage) (any, error) {
	return MatchEffectR2(op,
		func(*Log) (any, error) { return decodeAnswer[Unit](raw) },
		func(*Now) (any, error) { return decodeAnswer[time.Time](raw) },
		func(*ReadFile) (any, error) { return decodeAnswer[[]byte](raw) },
		func(*Random) (any, error) { return decodeAnswer[int](raw) },
		func(*Send) (any, error) { return decodeAnswer[string](raw) },
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
