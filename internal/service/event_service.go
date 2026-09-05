package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/LanreAkintayo/outpost/internal/models"
	"github.com/LanreAkintayo/outpost/internal/repository"
)

var (
	ErrInvalidEventType        = errors.New("event type name is required")
	ErrInvalidPayload          = errors.New("payload must be valid non-empty JSON")
	ErrTargetEventTypeNotFound = errors.New("event type does not exist for this application")
	ErrEventNotFound           = errors.New("event not found")
)

// SendEventParams carries transport-agnostic inputs required to ingest a webhook event.
type SendEventParams struct {
	EventType      string
	Payload        json.RawMessage
	RecipientID    string
	IdempotencyKey *string
}

// EventService defines the business logic operations for event ingestion and retrieval.
type EventService interface {
	SendEvent(ctx context.Context, appID uuid.UUID, params SendEventParams) (*models.IngestResult, error)
	GetEvent(ctx context.Context, appID, id uuid.UUID) (*models.Event, error)
}

type eventService struct {
	eventRepo        repository.EventRepository
	eventTypeRepo    repository.EventTypeRepository
	subscriptionRepo repository.SubscriptionRepository
}

// NewEventService constructs a new EventService with its required repositories injected.
func NewEventService(
	eventRepo repository.EventRepository,
	eventTypeRepo repository.EventTypeRepository,
	subscriptionRepo repository.SubscriptionRepository,
) EventService {
	return &eventService{
		eventRepo:        eventRepo,
		eventTypeRepo:    eventTypeRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

// SendEvent handles event validation, idempotency checks, subscription fan-out, and atomic database persistence.
func (s *eventService) SendEvent(ctx context.Context, appID uuid.UUID, params SendEventParams) (*models.IngestResult, error) {
	trimmedEventType := strings.TrimSpace(params.EventType)
	if trimmedEventType == "" {
		return nil, ErrInvalidEventType
	}

	trimmedRecipient := strings.TrimSpace(params.RecipientID)

	if len(params.Payload) == 0 || !json.Valid(params.Payload) {
		return nil, ErrInvalidPayload
	}

	var cleanedKey *string
	// Send existing event if there is an idempotency key that matches.
	if params.IdempotencyKey != nil {
		k := strings.TrimSpace(*params.IdempotencyKey)
		if k != "" {
			cleanedKey = &k

			existing, err := s.eventRepo.GetByIdempotencyKey(ctx, appID, k)
			if err == nil && existing != nil {
				return &models.IngestResult{
					Event:            existing,
					EventTypeName:    trimmedEventType,
					QueuedDeliveries: 0,
				}, nil
			}
		}
	}

	// Get event type by name.
	eventType, err := s.eventTypeRepo.GetByName(ctx, appID, trimmedEventType)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrTargetEventTypeNotFound
		}
		return nil, fmt.Errorf("failed to verify event type: %w", err)
	}

	// Get subscribed endpoints for the event type.
	subscribedEndpoints, err := s.subscriptionRepo.GetSubscribedEndpoints(ctx, appID, trimmedEventType, trimmedRecipient)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscribed endpoints: %w", err)
	}

	event := &models.Event{
		ApplicationID:  appID,
		EventTypeID:    eventType.ID,
		Payload:        params.Payload,
		IdempotencyKey: cleanedKey,
		RecipientID:    trimmedRecipient,
	}

	attempts := make([]*models.DeliveryAttempt, len(subscribedEndpoints))
	for i, ep := range subscribedEndpoints {
		attempts[i] = &models.DeliveryAttempt{
			EndpointID:    ep.ID,
			Status:        models.DeliveryStatusPending,
			AttemptNumber: 1,
		}
	}

	if err := s.eventRepo.CreateWithAttempts(ctx, event, attempts); err != nil {
		if errors.Is(err, repository.ErrDuplicateIdempotencyKey) && cleanedKey != nil {
			existing, getErr := s.eventRepo.GetByIdempotencyKey(ctx, appID, *cleanedKey)
			if getErr == nil && existing != nil {
				return &models.IngestResult{
					Event:            existing,
					EventTypeName:    trimmedEventType,
					QueuedDeliveries: 0,
				}, nil
			}
		}
		return nil, fmt.Errorf("failed to persist event and delivery attempts: %w", err)
	}

	return &models.IngestResult{
		Event:            event,
		EventTypeName:    trimmedEventType,
		QueuedDeliveries: len(attempts),
	}, nil
}

// GetEvent retrieves full details of a previously ingested event.
func (s *eventService) GetEvent(ctx context.Context, appID, id uuid.UUID) (*models.Event, error) {
	event, err := s.eventRepo.GetByID(ctx, appID, id)
	if err != nil {
		if errors.Is(err, repository.ErrEventNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return event, nil
}
