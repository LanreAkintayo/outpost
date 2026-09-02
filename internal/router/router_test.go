package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LanreAkintayo/outpost/internal/config"
	"github.com/LanreAkintayo/outpost/internal/logger"
	"github.com/LanreAkintayo/outpost/internal/router"
)

func TestHealthCheck(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			GinMode: "test",
		},
	}
	log := logger.New("test")

	r := router.New(cfg, log)

	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	assert.NoError(t, err)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"service":"outpost","status":"healthy"}`, rec.Body.String())
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}
