package machine

import "context"

func NewMachine[D, C, S any](d D, f func(context.Context, D, C, S) (S, error), state S) *Machine[D, C, S] {
	return &Machine[D, C, S]{
		di:     d,
		handle: f,
		state:  state,
	}
}

func NewSimpleMachine[C, S any](f func(C, S) (S, error)) *Machine[any, C, S] {
	var s S
	return NewSimpleMachineWithState(f, s)
}

func NewSimpleMachineWithState[C, S any](f func(C, S) (S, error), state S) *Machine[any, C, S] {
	return &Machine[any, C, S]{
		di: nil,
		handle: func(ctx context.Context, a any, c C, s S) (S, error) {
			return f(c, s)
		},
		state: state,
	}
}

type Machine[D, C, S any] struct {
	di     D
	state  S
	handle func(context.Context, D, C, S) (S, error)
}

func (o *Machine[D, C, S]) Handle(ctx context.Context, cmd C) error {
	state, err := o.handle(ctx, o.di, cmd, o.state)
	if err != nil {
		return err
	}

	o.state = state
	return nil
}

func (o *Machine[D, C, S]) State() S {
	return o.state
}

func (o *Machine[D, C, S]) Dep() D {
	return o.di
}
