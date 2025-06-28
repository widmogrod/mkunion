package machine

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/typedful"
)

// --8<-- [start:process-order]
// ProcessOrderCommandWithConcurrency demonstrates handling concurrent state updates
// with optimistic concurrency control and retry logic
func ProcessOrderCommandWithConcurrency(ctx context.Context, store schemaless.Repository[schema.Schema], orderID string, cmd Command) error {
	repo := typedful.NewTypedRepository[State](store)
	deps := Dependencies{}

	// Retry loop for handling version conflicts
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		// --8<-- [start:retry-loop]
		// Load current state with version
		record, err := repo.Get(orderID, "orders")
		if err != nil {
			if errors.Is(err, schemaless.ErrNotFound) {
				// First command - create new record
				record = schemaless.Record[State]{
					ID:      orderID,
					Type:    "orders",
					Data:    &OrderPending{}, // Initial state
					Version: 0,
				}
			} else {
				return fmt.Errorf("failed to load state: %w", err)
			}
		}

		// Create machine with current state
		machine := NewMachine(deps, record.Data)

		// Apply command
		err = machine.Handle(ctx, cmd)
		if err != nil {
			return fmt.Errorf("command failed: %w", err)
		}

		// Attempt to save with version check
		record.Data = machine.State()
		_, err = repo.UpdateRecords(schemaless.Save(record))

		if err == nil {
			// Success!
			return nil
		}

		if errors.Is(err, schemaless.ErrVersionConflict) {
			// Another process updated the state
			// Log and retry
			log.Printf("Version conflict on attempt %d, retrying...", attempt+1)

			// Exponential backoff
			backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
			time.Sleep(backoff)
			continue
		}
		// --8<-- [end:retry-loop]

		// Other error - don't retry
		return fmt.Errorf("failed to save state: %w", err)
	}

	return fmt.Errorf("max retries exceeded due to version conflicts")
}

// --8<-- [end:process-order]

// --8<-- [start:batch-operations]
// ProcessBulkOrdersWithConcurrency demonstrates batch operations with concurrency control
func ProcessBulkOrdersWithConcurrency(ctx context.Context, store schemaless.Repository[schema.Schema], updates map[string]Command) error {
	repo := typedful.NewTypedRepository[State](store)
	deps := Dependencies{}

	// Load all records first
	records := make(map[string]schemaless.Record[State])
	for orderID := range updates {
		record, err := repo.Get(orderID, "orders")
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", orderID, err)
		}
		records[orderID] = record
	}

	// Process all commands
	var recordsToSave []schemaless.Record[State]
	for orderID, cmd := range updates {
		record := records[orderID]
		machine := NewMachine(deps, record.Data)

		if err := machine.Handle(ctx, cmd); err != nil {
			return fmt.Errorf("command failed for %s: %w", orderID, err)
		}

		record.Data = machine.State()
		recordsToSave = append(recordsToSave, record)
	}

	// Save all at once with version checking
	_, err := repo.UpdateRecords(schemaless.Save(recordsToSave...))

	if errors.Is(err, schemaless.ErrVersionConflict) {
		// Handle partial failures
		// In a real implementation, you'd check which records failed and retry those
		return fmt.Errorf("version conflict in batch update: %w", err)
	}

	return err
}

// --8<-- [end:batch-operations]

// --8<-- [start:retry-helper]
// retryWithBackoff is a helper function for retry logic
func retryWithBackoff(ctx context.Context, maxRetries int, operation func() error) error {
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		if !errors.Is(err, schemaless.ErrVersionConflict) {
			return err // Don't retry non-conflict errors
		}

		if attempt < maxRetries-1 {
			backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return fmt.Errorf("max retries exceeded")
}

// --8<-- [end:retry-helper]
