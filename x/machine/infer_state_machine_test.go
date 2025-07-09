package machine

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInferStateMachine(t *testing.T) {
	infer := NewInferTransition[string, int]()
	infer.Record("inc", 10, 11, nil)
	infer.Record("dec", 11, 10, nil)
	infer.Record("unknown", 10, 10, fmt.Errorf("unknown cmd: unknown"))
	result := infer.ToMermaid()

	assert.Equal(t, `stateDiagram
	int: int

	int --> int: string
`, result)
}

func TestParseMermaid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []ParsedTransition
	}{
		{
			name: "simple transitions",
			input: `stateDiagram
	State1: *example.StateOne
	State2: *example.StateTwo
	
	[*] --> State1: StartCmd
	State1 --> State2: NextCmd
	State2 --> [*]: EndCmd`,
			expected: []ParsedTransition{
				{FromState: "", ToState: "*example.StateOne", Command: "StartCmd", IsError: false},
				{FromState: "*example.StateOne", ToState: "*example.StateTwo", Command: "NextCmd", IsError: false},
				{FromState: "*example.StateTwo", ToState: "", Command: "EndCmd", IsError: false},
			},
		},
		{
			name: "transitions with errors",
			input: `stateDiagram
	State1: *example.StateOne
	State2: *example.StateTwo
	
	%% error=something went wrong 
	State1 --> State1: ❌FailCmd
	State1 --> State2: SuccessCmd`,
			expected: []ParsedTransition{
				{FromState: "*example.StateOne", ToState: "*example.StateOne", Command: "FailCmd", IsError: true},
				{FromState: "*example.StateOne", ToState: "*example.StateTwo", Command: "SuccessCmd", IsError: false},
			},
		},
		{
			name: "with title and various formats",
			input: `---
title: Test State Machine
---
stateDiagram
	Initial: *test.Initial
	Processing: *test.Processing
	Complete: *test.Complete
	
	[*] --> Initial: Init
	Initial --> Processing: StartCmd
	Processing --> Complete: CompleteCmd
	Processing --> Processing: UpdateCmd`,
			expected: []ParsedTransition{
				{FromState: "", ToState: "*test.Initial", Command: "Init", IsError: false},
				{FromState: "*test.Initial", ToState: "*test.Processing", Command: "StartCmd", IsError: false},
				{FromState: "*test.Processing", ToState: "*test.Complete", Command: "CompleteCmd", IsError: false},
				{FromState: "*test.Processing", ToState: "*test.Processing", Command: "UpdateCmd", IsError: false},
			},
		},
		{
			name:     "empty diagram",
			input:    `stateDiagram`,
			expected: []ParsedTransition{},
		},
		{
			name: "ignores content before stateDiagram marker",
			input: `---
title: This should be ignored
someKey: someValue
---
# This is a comment that should be ignored
randomKey: randomValue

stateDiagram
	State1: *example.StateOne
	State2: *example.StateTwo
	
	State1 --> State2: NextCmd`,
			expected: []ParsedTransition{
				{FromState: "*example.StateOne", ToState: "*example.StateTwo", Command: "NextCmd", IsError: false},
			},
		},
		{
			name: "no stateDiagram marker",
			input: `State1: *example.StateOne
State2: *example.StateTwo
State1 --> State2: NextCmd`,
			expected: []ParsedTransition{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseMermaid(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferTransition_Deterministic(t *testing.T) {
	// Test that ToMermaid produces deterministic output
	infer1 := NewInferTransition[string, string]()
	infer2 := NewInferTransition[string, string]()

	// Record the same transitions in different order
	transitions := []struct {
		cmd  string
		from string
		to   string
		err  error
	}{
		{"cmd1", "state1", "state2", nil},
		{"cmd2", "state2", "state3", nil},
		{"cmd3", "state1", "state3", nil},
		{"cmd1", "state3", "state1", nil},
	}

	// Add to infer1 in forward order
	for _, tr := range transitions {
		infer1.Record(tr.cmd, tr.from, tr.to, tr.err)
	}

	// Add to infer2 in reverse order
	for i := len(transitions) - 1; i >= 0; i-- {
		tr := transitions[i]
		infer2.Record(tr.cmd, tr.from, tr.to, tr.err)
	}

	// Both should produce the same mermaid output
	diagram1 := infer1.ToMermaid()
	diagram2 := infer2.ToMermaid()

	assert.Equal(t, diagram1, diagram2, "Mermaid diagrams should be identical regardless of recording order")
}

func TestCreateAlias(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"*example.StateOne", "StateOne"},
		{"example.StateTwo", "StateTwo"},
		{"*github.com/example/pkg.ComplexState", "ComplexState"},
		{"SimpleState", "SimpleState"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := createAlias(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
