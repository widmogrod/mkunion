package machine

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shared"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"testing"
)

func NewTestSuite[D, C, S any](dep D, mkMachine func(dep D, init S) *Machine[D, C, S]) *Suite[D, C, S] {
	infer := NewInferTransition[C, S]()
	return &Suite[D, C, S]{
		dep:       dep,
		mkMachine: mkMachine,
		infer:     infer,
	}
}

type Suite[D, C, S any] struct {
	dep       D
	mkMachine func(dep D, init S) *Machine[D, C, S]
	infer     *InferTransition[C, S]
	cases     []*Case[D, C, S]
}

func (suite *Suite[D, C, S]) Case(t *testing.T, name string, f func(t *testing.T, c *Case[D, C, S])) *Suite[D, C, S] {
	t.Helper()
	c := &Case[D, C, S]{
		suit: suite,
		step: Step[D, C, S]{
			Name: name,
		},
	}
	f(t, c)

	suite.cases = append(suite.cases, c)
	return suite
}

func (suite *Suite[D, C, S]) fuzzy() {
	// Some commands or states can be more popular
	// when we randomly select them, we can increase the chance of selecting them, and skip less popular ones, which is not desired
	// we want to have a good coverage of all commands and states
	// to achieve this, we will group commands and states, and randomly select group, and then randomly select command or state from this group

	states := make(map[string][]Step[D, C, S])
	commands := make(map[string][]Step[D, C, S])
	// Change: Use content-based uniqueness instead of type-based
	uniqueStates := make(map[string]S)
	uniqueStatesList := []S{} // Keep ordered list of unique states

	for _, c := range suite.cases {
		if any(c.step.ExpectedState) != nil {
			stateName := reflect.TypeOf(c.step.ExpectedState).String()
			states[stateName] = append(states[stateName], c.step)

			// Use JSON marshalling for content-based uniqueness
			stateKey := suite.getStateKey(c.step.ExpectedState)
			if _, exists := uniqueStates[stateKey]; !exists {
				uniqueStates[stateKey] = c.step.ExpectedState
				uniqueStatesList = append(uniqueStatesList, c.step.ExpectedState)
			}
		}

		if any(c.step.GivenCommand) != nil {
			commandName := reflect.TypeOf(c.step.GivenCommand).String()
			commands[commandName] = append(commands[commandName], c.step)
		}

		if any(c.step.InitState) != nil {
			stateName := reflect.TypeOf(c.step.InitState).String()
			states[stateName] = append(states[stateName], c.step)

			// Use JSON marshalling for content-based uniqueness
			stateKey := suite.getStateKey(c.step.InitState)
			if _, exists := uniqueStates[stateKey]; !exists {
				uniqueStates[stateKey] = c.step.InitState
				uniqueStatesList = append(uniqueStatesList, c.step.InitState)
			}
		}
	}

	// Sort command and state names for deterministic iteration
	var commandNames []string
	for name := range commands {
		commandNames = append(commandNames, name)
	}
	sort.Strings(commandNames)

	var stateNames []string
	for name := range states {
		stateNames = append(stateNames, name)
	}
	sort.Strings(stateNames)

	// Add empty state to the list
	var zeroState S
	zeroStateKey := suite.getStateKey(zeroState)
	if _, exists := uniqueStates[zeroStateKey]; !exists {
		uniqueStates[zeroStateKey] = zeroState
		uniqueStatesList = append([]S{zeroState}, uniqueStatesList...) // Prepend zero state
	}

	// First, generate deterministic permutations of all state-command pairs
	for _, state := range uniqueStatesList {
		for _, commandName := range commandNames {
			if steps, ok := commands[commandName]; ok && len(steps) > 0 {
				// Try all command instances of this type
				for _, step := range steps {
					command := step.GivenCommand

					// Execute this state-command combination
					m := suite.mkMachine(suite.dep, state)
					err := m.Handle(context.Background(), command)
					newState := m.State()
					suite.infer.Record(command, state, newState, err)
				}
			}
		}
	}

	// Then add some controlled randomness for additional exploration
	r := rand.New(rand.NewSource(0))
	numRandomIterations := len(suite.cases) * 100

	for i := 0; i < numRandomIterations; i++ {
		// Select command deterministically based on index
		commandIdx := i % len(commandNames)
		commandName := commandNames[commandIdx]

		var command C
		if steps, ok := commands[commandName]; ok && len(steps) > 0 {
			// Use deterministic selection instead of random
			stepIdx := (i / len(commandNames)) % len(steps)
			command = steps[stepIdx].GivenCommand
		}

		// Select state with some controlled variation
		if len(uniqueStatesList) > 0 {
			stateIdx := (i + r.Intn(3)) % len(uniqueStatesList)
			state := uniqueStatesList[stateIdx]

			if any(command) != nil {
				m := suite.mkMachine(suite.dep, state)
				err := m.Handle(context.Background(), command)
				newState := m.State()
				suite.infer.Record(command, state, newState, err)
			}
		}
	}
}

