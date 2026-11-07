//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

type newUserBenefitSettingRepo struct {
	SettingRepository
	values map[string]string
}

func (r *newUserBenefitSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := r.values[key]; ok {
			out[key] = v
		}
	}
	return out, nil
}

func (r *newUserBenefitSettingRepo) SetMultiple(_ context.Context, settings map[string]string) error {
	for k, v := range settings {
		r.values[k] = v
	}
	return nil
}

func newUserBenefitConfigService(values map[string]string) *PaymentConfigService {
	return NewPaymentConfigService(nil, &newUserBenefitSettingRepo{values: values}, nil)
}

func TestNewUserRechargeCandidateRequiresFirstDayAndMinimumAmount(t *testing.T) {
	created := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	require.True(t, newUserRechargeCandidate(created, created.Add(23*time.Hour), 10, 10))
	require.True(t, newUserRechargeCandidate(created, created.Add(23*time.Hour), 20, 10))
	require.False(t, newUserRechargeCandidate(created, created.Add(23*time.Hour), 9.99, 10))
	require.False(t, newUserRechargeCandidate(created, created.Add(24*time.Hour), 10, 10))
	require.True(t, newUserRechargeCandidate(created, created.Add(time.Hour), 50, 50))
	require.False(t, newUserRechargeCandidate(created, created.Add(time.Hour), 49.99, 50))
}

func TestNewUserRechargeBonusConfigDefaultsToDisabled(t *testing.T) {
	ctx := context.Background()
	cfg, err := newUserBenefitConfigService(map[string]string{}).GetNewUserRechargeBonus(ctx)
	require.NoError(t, err)
	require.False(t, cfg.Enabled)
	require.Equal(t, 5.0, cfg.BonusAmount)
	require.Equal(t, 10.0, cfg.MinAmount)

	cfg, err = newUserBenefitConfigService(map[string]string{
		SettingNewUserRechargeBonusEnabled: "true", SettingNewUserRechargeBonusAmount: "bad", SettingNewUserRechargeMinAmount: "-1",
	}).GetNewUserRechargeBonus(ctx)
	require.NoError(t, err)
	require.True(t, cfg.Enabled)
	require.Equal(t, 5.0, cfg.BonusAmount)
	require.Equal(t, 10.0, cfg.MinAmount)
}

func TestUpdateNewUserRechargeBonusValidatesAmounts(t *testing.T) {
	ctx := context.Background()
	repo := &newUserBenefitSettingRepo{values: map[string]string{}}
	svc := NewPaymentConfigService(nil, repo, nil)
	require.NoError(t, svc.UpdateNewUserRechargeBonus(ctx, NewUserRechargeBonusConfig{Enabled: true, BonusAmount: 8.5, MinAmount: 30}))
	cfg, err := svc.GetNewUserRechargeBonus(ctx)
	require.NoError(t, err)
	require.Equal(t, NewUserRechargeBonusConfig{Enabled: true, BonusAmount: 8.5, MinAmount: 30}, *cfg)

	require.Error(t, svc.UpdateNewUserRechargeBonus(ctx, NewUserRechargeBonusConfig{Enabled: true, BonusAmount: 0, MinAmount: 10}))
	require.Error(t, svc.UpdateNewUserRechargeBonus(ctx, NewUserRechargeBonusConfig{Enabled: true, BonusAmount: 5, MinAmount: 0.001}))
	require.Error(t, svc.UpdateNewUserRechargeBonus(ctx, NewUserRechargeBonusConfig{Enabled: true, BonusAmount: 5, MinAmount: 1e9}))
}

