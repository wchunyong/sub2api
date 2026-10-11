//go:build unit

package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTenYuanScheduleRequiresOffsetAndAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		body string
		want int
	}{
		{`{"group_id":10,"starts_at":"2026-10-12T09:30:45"}`, http.StatusBadRequest},
		{`{"group_id":10,"starts_at":"2026-10-12T09:30:45+08:00"}`, http.StatusUnauthorized},
	} {
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = httptest.NewRequest(http.MethodPut, "/admin/payment/activities/ten-yuan-seven-days", strings.NewReader(tc.body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		(&PaymentHandler{}).ScheduleTenYuanActivity(ctx)
		require.Equal(t, tc.want, recorder.Code)
	}
}
