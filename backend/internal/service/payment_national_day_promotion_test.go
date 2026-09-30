//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestBuildNationalDayPromotionCheckoutActivePreview(t *testing.T) {
	t.Parallel()

	svc := &PaymentService{}
	preview, err := svc.BuildNationalDayPromotionCheckout(context.Background(), 42, time.Date(2026, 9, 30, 18, 0, 0, 0, nationalDayPromotionLocation()))

	require.NoError(t, err)
	require.NotNil(t, preview)
	require.True(t, preview.Active)
	require.Equal(t, 10.0, preview.DailyBenefitMinRecharge)
	require.Equal(t, 150.0, preview.CreditedAmountByPaymentAmount["100"])
	require.Equal(t, 20.0, preview.OldUserLimitedQuotaByPaymentAmount["100"])
}

func TestBuildNationalDayPromotionCheckoutOldUserRequiresCompletedBalanceOrderBeforeCutoff(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, nationalDayPromotionLocation())

	user, err := client.User.Create().
		SetEmail("national-day-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "@example.com").
		SetPasswordHash("hash").
		SetUsername("national-day-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("PAY-OLD-USER").
		SetOutTradeNo("sub2_old_user_" + strconv.FormatInt(time.Now().UnixNano(), 10)).
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-old-user").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetPaidAt(time.Date(2026, 9, 30, 17, 59, 0, 0, nationalDayPromotionLocation())).
		SetExpiresAt(time.Date(2026, 9, 30, 18, 10, 0, 0, nationalDayPromotionLocation())).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}
	preview, err := svc.BuildNationalDayPromotionCheckout(ctx, user.ID, now)

	require.NoError(t, err)
	require.True(t, preview.IsOldUser)
}

func TestBuildNationalDayPromotionCheckoutDoesNotTreatActivityRechargeAsOldUser(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, nationalDayPromotionLocation())

	user, err := client.User.Create().
		SetEmail("national-day-new-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "@example.com").
		SetPasswordHash("hash").
		SetUsername("national-day-new-user").
		Save(ctx)
	require.NoError(t, err)

	_, err = client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(10).
		SetPayAmount(10).
		SetFeeRate(0).
		SetRechargeCode("PAY-NEW-USER").
		SetOutTradeNo("sub2_new_user_" + strconv.FormatInt(time.Now().UnixNano(), 10)).
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-new-user").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetPaidAt(time.Date(2026, 9, 30, 18, 1, 0, 0, nationalDayPromotionLocation())).
		SetExpiresAt(time.Date(2026, 9, 30, 18, 10, 0, 0, nationalDayPromotionLocation())).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	svc := &PaymentService{entClient: client}
	preview, err := svc.BuildNationalDayPromotionCheckout(ctx, user.ID, now)

	require.NoError(t, err)
	require.False(t, preview.IsOldUser)
}
