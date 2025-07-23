package example

import (
	"errors"
	"testing"
)

func TestTransition(t *testing.T) {
	type testCase struct {
		name      string
		state     State
		command   Command
		expectErr error
		expectRes State
	}

	tests := []testCase{
		{
			name:      "InitialToProcessing",
			state:     &Initial{},
			command:   &StartCommand{ID: "123"},
			expectErr: nil,
			expectRes: &Processing{ID: "123"},
		},
		{
			name:      "ProcessingToComplete",
			state:     &Processing{ID: "123"},
			command:   &CompleteCommand{Result: "Success"},
			expectErr: nil,
			expectRes: &Complete{Result: "Success"},
		},
		{
			name:      "ProcessingToProcessingError",
			state:     &Processing{ID: "123"},
			command:   &StartCommand{ID: "123"},
			expectErr: errors.New("already processing"),
			expectRes: nil,
		},
		{
			name:      "InvalidTransitionError",
			state:     &Complete{Result: "Success"},
			command:   &StartCommand{ID: "456"},
			expectErr: errors.New("invalid transition"),
			expectRes: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Transition(tc.state, tc.command)

			if !errors.Is(err, tc.expectErr) {
				t.Errorf("expected error: %v, got: %v", tc.expectErr, err)
			}

			if res != nil && tc.expectRes != nil {
				if res1, ok := res.(*Processing); ok {
					if res2, ok := tc.expectRes.(*Processing); ok && res1.ID != res2.ID {
						t.Errorf("expected result: %v, got: %v", tc.expectRes, res)
					}
				} else if res1, ok := res.(*Complete); ok {
					if res2, ok := tc.expectRes.(*Complete); ok && res1.Result != res2.Result {
						t.Errorf("expected result: %v, got: %v", tc.expectRes, res)
					}
				}
			}
		})
	}
}
