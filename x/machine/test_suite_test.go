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

	suite := NewTestSuite(func() *Machine[string, int] { return m })
	suite.Case("inc", func(c *Case[string, int]) {
		c.GivenCommand("inc").ThenState(11)
		c.GivenCommand("inc").ThenState(12)
		c.GivenCommand("inc").ThenState(13)
	})

	suite.Run(t)
	suite.Fuzzy(t)

	suite.SelfDocumentTitle("SimpleMachine")
	if suite.AssertSelfDocumentStateDiagram(t, "test_suite_test.go") {
		suite.SelfDocumentStateDiagram(t, "test_suite_test.go")
	}
}
