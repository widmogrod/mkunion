package state

import "fmt"

var _ TransitionVisitor = (*Machine)(nil)

var ErrInvalidTransition = fmt.Errorf("invalid transition")

func NewMachine() *Machine {
	return &Machine{}
}

type Machine struct {
	state State
	err   error
}

func (m *Machine) LastError() error {
	return m.err
}

func (m *Machine) Apply(s Transition) error {
	if m.err != nil {
		return fmt.Errorf("cannot apply %T on state with error: %w", s, m.err)
	}
	res := s.Accept(m)
	if res != nil {
		if err, ok := res.(error); ok {
			m.err = err
		} else {
			m.err = fmt.Errorf("unexpected result %T. Expecting error ", res)
		}
	}

	return m.err
}

func (m *Machine) VisitCreateCandidate(v *CreateCandidate) any {
	if m.state != nil {
		return fmt.Errorf("%w VisitCreateCandidate: candidate can be created on new state only %T", ErrInvalidTransition, m.state)
	}

	m.state = &Candidate{
		ID:         v.ID,
		Attributes: nil,
	}

	return nil
}

func (m *Machine) VisitMarkAsCanonical(v *MarkAsCanonical) any {
	if candidate, ok := m.state.(*Candidate); ok {
		m.state = &Canonical{
			ID: candidate.ID,
		}
		return nil
	}

	return fmt.Errorf("%w VisitMarkAsCanonical: state %T cannot be marked as canonical", ErrInvalidTransition, m.state)
}

func (m *Machine) VisitMarkAsDuplicate(v *MarkAsDuplicate) any {
	if candidate, ok := m.state.(*Candidate); ok {
		m.state = &Duplicate{
			ID:          candidate.ID,
			CanonicalID: v.CanonicalID,
		}
		return nil
	}
	return fmt.Errorf("%w VisitMarkAsDuplicate: state %T cannot be marked as duplicate", ErrInvalidTransition, m.state)
}

func (m *Machine) VisitMarkAsUnique(v *MarkAsUnique) any {
	if candidate, ok := m.state.(*Candidate); ok {
		m.state = &Unique{
			ID: candidate.ID,
		}
		return nil
	}
	return fmt.Errorf("%w VisitMarkAsUnique: state %T cannot be marked as unique", ErrInvalidTransition, m.state)
}
