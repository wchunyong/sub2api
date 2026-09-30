package service

import (
	"context"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/shopspring/decimal"
)

const nationalDay2026DailyBenefitMinRecharge = 10
const nationalDayPromotionBalanceBonusAuditAction = "NATIONAL_DAY_2026_BALANCE_BONUS_APPLIED"
const nationalDayPromotionFailedAuditAction = "NATIONAL_DAY_2026_PROMOTION_FAILED"

var nationalDay2026RechargeAmounts = []float64{10, 20, 50, 100, 200, 400, 800}

type NationalDayPromotionCheckout struct {
	Active                             bool               `json:"active"`
	IsOldUser                          bool               `json:"is_old_user"`
	DailyBenefitMinRecharge            float64            `json:"daily_benefit_min_recharge"`
	CreditedAmountByPaymentAmount      map[string]float64 `json:"credited_amount_by_payment_amount"`
	OldUserLimitedQuotaByPaymentAmount map[string]float64 `json:"old_user_limited_quota_by_payment_amount"`
}

func nationalDayPromotionLocation() *time.Location {
	return time.FixedZone("Asia/Shanghai", 8*60*60)
}

func nationalDayPromotionStart() time.Time {
	loc := nationalDayPromotionLocation()
	return time.Date(2026, 9, 30, 18, 0, 0, 0, loc)
}

func nationalDayPromotionEnd() time.Time {
	loc := nationalDayPromotionLocation()
	return time.Date(2026, 10, 8, 0, 0, 0, 0, loc)
}

func nationalDayPromotionActiveAt(now time.Time) bool {
	start := nationalDayPromotionStart()
	end := nationalDayPromotionEnd()
	inShanghai := now.In(nationalDayPromotionLocation())
	return !inShanghai.Before(start) && inShanghai.Before(end)
}

func nationalDayPromotionAmountKey(amount float64) string {
	return strconv.FormatFloat(amount, 'f', -1, 64)
}

func nationalDayPromotionPreviewMaps() (map[string]float64, map[string]float64) {
	credited := make(map[string]float64, len(nationalDay2026RechargeAmounts))
	oldUserQuota := make(map[string]float64, len(nationalDay2026RechargeAmounts))
	for _, amount := range nationalDay2026RechargeAmounts {
		key := nationalDayPromotionAmountKey(amount)
		credited[key] = decimal.NewFromFloat(amount).Mul(decimal.NewFromFloat(1.5)).Round(2).InexactFloat64()
		oldUserQuota[key] = decimal.NewFromFloat(amount).Mul(decimal.NewFromFloat(0.2)).Round(2).InexactFloat64()
	}
	return credited, oldUserQuota
}

func (s *PaymentService) BuildNationalDayPromotionCheckout(ctx context.Context, userID int64, now time.Time) (*NationalDayPromotionCheckout, error) {
	credited, oldUserQuota := nationalDayPromotionPreviewMaps()
	preview := &NationalDayPromotionCheckout{
		Active:                             nationalDayPromotionActiveAt(now),
		DailyBenefitMinRecharge:            nationalDay2026DailyBenefitMinRecharge,
		CreditedAmountByPaymentAmount:      credited,
		OldUserLimitedQuotaByPaymentAmount: oldUserQuota,
	}
	if !preview.Active {
		return preview, nil
	}
	isOldUser, err := s.nationalDayPromotionIsOldUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	preview.IsOldUser = isOldUser
	return preview, nil
}

func (s *PaymentService) nationalDayPromotionIsOldUser(ctx context.Context, userID int64) (bool, error) {
	if s == nil || s.entClient == nil || userID <= 0 {
		return false, nil
	}
	return s.entClient.PaymentOrder.Query().
		Where(
			paymentorder.UserIDEQ(userID),
			paymentorder.OrderTypeEQ(payment.OrderTypeBalance),
			paymentorder.StatusEQ(OrderStatusCompleted),
			paymentorder.PaidAtLT(nationalDayPromotionStart()),
		).
		Exist(ctx)
}

func (s *PaymentService) applyNationalDayPromotionForOrder(ctx context.Context, o *dbent.PaymentOrder) error {
	if o == nil || o.OrderType != payment.OrderTypeBalance {
		return nil
	}
	paidAt := time.Now()
	if o.PaidAt != nil {
		paidAt = *o.PaidAt
	}
	if !nationalDayPromotionActiveAt(paidAt) {
		return nil
	}
	if err := s.applyNationalDayBalanceBonus(ctx, o); err != nil {
		return err
	}
	return nil
}

func nationalDayBalanceBonusAmount(paymentAmount float64) float64 {
	return decimal.NewFromFloat(paymentAmount).
		Mul(decimal.NewFromFloat(0.5)).
		Round(2).
		InexactFloat64()
}

func (s *PaymentService) applyNationalDayBalanceBonus(ctx context.Context, o *dbent.PaymentOrder) error {
	if s == nil || s.entClient == nil || s.userRepo == nil || o == nil {
		return nil
	}
	if s.hasAuditLog(ctx, o.ID, nationalDayPromotionBalanceBonusAuditAction) {
		return nil
	}
	bonus := nationalDayBalanceBonusAmount(o.Amount)
	if bonus <= 0 {
		return nil
	}
	change, err := s.userRepo.AdjustBalance(ctx, o.UserID, bonus)
	if err != nil {
		return err
	}
	s.writeAuditLog(ctx, o.ID, nationalDayPromotionBalanceBonusAuditAction, "system", map[string]any{
		"baseAmount": o.Amount,
		"bonus":      bonus,
		"oldBalance": change.Old,
		"newBalance": change.New,
	})
	return nil
}

func (s *PaymentService) logNationalDayPromotionFailure(ctx context.Context, o *dbent.PaymentOrder, err error) {
	if s == nil || s.entClient == nil || o == nil || err == nil {
		return
	}
	s.writeAuditLog(ctx, o.ID, nationalDayPromotionFailedAuditAction, "system", map[string]any{
		"error": err.Error(),
	})
}
