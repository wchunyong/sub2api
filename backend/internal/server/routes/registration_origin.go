package routes

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"regexp"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const registrationDeviceCookie = "sub2api_registration_device"

var registrationDevicePattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

func registrationOriginMiddleware(c *gin.Context) {
	deviceID, err := c.Cookie(registrationDeviceCookie)
	if err != nil || !registrationDevicePattern.MatchString(deviceID) {
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		deviceID = hex.EncodeToString(bytes)
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie(registrationDeviceCookie, deviceID, 365*24*3600, "/api/v1/auth", "", c.Request.TLS != nil, true)
	}
	ctx := service.WithRegistrationOrigin(c.Request.Context(), ip.GetSecurityClientIP(c, false), deviceID)
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}
