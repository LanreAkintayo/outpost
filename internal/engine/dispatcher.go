package engine

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	// DefaultPollInterval is the default time between database polling cycles.
	DefaultPollInterval = 2 * time.Second

	// DefaultBatchSize is the default maximum number of delivery attempts claimed per cycle.
	DefaultBatchSize = 50

	// DefaultEnqueueTimeout is the maximum duration to wait for capacity in the worker pool queue.
	DefaultEnqueueTimeout = 2 * time.Second
)

// TaskSource abstracts the persistence layer for claiming and managing delivery attempts.
// This interface is satisfied by repository.PostgresDeliveryRepository.
type TaskSource interface {
	FetchAndClaimPending(ctx context.Context, batchSize int) ([]DeliveryTask, error)
	RevertToPending(ctx context.Context, attemptIDs []uuid.UUID) error
}

// DispatcherConfig tunes the database polling behavior of the background dispatcher.
type DispatcherConfig struct {
	PollInterval   time.Duration
	BatchSize      int
	EnqueueTimeout time.Duration
}

// Dispatcher manages background polling of pending webhook tasks and enqueues them into the worker pool.
type Dispatcher interface {
	Start()
	Stop()
}

type dispatcher struct {
	source TaskSource
	pool   WorkerPool
	cfg    DispatcherConfig
	logger zerolog.Logger

	stopChan  chan struct{}
	wg        sync.WaitGroup
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewDispatcher constructs a new Dispatcher with configurable polling parameters.
func NewDispatcher(source TaskSource, pool WorkerPool, cfg DispatcherConfig, log zerolog.Logger) Dispatcher {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = DefaultPollInterval
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if cfg.EnqueueTimeout <= 0 {
		cfg.EnqueueTimeout = DefaultEnqueueTimeout
	}

	return &dispatcher{
		source:   source,
		pool:     pool,
		cfg:      cfg,
		logger:   log,
		stopChan: make(chan struct{}),
	}
}

// Start launches the background polling loop in a dedicated goroutine.
func (d *dispatcher) Start() {
	d.startOnce.Do(func() {
		d.logger.Info().
			Dur("poll_interval", d.cfg.PollInterval).
			Int("batch_size", d.cfg.BatchSize).
			Msg("starting webhook delivery dispatcher")
		d.wg.Add(1)
		go d.dispatchLoop()
	})
}

// Stop gracefully signals the polling loop to stop and waits for in-flight DB operations to complete.
func (d *dispatcher) Stop() {
	d.stopOnce.Do(func() {
		d.logger.Info().Msg("stopping webhook delivery dispatcher")
		close(d.stopChan)
		d.wg.Wait()
		d.logger.Info().Msg("webhook delivery dispatcher stopped cleanly")
	})
}

// dispatchLoop continuously polls for pending tasks and enqueues them into the worker pool.
// It implements an adaptive drain pattern: if a full batch is retrieved, it immediately queries
// again without waiting for the next ticker tick.
func (d *dispatcher) dispatchLoop() {
	defer d.wg.Done()

	// Ticks every <pollinterval> time
	ticker := time.NewTicker(d.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.stopChan:
			d.logger.Info().Msg("dispatcher loop exiting on stop signal")
			return
		default:
		}

		// fetch delivery attempts and then dispatch to workers pool
		drainedCounts, err := d.pollAndDispatch()

		if err != nil {
			d.logger.Error().Err(err).Msg("failed to poll and dispatch")

		}

		// Straight up fetch more if there is still a lot in the database
		if drainedCounts >= d.cfg.BatchSize {
			continue
		}

		select {
		case <-d.stopChan:
			d.logger.Info().Msg("dispatcher loop exiting on stop signal")
			return
		case <-ticker.C:
		}

	}

}


// pollAndDispatch fetches a batch of pending tasks and submits each to the worker pool.
// Any task that fails to enqueue (e.g. due to queue backpressure or pool shutdown)
// is reverted to 'pending' in the database.
func (d *dispatcher) pollAndDispatch() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tasks, err := d.source.FetchAndClaimPending(ctx, d.cfg.BatchSize)
	if err != nil {
		return 0, err
	}

	if len(tasks) == 0 {
		return 0, nil
	}

	d.logger.Debug().Int("task_count", len(tasks)).Msg("claimed pending delivery tasks")

	var failedAttemptIDs []uuid.UUID

	for i, task := range tasks {
		select {
		case <-d.stopChan:
			// Dispatcher is stopping: collect all remaining unenqueued tasks to revert
			for _, remaining := range tasks[i:] {
				failedAttemptIDs = append(failedAttemptIDs, remaining.AttemptID)
			}
			goto revert
		default:
		}

		enqueueCtx, enqueueCancel := context.WithTimeout(context.Background(), d.cfg.EnqueueTimeout)
		err := d.pool.Enqueue(enqueueCtx, task)
		enqueueCancel()

		if err != nil {
			d.logger.Warn().
				Err(err).
				Str("attempt_id", task.AttemptID.String()).
				Msg("failed to enqueue delivery task to worker pool, will revert to pending")
			failedAttemptIDs = append(failedAttemptIDs, task.AttemptID)
		}
	}

revert:
	if len(failedAttemptIDs) > 0 {
		revertCtx, revertCancel := context.WithTimeout(context.Background(), 5*time.Second)
		if revertErr := d.source.RevertToPending(revertCtx, failedAttemptIDs); revertErr != nil {
			d.logger.Error().
				Err(revertErr).
				Int("count", len(failedAttemptIDs)).
				Msg("critical: failed to revert un-enqueued tasks to pending")
		}
		revertCancel()
	}

	return len(tasks), nil
}