// findMatchingCommands returns all command steps that match the given command name
func (suite *Suite[D, C, S]) findMatchingCommands(commandSteps map[string][]Step[D, C, S], commandName string) []C {
	var commands []C
	if steps, ok := commandSteps[commandName]; ok {
		for _, step := range steps {
			if any(step.GivenCommand) != nil {
				commands = append(commands, step.GivenCommand)
			}
		}
	}
	return commands
}

// findMatchingStates returns all states that match the given state name
func (suite *Suite[D, C, S]) findMatchingStates(stateSteps map[string][]Step[D, C, S], stateName string) []S {
	var states []S
	seen := make(map[string]bool) // Track unique states by content

	if steps, ok := stateSteps[stateName]; ok {
		for _, step := range steps {
			state := suite.extractStateFromStep(step)
			if any(state) != nil {
				// Use content-based uniqueness
				key := suite.getStateKey(state)
				if !seen[key] {
					seen[key] = true
					states = append(states, state)
				}
			}
		}
	}
	return states
}

// extractStateFromStep extracts a state from a step, preferring ExpectedState over InitState
func (suite *Suite[D, C, S]) extractStateFromStep(step Step[D, C, S]) S {
	if any(step.ExpectedState) != nil {
		return step.ExpectedState
	}
	if any(step.InitState) != nil {
		return step.InitState
	}
	var zeroState S
	return zeroState
}

// fuzzyWithKnownTransitions runs fuzzy testing with a feedback loop that prioritizes known transitions
func (suite *Suite[D, C, S]) fuzzyWithKnownTransitions(knownTransitions []ParsedTransition) {
	// First, build maps to find steps by command and state names
	commandSteps := make(map[string][]Step[D, C, S])
	stateSteps := make(map[string][]Step[D, C, S])

	for _, c := range suite.cases {
		if any(c.step.GivenCommand) != nil {
			commandName := reflect.TypeOf(c.step.GivenCommand).String()
			commandSteps[commandName] = append(commandSteps[commandName], c.step)
		}

		if any(c.step.ExpectedState) != nil {
			stateName := reflect.TypeOf(c.step.ExpectedState).String()
			stateSteps[stateName] = append(stateSteps[stateName], c.step)
		}

		if any(c.step.InitState) != nil {
			stateName := reflect.TypeOf(c.step.InitState).String()
			stateSteps[stateName] = append(stateSteps[stateName], c.step)
		}
	}

	// Execute all known transitions first to ensure coverage
	for _, knownTrans := range knownTransitions {
		// Find all matching commands
		commands := suite.findMatchingCommands(commandSteps, knownTrans.Command)
		if len(commands) == 0 {
			continue
		}

		// Find all matching initial states
		var initStates []S
		if knownTrans.FromState != "" {
			initStates = suite.findMatchingStates(stateSteps, knownTrans.FromState)
		}

		// If no states found or FromState is empty, use zero state
		if len(initStates) == 0 {
			var zeroState S
			initStates = append(initStates, zeroState)
		}

		// Try all combinations of commands and initial states
		for _, command := range commands {
			for _, initState := range initStates {
				// Execute the transition
				m := suite.mkMachine(suite.dep, initState)
				err := m.Handle(context.Background(), command)
				newState := m.State()

				// Record the transition
				suite.infer.Record(command, initState, newState, err)
			}
		}
	}

	// Then run the regular fuzzy testing for additional exploration
	suite.fuzzy()
}

