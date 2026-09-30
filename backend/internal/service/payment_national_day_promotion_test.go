//go:build unit

package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
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

func TestCalculateBalanceOrderAmountKeepsPaidAmountDuringNationalDayPromotion(t *testing.T) {
	t.Parallel()

	activeAt := time.Date(2026, 10, 1, 12, 0, 0, 0, nationalDayPromotionLocation())
	before := time.Date(2026, 9, 30, 17, 59, 59, 0, nationalDayPromotionLocation())

	require.Equal(t, 200.0, calculateBalanceOrderAmount(200, 1, activeAt))
	require.Equal(t, 210.0, calculateBalanceOrderAmount(200, 1, before))
}

func TestApplyNationalDayPromotionBalanceBonusIsIdempotent(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, nationalDayPromotionLocation())

	user, err := client.User.Create().
		SetEmail("national-day-bonus-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "@example.com").
		SetPasswordHash("hash").
		SetUsername("national-day-bonus-user").
		Save(ctx)
	require.NoError(t, err)

	order, err := client.PaymentOrder.Create().
		SetUserID(user.ID).
		SetUserEmail(user.Email).
		SetUserName(user.Username).
		SetAmount(100).
		SetPayAmount(100).
		SetFeeRate(0).
		SetRechargeCode("PAY-NATIONAL-DAY-BONUS").
		SetOutTradeNo("sub2_national_day_bonus_" + strconv.FormatInt(time.Now().UnixNano(), 10)).
		SetPaymentType(payment.TypeAlipay).
		SetPaymentTradeNo("trade-national-day-bonus").
		SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).
		SetPaidAt(now).
		SetExpiresAt(now.Add(10 * time.Minute)).
		SetClientIP("127.0.0.1").
		SetSrcHost("api.example.com").
		Save(ctx)
	require.NoError(t, err)

	balance := 100.0
	adjustCalls := 0
	userRepo := &mockUserRepo{}
	userRepo.adjustBalanceFn = func(_ context.Context, id int64, delta float64) (BalanceChange, error) {
		require.Equal(t, user.ID, id)
		adjustCalls++
		old := balance
		balance += delta
		return BalanceChange{Old: old, New: balance}, nil
	}

	svc := &PaymentService{entClient: client, userRepo: userRepo}
	require.NoError(t, svc.applyNationalDayPromotionForOrder(ctx, order))
	require.NoError(t, svc.applyNationalDayPromotionForOrder(ctx, order))

	require.Equal(t, 1, adjustCalls)
	require.Equal(t, 150.0, balance)

	count, err := client.PaymentAuditLog.Query().
		Where(paymentauditlog.OrderIDEQ(strconv.FormatInt(order.ID, 10)), paymentauditlog.ActionEQ(nationalDayPromotionBalanceBonusAuditAction)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
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
