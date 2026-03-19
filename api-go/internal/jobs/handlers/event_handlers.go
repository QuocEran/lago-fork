package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
)

const TaskTypeValidateEvents = "events:validate"

// HandleValidateEvents is a stub for the events post-ingestion validation job.
// In Rails this refreshes a materialized view and dispatches per-org validation jobs.
// Here it logs and returns nil — the full implementation depends on ClickHouse integration (lago-fork-2bg).
//
// Dead-letter: returns SkipRetry on non-retryable context cancellation.
func HandleValidateEvents() asynq.HandlerFunc {
	return func(ctx context.Context, _ *asynq.Task) error {
		if ctx.Err() != nil {
			slog.Error("validate_events: context cancelled, sending to dead-letter",
				slog.String("error", ctx.Err().Error()),
			)
			return fmt.Errorf("context cancelled: %w", asynq.SkipRetry)
		}

		// TODO(lago-fork-2bg): refresh events materialized view and dispatch org-level validation.
		slog.Info("validate_events: stub executed (ClickHouse integration pending)")
		return nil
	}
}
