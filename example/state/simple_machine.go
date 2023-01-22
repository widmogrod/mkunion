package state

import (
	"fmt"
	"github.com/widmogrod/mkunion/x/machine"
)

var (
	ErrInvalidTransition = fmt.Errorf("invalid cmds")
)

func NewMachine() *machine.Machine[Command, State] {
	return machine.NewSimpleMachine(Transition)
}

func Transition(cmd Command, state State) (State, error) {
	return MustMatchCommandR2(
		cmd,
		func(x *CreateCandidateCMD) (State, error) {
			if state != nil {
				return nil, fmt.Errorf("candidate already created, state: %T; %w", state, ErrInvalidTransition)
			}

			newState := &Candidate{
				ID:         x.ID,
				Attributes: nil,
			}

			return newState, nil
		},
		func(x *MarkAsCanonicalCMD) (State, error) {
			stateCandidate, ok := state.(*Candidate)
			if !ok {
				return nil, fmt.Errorf("state is not candidate, state: %T; %w", state, ErrInvalidTransition)
			}

			return &Canonical{
				ID: stateCandidate.ID,
			}, nil
		},
		func(x *MarkAsDuplicateCMD) (State, error) {
			stateCandidate, ok := state.(*Candidate)
			if !ok {
				return nil, fmt.Errorf("state is not candidate, state: %T; %w", state, ErrInvalidTransition)
			}

			return &Duplicate{
				ID:          stateCandidate.ID,
				CanonicalID: x.CanonicalID,
			}, nil
		},
		func(x *MarkAsUniqueCMD) (State, error) {
			stateCandidate, ok := state.(*Candidate)
			if !ok {
				return nil, fmt.Errorf("state is not candidate, state: %T; %w", state, ErrInvalidTransition)
			}
			return &Unique{
				ID: stateCandidate.ID,
			}, nil
		},
	)
}
