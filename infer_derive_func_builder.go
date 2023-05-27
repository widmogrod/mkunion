package mkunion

import "fmt"

type MatchBuilder struct {
	name       string
	inputTypes []string
	cases      [][]string
	names      []string
}

func (b *MatchBuilder) SetName(name string) error {
	if b.name == "" {
		b.name = name
	} else {
		return fmt.Errorf("match.SetName cannot declare name more than once")
	}

	return nil
}

func (b *MatchBuilder) SetInputs(types ...string) error {
	if len(types) == 0 {
		return fmt.Errorf("match.SetInputs is empty")
	}

	if b.inputTypes == nil {
		b.inputTypes = types
	} else {
		return fmt.Errorf("match.SetInputs cannot declare inputs more than once")
	}

	return nil
}

func (b *MatchBuilder) AddCase(name string, inputs ...string) error {
	if len(inputs) == 0 {
		return fmt.Errorf("match.AddCase is empty; case name: %s", name)
	}

	if len(inputs) != len(b.inputTypes) {
		return fmt.Errorf("match.AddCase function must have same number of arguments as inputs; case name: %s", name)
	}

	// check if there are no duplicates in other cases
	for cid, caseInputs := range b.cases {
		same := len(caseInputs)
		for i, input := range caseInputs {
			if input == inputs[i] {
				same--
			}
		}
		if same == 0 {
			return fmt.Errorf("match.AddCase cannot have duplicate; cases name: %s", b.names[cid])
		}
	}
	b.cases = append(b.cases, inputs)

	// check if there are no duplicates in names
	for _, caseName := range b.names {
		if caseName == name {
			return fmt.Errorf("match.AddCase cannot have duplicate; case name: %s", caseName)
		}
	}
	b.names = append(b.names, name)

	return nil
}

type MatchSpec struct {
	Name   string
	Names  []string
	Inputs []string
	Cases  [][]string
}

func (b *MatchBuilder) Build() (*MatchSpec, error) {
	return &MatchSpec{
		Name:   b.name,
		Names:  b.names,
		Inputs: b.inputTypes,
		Cases:  b.cases,
	}, nil
}

func NewMatchBuilder() *MatchBuilder {
	return &MatchBuilder{}
}
