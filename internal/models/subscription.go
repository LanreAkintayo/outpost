package models

import (
	"time"

	"github.com/google/uuid"
)

// Subscription represents the many-to-many link between an Endpoint and an EventType.
type Subscription struct {
	ID          uuid.UUID `json:"id" db:"id"`
	EndpointID  uuid.UUID `json:"endpoint_id" db:"endpoint_id"`
	EventTypeID uuid.UUID `json:"event_type_id" db:"event_type_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// SubscriptionWithDetails embeds the subscription along with its linked EventType details.
type SubscriptionWithDetails struct {
	Subscription
	EventTypeName        string `json:"event_type_name" db:"event_type_name"`
	EventTypeDescription string `json:"event_type_description" db:"event_type_description"`
}
