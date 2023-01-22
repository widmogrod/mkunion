package state

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestStateTransition(t *testing.T) {
	useCases := map[string]struct {
		cmds   []Command
		state  []State
		errors []error
	}{
		"create candidate (valid)": {
			cmds: []Command{
				&CreateCandidateCMD{ID: "123"},
			},
			state: []State{
				&Candidate{ID: "123"},
			},
			errors: []error{
				nil,
			},
		},
		"candidate state and transit to duplicate  (valid)": {
			cmds: []Command{
				&CreateCandidateCMD{ID: "123"},
				&MarkAsDuplicateCMD{CanonicalID: "456"},
			},
			state: []State{
				&Candidate{ID: "123"},
				&Duplicate{ID: "123", CanonicalID: "456"},
			},
			errors: []error{
				nil,
				nil,
			},
		},
		"candidate state and transit to canonical  (valid)": {
			cmds: []Command{
				&CreateCandidateCMD{ID: "123"},
				&MarkAsCanonicalCMD{},
			},
			state: []State{
				&Candidate{ID: "123"},
				&Canonical{ID: "123"},
			},
			errors: []error{
				nil,
				nil,
			},
		},
		"candidate state and transit to unique  (valid)": {
			cmds: []Command{
				&CreateCandidateCMD{ID: "123"},
				&MarkAsUniqueCMD{},
			},
			state: []State{
				&Candidate{ID: "123"},
				&Unique{ID: "123"},
			},
			errors: []error{
				nil,
				nil,
			},
		},
		"initial state cannot be market as duplicate (invalid)": {
			cmds: []Command{
				&MarkAsDuplicateCMD{CanonicalID: "456"},
			},
			state: []State{
				nil,
			},
			errors: []error{
				ErrInvalidTransition,
			},
		},
		"candidate state and transit to canonical and duplicate  (invalid)": {
			cmds: []Command{
				&CreateCandidateCMD{ID: "123"},
				&MarkAsCanonicalCMD{},
				&MarkAsDuplicateCMD{CanonicalID: "456"},
			},
			state: []State{
				&Candidate{ID: "123"},
				&Canonical{ID: "123"},
				&Canonical{ID: "123"},
			},
			errors: []error{
				nil,
				nil,
				ErrInvalidTransition,
			},
		},
	}

	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			m := NewMachine()
			for i, tr := range uc.cmds {
				err := m.Handle(tr)
				if uc.errors[i] == nil {
					assert.NoError(t, err)
				} else {
					assert.Error(t, uc.errors[i], err)
				}
				assert.Equal(t, uc.state[i], m.State())
			}
		})
	}
}
