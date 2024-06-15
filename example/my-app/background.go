package main

import (
	"context"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
	"github.com/widmogrod/mkunion/x/taskqueue"
	"github.com/widmogrod/mkunion/x/workflow"
)

func backgroundScheduled(di *workflow.DI, statesRepo *typedful.TypedRepoWithAggregator[workflow.State, any]) (*taskqueue.FunctionProcessor[schemaless.Record[workflow.State]], *taskqueue.Description) {
	procScheduled := &taskqueue.FunctionProcessor[schemaless.Record[workflow.State]]{
		F: func(task taskqueue.Task[schemaless.Record[workflow.State]]) {
			log := log.
				WithField("op", "scheduled").
				WithField("runID", task.Data.ID).
				WithField("type", fmt.Sprintf("%T", task.Data.Data))

			work := workflow.NewMachine(di, task.Data.Data)
			err := work.Handle(context.TODO(), &workflow.Run{})
			if err != nil {
				log.Errorf("err: %s", err)
				return
			}

			newState := work.State()

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

			log.Infof("successful run")
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
		Filter: `Type == "process" 
            AND  Data["workflow.Scheduled"].ExpectedRunTimestamp <= :now 
			AND  Data["workflow.Scheduled"].ExpectedRunTimestamp > 0`,
	}
	return procScheduled, descScheduled
}

func backgroundRetry(di *workflow.DI, statesRepo *typedful.TypedRepoWithAggregator[workflow.State, any]) (*taskqueue.FunctionProcessor[schemaless.Record[workflow.State]], *taskqueue.Description) {
	procRetry := &taskqueue.FunctionProcessor[schemaless.Record[workflow.State]]{
		F: func(task taskqueue.Task[schemaless.Record[workflow.State]]) {
			log := log.
				WithField("op", "retry").
				WithField("runID", task.Data.ID).
				WithField("type", fmt.Sprintf("%T", task.Data.Data))

			switch sss := task.Data.Data.(type) {
			case *workflow.Error:
				log = log.
					WithField("retries", sss.Retried).
					WithField("version", task.Data.Version)
			}

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
		Filter: `Type == "process" 
			AND  Data["workflow.Error"].Retried < Data["workflow.Error"].BaseState.DefaultMaxRetries`,
		//AND Data["workflow.Error"].Code != "async-timeout"`,
	}

	return procRetry, descRetry
}

func backgroundTimeout(di *workflow.DI, statesRepo *typedful.TypedRepoWithAggregator[workflow.State, any]) (*taskqueue.FunctionProcessor[schemaless.Record[workflow.State]], *taskqueue.Description) {
	procTimeout := &taskqueue.FunctionProcessor[schemaless.Record[workflow.State]]{
		F: func(task taskqueue.Task[schemaless.Record[workflow.State]]) {
			log := log.
				WithField("op", "timeout").
				WithField("runID", task.Data.ID).
				WithField("type", fmt.Sprintf("%T", task.Data.Data))

			switch sss := task.Data.Data.(type) {
			case *workflow.Await:
				log = log.
					WithField("expectedTimeout", sss.ExpectedTimeoutTimestamp).
					WithField("version", task.Data.Version)
			}

			log.Infof("start timeout operation")
			work := workflow.NewMachine(di, task.Data.Data)
			err := work.Handle(context.TODO(), &workflow.ExpireAsync{
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
					log.Panicf("panic: %s", err)
				}
			}

			log.Infof("successful expire")
		},
	}

	descTimeout := &taskqueue.Description{
		Change: []string{"create", "update"},
		Entity: "process",
		Filter: `Type == "process" 
			AND  Data["workflow.Await"].ExpectedTimeoutTimestamp <= :now`,
	}

	return procTimeout, descTimeout
}