func TestPrepareNewUserRechargeBonusCreditsOnlyFirstPaidOrder(t *testing.T) {
	ctx := context.Background()
	svc := tenYuanScheduleService(t)
	created := time.Now().Add(-time.Hour).Truncate(time.Second)
	user, err := svc.entClient.User.Create().
		SetEmail("new-user-benefit@example.com").SetPasswordHash("hash").
		SetRole(RoleUser).SetStatus(StatusActive).SetCreatedAt(created).Save(ctx)
	require.NoError(t, err)
	createOrder := func(code string, paidAt time.Time) *dbent.PaymentOrder {
		order, createErr := svc.entClient.PaymentOrder.Create().
			SetUserID(user.ID).SetUserEmail(user.Email).SetUserName("").
			SetAmount(20).SetPayAmount(20).SetRechargeCode(code).
			SetOutTradeNo(code).SetPaymentType("alipay").SetPaymentTradeNo("").
			SetOrderType(payment.OrderTypeBalance).SetStatus(OrderStatusRecharging).
			SetExpiresAt(paidAt.Add(time.Hour)).SetPaidAt(paidAt).
			SetClientIP("").SetSrcHost("").
			SetProviderSnapshot(map[string]any{newUserRechargeCandidateKey: true}).Save(ctx)
		require.NoError(t, createErr)
		return order
	}
	first := createOrder("BENEFIT-1", created.Add(30*time.Minute))
	second := createOrder("BENEFIT-2", created.Add(40*time.Minute))
	lease := &paymentFulfillmentLease{version: first.UpdatedAt}
	first, err = svc.prepareNewUserRechargeBonus(ctx, first, lease)
	require.NoError(t, err)
	require.Equal(t, 30.0, first.Amount)
	require.Equal(t, 10.0, first.BonusAmount)
	require.Equal(t, first.UpdatedAt, lease.version)
	persistedFirst, err := svc.entClient.PaymentOrder.Get(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, persistedFirst.UpdatedAt, lease.version)
	first, err = svc.prepareNewUserRechargeBonus(ctx, first, lease)
	require.NoError(t, err)
	require.Equal(t, 30.0, first.Amount)
	second, err = svc.prepareNewUserRechargeBonus(ctx, second, &paymentFulfillmentLease{version: second.UpdatedAt})
	require.NoError(t, err)
	require.Equal(t, 20.0, second.Amount)
	storedUser, err := svc.entClient.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.Equal(t, first.ID, *storedUser.WelcomeRechargeOrderID)
	rows, err := svc.entClient.PaymentOrder.Update().Where(
		paymentorder.IDEQ(first.ID), paymentorder.StatusEQ(OrderStatusRecharging),
		paymentorder.UpdatedAtEQ(lease.version),
	).SetStatus(OrderStatusCompleted).Save(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, rows, "the bonus must preserve the fulfillment lease")
}

func TestNewUserRechargeAvailableClosesAtTwentyFourHours(t *testing.T) {
	ctx := context.Background()
	svc := tenYuanScheduleService(t)
	svc.configService = newUserBenefitConfigService(map[string]string{SettingNewUserRechargeBonusEnabled: "true"})
	created := time.Now().Add(-time.Hour).Truncate(time.Second)
	user, err := svc.entClient.User.Create().SetEmail("offer@example.com").SetPasswordHash("hash").
		SetRole(RoleUser).SetStatus(StatusActive).SetCreatedAt(created).Save(ctx)
	require.NoError(t, err)
	available, err := svc.NewUserRechargeAvailable(ctx, user.ID, created.Add(23*time.Hour))
	require.NoError(t, err)
	require.True(t, available)
	available, err = svc.NewUserRechargeAvailable(ctx, user.ID, created.Add(24*time.Hour))
	require.NoError(t, err)
	require.False(t, available)
}

func TestNewUserRechargeAvailableRespectsDisabledSwitch(t *testing.T) {
	ctx := context.Background()
	svc := tenYuanScheduleService(t)
	svc.configService = newUserBenefitConfigService(map[string]string{SettingNewUserRechargeBonusEnabled: "false"})
	created := time.Now().Add(-time.Hour).Truncate(time.Second)
	user, err := svc.entClient.User.Create().SetEmail("offer-disabled@example.com").SetPasswordHash("hash").
		SetRole(RoleUser).SetStatus(StatusActive).SetCreatedAt(created).Save(ctx)
	require.NoError(t, err)
	available, err := svc.NewUserRechargeAvailable(ctx, user.ID, created.Add(time.Hour))
	require.NoError(t, err)
	require.False(t, available)
}

