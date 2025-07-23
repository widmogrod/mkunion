package example

import "errors"

//go:tag mkunion:"Command"
type (
	StartCommand struct {
		ID string
	}
	CompleteCommand struct {
		Result string
	}
)

//go:tag mkunion:"State"
type (
	Initial    struct{}
	Processing struct {
		ID string
	}
	Complete struct{ Result string }
)

// --8<-- [start:match-def]

//go:tag mkmatch:"TransitionMatch"
type TransitionMatch[S State, C Command] interface {
	ProcessingStart(*Processing, *StartCommand)
	ProcessingComplete(*Processing, *CompleteCommand)
	InitialStart(*Initial, *StartCommand)
	Default(State, Command)
}

// --8<-- [end:match-def]
// --8<-- [start:match-use]

func Transition(state State, cmd Command) (State, error) {
	return TransitionMatchR2(
		state, cmd,
		func(s *Processing, c *StartCommand) (State, error) {
			return nil, errors.New("already processing")
		},
		func(s *Processing, c *CompleteCommand) (State, error) {
			return &Complete{Result: c.Result}, nil
		},
		func(s *Initial, c *StartCommand) (State, error) {
			return &Processing{ID: c.ID}, nil
		},
		func(s State, c Command) (State, error) {
			return nil, errors.New("invalid transition")
		},
	)
}

// --8<-- [start:match-use]
