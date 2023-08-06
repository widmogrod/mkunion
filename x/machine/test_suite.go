package machine

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"testing"
)

func NewTestSuite[TCommand, TState any](mkMachine func() *Machine[TCommand, TState]) *Suite[TCommand, TState] {
	infer := NewInferTransition[TCommand, TState]()
	return &Suite[TCommand, TState]{
		mkMachine: mkMachine,
		infer:     infer,
	}
}

type Suite[TCommand, TState any] struct {
	mkMachine func() *Machine[TCommand, TState]
	infer     *InferTransition[TCommand, TState]
	then      []*Case[TCommand, TState]
}

type Case[TCommand, TState any] struct {
	name    string
	command []TCommand
	state   []TState
	err     []error
	then    [][]*Case[TCommand, TState]
}

func (c *Case[TCommand, TState]) next() {
	var zeroCmd TCommand
	var zeroState TState
	var zeroErr error

	c.command = append(c.command, zeroCmd)
	c.state = append(c.state, zeroState)
	c.err = append(c.err, zeroErr)
	c.then = append(c.then, nil)
}

func (c *Case[TCommand, TState]) index() int {
	return len(c.command) - 1
}

// GivenCommand starts building assertion that when command is applied to machine, it will result in given state or error.
// Use this method always with ThenState or ThenStateAndError
func (c *Case[TCommand, TState]) GivenCommand(cmd TCommand) *Case[TCommand, TState] {
	c.next()
	c.command[c.index()] = cmd
	return c
}

// ThenState asserts that command applied to machine will result in given state
func (c *Case[TCommand, TState]) ThenState(state TState) *Case[TCommand, TState] {
	c.state[c.index()] = state
	c.err[c.index()] = nil
	return c
}

// ForkCase takes previous state of machine and allows to apply another case from this point onward
// there can be many forks from one state
func (c *Case[TCommand, TState]) ForkCase(name string, definition func(c *Case[TCommand, TState])) *Case[TCommand, TState] {
	useCase := &Case[TCommand, TState]{name: name}
	definition(useCase)
	c.then[c.index()] = append(c.then[c.index()], useCase)
	return c
}

func (c *Case[TCommand, TState]) ThenStateAndError(state TState, err error) {
	c.state[c.index()] = state
	c.err[c.index()] = err
}

func (suite *Suite[TCommand, TState]) Case(name string, init func(c *Case[TCommand, TState])) {
	useCase := &Case[TCommand, TState]{
		name: name,
	}
	init(useCase)

	suite.then = append(suite.then, useCase)
}

// Run runs all test then that describe state machine transitions
func (suite *Suite[TCommand, TState]) Run(t *testing.T) {
	t.Helper()
	for _, c := range suite.then {
		m := suite.mkMachine()
		suite.assert(t, c, m)
	}
}

func (suite *Suite[TCommand, TState]) assert(t *testing.T, c *Case[TCommand, TState], m *Machine[TCommand, TState]) bool {
	return t.Run(c.name, func(t *testing.T) {
		for idx, cmd := range c.command {
			state := m.State()
			t.Run(fmt.Sprintf("Apply(cmd=%T, state=%T)", cmd, state), func(t *testing.T) {
				err := m.Handle(cmd)
				newState := m.State()

				if c.err[idx] == nil {
					assert.NoError(t, err)
				} else {
					assert.ErrorAs(t, err, &c.err[idx])
				}

				assert.Equal(t, c.state[idx], newState)

				suite.infer.Record(cmd, state, newState, err)

				if len(c.then[idx]) > 0 {
					for _, then := range c.then[idx] {
						m := *m
						suite.assert(t, then, &m)
					}
				}
			})
		}
	})
}

// Fuzzy takes commands and states from recorded transitions and tries to find all possible combinations of commands and states.
// This can help complete state diagrams with missing transitions, or find errors in state machine that haven't been tested yet.
// It's useful when connected with AssertSelfDocumentStateDiagram, to automatically update state diagram.
func (suite *Suite[TCommand, TState]) Fuzzy(t *testing.T) {
	t.Helper()

	r := rand.New(rand.NewSource(0))

	m := suite.mkMachine()
	var states []TState
	var commands []TCommand

	then := suite.then
	for len(then) > 0 {
		c := then[0]
		then = then[1:]
		for _, cmd := range c.command {
			commands = append(commands, cmd)
		}
		for _, state := range c.state {
			states = append(states, state)
		}
		for _, t := range c.then {
			for _, tt := range t {
				then = append(then, tt)
			}
		}
	}

	for _, seed := range rand.Perm(len(states) * len(commands) * 10) {
		//r.Seed(int64(seed))
		_ = seed
		// randomly select command and state
		cmd := commands[r.Intn(len(commands))]

		// with some chance keep previous state, or randomly select new state
		// this helps to generate new states, that can succeed after applying command
		prob := r.Float64()
		if prob < 0.3 {
			m.state = states[r.Intn(len(states))]
		} else if prob < 0.6 {
			// explore also initial states
			var zeroState TState
			m.state = zeroState
		}

		state := m.State()
		err := m.Handle(cmd)
		newState := m.State()
		suite.infer.Record(cmd, state, newState, err)
	}
}

// AssertSelfDocumentStateDiagram help to self document state machine transitions, just by running tests.
// It will compare current state diagram with stored in file.
// It will fail assertion if they are not equal. This may happen, when tests are changed, or state machine is changed.
// In both then visual inspection of state diagram helps to double-check if changes are correct. And use diagrams in documentation.
//
// If file does not exist, function will return true, to indicate that file should be created.
// For this purpose call SelfDocumentStateDiagram.
func (suite *Suite[TCommand, TState]) AssertSelfDocumentStateDiagram(t *testing.T, baseFileName string) (shouldSelfDocument bool) {
	// extract fine name from file, if there is extension remove it
	fileName := baseFileName + ".state_diagram.mmd"
	fileNameWithErrorTransitions := baseFileName + ".state_diagram_with_errors.mmd"

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
		assert.Equalf(t, string(date), mermaidDiagram, "state diagram is not equal to stored in file %s", f.filename)
	}

	return false
}

// SelfDocumentStateDiagram help to self document state machine transitions, just by running tests.
// It will always overwrite stored state diagram files, useful in TDD loop, when tests are being written.
func (suite *Suite[TCommand, TState]) SelfDocumentStateDiagram(t *testing.T, baseFileName string) {
	// extract fine name from file, if there is extension remove it
	fileName := baseFileName + ".state_diagram.mmd"
	fileNameWithErrorTransitions := baseFileName + ".state_diagram_with_errors.mmd"

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

func (suite *Suite[TCommand, TState]) SelfDocumentTitle(title string) {
	suite.infer.WithTitle(title)
}
