package main

import (
	"context"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
	"github.com/widmogrod/mkunion/x/taskqueue"
	"github.com/widmogrod/mkunion/x/workflow"
)

func backgroundScheduled(di *workflow.DI, statesRepo *typedful.TypedRepoWithAggregator[workflow.State, any]) (*taskqueue.FunctionProcessor[schemaless.Record[workflow.State]], *taskqueue.Description) {
	procScheduled := &taskqueue.FunctionProcessor[schemaless.Record[workflow.State]]{
		F: func(task taskqueue.Task[schemaless.Record[workflow.State]]) {
			work := workflow.NewMachine(di, task.Data.Data)
			err := work.Handle(context.TODO(), &workflow.Run{})
			if err != nil {
				log.Errorf("err: %s", err)
				return
			}

			newState := work.State()
			//log.Infof("newState: %T", newState)

			saving := []schemaless.Record[workflow.State]{
				{
					ID:      task.Data.ID,
					Data:    newState,
					Type:    task.Data.Type,
					Version: task.Data.Version,
				},
			}

			if next := workflow.ScheduleNext(newState, di); next != nil {
				work := workflow.NewMachine(di, nil)
				err := work.Handle(context.TODO(), next)
				if err != nil {
					log.Infof("err: %s", err)
					return
				}

				//log.Infof("next id=%s", workflow.GetRunIDFromBaseState(work.State()))

				saving = append(saving, schemaless.Record[workflow.State]{
					ID:   workflow.GetRunIDFromBaseState(work.State()),
					Type: task.Data.Type,
					Data: work.State(),
				})
			}

			_, err = statesRepo.UpdateRecords(schemaless.Save(saving...))
			if err != nil {
				if errors.Is(err, schemaless.ErrVersionConflict) {
					// make it configurable, but by default we should
					// just ignore conflicts, since that means we may have duplicate,
					// or some other process already update it.
					// assuming that queue is populated from stream of changes
					// it such case (there was update) new message with new version
					// will land in queue soon (if it pass selector)
					log.Warnf("version conflict, ignoring: %s", err.Error())
				} else {
					panic(err)
				}
			}
		},
	}

	// there can be few process,
	// - timeout out workflow (command to timeout)
	// - retry workflow (command to retry)
	// - run workflow (command to run)
	// - callback workflow (command to callback)
	// - terminate workflow (command to terminate)
	// - complete workflow (command to complete)

	descScheduled := &taskqueue.Description{
		Change: []string{"create"},
		Entity: "process",
		Filter: `Data["workflow.Scheduled"].ExpectedRunTimestamp <= :now 
			AND  Data["workflow.Scheduled"].ExpectedRunTimestamp > 0`,
	}
	return procScheduled, descScheduled
}

func backgroundRetry(di *workflow.DI, statesRepo *typedful.TypedRepoWithAggregator[workflow.State, any]) (*taskqueue.FunctionProcessor[schemaless.Record[workflow.State]], *taskqueue.Description) {
	procRetry := &taskqueue.FunctionProcessor[schemaless.Record[workflow.State]]{
		F: func(task taskqueue.Task[schemaless.Record[workflow.State]]) {
			log := log.WithField("runID", task.Data.ID)
			log.Infof("start retry")

			work := workflow.NewMachine(di, task.Data.Data)
			err := work.Handle(context.TODO(), &workflow.TryRecover{
				RunID: task.Data.ID,
			})
			if err != nil {
				log.Errorf("err: %s", err)
				return
			}

			newState := work.State()

			_, err = statesRepo.UpdateRecords(schemaless.Save(schemaless.Record[workflow.State]{
				ID:      task.Data.ID,
				Data:    newState,
				Type:    task.Data.Type,
				Version: task.Data.Version,
			}))
			if err != nil {
				if errors.Is(err, schemaless.ErrVersionConflict) {
					// make it configurable, but by default we should
					// just ignore conflicts, since that means we may have duplicate,
					// or some other process already update it.
					// assuming that queue is populated from stream of changes
					// it such case (there was update) new message with new version
					// will land in queue soon (if it pass selector)
					log.Warnf("version conflict, ignoring: %s", err.Error())
				} else {
					log.Panicf("err: %s", err)
				}
			}

			log.Infof("successful retry")
		},
	}

	descRetry := &taskqueue.Description{
		Change: []string{"create", "update"},
		Entity: "process",
		Filter: `Data["workflow.Error"].Retried < Data["workflow.Error"].BaseState.DefaultMaxRetries 
			AND  Version < :now`,
		//Filter: `After.Data["workflow.Error"].Retried < After.Data["workflow.Error"].BaseState.DefaultMaxRetries
		//	AND  After.Version <> :now
		//	AND  After.Type = "process"`,
	}

	return procRetry, descRetry
}
