package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

const (
	SettingNewUserRechargeBonusEnabled = "NEW_USER_RECHARGE_BONUS_ENABLED"
	SettingNewUserRechargeBonusAmount  = "NEW_USER_RECHARGE_BONUS_AMOUNT"
	SettingNewUserRechargeMinAmount    = "NEW_USER_RECHARGE_MIN_AMOUNT"

	newUserRechargeCandidateKey   = "new_user_recharge_candidate"
	newUserRechargeBonusAmountKey = "new_user_recharge_bonus_amount"
	newUserRechargeWindow         = 24 * time.Hour
	defaultNewUserRechargeBonus   = 5.0
	defaultNewUserRechargeMin     = 10.0
	maxNewUserRechargeAmount      = 100000.0
	legacyNewUserRechargeBonus    = 10.0
)

// NewUserRechargeBonusConfig is the admin-configurable first-recharge gift for
// new users. The gift window is fixed at 24 hours after registration.
type NewUserRechargeBonusConfig struct {
	Enabled     bool    `json:"enabled"`
	BonusAmount float64 `json:"bonus_amount"`
	MinAmount   float64 `json:"min_amount"`
}

func validateNewUserRechargeAmount(value float64) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 || value > maxNewUserRechargeAmount {
		return false
	}
	return math.Abs(value*100-math.Round(value*100)) < 1e-6
}

func validateNewUserRechargeBonusConfig(cfg NewUserRechargeBonusConfig) error {
	if !validateNewUserRechargeAmount(cfg.BonusAmount) {
		return fmt.Errorf("bonus amount must be greater than 0 and at most %.0f with at most two decimal places", maxNewUserRechargeAmount)
	}
	if !validateNewUserRechargeAmount(cfg.MinAmount) {
		return fmt.Errorf("min amount must be greater than 0 and at most %.0f with at most two decimal places", maxNewUserRechargeAmount)
	}
	return nil
}

func parseNewUserRechargeAmount(raw string, fallback float64) float64 {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || !validateNewUserRechargeAmount(value) {
		return fallback
	}
	return value
}

func parseNewUserRechargeBonusConfig(vals map[string]string) NewUserRechargeBonusConfig {
	return NewUserRechargeBonusConfig{
		Enabled:     vals[SettingNewUserRechargeBonusEnabled] == "true",
		BonusAmount: parseNewUserRechargeAmount(vals[SettingNewUserRechargeBonusAmount], defaultNewUserRechargeBonus),
		MinAmount:   parseNewUserRechargeAmount(vals[SettingNewUserRechargeMinAmount], defaultNewUserRechargeMin),
	}
}

// GetNewUserRechargeBonus returns the admin-configured new user recharge gift.
// Missing settings fall back to disabled with the default amounts.
func (s *PaymentConfigService) GetNewUserRechargeBonus(ctx context.Context) (*NewUserRechargeBonusConfig, error) {
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingNewUserRechargeBonusEnabled, SettingNewUserRechargeBonusAmount, SettingNewUserRechargeMinAmount,
	})
	if err != nil {
		return nil, err
	}
	cfg := parseNewUserRechargeBonusConfig(values)
	return &cfg, nil
}

func (s *PaymentConfigService) UpdateNewUserRechargeBonus(ctx context.Context, cfg NewUserRechargeBonusConfig) error {
	if err := validateNewUserRechargeBonusConfig(cfg); err != nil {
		return err
	}
	return s.settingRepo.SetMultiple(ctx, map[string]string{
		SettingNewUserRechargeBonusEnabled: strconv.FormatBool(cfg.Enabled),
		SettingNewUserRechargeBonusAmount:  strconv.FormatFloat(cfg.BonusAmount, 'f', -1, 64),
		SettingNewUserRechargeMinAmount:    strconv.FormatFloat(cfg.MinAmount, 'f', -1, 64),
	})
}

func newUserRechargeCandidate(userCreatedAt, orderCreatedAt time.Time, faceAmount, minAmount float64) bool {
	return !userCreatedAt.IsZero() && !orderCreatedAt.Before(userCreatedAt) &&
		orderCreatedAt.Before(userCreatedAt.Add(newUserRechargeWindow)) && faceAmount >= minAmount
}

