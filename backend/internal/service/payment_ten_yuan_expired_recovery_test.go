//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func tenYuanExpiredRecoveryFixture(t *testing.T) (*PaymentService, *dbent.PaymentOrder, time.Time) {
	t.Helper()
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	created := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	paidAt := created.Add(10 * time.Minute)
	user, err := client.User.Create().SetEmail("activity-expired@example.com").SetPasswordHash("hash").SetUsername("activity-expired").Save(ctx)
	require.NoError(t, err)
	_, err = client.ActivityRun.Create().SetCode(tenYuanActivityCode).SetCurrentMarker(tenYuanActivityCode).SetGroupID(10).
		SetStartsAt(created.Add(-time.Minute)).SetEndsAt(created.Add(168 * time.Hour)).SetDisabledAt(created.Add(20 * time.Minute)).SetCreatedBy(user.ID).Save(ctx)
	require.NoError(t, err)
	order, err := client.PaymentOrder.Create().SetUserID(user.ID).SetUserEmail(user.Email).SetUserName(user.Username).
		SetAmount(10).SetPayAmount(10).SetFeeRate(0).SetRechargeCode("PAY-ACTIVITY-EXPIRED").SetOutTradeNo("activity-expired-order").
		SetPaymentType(payment.TypeAlipay).SetPaymentTradeNo("trade-activity-expired").SetOrderType(payment.OrderTypeBalance).SetStatus(OrderStatusExpired).
		SetExpiresAt(created.Add(5 * time.Minute)).SetClientIP("127.0.0.1").SetSrcHost("api.example.com").
		SetCreatedAt(created).SetUpdatedAt(created.Add(30 * time.Minute)).Save(ctx)
	require.NoError(t, err)
	usedBy := user.ID
	redeemRepo := &redeemCodeRepoStub{codesByCode: map[string]*RedeemCode{order.RechargeCode: {
		ID: 101, Code: order.RechargeCode, Type: RedeemTypeBalance, Value: order.Amount, Status: StatusUsed, UsedBy: &usedBy,
	}}}
	return &PaymentService{entClient: client, redeemService: &RedeemService{redeemRepo: redeemRepo}}, order, paidAt
}

func TestTenYuanExpiredOrderRecoversWithVerifiedPreClosePaymentTime(t *testing.T) {
	svc, order, paidAt := tenYuanExpiredRecoveryFixture(t)
	_ = svc.HandlePaymentNotification(context.Background(), &payment.PaymentNotification{
		TradeNo: "verified-paid-before-close", OrderID: order.OutTradeNo, Amount: 10,
		Status: payment.NotificationStatusSuccess, PaidAt: paidAt.Format(time.RFC3339),
	}, payment.TypeAlipay)
	reloaded, err := svc.entClient.PaymentOrder.Get(context.Background(), order.ID)
	require.NoError(t, err)
	require.NotEqual(t, OrderStatusExpired, reloaded.Status)
	require.NotNil(t, reloaded.PaidAt)
	require.True(t, paidAt.Equal(*reloaded.PaidAt))
}

func TestTenYuanExpiredOrderStaysExpiredWithoutVerifiedPaymentTime(t *testing.T) {
	svc, order, _ := tenYuanExpiredRecoveryFixture(t)
	require.NoError(t, svc.HandlePaymentNotification(context.Background(), &payment.PaymentNotification{
		TradeNo: "no-provider-payment-time", OrderID: order.OutTradeNo, Amount: 10,
		Status: payment.NotificationStatusSuccess,
	}, payment.TypeAlipay))
	reloaded, err := svc.entClient.PaymentOrder.Get(context.Background(), order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusExpired, reloaded.Status)
}
