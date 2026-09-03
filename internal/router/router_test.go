package router_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/LanreAkintayo/outpost/internal/config"
	"github.com/LanreAkintayo/outpost/internal/router"
)

type mockRegistrar struct {
	path    string
	handler gin.HandlerFunc
}

func (m *mockRegistrar) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET(m.path, m.handler)
}

func testConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			GinMode: "test",
		},
	}
}

func TestHealthCheck(t *testing.T) {
	r := router.New(router.RouterParams{
		Config: testConfig(),
	})

	req, err := http.NewRequest(http.MethodGet, "/health", nil)
	assert.NoError(t, err)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"service":"outpost","status":"healthy"}`, rec.Body.String())
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestRouteRegistrar_Mounting(t *testing.T) {
	public := &mockRegistrar{
		path: "/public-endpoint",
		handler: func(c *gin.Context) {
			c.String(http.StatusOK, "public-ok")
		},
	}

	protected := &mockRegistrar{
		path: "/protected-endpoint",
		handler: func(c *gin.Context) {
			c.String(http.StatusOK, "protected-ok")
		},
	}

	// Middleware that checks for an "X-Auth" header
	authMiddleware := func(c *gin.Context) {
		if c.GetHeader("X-Auth") != "valid" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}

	r := router.New(router.RouterParams{
		Config:          testConfig(),
		AuthMiddleware:  authMiddleware,
		PublicRoutes:    []router.RouteRegistrar{public},
		ProtectedRoutes: []router.RouteRegistrar{protected},
	})

	// 1. Public route test
	pubReq, _ := http.NewRequest(http.MethodGet, "/api/v1/public-endpoint", nil)
	pubRec := httptest.NewRecorder()
	r.ServeHTTP(pubRec, pubReq)

	assert.Equal(t, http.StatusOK, pubRec.Code)
	assert.Equal(t, "public-ok", pubRec.Body.String())

	// 2. Protected route without auth header -> 401
	protReqNoAuth, _ := http.NewRequest(http.MethodGet, "/api/v1/protected-endpoint", nil)
	protRecNoAuth := httptest.NewRecorder()
	r.ServeHTTP(protRecNoAuth, protReqNoAuth)

	assert.Equal(t, http.StatusUnauthorized, protRecNoAuth.Code)

	// 3. Protected route with valid auth header -> 200
	protReqAuth, _ := http.NewRequest(http.MethodGet, "/api/v1/protected-endpoint", nil)
	protReqAuth.Header.Set("X-Auth", "valid")
	protRecAuth := httptest.NewRecorder()
	r.ServeHTTP(protRecAuth, protReqAuth)

	assert.Equal(t, http.StatusOK, protRecAuth.Code)
	assert.Equal(t, "protected-ok", protRecAuth.Body.String())
}
