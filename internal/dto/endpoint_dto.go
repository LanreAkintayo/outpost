package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/LanreAkintayo/outpost/internal/models"
)

// CreateEndpointRequest defines the expected JSON payload for creating an endpoint.
type CreateEndpointRequest struct {
	URL         string `json:"url" binding:"required"`
	Description string `json:"description" binding:"max=500"`
}

// UpdateEndpointRequest defines the expected JSON payload for modifying an endpoint.
type UpdateEndpointRequest struct {
	URL         *string                `json:"url,omitempty"`
	Description *string                `json:"description,omitempty"`
	Status      *models.EndpointStatus `json:"status,omitempty"`
}

// EndpointResponse represents the public API response for an endpoint.
type EndpointResponse struct {
	ID            uuid.UUID             `json:"id"`
	ApplicationID uuid.UUID             `json:"application_id"`
	URL           string                `json:"url"`
	Secret        string                `json:"secret"`
	Description   string                `json:"description"`
	Status        models.EndpointStatus `json:"status"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// ToEndpointResponse maps an internal Endpoint domain model to an EndpointResponse DTO.
func ToEndpointResponse(e *models.Endpoint) EndpointResponse {
	return EndpointResponse{
		ID:            e.ID,
		ApplicationID: e.ApplicationID,
		URL:           e.URL,
		Secret:        e.Secret,
		Description:   e.Description,
		Status:        e.Status,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

// ToEndpointResponses maps a slice of internal Endpoint models to EndpointResponse DTOs.
func ToEndpointResponses(endpoints []*models.Endpoint) []EndpointResponse {
	res := make([]EndpointResponse, len(endpoints))
	for i, e := range endpoints {
		res[i] = ToEndpointResponse(e)
	}
	return res
}
