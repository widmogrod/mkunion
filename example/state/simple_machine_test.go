package state

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/machine"
	"testing"
)

func TestStateTransition(t *testing.T) {
	useCases := []struct {
		name   string
		cmds   []Command
		state  []State
		errors []error
	}{
		{
			name: "create candidate (valid)",
			cmds: []Command{
				&CreateCandidateCMD{ID: "123"},
			},
			state: []State{
				&Candidate{ID: "123"},
			},
			errors: []error{
				nil,
			},
		}, {
			name: "candidate state and transit to duplicate  (valid)",
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
		{
			name: "candidate state and transit to canonical  (valid)",
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
		{
			name: "candidate state and transit to unique  (valid)",
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
		{
			name: "initial state cannot be market as duplicate (invalid)",
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
		{
			name: "candidate state and transit to canonical and duplicate  (invalid)",
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

	infer := machine.NewInferTransition[Command, State]()
	infer.WithTitle("Canonical question transition")

	for _, uc := range useCases {
		t.Run(uc.name, func(t *testing.T) {
			m := NewMachine()
			for i, tr := range uc.cmds {
				prev := m.State()
				err := m.Handle(tr)
				if uc.errors[i] == nil {
					assert.NoError(t, err)
				} else {
					assert.Error(t, uc.errors[i], err)
				}
				assert.Equal(t, uc.state[i], m.State())
				infer.Record(tr, prev, m.State(), err)
			}
		})
	}

	infer.WithErrorTransitions()
	result := infer.ToMermaid()
	fmt.Println(result)
	assert.Equal(t, `---
title: Canonical question transition
---
stateDiagram
	[*] --> "*state.Candidate": "*state.CreateCandidateCMD"
	"*state.Candidate" --> "*state.Duplicate": "*state.MarkAsDuplicateCMD"
	"*state.Candidate" --> "*state.Canonical": "*state.MarkAsCanonicalCMD"
	"*state.Candidate" --> "*state.Unique": "*state.MarkAsUniqueCMD"
	[*] --> [*]: "❌*state.MarkAsDuplicateCMD"
	"*state.Canonical" --> "*state.Canonical": "❌*state.MarkAsDuplicateCMD"
`, result)
}
