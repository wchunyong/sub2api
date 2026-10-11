package admin

import (
	"errors"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *PaymentHandler) GetTenYuanActivity(c *gin.Context) {
	state, err := h.paymentService.GetTenYuanActivityAdmin(c.Request.Context(), time.Now())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, state)
}

func (h *PaymentHandler) ScheduleTenYuanActivity(c *gin.Context) {
	var req struct {
		GroupID  int64  `json:"group_id" binding:"required"`
		StartsAt string `json:"starts_at" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	start, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil || start.Nanosecond() != 0 {
		response.BadRequest(c, "starts_at must include a timezone and whole seconds")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "authentication required")
		return
	}
	run, err := h.paymentService.ScheduleTenYuanActivity(c.Request.Context(), service.TenYuanActivityScheduleInput{GroupID: req.GroupID, StartsAt: start}, subject.UserID, time.Now())
	if err != nil {
		if errors.Is(err, service.ErrTenYuanActivityInvalid) {
			response.BadRequest(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrTenYuanActivityConflict) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, run)
}

func (h *PaymentHandler) CloseTenYuanActivity(c *gin.Context) {
	var req struct {
		RunID int64 `json:"run_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.paymentService.CloseTenYuanActivity(c.Request.Context(), req.RunID, time.Now()); err != nil {
		if errors.Is(err, service.ErrTenYuanActivityConflict) {
			response.Error(c, http.StatusConflict, err.Error())
			return
		}
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"closed": true})
}
