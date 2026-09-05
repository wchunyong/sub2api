package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	ratelimit "github.com/Wei-Shaw/sub2api/internal/middleware"
	"github.com/Wei-Shaw/sub2api/internal/quickimport"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"time"
)

// RegisterQuickImportRoutes keeps public redemption separate from JWT sessions.
func RegisterQuickImportRoutes(v1 *gin.RouterGroup, keys *service.APIKeyService, settings *service.SettingService, redisClient *redis.Client, jwt middleware.JWTAuthMiddleware, audit middleware.AuditLogMiddleware, panel *middleware.PanelRateLimiter) {
	h := handler.NewQuickImportHandler(keys, settings, quickimport.NewTicketStore(redisClient))
	limiter := ratelimit.NewRateLimiter(redisClient)
	group := v1.Group("/quick-import")
	group.GET("/assets/:name", h.Asset)
	group.POST("/exchange", limiter.LimitWithOptions("quick-import-exchange", 30, time.Minute, ratelimit.RateLimitOptions{FailureMode: ratelimit.RateLimitFailClose}), h.Exchange)
	group.POST("/tickets", gin.HandlerFunc(jwt), middleware.BackendModeUserGuard(settings), panel.Global(), panel.Heavy(), gin.HandlerFunc(audit), h.Issue)
}
