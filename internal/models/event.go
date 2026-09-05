package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DeliveryStatus represents the delivery lifecycle state of a webhook attempt.
type DeliveryStatus string

const (
	DeliveryStatusPending    DeliveryStatus = "pending"
	DeliveryStatusProcessing DeliveryStatus = "processing"
	DeliveryStatusDelivered  DeliveryStatus = "delivered"
	DeliveryStatusFailed     DeliveryStatus = "failed"
	DeliveryStatusDeadLetter DeliveryStatus = "dead_letter"
)

// Event represents an ingested webhook event payload published by an Application tenant.
type Event struct {
	ID             uuid.UUID       `json:"id" db:"id"`
	ApplicationID  uuid.UUID       `json:"application_id" db:"application_id"`
	EventTypeID    uuid.UUID       `json:"event_type_id" db:"event_type_id"`
	Payload        json.RawMessage `json:"payload" db:"payload"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty" db:"idempotency_key"`
	RecipientID    string          `json:"recipient_id" db:"recipient_id"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
}

// DeliveryAttempt represents a single webhook delivery dispatch to an Endpoint.
type DeliveryAttempt struct {
	ID                  uuid.UUID      `json:"id" db:"id"`
	EventID             uuid.UUID      `json:"event_id" db:"event_id"`
	EndpointID          uuid.UUID      `json:"endpoint_id" db:"endpoint_id"`
	Status              DeliveryStatus `json:"status" db:"status"`
	AttemptNumber       int            `json:"attempt_number" db:"attempt_number"`
	HTTPStatus          *int           `json:"http_status,omitempty" db:"http_status"`
	ResponseBody        *string        `json:"response_body,omitempty" db:"response_body"`
	ErrorMessage        *string        `json:"error_message,omitempty" db:"error_message"`
	ExecutionDurationMS *int           `json:"execution_duration_ms,omitempty" db:"execution_duration_ms"`
	NextRetryAt         *time.Time     `json:"next_retry_at,omitempty" db:"next_retry_at"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

// IngestResult represents the domain outcome of an event ingestion, bundling the event and its queued delivery count.
type IngestResult struct {
	Event            *Event
	EventTypeName    string
	QueuedDeliveries int
}
