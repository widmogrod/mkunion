package state

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStateTransition(t *testing.T) {
	useCases := map[string]struct {
		transition    []Transition
		expectedState []State
		expectedError []error
	}{
		"create candidate (valid)": {
			transition: []Transition{
				&CreateCandidate{ID: "123"},
			},
			expectedState: []State{
				&Candidate{ID: "123"},
			},
			expectedError: []error{
				nil,
			},
		},
		"candidate state and transit to duplicate  (valid)": {
			transition: []Transition{
				&CreateCandidate{ID: "123"},
				&MarkAsDuplicate{CanonicalID: "456"},
			},
			expectedState: []State{
				&Candidate{ID: "123"},
				&Duplicate{ID: "123", CanonicalID: "456"},
			},
			expectedError: []error{
				nil,
				nil,
			},
		},
		"candidate state and transit to canonical  (valid)": {
			transition: []Transition{
				&CreateCandidate{ID: "123"},
				&MarkAsCanonical{},
			},
			expectedState: []State{
				&Candidate{ID: "123"},
				&Canonical{ID: "123"},
			},
			expectedError: []error{
				nil,
				nil,
			},
		},
		"candidate state and transit to unique  (valid)": {
			transition: []Transition{
				&CreateCandidate{ID: "123"},
				&MarkAsUnique{},
			},
			expectedState: []State{
				&Candidate{ID: "123"},
				&Unique{ID: "123"},
			},
			expectedError: []error{
				nil,
				nil,
			},
		},
		"initial state cannot be market as duplicate (invalid)": {
			transition: []Transition{
				&MarkAsDuplicate{CanonicalID: "456"},
			},
			expectedState: []State{
				nil,
			},
			expectedError: []error{
				ErrInvalidTransition,
			},
		},
		"candidate state and transit to canonical and duplicate  (invalid)": {
			transition: []Transition{
				&CreateCandidate{ID: "123"},
				&MarkAsCanonical{},
				&MarkAsDuplicate{CanonicalID: "456"},
			},
			expectedState: []State{
				&Candidate{ID: "123"},
				&Canonical{ID: "123"},
				&Canonical{ID: "123"},
			},
			expectedError: []error{
				nil,
				nil,
				ErrInvalidTransition,
			},
		},
	}

	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			m := &Machine{}
			for i, tr := range uc.transition {
				err := m.Apply(tr)
				if uc.expectedError[i] == nil {
					assert.NoError(t, err)
				} else {
					assert.Error(t, uc.expectedError[i], err)
				}
				assert.Equal(t, uc.expectedState[i], m.state)
			}
		})
	}

}
