package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/LanreAkintayo/outpost/internal/models"
)

// SendEventRequest represents the incoming JSON payload to ingest a new webhook event.
type SendEventRequest struct {
	EventType      string          `json:"event_type" binding:"required,min=1,max=255"`
	Payload        json.RawMessage `json:"payload" binding:"required"`
	RecipientID    string          `json:"recipient_id,omitempty"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
}

// IngestEventResponse represents the 202 Accepted response returned after successfully scheduling an event.
type IngestEventResponse struct {
	ID               uuid.UUID `json:"id"`
	EventType        string    `json:"event_type"`
	RecipientID      string    `json:"recipient_id,omitempty"`
	IdempotencyKey   *string   `json:"idempotency_key,omitempty"`
	QueuedDeliveries int       `json:"queued_deliveries"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
}

// EventResponse represents the full details of an ingested event.
type EventResponse struct {
	ID             uuid.UUID       `json:"id"`
	ApplicationID  uuid.UUID       `json:"application_id"`
	EventTypeID    uuid.UUID       `json:"event_type_id"`
	Payload        json.RawMessage `json:"payload"`
	IdempotencyKey *string         `json:"idempotency_key,omitempty"`
	RecipientID    string          `json:"recipient_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ToEventResponse maps an internal Event domain model to an EventResponse DTO.
func ToEventResponse(e *models.Event) EventResponse {
	return EventResponse{
		ID:             e.ID,
		ApplicationID:  e.ApplicationID,
		EventTypeID:    e.EventTypeID,
		Payload:        e.Payload,
		IdempotencyKey: e.IdempotencyKey,
		RecipientID:    e.RecipientID,
		CreatedAt:      e.CreatedAt,
	}
}

// ToIngestEventResponse maps an internal IngestResult domain model to an IngestEventResponse DTO.
func ToIngestEventResponse(r *models.IngestResult) IngestEventResponse {
	return IngestEventResponse{
		ID:               r.Event.ID,
		EventType:        r.EventTypeName,
		RecipientID:      r.Event.RecipientID,
		IdempotencyKey:   r.Event.IdempotencyKey,
		QueuedDeliveries: r.QueuedDeliveries,
		Status:           "accepted",
		CreatedAt:        r.Event.CreatedAt,
	}
}