// AssertSelfDocumentStateDiagram help to self document state machine transitions, just by running tests.
// It will compare current state diagram with stored in file.
// It will fail assertion if they are not equal. This may happen, when tests are changed, or state machine is changed.
// In both then visual inspection of state diagram helps to double-check if changes are correct. And use diagrams in documentation.
//
// If file does not exist, function will return true, to indicate that file should be created.
// For this purpose call SelfDocumentStateDiagram.
func (suite *Suite[D, C, S]) AssertSelfDocumentStateDiagram(t *testing.T, filename string) bool {
	t.Helper()
	// extract fine name from file, if there is extension remove it
	fileName := filename + ".state_diagram.mmd"
	fileNameWithErrorTransitions := filename + ".state_diagram_with_errors.mmd"

	knownTransitions := suite.extractKnownTransitions(fileName, fileNameWithErrorTransitions)

	// Run fuzzy testing with feedback loop if we have known transitions
	if len(knownTransitions) > 0 {
		suite.fuzzyWithKnownTransitions(knownTransitions)
	} else {
		suite.fuzzy()
	}

	for _, f := range []struct {
		filename  string
		withError bool
	}{
		{fileName, false},
		{fileNameWithErrorTransitions, true},
	} {
		suite.infer.WithErrorTransitions(f.withError)
		mermaidDiagram := suite.infer.ToMermaid()

		// if file exists, read content and compare with mermaidDiagram
		data, err := os.ReadFile(f.filename)
		if err != nil {
			if os.IsNotExist(err) {
				return true
			} else {
				assert.NoErrorf(t, err, "failed to read file %s", f.filename)
			}
		}

		// if stored content is not equal, fail assertion
		if diff := cmp.Diff(string(data), mermaidDiagram); diff != "" {
			t.Fatalf("unexpected state diagram (-want +got):\n%s", diff)
			return false
		}
	}

	return false
}

func (suite *Suite[D, C, S]) extractKnownTransitions(fileName string, fileNameWithErrorTransitions string) []ParsedTransition {
	// Try to read existing diagrams and parse known transitions
	var knownTransitions []ParsedTransition

	// Read the main diagram file
	if data, err := os.ReadFile(fileName); err == nil {
		if parsed, err := ParseMermaid(string(data)); err == nil {
			knownTransitions = append(knownTransitions, parsed...)
		}
	}

	// Read the error transitions diagram if it exists
	if data, err := os.ReadFile(fileNameWithErrorTransitions); err == nil {
		if parsed, err := ParseMermaid(string(data)); err == nil {
			// Add error transitions too
			for _, t := range parsed {
				if t.IsError {
					knownTransitions = append(knownTransitions, t)
				}
			}
		}
	}

	return knownTransitions
}

// SelfDocumentStateDiagram help to self document state machine transitions, just by running tests.
// It will always overwrite stored state diagram files, useful in TDD loop, when tests are being written.
func (suite *Suite[D, C, S]) SelfDocumentStateDiagram(t *testing.T, filename string) {
	// extract fine name from file, if there is extension remove it
	fileName := filename + ".state_diagram.mmd"
	fileNameWithErrorTransitions := filename + ".state_diagram_with_errors.mmd"

	knownTransitions := suite.extractKnownTransitions(fileName, fileNameWithErrorTransitions)

	// Run fuzzy testing with feedback loop if we have known transitions
	if len(knownTransitions) > 0 {
		suite.fuzzyWithKnownTransitions(knownTransitions)
	} else {
		suite.fuzzy()
	}

	for _, f := range []struct {
		filename  string
		withError bool
	}{
		{fileName, false},
		{fileNameWithErrorTransitions, true},
	} {
		suite.infer.WithErrorTransitions(f.withError)
		mermaidDiagram := suite.infer.ToMermaid()

		// create file if not exists, use mermaidDiagram as content
		err := os.WriteFile(f.filename, []byte(mermaidDiagram), 0644)
		assert.NoError(t, err, "failed to write file %s", f.filename)
	}
}

type Case[D, C, S any] struct {
	suit *Suite[D, C, S]

	step Step[D, C, S]

	process     bool
	resultErr   error
	resultState S
}

// GivenCommand starts building assertion that when command is applied to machine, it will result in given state or error.
func (suitcase *Case[D, C, S]) GivenCommand(c C) *Case[D, C, S] {
	suitcase.step.GivenCommand = c
	return suitcase
}

// BeforeCommand is optional, if provided it will be called before command is executed
// useful when you want to prepare some data before command is executed,
// like change dependency to return error, or change some state
func (suitcase *Case[D, C, S]) BeforeCommand(f func(testing.TB, D)) *Case[D, C, S] {
	suitcase.step.BeforeCommand = f
	return suitcase
}

// AfterCommand is optional, if provided it will be called after command is executed
// useful when you want to assert some data after command is executed,
// like what function were called, and with what arguments
func (suitcase *Case[D, C, S]) AfterCommand(f func(testing.TB, D)) *Case[D, C, S] {
	suitcase.step.AfterCommand = f
	return suitcase
}

