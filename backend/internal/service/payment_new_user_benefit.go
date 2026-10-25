package service

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/payment"
)

const (
	newUserRechargeCandidateKey   = "new_user_recharge_candidate"
	newUserRechargeBonusAmountKey = "new_user_recharge_bonus_amount"
	newUserRechargeBonusAmount    = 5.0
	legacyNewUserRechargeBonus    = 10.0
)

func newUserRechargeCandidate(userCreatedAt, orderCreatedAt time.Time, faceAmount float64) bool {
	return !userCreatedAt.IsZero() && !orderCreatedAt.Before(userCreatedAt) &&
		orderCreatedAt.Before(userCreatedAt.Add(24*time.Hour)) && faceAmount >= 10
}

func (s *PaymentService) NewUserRechargeAvailable(ctx context.Context, userID int64, now time.Time) (bool, error) {
	user, err := s.entClient.User.Get(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("load new user benefit: %w", err)
	}
	if user.WelcomeRechargeOrderID != nil || user.CreatedAt.After(now) || !now.Before(user.CreatedAt.Add(24*time.Hour)) {
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
	if order.PaidAt.Before(user.CreatedAt) || !order.PaidAt.Before(user.CreatedAt.Add(24*time.Hour)) {
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
