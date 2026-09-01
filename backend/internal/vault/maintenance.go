package vault

import (
	"context"
	"log"
	"sort"
	"time"

	"github.com/shxntanu/aether/backend/internal/domain"
)

const (
	defaultPurgeInterval = time.Hour
	defaultPurgeBatch    = 100
)

// PurgeRunner performs one bounded attempt to permanently remove due
// documents. Implementations must honor ctx and return document-scoped
// failures in PurgeResult so one bad document does not stop maintenance.
type PurgeRunner interface {
	PurgeDue(context.Context, int) (PurgeResult, error)
}

// RunPurgeLoop performs a bounded purge immediately and repeats it at interval
// until ctx is canceled. It uses one goroutine owned by the caller and never
// creates a goroutine per document.
func RunPurgeLoop(
	ctx context.Context,
	runner PurgeRunner,
	interval time.Duration,
	batchSize int,
	logger *log.Logger,
) {
	if runner == nil {
		return
	}
	if interval <= 0 {
		interval = defaultPurgeInterval
	}
	if batchSize <= 0 {
		batchSize = defaultPurgeBatch
	}

	runPurgePass(ctx, runner, batchSize, logger)
	if ctx.Err() != nil {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runPurgePass(ctx, runner, batchSize, logger)
		}
	}
}

func runPurgePass(
	ctx context.Context,
	runner PurgeRunner,
	batchSize int,
	logger *log.Logger,
) {
	if ctx.Err() != nil {
		return
	}
	result, err := runner.PurgeDue(ctx, batchSize)
	if err != nil {
		logPurgeFailure(logger, "purge pass", err)
		return
	}

	failedIDs := make([]domain.DocumentID, 0, len(result.Failed))
	for id := range result.Failed {
		failedIDs = append(failedIDs, id)
	}
	sort.Slice(failedIDs, func(i, j int) bool { return failedIDs[i] < failedIDs[j] })
	for _, id := range failedIDs {
		logPurgeFailure(logger, "document "+string(id), result.Failed[id])
	}
	if logger != nil {
		logger.Printf(
			"purge pass completed purged=%d failed=%d",
			len(result.Purged),
			len(result.Failed),
		)
	}
}

func logPurgeFailure(logger *log.Logger, target string, err error) {
	if logger == nil {
		return
	}
	logger.Printf("purge failed target=%s error=%v", target, err)
}
