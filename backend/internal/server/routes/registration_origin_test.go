package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRegistrationOriginMiddlewareKeepsDeviceAcrossRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(registrationOriginMiddleware)
	router.GET("/auth/probe", func(c *gin.Context) {
		origin, ok := service.RegistrationOriginFromContext(c.Request.Context())
		require.True(t, ok)
		require.NotEmpty(t, origin.IPHash)
		require.NotEmpty(t, origin.DeviceHash)
		c.String(http.StatusOK, origin.DeviceHash)
	})
	first := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/probe", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	router.ServeHTTP(first, req)
	require.Equal(t, http.StatusOK, first.Code)
	require.NotEmpty(t, first.Result().Cookies())

	second := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/auth/probe", nil)
	req.RemoteAddr = "203.0.113.10:5678"
	for _, cookie := range first.Result().Cookies() {
		req.AddCookie(cookie)
	}
	router.ServeHTTP(second, req)
	require.Equal(t, first.Body.String(), second.Body.String())
}
