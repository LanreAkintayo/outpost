package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/LanreAkintayo/outpost/internal/engine"
	"github.com/LanreAkintayo/outpost/internal/models"
)

var (
	ErrDeliveryAttemptNotFound = errors.New("delivery attempt not found")
)

// DeliveryRepository defines database operations for webhook delivery attempt lifecycle management.
type DeliveryRepository interface {
	// FetchAndClaimPending atomically selects pending (or stale processing) delivery attempts
	FetchAndClaimPending(ctx context.Context, batchSize int) ([]engine.DeliveryTask, error)

	// RecordResult updates the delivery attempt record with the outcome of an HTTP execution attempt.
	RecordResult(ctx context.Context, attemptID uuid.UUID, result *engine.DeliveryResult) error

	// RevertToPending safely transitions in-flight claimed tasks back to 'pending' if the worker queue is saturated.
	RevertToPending(ctx context.Context, attemptIDs []uuid.UUID) error
}

// PostgresDeliveryRepository implements DeliveryRepository backed by PostgreSQL with pgxpool.
type PostgresDeliveryRepository struct {
	db *pgxpool.Pool
}

// NewPostgresDeliveryRepository constructs a new PostgresDeliveryRepository.
func NewPostgresDeliveryRepository(db *pgxpool.Pool) *PostgresDeliveryRepository {
	return &PostgresDeliveryRepository{db: db}
}

// FetchAndClaimPending claims a batch of pending webhook attempts in a single atomic SQL query.
func (r *PostgresDeliveryRepository) FetchAndClaimPending(ctx context.Context, batchSize int) ([]engine.DeliveryTask, error) {
	if batchSize <= 0 {
		return nil, nil
	}

	query := `
		WITH claimable AS (	
			SELECT da.id
			FROM delivery_attempts da
			JOIN endpoints ep ON da.endpoint_id = ep.id
			WHERE ep.is_active = true
			  AND (
				  (da.status = 'pending' AND (da.next_retry_at IS NULL OR da.next_retry_at <= NOW()))
				  OR
				  (da.status = 'processing' AND da.updated_at < NOW() - INTERVAL '5 minutes')
			  )
			ORDER BY da.created_at ASC
			LIMIT $1
			FOR UPDATE OF da SKIP LOCKED
		),
		claimed AS (
			UPDATE delivery_attempts da
			SET status = 'processing',
			    updated_at = NOW()
			FROM claimable c
			WHERE da.id = c.id
			RETURNING da.id, da.event_id, da.endpoint_id, da.attempt_number
		)
		SELECT 
			c.id,
			c.event_id,
			et.name,
			ep.url,
			ep.secret,
			e.payload,
			c.attempt_number
		FROM claimed c
		JOIN events e ON c.event_id = e.id
		JOIN event_types et ON e.event_type_id = et.id
		JOIN endpoints ep ON c.endpoint_id = ep.id
		ORDER BY c.attempt_number ASC;
	`

	rows, err := r.db.Query(ctx, query, batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch and claim pending deliveries: %w", err)
	}
	defer rows.Close()

	var tasks []engine.DeliveryTask
	for rows.Next() {
		var task engine.DeliveryTask
		err := rows.Scan(
			&task.AttemptID,
			&task.EventID,
			&task.EventType,
			&task.EndpointURL,
			&task.Secret,
			&task.Payload,
			&task.AttemptNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan claimed delivery task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading claimed delivery rows: %w", err)
	}

	return tasks, nil
}

// RecordResult writes the HTTP outcome (status, response, duration, error) to the delivery_attempts table.
func (r *PostgresDeliveryRepository) RecordResult(ctx context.Context, attemptID uuid.UUID, result *engine.DeliveryResult) error {
	if result == nil {
		return errors.New("result cannot be nil")
	}

	status := models.DeliveryStatusDelivered
	if !result.Success {
		status = models.DeliveryStatusFailed
	}

	query := `
		UPDATE delivery_attempts
		SET status = $2,
		    http_status = $3,
		    response_body = $4,
		    error_message = $5,
		    execution_duration_ms = $6,
		    updated_at = NOW()
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query,
		attemptID,
		status,
		result.HTTPStatus,
		result.ResponseBody,
		result.ErrorMessage,
		result.ExecutionDurationMS,
	)
	if err != nil {
		return fmt.Errorf("failed to update delivery attempt status: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return ErrDeliveryAttemptNotFound
	}

	return nil
}

// RevertToPending sets the status of un-enqueued tasks back to 'pending'.
func (r *PostgresDeliveryRepository) RevertToPending(ctx context.Context, attemptIDs []uuid.UUID) error {
	if len(attemptIDs) == 0 {
		return nil
	}

	query := `
		UPDATE delivery_attempts
		SET status = 'pending',
		    updated_at = NOW()
		WHERE id = ANY($1)
	`

	_, err := r.db.Exec(ctx, query, attemptIDs)
	if err != nil {
		return fmt.Errorf("failed to revert delivery attempts to pending: %w", err)
	}

	return nil
}
