package handler

import (
	"context"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/quickimport"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type quickImportKeys interface {
	GetByID(context.Context, int64) (*service.APIKey, error)
}
type quickImportSettings interface {
	GetPublicSettings(context.Context) (*service.PublicSettings, error)
}
type QuickImportHandler struct {
	keys     quickImportKeys
	settings quickImportSettings
	tickets  *quickimport.TicketStore
}

func NewQuickImportHandler(keys *service.APIKeyService, settings *service.SettingService, tickets *quickimport.TicketStore) *QuickImportHandler {
	return &QuickImportHandler{keys: keys, settings: settings, tickets: tickets}
}
func (h *QuickImportHandler) validKey(ctx context.Context, id, user int64) (*service.APIKey, bool) {
	key, err := h.keys.GetByID(ctx, id)
	return key, err == nil && key != nil && key.UserID == user && key.IsActive() && !key.IsExpired() && !key.IsQuotaExhausted() && key.Group != nil && key.Group.Status == service.StatusActive && (key.User == nil || key.User.Status == service.StatusActive)
}
func (h *QuickImportHandler) Issue(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var request struct {
		KeyID int64  `json:"key_id" binding:"required"`
		Agent string `json:"agent" binding:"required"`
		Model string `json:"model"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if c.ShouldBindJSON(&request) != nil {
		response.BadRequest(c, "Invalid import request")
		return
	}
	key, ok := h.validKey(c.Request.Context(), request.KeyID, subject.UserID)
	if !ok {
		response.BadRequest(c, "Key is unavailable for import")
		return
	}
	settings, err := h.settings.GetPublicSettings(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Cannot load gateway settings")
		return
	}
	base := settings.APIBaseURL
	if base == "" {
		base = c.GetHeader("Origin")
	}
	config, err := quickimport.BuildConfig(request.Agent, key.Group.Platform, key.Group.AllowMessagesDispatch, base, key.Key, request.Model)
	if err != nil {
		response.BadRequest(c, "Unsupported Agent or group; configure an HTTPS API base URL")
		return
	}
	code, err := h.tickets.Issue(c.Request.Context(), quickimport.Ticket{UserID: subject.UserID, KeyID: key.ID, Agent: request.Agent, Model: config.Model, BaseURL: base})
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Cannot generate configuration code")
		return
	}
	response.Success(c, gin.H{"ticket": code, "expires_in": int(quickimport.TicketTTL.Seconds()), "agent": request.Agent, "model": config.Model})
}
func (h *QuickImportHandler) Exchange(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	var request struct {
		Ticket string `json:"ticket"`
		Agent  string `json:"agent"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	if c.ShouldBindJSON(&request) != nil {
		response.BadRequest(c, "Invalid configuration code")
		return
	}
	ticket, err := h.tickets.Consume(c.Request.Context(), request.Ticket)
	if err != nil || request.Agent != ticket.Agent {
		response.BadRequest(c, "Configuration code expired or unavailable; generate a new command")
		return
	}
	key, ok := h.validKey(c.Request.Context(), ticket.KeyID, ticket.UserID)
	if !ok {
		response.BadRequest(c, "Key is no longer available")
		return
	}
	config, err := quickimport.BuildConfig(ticket.Agent, key.Group.Platform, key.Group.AllowMessagesDispatch, ticket.BaseURL, key.Key, ticket.Model)
	if err != nil {
		response.BadRequest(c, "Group no longer supports this Agent")
		return
	}
	response.Success(c, config)
}
func (h *QuickImportHandler) Asset(c *gin.Context) {
	name := c.Param("name")
	if name != "installer.py" && name != "install.ps1" && name != "install.sh" {
		c.Status(http.StatusNotFound)
		return
	}
	body, err := quickimport.Assets.ReadFile("assets/" + name)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/plain; charset=utf-8", body)
}
