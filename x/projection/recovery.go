package projection

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"time"
)

var (
	ErrMaxRecoveryAttemptsReached = errors.New("max recovery attempts reached")
)

const (
	RecoveryRecordType = "recovery-state"
)

func NewRecoveryOptions(id string, init func() SnapshotState, store schemaless.Repository[SnapshotState]) *RecoveryOptions[SnapshotState] {
	return &RecoveryOptions[SnapshotState]{
		id:                  id,
		init:                init,
		store:               store,
		maxRecoveryAttempts: 3,
		autoSnapshot:        true,
	}
}

type RecoveryOptions[A SnapshotState] struct {
	id                  string
	init                func() A
	store               schemaless.Repository[A]
	maxRecoveryAttempts uint8
	autoSnapshot        bool
}

func (options *RecoveryOptions[A]) WithAutoSnapshot(x bool) *RecoveryOptions[A] {
	options.autoSnapshot = x
	return options
}

func (options *RecoveryOptions[A]) WithMaxRecoveryAttempts(x uint8) *RecoveryOptions[A] {
	options.maxRecoveryAttempts = x
	return options
}

func (options *RecoveryOptions[A]) SnapshotFrom(x SnapshotContext) error {
	if x == nil {
		return fmt.Errorf("projection.RecoveryOptions: SnapshotFrom: nil context")
	}

	y, ok := any(x.CurrentState()).(A)
	if !ok {
		return fmt.Errorf("projection.RecoveryOptions: SnapshotFrom: invalid context type %T, expects %T", x, options.init)
	}
	return options.Snapshot(y)
}

func (options *RecoveryOptions[A]) Snapshot(x A) error {
	record := schemaless.Record[A]{
		ID:      options.id,
		Type:    RecoveryRecordType,
		Data:    x,
		Version: 0,
	}
	saving := schemaless.Save(record)
	saving.UpdatingPolicy = schemaless.PolicyOverwriteServerChanges

	updated, err := options.store.UpdateRecords(saving)
	if err != nil {
		return fmt.Errorf("projection.RecoveryOptions: save last snapthot in store; %w", err)
	}

	for _, v := range updated.Saved {
		log.Debugf("projection.RecoveryOptions: save last snapthot in store; %d, %#v", v.Version, v.Data)
		MatchSnapshotStateR0(
			v.Data,
			func(x *PullPushContextState) {
				if x.Offset != nil {
					log.Debugf("projection.RecoveryOptions: offset: %s", *x.Offset)
				}
			},
			func(x *JoinContextState) {
				if x.Offset1 != nil {
					log.Debugf("projection.RecoveryOptions: offset1: %s", *x.Offset1)
				}
				if x.Offset2 != nil {
					log.Debugf("projection.RecoveryOptions: offset2: %s", *x.Offset2)
				}
			},
		)
	}

	return nil
}

func (options *RecoveryOptions[A]) LatestSnapshot() (A, error) {
	record, err := options.store.Get(options.id, RecoveryRecordType)
	if err != nil {
		if errors.Is(err, schemaless.ErrNotFound) {
			return options.init(), nil
		}
		var zero A
		return zero, fmt.Errorf("projection.RecoveryOptions: load last snapshot in store; %w", err)
	}

	MatchSnapshotStateR0(
		record.Data,
		func(x *PullPushContextState) {
			if x.Offset != nil {
				log.Debugf("projection.RecoveryOptions: load offset: %s", *x.Offset)
			}
		},
		func(x *JoinContextState) {
			if x.Offset1 != nil {
				log.Debugf("projection.RecoveryOptions: load offset1: %s", *x.Offset1)
			}
			if x.Offset2 != nil {
				log.Debugf("projection.RecoveryOptions: load offset2: %s", *x.Offset2)
			}
		},
	)
	return any(record.Data).(A), nil
}

func Recovery[T SnapshotState, A, B any](
	recovery *RecoveryOptions[SnapshotState],
	buildCtx func(T) (*PushAndPullInMemoryContext[A, B], error),
	f func(ctx *PushAndPullInMemoryContext[A, B]) error,
) error {
	maxAttempts := recovery.maxRecoveryAttempts

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			log.Debugf("projection.Recovery: context done")
			return nil

		default:
			state, err := recovery.LatestSnapshot()
			if err != nil {
				return fmt.Errorf("projection.Recovery: load last state in store; %w", err)
			}

			context, err := buildCtx(any(state).(T))
			if err != nil {
				return fmt.Errorf("projection.Recovery: build context; %w", err)
			}

			if recovery.autoSnapshot {
				go func() {
					for {
						select {
						case <-ctx.Done():
							return
						case <-time.After(100 * time.Millisecond):
							err := recovery.SnapshotFrom(context)
							if err != nil {
								log.Errorf("projection.Recovery: save last state in store; %s", err)
								cancel()
								return
							}
						}
					}
				}()
			}

			err = f(context)
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
}
