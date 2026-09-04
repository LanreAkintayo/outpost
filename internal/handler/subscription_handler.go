package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/LanreAkintayo/outpost/internal/dto"
	"github.com/LanreAkintayo/outpost/internal/middleware"
	"github.com/LanreAkintayo/outpost/internal/repository"
	"github.com/LanreAkintayo/outpost/internal/response"
	"github.com/LanreAkintayo/outpost/internal/service"
)

// SubscriptionHandler handles HTTP requests for subscription management.
type SubscriptionHandler struct {
	service service.SubscriptionService
}

// NewSubscriptionHandler creates a new SubscriptionHandler instance.
func NewSubscriptionHandler(s service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{service: s}
}

// RegisterRoutes registers subscription endpoints onto the protected router group.
func (h *SubscriptionHandler) RegisterRoutes(rg *gin.RouterGroup) {
	subscriptions := rg.Group("/endpoints/:id/subscriptions")
	{
		subscriptions.POST("", h.Subscribe)
		subscriptions.GET("", h.List)
		subscriptions.DELETE("/:event_type_id", h.Unsubscribe)
	}
}

// Subscribe handles POST /api/v1/endpoints/:id/subscriptions
func (h *SubscriptionHandler) Subscribe(c *gin.Context) {
	app, ok := middleware.GetApplication(c)
	if !ok || app == nil {
		response.Unauthorized(c, "unauthenticated")
		return
	}

	endpointIDStr := c.Param("id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		response.BadRequest(c, "invalid endpoint ID format (must be UUID)")
		return
	}

	var req dto.CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	sub, err := h.service.Subscribe(
		c.Request.Context(),
		app.ID,
		endpointID,
		service.SubscribeParams{
			EventTypeID: req.EventTypeID,
		},
	)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicateSubscription) {
			response.Conflict(c, err.Error())
			return
		}
		if errors.Is(err, repository.ErrEndpointNotFound) {
			response.NotFound(c, "endpoint not found")
			return
		}
		if errors.Is(err, repository.ErrEventTypeNotFound) {
			response.NotFound(c, "event type not found")
			return
		}
		response.InternalServerError(c)
		return
	}

	response.Created(c, dto.ToSubscriptionResponse(sub))
}

// List handles GET /api/v1/endpoints/:id/subscriptions
func (h *SubscriptionHandler) List(c *gin.Context) {
	app, ok := middleware.GetApplication(c)
	if !ok || app == nil {
		response.Unauthorized(c, "unauthenticated")
		return
	}

	endpointIDStr := c.Param("id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		response.BadRequest(c, "invalid endpoint ID format (must be UUID)")
		return
	}

	list, err := h.service.ListSubscriptions(c.Request.Context(), app.ID, endpointID)
	if err != nil {
		if errors.Is(err, repository.ErrEndpointNotFound) {
			response.NotFound(c, "endpoint not found")
			return
		}
		response.InternalServerError(c)
		return
	}

	response.OK(c, dto.ToSubscriptionResponses(list))
}

// Unsubscribe handles DELETE /api/v1/endpoints/:id/subscriptions/:event_type_id
func (h *SubscriptionHandler) Unsubscribe(c *gin.Context) {
	app, ok := middleware.GetApplication(c)
	if !ok || app == nil {
		response.Unauthorized(c, "unauthenticated")
		return
	}

	endpointIDStr := c.Param("id")
	endpointID, err := uuid.Parse(endpointIDStr)
	if err != nil {
		response.BadRequest(c, "invalid endpoint ID format (must be UUID)")
		return
	}

	eventTypeIDStr := c.Param("event_type_id")
	eventTypeID, err := uuid.Parse(eventTypeIDStr)
	if err != nil {
		response.BadRequest(c, "invalid event type ID format (must be UUID)")
		return
	}

	if err := h.service.Unsubscribe(c.Request.Context(), app.ID, endpointID, eventTypeID); err != nil {
		if errors.Is(err, repository.ErrEndpointNotFound) {
			response.NotFound(c, "endpoint not found")
			return
		}
		if errors.Is(err, repository.ErrSubscriptionNotFound) {
			response.NotFound(c, "subscription not found")
			return
		}
		response.InternalServerError(c)
		return
	}

	response.NoContent(c)
}