// ThenState asserts that command applied to machine will result in given state
// implicitly assumes that error is nil
func (suitcase *Case[D, C, S]) ThenState(t *testing.T, o S) *Case[D, C, S] {
	t.Helper()

	suitcase.step.ExpectedState = o
	suitcase.step.ExpectedErr = nil
	suitcase.run(t)

	return suitcase
}

// ThenStateAndError asserts that command applied to machine will result in given state and error
// state is required because we want to know what is the expected state after command fails to be applied, and return error.
// state most of the time shouldn't be modified, and explicit definition of state help to make this behaviour explicit.
func (suitcase *Case[D, C, S]) ThenStateAndError(t *testing.T, state S, err error) *Case[D, C, S] {
	t.Helper()
	suitcase.step.ExpectedState = state
	suitcase.step.ExpectedErr = err
	suitcase.run(t)

	return suitcase
}

// ForkCase takes previous state of machine and allows to apply another case from this point onward
// it's useful when you want to test multiple scenarios from one state
func (suitcase *Case[D, C, S]) ForkCase(t *testing.T, name string, f func(t *testing.T, c *Case[D, C, S])) *Case[D, C, S] {
	t.Helper()

	// We have to run the current test case,
	// if we want to have state to form from
	suitcase.run(t)

	newState := suitcase.deepCopy(suitcase.resultState)

	newCase := &Case[D, C, S]{
		suit: suitcase.suit,
		step: Step[D, C, S]{
			Name:      name,
			InitState: newState,
		},
	}

	f(t, newCase)

	suitcase.suit.cases = append(suitcase.suit.cases, newCase)
	return suitcase
}

func (suitcase *Case[D, C, S]) run(t *testing.T) {
	if suitcase.process {
		return
	}
	suitcase.process = true

	t.Helper()
	machine := suitcase.suit.mkMachine(suitcase.suit.dep, suitcase.step.InitState)
	if suitcase.step.BeforeCommand != nil {
		suitcase.step.BeforeCommand(t, suitcase.suit.dep)
	}

	err := machine.Handle(context.Background(), suitcase.step.GivenCommand)
	suitcase.resultErr = err
	suitcase.resultState = machine.State()

	if suitcase.step.AfterCommand != nil {
		suitcase.step.AfterCommand(t, suitcase.suit.dep)
	}

	suitcase.suit.infer.Record(suitcase.step.GivenCommand, suitcase.step.InitState, suitcase.resultState, err)

	if !errors.Is(err, suitcase.step.ExpectedErr) {
		t.Fatalf("unexpected error \n  expect: %v \n     got: %v\n", suitcase.step.ExpectedErr, err)
	}

	if diff := cmp.Diff(suitcase.step.ExpectedState, suitcase.resultState); diff != "" {
		t.Fatalf("unexpected state (-want +got):\n%suitcase", diff)
	}
}

func (suitcase *Case[D, C, S]) deepCopy(state S) S {
	data, err := shared.JSONMarshal[S](state)
	if err != nil {
		panic(fmt.Errorf("failed deep copying state %T, reason: %w", state, err))
	}
	result, err := shared.JSONUnmarshal[S](data)
	if err != nil {
		panic(fmt.Errorf("failed deep copying state %T, reason: %w", state, err))
	}
	return result
}

// getStateKey generates a unique key for a state based on its content
func (suite *Suite[D, C, S]) getStateKey(state S) string {
	// If state is nil, return a special key
	if any(state) == nil {
		return "<nil>"
	}

	// Try to marshal state to JSON for content-based comparison
	data, err := shared.JSONMarshal[S](state)
	if err != nil {
		// Fallback to type name if marshalling fails
		return reflect.TypeOf(state).String()
	}

	// Use the JSON representation as the unique key
	return string(data)
}

// Step is a single test case that describe state machine transition
type Step[D, C, S any] struct {
	// Name human readable description of the test case. It's required
	Name string

	// InitState is optional, if not provided it will be nil
	// and when step is part of sequence, then state will be inherited from previous step
	InitState S

	// GivenCommand is the command that will be applied to the machine. It's required
	GivenCommand C
	// BeforeCommand is optional, if provided it will be called before command is executed
	BeforeCommand func(t testing.TB, x D)
	// AfterCommand is optional, if provided it will be called after command is executed
	AfterCommand func(t testing.TB, x D)

	// ExpectedState is the expected state after command is executed. It's required, but can be nil
	ExpectedState S
	// ExpectedErr is the expected error after command is executed. It's required, but can be nil
	ExpectedErr error
}
