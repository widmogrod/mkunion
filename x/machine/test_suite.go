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
	r := rand.New(rand.NewSource(0))

	// Some commands or states can be more popular
	// when we randomly select them, we can increase the chance of selecting them, and skip less popular ones, which is not desired
	// we want to have a good coverage of all commands and states
	// to achieve this, we will group commands and states, and randomly select group, and then randomly select command or state from this group

	states := make(map[string][]Step[D, C, S])
	commands := make(map[string][]Step[D, C, S])

	for _, c := range suite.cases {
		if any(c.step.ExpectedState) != nil {
			stateName := reflect.TypeOf(c.step.ExpectedState).String()
			states[stateName] = append(states[stateName], c.step)
		}

		if any(c.step.GivenCommand) != nil {
			commandName := reflect.TypeOf(c.step.GivenCommand).String()
			commands[commandName] = append(commands[commandName], c.step)
		}

		if any(c.step.InitState) != nil {
			stateName := reflect.TypeOf(c.step.InitState).String()
			states[stateName] = append(states[stateName], c.step)
		}
	}

	for _, seed := range rand.Perm(len(suite.cases) * 100) {
		r.Seed(int64(seed))

		var step Step[D, C, S]
		var state S

		// randomly select step with command
		for _, steps := range commands {
			if r.Float64() < 0.1 {
				step = steps[r.Intn(len(steps))]
				break
			}
		}

		// randomly select step with state
		for _, steps := range states {
			if r.Float64() < 0.1 {
				state = steps[r.Intn(len(steps))].ExpectedState
				break
			}
		}

		// with some chance keep previous state, or randomly select new state
		// this helps to generate new states, that can succeed after applying command
		prob := r.Float64()
		if prob < 0.3 {
			state = suite.cases[r.Intn(len(suite.cases))].step.ExpectedState
		} else if prob < 0.6 {
			// explore also initial states
			var zeroState S
			state = zeroState
		}

		m := suite.mkMachine(suite.dep, state)
		// Before and After commands can have assertions, when we fuzzing we don't want to run them
		if step.BeforeCommand != nil {
			step.BeforeCommand(zeroT, suite.dep)
		}
		if step.AfterCommand != nil {
			step.AfterCommand(zeroT, suite.dep)
		}
		err := m.Handle(context.Background(), step.GivenCommand)
		newState := m.State()
		suite.infer.Record(step.GivenCommand, state, newState, err)
	}
}

// AssertSelfDocumentStateDiagram help to self document state machine transitions, just by running tests.
// It will compare current state diagram with stored in file.
// It will fail assertion if they are not equal. This may happen, when tests are changed, or state machine is changed.
// In both then visual inspection of state diagram helps to double-check if changes are correct. And use diagrams in documentation.
//
// If file does not exist, function will return true, to indicate that file should be created.
// For this purpose call SelfDocumentStateDiagram.
func (suite *Suite[D, C, S]) AssertSelfDocumentStateDiagram(t *testing.T, filename string) bool {
	suite.fuzzy()

	// extract fine name from file, if there is extension remove it
	fileName := filename + ".state_diagram.mmd"
	fileNameWithErrorTransitions := filename + ".state_diagram_with_errors.mmd"

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
		date, err := os.ReadFile(f.filename)
		if err != nil {
			if os.IsNotExist(err) {
				return true
			} else {
				assert.NoErrorf(t, err, "failed to read file %s", f.filename)
			}
		}

		// if stored content is not equal, fail assertion
		if diff := cmp.Diff(string(date), mermaidDiagram); diff != "" {
			t.Fatalf("unexpected state diagram (-want +got):\n%s", diff)
			return false
		}
	}

	return false
}

// SelfDocumentStateDiagram help to self document state machine transitions, just by running tests.
// It will always overwrite stored state diagram files, useful in TDD loop, when tests are being written.
func (suite *Suite[D, C, S]) SelfDocumentStateDiagram(t *testing.T, filename string) {
	suite.fuzzy()

	// extract fine name from file, if there is extension remove it
	fileName := filename + ".state_diagram.mmd"
	fileNameWithErrorTransitions := filename + ".state_diagram_with_errors.mmd"

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

type TestingT interface {
	Errorf(format string, args ...interface{})
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

var zeroT testing.TB = &testing.T{}
