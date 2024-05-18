package machine

import (
	"fmt"
	"testing"
)

func TestSuite_Run(t *testing.T) {
	m := NewSimpleMachineWithState(func(cmd string, state int) (int, error) {
		switch cmd {
		case "inc":
			return state + 1, nil
		case "dec":
			return state - 1, nil
		default:
			return 0, fmt.Errorf("unknown cmd: %s", cmd)
		}
	}, 10)

	suite := NewTestSuite(nil, func(dep any, init int) *Machine[any, string, int] {
		return m
	})
	suite.Case(t, "inc", func(t *testing.T, c *Case[any, string, int]) {
		c.GivenCommand("inc").ThenState(t, 11)
		c.GivenCommand("inc").ThenState(t, 12)
		c.GivenCommand("inc").ThenState(t, 13)
	})

	if suite.AssertSelfDocumentStateDiagram(t, "test_suite_test.go") {
		suite.SelfDocumentStateDiagram(t, "test_suite_test.go")
	}
}
