---
title: Simple State Machine Examples
---

# Simple State Machine Examples

This document provides simple, easy-to-understand examples of state machines using mkunion. These examples are designed to help you grasp the core concepts before moving on to more complex scenarios.

## Traffic Light Example

A traffic light is a classic example of a state machine with three states: Red, Yellow, and Green.

### Model Definition

```go
package traffic

//go:tag mkunion:"TrafficState"
type (
    RedLight    struct{}
    YellowLight struct{}
    GreenLight  struct{}
)

//go:tag mkunion:"TrafficCommand"
type (
    NextCMD struct{} // Move to next state in sequence
)
```

### Transition Function

```go
package traffic

import (
    "context"
    "fmt"
)

// Simple traffic light with no dependencies
type Dependencies struct{}

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
```

### Testing

```go
package traffic

import (
    "testing"
    "github.com/widmogrod/mkunion/x/machine"
)

func TestTrafficLight(t *testing.T) {
    deps := Dependencies{}
    suite := machine.NewTestSuite(func() *machine.Machine[Dependencies, TrafficCommand, TrafficState] {
        return machine.NewMachine(deps, Transition, &RedLight{})
    })

    // Test the cycle: Red -> Green -> Yellow -> Red
    suite.Case("Traffic light cycle",
        suite.GivenCommand(&NextCMD{}),
        suite.ThenState(&GreenLight{}),
        
        suite.GivenCommand(&NextCMD{}),
        suite.ThenState(&YellowLight{}),
        
        suite.GivenCommand(&NextCMD{}),
        suite.ThenState(&RedLight{}),
    )

    suite.Run(t)
}
```

## Toggle Switch Example

An even simpler example: a toggle switch with just two states.

### Model Definition

```go
package toggle

//go:tag mkunion:"SwitchState"
type (
    On  struct{}
    Off struct{}
)

//go:tag mkunion:"SwitchCommand"
type (
    ToggleCMD struct{}
)
```

### Transition Function

```go
package toggle

import "context"

type Dependencies struct{}

func Transition(ctx context.Context, deps Dependencies, cmd SwitchCommand, state SwitchState) (SwitchState, error) {
    return MatchSwitchCommandR2(cmd,
        func(c *ToggleCMD) (SwitchState, error) {
            return MatchSwitchStateR2(state,
                func(s *On) (SwitchState, error) {
                    return &Off{}, nil
                },
                func(s *Off) (SwitchState, error) {
                    return &On{}, nil
                },
            )
        },
    )
}
```

## Door Lock Example with State Data

This example shows how to include data in states and validate commands.

### Model Definition

```go
package door

import "time"

//go:tag mkunion:"DoorState"
type (
    Locked struct {
        LockedAt time.Time
        LockedBy string
    }
    Unlocked struct {
        UnlockedAt time.Time
        UnlockedBy string
    }
)

//go:tag mkunion:"DoorCommand"
type (
    LockCMD struct {
        UserID string
    }
    UnlockCMD struct {
        UserID string
        Code   string
    }
)
```

### Transition Function with Validation

```go
package door

import (
    "context"
    "fmt"
    "time"
)

type Dependencies struct {
    ValidateLockCode func(userID, code string) bool
}

func Transition(ctx context.Context, deps Dependencies, cmd DoorCommand, state DoorState) (DoorState, error) {
    return MatchDoorCommandR2(cmd,
        func(c *LockCMD) (DoorState, error) {
            if c.UserID == "" {
                return nil, fmt.Errorf("user ID is required")
            }
            
            // Can only lock if currently unlocked
            return MatchDoorStateR2(state,
                func(s *Locked) (DoorState, error) {
                    return nil, fmt.Errorf("door is already locked")
                },
                func(s *Unlocked) (DoorState, error) {
                    return &Locked{
                        LockedAt: time.Now(),
                        LockedBy: c.UserID,
                    }, nil
                },
            )
        },
        func(c *UnlockCMD) (DoorState, error) {
            if c.UserID == "" || c.Code == "" {
                return nil, fmt.Errorf("user ID and code are required")
            }
            
            // Validate the unlock code
            if !deps.ValidateLockCode(c.UserID, c.Code) {
                return nil, fmt.Errorf("invalid unlock code")
            }
            
            // Can only unlock if currently locked
            return MatchDoorStateR2(state,
                func(s *Locked) (DoorState, error) {
                    return &Unlocked{
                        UnlockedAt: time.Now(),
                        UnlockedBy: c.UserID,
                    }, nil
                },
                func(s *Unlocked) (DoorState, error) {
                    return nil, fmt.Errorf("door is already unlocked")
                },
            )
        },
    )
}
```

## Key Takeaways

1. **Start Simple**: Begin with state machines that have few states and commands
2. **States Can Hold Data**: States aren't limited to empty structs - they can contain relevant data
3. **Validation is Important**: Always validate commands before performing transitions
4. **Exhaustive Matching**: The generated match functions ensure you handle all cases
5. **Dependencies are Optional**: Simple state machines might not need any dependencies

## Next Steps

- Review the [comprehensive Order Service example](state_machine.md) for a more complex scenario
- Learn about [testing strategies](state_machine.md#testing-state-machines--self-documenting) for state machines
- Explore [advanced patterns](#advanced-patterns) for composition and async operations