func TestNewUserRechargeBonusRequiresFirstSuccessfulOrder(t *testing.T) {
	ctx := context.Background()
	svc := tenYuanScheduleService(t)
	created := time.Now().Add(-time.Hour).Truncate(time.Second)
	user, err := svc.entClient.User.Create().SetEmail("first-order-benefit@example.com").SetPasswordHash("hash").
		SetRole(RoleUser).SetStatus(StatusActive).SetCreatedAt(created).Save(ctx)
	require.NoError(t, err)
	_, err = svc.entClient.PaymentOrder.Create().SetUserID(user.ID).SetUserEmail(user.Email).SetUserName("").
		SetAmount(5).SetPayAmount(5).SetRechargeCode("BENEFIT-SMALL").SetOutTradeNo("benefit-small").
		SetPaymentType("alipay").SetPaymentTradeNo("").SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusCompleted).SetExpiresAt(created.Add(time.Hour)).SetPaidAt(created.Add(10 * time.Minute)).
		SetClientIP("").SetSrcHost("").Save(ctx)
	require.NoError(t, err)
	order, err := svc.entClient.PaymentOrder.Create().SetUserID(user.ID).SetUserEmail(user.Email).SetUserName("").
		SetAmount(20).SetPayAmount(20).SetRechargeCode("BENEFIT-LATER").SetOutTradeNo("benefit-later").
		SetPaymentType("alipay").SetPaymentTradeNo("").SetOrderType(payment.OrderTypeBalance).
		SetStatus(OrderStatusRecharging).SetExpiresAt(created.Add(time.Hour)).SetPaidAt(created.Add(20 * time.Minute)).
		SetProviderSnapshot(map[string]any{newUserRechargeCandidateKey: true}).SetClientIP("").SetSrcHost("").Save(ctx)
	require.NoError(t, err)
	credited, err := svc.prepareNewUserRechargeBonus(ctx, order, &paymentFulfillmentLease{version: order.UpdatedAt})
	require.NoError(t, err)
	require.Equal(t, 20.0, credited.Amount)
	require.Equal(t, 0.0, credited.BonusAmount)
	storedUser, err := svc.entClient.User.Get(ctx, user.ID)
	require.NoError(t, err)
	require.Nil(t, storedUser.WelcomeRechargeOrderID)
}

func TestExecuteBalanceFulfillmentAddsNewUserGiftOnce(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	ensurePaymentAuditOrderActionUniqueIndex(t, ctx, client)
	created := time.Now().Add(-time.Hour).Truncate(time.Second)
	user, err := client.User.Create().SetEmail("full-benefit@example.com").SetPasswordHash("hash").
		SetRole(RoleUser).SetStatus(StatusActive).SetCreatedAt(created).Save(ctx)
	require.NoError(t, err)
	order, err := client.PaymentOrder.Create().SetUserID(user.ID).SetUserEmail(user.Email).SetUserName("").
		SetAmount(20).SetPayAmount(20).SetRechargeCode("WELCOME-FULL-1").SetOutTradeNo("welcome-full-1").
		SetPaymentType("alipay").SetPaymentTradeNo("trade-welcome-full-1").
		SetOrderType(payment.OrderTypeBalance).SetStatus(OrderStatusPaid).
		SetExpiresAt(time.Now().Add(time.Hour)).SetPaidAt(created.Add(30 * time.Minute)).
		SetClientIP("").SetSrcHost("").SetProviderSnapshot(map[string]any{
		newUserRechargeCandidateKey: true, "new_user_recharge_bonus_amount": 5.0,
	}).Save(ctx)
	require.NoError(t, err)
	redeemRepo := &paymentFulfillmentRedeemRepo{}
	credited := 0.0
	userRepo := &mockUserRepo{getByIDUser: &User{ID: user.ID, Balance: 0}}
	userRepo.updateBalanceFn = func(_ context.Context, _ int64, amount float64) error { credited += amount; return nil }
	cache := &paymentFulfillmentRedeemCacheStub{}
	redeemService := NewRedeemService(redeemRepo, userRepo, nil, cache, nil, client, nil, nil)
	svc := &PaymentService{entClient: client, redeemService: redeemService, userRepo: userRepo}
	require.NoError(t, svc.ExecuteBalanceFulfillment(ctx, order.ID))
	require.Equal(t, 25.0, credited)
	stored, err := client.PaymentOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, OrderStatusCompleted, stored.Status)
	require.Equal(t, 25.0, stored.Amount)
	require.Equal(t, 5.0, stored.BonusAmount)
	require.NoError(t, svc.ExecuteBalanceFulfillment(ctx, order.ID))
	require.Equal(t, 25.0, credited)
}