func (s *PaymentService) NewUserRechargeAvailable(ctx context.Context, userID int64, now time.Time) (bool, error) {
	benefit, err := s.configService.GetNewUserRechargeBonus(ctx)
	if err != nil {
		return false, fmt.Errorf("load new user recharge bonus config: %w", err)
	}
	if !benefit.Enabled {
		return false, nil
	}
	user, err := s.entClient.User.Get(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("load new user benefit: %w", err)
	}
	if user.WelcomeRechargeOrderID != nil || user.CreatedAt.After(now) || !now.Before(user.CreatedAt.Add(newUserRechargeWindow)) {
		return false, nil
	}
	prior, err := s.entClient.PaymentOrder.Query().Where(
		paymentorder.UserIDEQ(userID), paymentorder.OrderTypeEQ(payment.OrderTypeBalance),
		paymentorder.StatusIn(OrderStatusPaid, OrderStatusRecharging, OrderStatusCompleted),
	).Exist(ctx)
	if err != nil {
		return false, fmt.Errorf("check new user recharge history: %w", err)
	}
	return !prior, nil
}

// prepareNewUserRechargeBonus reserves the one-time gift and updates the order
// amount in a single transaction before its redeem code is created. The
// conditional user update makes concurrent fulfillment idempotent per user.
func (s *PaymentService) prepareNewUserRechargeBonus(ctx context.Context, order *dbent.PaymentOrder, lease *paymentFulfillmentLease) (*dbent.PaymentOrder, error) {
	if order == nil || lease == nil || order.OrderType != payment.OrderTypeBalance || order.PaidAt == nil ||
		order.ProviderSnapshot[newUserRechargeCandidateKey] != true {
		return order, nil
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin new user benefit transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	user, err := tx.User.Get(ctx, order.UserID)
	if err != nil {
		return nil, fmt.Errorf("load new user benefit account: %w", err)
	}
	if user.WelcomeRechargeOrderID != nil {
		return order, nil
	}
	if order.PaidAt.Before(user.CreatedAt) || !order.PaidAt.Before(user.CreatedAt.Add(newUserRechargeWindow)) {
		return order, nil
	}
	// A smaller or earlier balance recharge consumes the first-order opportunity.
	earlier, err := tx.PaymentOrder.Query().Where(
		paymentorder.UserIDEQ(order.UserID), paymentorder.OrderTypeEQ(payment.OrderTypeBalance),
		paymentorder.StatusIn(OrderStatusPaid, OrderStatusRecharging, OrderStatusCompleted),
		paymentorder.Or(paymentorder.PaidAtLT(*order.PaidAt),
			paymentorder.And(paymentorder.PaidAtEQ(*order.PaidAt), paymentorder.IDLT(order.ID))),
	).Exist(ctx)
	if err != nil {
		return nil, fmt.Errorf("check first recharge: %w", err)
	}
	if earlier {
		return order, nil
	}
	updated, err := tx.User.Update().Where(dbuser.IDEQ(order.UserID), dbuser.WelcomeRechargeOrderIDIsNil()).
		SetWelcomeRechargeOrderID(order.ID).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("reserve new user benefit: %w", err)
	}
	if updated == 0 {
		return order, nil
	}
	bonusAmount := legacyNewUserRechargeBonus
	if snapshotAmount, ok := order.ProviderSnapshot[newUserRechargeBonusAmountKey].(float64); ok && snapshotAmount > 0 {
		bonusAmount = snapshotAmount
	}
	credited, err := tx.PaymentOrder.UpdateOneID(order.ID).AddAmount(bonusAmount).AddBonusAmount(bonusAmount).
		SetUpdatedAt(time.Now().UTC().Truncate(time.Microsecond)).Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("add new user recharge bonus: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit new user recharge bonus: %w", err)
	}
	credited, err = s.entClient.PaymentOrder.Get(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("reload new user recharge bonus: %w", err)
	}
	lease.version = credited.UpdatedAt
	return credited, nil
}
