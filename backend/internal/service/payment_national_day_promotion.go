package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/shopspring/decimal"
)

const nationalDay2026DailyBenefitMinRecharge = 10
const nationalDayPromotionDailyBenefitGroupName = "10 元解锁国庆七天福利"
const nationalDayPromotionBalanceBonusAuditAction = "NATIONAL_DAY_2026_BALANCE_BONUS_APPLIED"
const nationalDayPromotionDailyBenefitAuditAction = "NATIONAL_DAY_2026_DAILY_BENEFIT_ASSIGNED"
const nationalDayPromotionOldUserLimitedQuotaAuditAction = "NATIONAL_DAY_2026_OLD_USER_LIMITED_QUOTA_ASSIGNED"
const nationalDayPromotionFailedAuditAction = "NATIONAL_DAY_2026_PROMOTION_FAILED"

var nationalDay2026RechargeAmounts = []float64{10, 20, 50, 100, 200, 400, 800}
var nationalDayPromotionOldUserGroupNames = map[string]string{
	"10":  "国庆老用户限时额度-2",
	"20":  "国庆老用户限时额度-4",
	"50":  "国庆老用户限时额度-10",
	"100": "国庆老用户限时额度-20",
	"200": "国庆老用户限时额度-40",
	"400": "国庆老用户限时额度-80",
	"800": "国庆老用户限时额度-160",
}

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
	if err := s.applyNationalDayDailyBenefit(ctx, o, paidAt); err != nil {
		return err
	}
	isOldUser, err := s.nationalDayPromotionIsOldUser(ctx, o.UserID)
	if err != nil {
		return err
	}
	if isOldUser {
		if err := s.applyNationalDayOldUserLimitedQuota(ctx, o); err != nil {
			return err
		}
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

func (s *PaymentService) applyNationalDayDailyBenefit(ctx context.Context, o *dbent.PaymentOrder, paidAt time.Time) error {
	if s == nil || s.entClient == nil || s.groupRepo == nil || s.subscriptionSvc == nil || o == nil {
		return nil
	}
	if o.Amount < nationalDay2026DailyBenefitMinRecharge {
		return nil
	}
	if s.hasAuditLog(ctx, o.ID, nationalDayPromotionDailyBenefitAuditAction) {
		return nil
	}
	group, err := s.nationalDayPromotionGroupByName(ctx, nationalDayPromotionDailyBenefitGroupName)
	if err != nil {
		return err
	}
	validityDays := nationalDayPromotionDaysUntilEnd(paidAt)
	_, err = s.subscriptionSvc.AssignSubscription(ctx, &AssignSubscriptionInput{
		UserID:       o.UserID,
		GroupID:      group.ID,
		ValidityDays: validityDays,
		Notes:        "national_day_2026_daily_benefit",
	})
	if err != nil && !nationalDayPromotionIgnoreExistingDailyBenefitError(err) {
		return err
	}
	s.writeAuditLog(ctx, o.ID, nationalDayPromotionDailyBenefitAuditAction, "system", map[string]any{
		"groupID":      group.ID,
		"groupName":    group.Name,
		"validityDays": validityDays,
	})
	return nil
}

func (s *PaymentService) applyNationalDayOldUserLimitedQuota(ctx context.Context, o *dbent.PaymentOrder) error {
	if s == nil || s.entClient == nil || s.groupRepo == nil || s.subscriptionSvc == nil || o == nil {
		return nil
	}
	if s.hasAuditLog(ctx, o.ID, nationalDayPromotionOldUserLimitedQuotaAuditAction) {
		return nil
	}
	groupName := nationalDayPromotionOldUserGroupNames[nationalDayPromotionAmountKey(o.Amount)]
	if groupName == "" {
		return nil
	}
	group, err := s.nationalDayPromotionGroupByName(ctx, groupName)
	if err != nil {
		return err
	}
	_, _, err = s.subscriptionSvc.AssignOrExtendSubscription(ctx, &AssignSubscriptionInput{
		UserID:       o.UserID,
		GroupID:      group.ID,
		ValidityDays: 30,
		Notes:        "national_day_2026_old_user_limited_quota order " + strconv.FormatInt(o.ID, 10),
	})
	if err != nil {
		return err
	}
	s.writeAuditLog(ctx, o.ID, nationalDayPromotionOldUserLimitedQuotaAuditAction, "system", map[string]any{
		"groupID":   group.ID,
		"groupName": group.Name,
		"quota":     nationalDayBalanceBonusAmount(o.Amount) * 0.4,
	})
	return nil
}

func (s *PaymentService) nationalDayPromotionGroupByName(ctx context.Context, name string) (*Group, error) {
	groups, _, err := s.groupRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 50}, "", StatusActive, name, nil)
	if err != nil {
		return nil, err
	}
	for i := range groups {
		if groups[i].Name == name {
			group := groups[i]
			return &group, nil
		}
	}
	return nil, ErrGroupNotFound
}

func nationalDayPromotionDaysUntilEnd(now time.Time) int {
	remaining := nationalDayPromotionEnd().Sub(now.In(nationalDayPromotionLocation()))
	if remaining <= 0 {
		return 1
	}
	days := int(remaining / (24 * time.Hour))
	if remaining%(24*time.Hour) != 0 {
		days++
	}
	if days < 1 {
		return 1
	}
	return days
}

func nationalDayPromotionIgnoreExistingDailyBenefitError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrSubscriptionAlreadyExists) || errors.Is(err, ErrSubscriptionAssignConflict) {
		return true
	}
	code := infraerrors.Code(err)
	return code == infraerrors.Code(ErrSubscriptionAlreadyExists) || code == infraerrors.Code(ErrSubscriptionAssignConflict)
}

func (s *PaymentService) logNationalDayPromotionFailure(ctx context.Context, o *dbent.PaymentOrder, err error) {
	if s == nil || s.entClient == nil || o == nil || err == nil {
		return
	}
	s.writeAuditLog(ctx, o.ID, nationalDayPromotionFailedAuditAction, "system", map[string]any{
		"error": err.Error(),
	})
}
