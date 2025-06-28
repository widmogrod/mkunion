package traffic

import (
	"context"
	"fmt"

	"github.com/widmogrod/mkunion/x/machine"
)

// NewMachine creates a new traffic light state machine
func NewMachine(deps Dependencies, state TrafficState) *machine.Machine[Dependencies, TrafficCommand, TrafficState] {
	if state == nil {
		state = &RedLight{} // Default to red light
	}
	return machine.NewMachine(deps, Transition, state)
}

// --8<-- [start:transition]
// Transition defines the traffic light state transitions
func Transition(ctx context.Context, deps Dependencies, cmd TrafficCommand, state TrafficState) (TrafficState, error) {
	return MatchTrafficCommandR2(cmd,
		func(c *NextCMD) (TrafficState, error) {
			return MatchTrafficStateR2(state,
				func(s *RedLight) (TrafficState, error) {
					return &GreenLight{}, nil
				},
				func(s *YellowLight) (TrafficState, error) {
					return &RedLight{}, nil
				},
				func(s *GreenLight) (TrafficState, error) {
					return &YellowLight{}, nil
				},
			)
		},
	)
}

// --8<-- [end:transition]

// --8<-- [start:example]
// Example demonstrates using the traffic light state machine
func Example() {
	// Create a new traffic light starting at red
	m := NewMachine(Dependencies{}, &RedLight{})

	// Cycle through the lights
	ctx := context.Background()
	for i := 0; i < 6; i++ {
		state := m.State()
		fmt.Printf("Current light: %T\n", state)

		err := m.Handle(ctx, &NextCMD{})
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			break
		}
	}
	// Output:
	// Current light: *traffic.RedLight
	// Current light: *traffic.GreenLight
	// Current light: *traffic.YellowLight
	// Current light: *traffic.RedLight
	// Current light: *traffic.GreenLight
	// Current light: *traffic.YellowLight
}

// --8<-- [end:example]
