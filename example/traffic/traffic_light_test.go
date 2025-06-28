package traffic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/machine"
)

// --8<-- [start:basic-test]
func TestTrafficLightTransitions(t *testing.T) {
	deps := Dependencies{}
	ctx := context.Background()

	// Test red -> green transition
	state, err := Transition(ctx, deps, &NextCMD{}, &RedLight{})
	assert.NoError(t, err)
	assert.IsType(t, &GreenLight{}, state)

	// Test green -> yellow transition
	state, err = Transition(ctx, deps, &NextCMD{}, &GreenLight{})
	assert.NoError(t, err)
	assert.IsType(t, &YellowLight{}, state)

	// Test yellow -> red transition
	state, err = Transition(ctx, deps, &NextCMD{}, &YellowLight{})
	assert.NoError(t, err)
	assert.IsType(t, &RedLight{}, state)
}

// --8<-- [end:basic-test]

// --8<-- [start:test-suite]
func TestTrafficLightMachine(t *testing.T) {
	suite := machine.NewTestSuite[Dependencies](Dependencies{}, NewMachine)

	suite.Case(t, "traffic light cycle", func(t *testing.T, c *machine.Case[Dependencies, TrafficCommand, TrafficState]) {
		// Start with a red light (default)
		c.
			GivenCommand(&NextCMD{}).
			ThenState(t, &GreenLight{}).
			ForkCase(t, "continue cycle", func(t *testing.T, c *machine.Case[Dependencies, TrafficCommand, TrafficState]) {
				c.
					GivenCommand(&NextCMD{}).
					ThenState(t, &YellowLight{}).
					ForkCase(t, "complete cycle", func(t *testing.T, c *machine.Case[Dependencies, TrafficCommand, TrafficState]) {
						c.
							GivenCommand(&NextCMD{}).
							ThenState(t, &RedLight{})
					})
			})
	})

	// Generate state diagrams
	if suite.AssertSelfDocumentStateDiagram(t, "traffic_light_test.go") {
		suite.SelfDocumentStateDiagram(t, "traffic_light_test.go")
	}
}

// --8<-- [end:test-suite]
