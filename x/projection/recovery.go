package projection

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

var (
	ErrMaxRecoveryAttemptsReached = errors.New("max recovery attempts reached")
)

func NewRecoveryOptions[A any](id string, init A, store schemaless.Repository[A]) *RecoveryOptions[A] {
	return &RecoveryOptions[A]{
		id:                  id,
		init:                init,
		store:               store,
		maxRecoveryAttempts: 3,
	}
}

type RecoveryOptions[A any] struct {
	id                  string
	init                A
	store               schemaless.Repository[A]
	maxRecoveryAttempts uint8
}

func (ctx *RecoveryOptions[A]) WithMaxRecoveryAttempts(x uint8) *RecoveryOptions[A] {
	ctx.maxRecoveryAttempts = x
	return ctx
}

func Recovery[A any](ctx *RecoveryOptions[A], f func(state A) error) error {
	maxAttempts := ctx.maxRecoveryAttempts

	for {
		state := ctx.init
		record, err := ctx.store.Get(ctx.id, "recovery-state")
		if err != nil {
			if !errors.Is(err, schemaless.ErrNotFound) {
				return fmt.Errorf("projection.Recovery: load last state in store; %w", err)
			}
		} else {
			state = record.Data
		}

		err = f(state)
		if err == nil {
			return nil
		}

		log.Debugf("projection.Recovery: recent operation error %s; %d attempts left", err, maxAttempts)

		if maxAttempts == 0 {
			return fmt.Errorf("projection.Recovery: last operation error %w; %w", err, ErrMaxRecoveryAttemptsReached)
		}

		maxAttempts--
	}
}
