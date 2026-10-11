package handler

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *PaymentHandler) GetTenYuanActivity(c *gin.Context) {
	subject, ok := requireAuth(c)
	if !ok {
		return
	}
	state, err := h.paymentService.GetTenYuanActivityForUser(c.Request.Context(), subject.UserID, time.Now())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, state)
}
