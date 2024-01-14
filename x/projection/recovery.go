package projection

import (
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
)

var (
	ErrMaxRecoveryAttemptsReached = errors.New("max recovery attempts reached")
)

func NewRecoveryOptions(id, pullTopic, pushTopic string, snapshotStore *SnapshotStore) *RecoveryOptions {
	return &RecoveryOptions{
		id:                  id,
		pullTopic:           pullTopic,
		pushTopic:           pushTopic,
		snapshotStore:       snapshotStore,
		maxRecoveryAttempts: 3,
	}
}

type RecoveryOptions struct {
	id                  string
	pullTopic           string
	pushTopic           string
	snapshotStore       *SnapshotStore
	maxRecoveryAttempts uint8
}

func (ctx *RecoveryOptions) WithMaxRecoveryAttempts(x uint8) *RecoveryOptions {
	ctx.maxRecoveryAttempts = x
	return ctx
}

func Recovery(ctx *RecoveryOptions, f func(state SnapshotState) error) error {
	maxAttempts := ctx.maxRecoveryAttempts

	for {
		state, err := ctx.snapshotStore.LoadLastSnapshot(ctx.id)
		if err != nil {
			if !errors.Is(err, ErrSnapshotNotFound) {
				return fmt.Errorf("projection.Recovery: load last state; %w", err)
			}
		}

		if state == nil {
			state = ctx.snapshotStore.InitSnapshot(ctx.id, ctx.pullTopic, ctx.pushTopic)
		}

		err = f(*state)
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
