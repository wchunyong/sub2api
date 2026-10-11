//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestPaymentNotificationPersistsVerifiedProviderPaymentTime(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	order := createPaymentFulfillmentSubscriptionOrder(t, ctx, client, OrderStatusPending, time.Now())
	paidAt := order.CreatedAt
	svc := &PaymentService{entClient: client, groupRepo: &nationalDayPromotionGroupRepoStub{byName: map[string]*Group{"test": {ID: 7, Name: "test", Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}}}}
	_ = svc.HandlePaymentNotification(ctx, &payment.PaymentNotification{
		TradeNo: "verified-provider-trade", OrderID: order.OutTradeNo,
		Amount: order.PayAmount, Status: payment.NotificationStatusSuccess,
		PaidAt: paidAt.Format(time.RFC3339Nano),
	}, payment.TypeAlipay)
	reloaded, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.NotNil(t, reloaded.PaidAt)
	require.True(t, paidAt.Equal(*reloaded.PaidAt), "got %s, want %s", reloaded.PaidAt, paidAt)
}

func TestVerifiedProviderPaidAtFallsBackWhenTimestampUnavailableOrInvalid(t *testing.T) {
	created := time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC)
	received := created.Add(time.Hour)
	paid := created.Add(10 * time.Minute)
	require.True(t, paid.Equal(verifiedProviderPaidAt("2026-10-11 16:10:00", created, received)))
	require.True(t, paid.Equal(verifiedProviderPaidAt(paid.Format(time.RFC3339), created, received)))
	for _, raw := range []string{"", "invalid", created.Add(-time.Hour).Format(time.RFC3339), received.Add(time.Hour).Format(time.RFC3339)} {
		require.True(t, received.Equal(verifiedProviderPaidAt(raw, created, received)), raw)
	}
}
