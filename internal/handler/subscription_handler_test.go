package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/LanreAkintayo/outpost/internal/dto"
	"github.com/LanreAkintayo/outpost/internal/handler"
	"github.com/LanreAkintayo/outpost/internal/middleware"
	"github.com/LanreAkintayo/outpost/internal/models"
	"github.com/LanreAkintayo/outpost/internal/repository"
	"github.com/LanreAkintayo/outpost/internal/service"
)

type mockSubscriptionService struct {
	subs map[uuid.UUID]*models.SubscriptionWithDetails
}

func newMockSubscriptionService() *mockSubscriptionService {
	return &mockSubscriptionService{
		subs: make(map[uuid.UUID]*models.SubscriptionWithDetails),
	}
}

func (m *mockSubscriptionService) Subscribe(ctx context.Context, appID, endpointID uuid.UUID, params service.SubscribeParams) (*models.SubscriptionWithDetails, error) {
	for _, s := range m.subs {
		if s.EndpointID == endpointID && s.EventTypeID == params.EventTypeID {
			return nil, repository.ErrDuplicateSubscription
		}
	}

	sub := &models.SubscriptionWithDetails{
		Subscription: models.Subscription{
			ID:          uuid.New(),
			EndpointID:  endpointID,
			EventTypeID: params.EventTypeID,
			CreatedAt:   time.Now(),
		},
		EventTypeName:        "payment.succeeded",
		EventTypeDescription: "Payment captured",
	}
	m.subs[sub.ID] = sub
	return sub, nil
}

func (m *mockSubscriptionService) Unsubscribe(ctx context.Context, appID, endpointID, eventTypeID uuid.UUID) error {
	for id, s := range m.subs {
		if s.EndpointID == endpointID && s.EventTypeID == eventTypeID {
			delete(m.subs, id)
			return nil
		}
	}
	return repository.ErrSubscriptionNotFound
}

func (m *mockSubscriptionService) ListSubscriptions(ctx context.Context, appID, endpointID uuid.UUID) ([]*models.SubscriptionWithDetails, error) {
	var list []*models.SubscriptionWithDetails
	for _, s := range m.subs {
		if s.EndpointID == endpointID {
			list = append(list, s)
		}
	}
	return list, nil
}

func (m *mockSubscriptionService) GetSubscribedEndpoints(ctx context.Context, appID uuid.UUID, eventTypeName string, recipientID string) ([]*models.Endpoint, error) {
	return []*models.Endpoint{}, nil
}

func setupSubscriptionTestRouter(svc service.SubscriptionService, authApp *models.Application) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	if authApp != nil {
		r.Use(func(c *gin.Context) {
			c.Set(middleware.ApplicationContextKey, authApp)
			c.Next()
		})
	}

	h := handler.NewSubscriptionHandler(svc)
	v1 := r.Group("/api/v1")
	h.RegisterRoutes(v1)

	return r
}

func TestSubscriptionHandler(t *testing.T) {
	app := &models.Application{
		ID:   uuid.New(),
		Name: "PayNova Payments",
	}
	endpointID := uuid.New()
	eventTypeID := uuid.New()

	t.Run("POST /api/v1/endpoints/:id/subscriptions subscribes endpoint successfully", func(t *testing.T) {
		svc := newMockSubscriptionService()
		r := setupSubscriptionTestRouter(svc, app)

		reqBody := dto.CreateSubscriptionRequest{
			EventTypeID: eventTypeID,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		url := "/api/v1/endpoints/" + endpointID.String() + "/subscriptions"
		req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var resp dto.SubscriptionResponse
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, endpointID, resp.EndpointID)
		assert.Equal(t, eventTypeID, resp.EventTypeID)
		assert.NotNil(t, resp.EventType)
		assert.Equal(t, "payment.succeeded", resp.EventType.Name)
	})

	t.Run("POST /api/v1/endpoints/:id/subscriptions rejects duplicate with 409 Conflict", func(t *testing.T) {
		svc := newMockSubscriptionService()
		r := setupSubscriptionTestRouter(svc, app)

		reqBody := dto.CreateSubscriptionRequest{
			EventTypeID: eventTypeID,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		url := "/api/v1/endpoints/" + endpointID.String() + "/subscriptions"

		// First call -> 201
		req1, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
		rec1 := httptest.NewRecorder()
		r.ServeHTTP(rec1, req1)
		assert.Equal(t, http.StatusCreated, rec1.Code)

		// Duplicate call -> 409
		req2, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
		rec2 := httptest.NewRecorder()
		r.ServeHTTP(rec2, req2)
		assert.Equal(t, http.StatusConflict, rec2.Code)
	})

	t.Run("POST /api/v1/endpoints/:id/subscriptions rejects invalid endpoint UUID with 400", func(t *testing.T) {
		svc := newMockSubscriptionService()
		r := setupSubscriptionTestRouter(svc, app)

		reqBody := dto.CreateSubscriptionRequest{
			EventTypeID: eventTypeID,
		}
		bodyBytes, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/endpoints/not-a-uuid/subscriptions", bytes.NewReader(bodyBytes))
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("GET /api/v1/endpoints/:id/subscriptions lists subscriptions", func(t *testing.T) {
		svc := newMockSubscriptionService()
		_, _ = svc.Subscribe(context.Background(), app.ID, endpointID, service.SubscribeParams{
			EventTypeID: eventTypeID,
		})
		r := setupSubscriptionTestRouter(svc, app)

		url := "/api/v1/endpoints/" + endpointID.String() + "/subscriptions"
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var list []dto.SubscriptionResponse
		err := json.Unmarshal(rec.Body.Bytes(), &list)
		assert.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, "payment.succeeded", list[0].EventType.Name)
	})

	t.Run("DELETE /api/v1/endpoints/:id/subscriptions/:event_type_id unsubscribes", func(t *testing.T) {
		svc := newMockSubscriptionService()
		_, _ = svc.Subscribe(context.Background(), app.ID, endpointID, service.SubscribeParams{
			EventTypeID: eventTypeID,
		})
		r := setupSubscriptionTestRouter(svc, app)

		url := "/api/v1/endpoints/" + endpointID.String() + "/subscriptions/" + eventTypeID.String()
		req, _ := http.NewRequest(http.MethodDelete, url, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
	})

	t.Run("Rejects unauthenticated requests with 401", func(t *testing.T) {
		svc := newMockSubscriptionService()
		r := setupSubscriptionTestRouter(svc, nil)

		url := "/api/v1/endpoints/" + endpointID.String() + "/subscriptions"
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
