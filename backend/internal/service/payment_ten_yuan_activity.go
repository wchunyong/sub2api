package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/activityclaim"
	"github.com/Wei-Shaw/sub2api/ent/activityrun"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const tenYuanActivityCode = "ten_yuan_seven_days"
const tenYuanActivityGroupName = "十元七天乐"
const tenYuanActivityDuration = 168 * time.Hour

var ErrTenYuanActivityInvalid = errors.New("invalid ten yuan activity configuration")
var ErrTenYuanActivityConflict = errors.New("ten yuan activity cannot be changed")

type TenYuanActivityScheduleInput struct {
	GroupID  int64     `json:"group_id"`
	StartsAt time.Time `json:"starts_at"`
}

type TenYuanActivityRun struct {
	ID         int64      `json:"id"`
	GroupID    int64      `json:"group_id"`
	StartsAt   time.Time  `json:"starts_at"`
	EndsAt     time.Time  `json:"ends_at"`
	DisabledAt *time.Time `json:"disabled_at,omitempty"`
	Status     string     `json:"status"`
}

type TenYuanActivityAdminState struct {
	Current        *TenYuanActivityRun  `json:"current"`
	Recent         []TenYuanActivityRun `json:"recent"`
	DefaultGroupID int64                `json:"default_group_id"`
	ServerNow      time.Time            `json:"server_now"`
}

type TenYuanActivityUserState struct {
	Enabled       bool       `json:"enabled"`
	Status        string     `json:"status"`
	RunID         int64      `json:"run_id,omitempty"`
	StartsAt      *time.Time `json:"starts_at,omitempty"`
	EndsAt        *time.Time `json:"ends_at,omitempty"`
	ServerNow     time.Time  `json:"server_now"`
	Claimed       bool       `json:"claimed"`
	GroupName     string     `json:"group_name,omitempty"`
	DailyLimitUSD *float64   `json:"daily_limit_usd,omitempty"`
	MinRecharge   float64    `json:"min_recharge"`
}

func (s *PaymentService) GetTenYuanActivityForUser(ctx context.Context, userID int64, now time.Time) (*TenYuanActivityUserState, error) {
	state := &TenYuanActivityUserState{Status: "off", ServerNow: now, MinRecharge: 10}
	if s == nil || s.entClient == nil {
		return nil, errors.New("activity storage unavailable")
	}
	row, err := s.entClient.ActivityRun.Query().Where(activityrun.CurrentMarkerEQ(tenYuanActivityCode)).Only(ctx)
	if dbent.IsNotFound(err) {
		return state, nil
	}
	if err != nil {
		return nil, err
	}
	run := tenYuanActivityRun(row, now)
	state.Status, state.RunID, state.StartsAt, state.EndsAt = run.Status, run.ID, &run.StartsAt, &run.EndsAt
	state.Enabled = run.Status == "scheduled" || run.Status == "active"
	if s.groupRepo != nil {
		if group, groupErr := s.groupRepo.GetByID(ctx, row.GroupID); groupErr == nil && group != nil {
			state.GroupName, state.DailyLimitUSD = group.Name, group.DailyLimitUSD
		}
	}
	if userID > 0 {
		state.Claimed, err = s.entClient.ActivityClaim.Query().Where(activityclaim.RunIDEQ(row.ID), activityclaim.UserIDEQ(userID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
	}
	return state, nil
}

func tenYuanActivityRun(row *dbent.ActivityRun, now time.Time) *TenYuanActivityRun {
	if row == nil {
		return nil
	}
	run := &TenYuanActivityRun{ID: row.ID, GroupID: row.GroupID, StartsAt: row.StartsAt, EndsAt: row.EndsAt, DisabledAt: row.DisabledAt}
	run.Status = tenYuanActivityStatus(run, now)
	return run
}

func tenYuanActivityStatus(run *TenYuanActivityRun, now time.Time) string {
	if run == nil {
		return "off"
	}
	if !now.Before(run.EndsAt) {
		return "ended"
	}
	if run.DisabledAt != nil && !now.Before(*run.DisabledAt) {
		return "closed"
	}
	if now.Before(run.StartsAt) {
		return "scheduled"
	}
	return "active"
}

func (s *PaymentService) tenYuanDefaultGroup(ctx context.Context) int64 {
	if s == nil || s.groupRepo == nil {
		return 0
	}
	groups, _, err := s.groupRepo.ListWithFilters(ctx, pagination.PaginationParams{Page: 1, PageSize: 100}, "", "", tenYuanActivityGroupName, nil)
	if err != nil {
		return 0
	}
	for i := range groups {
		group := &groups[i]
		if group.Name == tenYuanActivityGroupName && group.IsActive() && group.IsSubscriptionType() {
			return group.ID
		}
	}
	return 0
}

func (s *PaymentService) GetTenYuanActivityAdmin(ctx context.Context, now time.Time) (*TenYuanActivityAdminState, error) {
	if s == nil || s.entClient == nil {
		return nil, errors.New("activity storage unavailable")
	}
	state := &TenYuanActivityAdminState{DefaultGroupID: s.tenYuanDefaultGroup(ctx), ServerNow: now, Recent: []TenYuanActivityRun{}}
	rows, err := s.entClient.ActivityRun.Query().Where(activityrun.CodeEQ(tenYuanActivityCode)).Order(dbent.Desc(activityrun.FieldStartsAt)).Limit(10).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		run := tenYuanActivityRun(row, now)
		state.Recent = append(state.Recent, *run)
		if row.CurrentMarker != nil {
			state.Current = run
		}
	}
	return state, nil
}

func (s *PaymentService) ScheduleTenYuanActivity(ctx context.Context, input TenYuanActivityScheduleInput, adminID int64, now time.Time) (*TenYuanActivityRun, error) {
	if s == nil || s.entClient == nil || s.groupRepo == nil {
		return nil, errors.New("activity service unavailable")
	}
	if adminID <= 0 || input.GroupID <= 0 || !input.StartsAt.After(now) || input.StartsAt.Nanosecond() != 0 {
		return nil, fmt.Errorf("%w: start must be in the future to the second", ErrTenYuanActivityInvalid)
	}
	group, err := s.groupRepo.GetByID(ctx, input.GroupID)
	if err != nil || group == nil || !group.IsActive() || !group.IsSubscriptionType() {
		return nil, fmt.Errorf("%w: group must be an active subscription group", ErrTenYuanActivityInvalid)
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	current, err := tx.ActivityRun.Query().Where(activityrun.CurrentMarkerEQ(tenYuanActivityCode)).Only(ctx)
	if err != nil && !dbent.IsNotFound(err) {
		return nil, err
	}
	if err == nil {
		if !now.Before(current.EndsAt) {
			if err = tx.ActivityRun.UpdateOneID(current.ID).ClearCurrentMarker().Exec(ctx); err != nil {
				return nil, err
			}
		} else if now.Before(current.StartsAt) && current.DisabledAt == nil {
			// Keep an edit from reopening a round closed by another administrator.
			updateErr := tx.ActivityRun.UpdateOneID(current.ID).
				Where(activityrun.CurrentMarkerEQ(tenYuanActivityCode), activityrun.DisabledAtIsNil(), activityrun.StartsAtGT(now)).
				SetGroupID(input.GroupID).SetStartsAt(input.StartsAt).SetEndsAt(input.StartsAt.Add(tenYuanActivityDuration)).Exec(ctx)
			if updateErr != nil {
				if dbent.IsNotFound(updateErr) {
					return nil, ErrTenYuanActivityConflict
				}
				return nil, updateErr
			}
			if err = tx.Commit(); err != nil {
				return nil, err
			}
			return &TenYuanActivityRun{ID: current.ID, GroupID: input.GroupID, StartsAt: input.StartsAt, EndsAt: input.StartsAt.Add(tenYuanActivityDuration), Status: "scheduled"}, nil
		} else {
			return nil, ErrTenYuanActivityConflict
		}
	}
	row, err := tx.ActivityRun.Create().SetCode(tenYuanActivityCode).SetCurrentMarker(tenYuanActivityCode).SetGroupID(input.GroupID).SetStartsAt(input.StartsAt).SetEndsAt(input.StartsAt.Add(tenYuanActivityDuration)).SetCreatedBy(adminID).Save(ctx)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return tenYuanActivityRun(row, now), nil
}

func (s *PaymentService) CloseTenYuanActivity(ctx context.Context, runID int64, now time.Time) error {
	if s == nil || s.entClient == nil {
		return errors.New("activity storage unavailable")
	}
	run, err := s.entClient.ActivityRun.Query().Where(activityrun.IDEQ(runID), activityrun.CurrentMarkerEQ(tenYuanActivityCode)).Only(ctx)
	if err != nil {
		return ErrTenYuanActivityConflict
	}
	if !now.Before(run.EndsAt) {
		return ErrTenYuanActivityConflict
	}
	if run.DisabledAt != nil {
		return nil
	}
	err = s.entClient.ActivityRun.UpdateOneID(run.ID).
		Where(activityrun.CurrentMarkerEQ(tenYuanActivityCode), activityrun.DisabledAtIsNil(), activityrun.EndsAtGT(now)).
		SetDisabledAt(now).Exec(ctx)
	if dbent.IsNotFound(err) {
		latest, latestErr := s.entClient.ActivityRun.Get(ctx, run.ID)
		if latestErr == nil && latest.CurrentMarker != nil && latest.DisabledAt != nil {
			return nil
		}
		return ErrTenYuanActivityConflict
	}
	return err
}

// applyTenYuanActivityForOrder runs before marking a balance order completed.
// The claim and subscription change share a transaction, so a failed grant can
// be retried through the existing paid-order fulfillment path.
func (s *PaymentService) applyTenYuanActivityForOrder(ctx context.Context, order *dbent.PaymentOrder) error {
	if s == nil || s.entClient == nil || order == nil || order.OrderType != payment.OrderTypeBalance || order.Amount < 10 || order.PaidAt == nil {
		return nil
	}
	now := time.Now()
	run, err := s.entClient.ActivityRun.Query().Where(
		activityrun.CodeEQ(tenYuanActivityCode),
		activityrun.StartsAtLTE(*order.PaidAt),
		activityrun.EndsAtGT(*order.PaidAt),
	).Order(dbent.Desc(activityrun.FieldStartsAt)).First(ctx)
	if dbent.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if run.DisabledAt != nil && !order.PaidAt.Before(*run.DisabledAt) {
		if !s.hasAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_SKIPPED") {
			s.writeAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_SKIPPED", "system", map[string]any{"run_id": run.ID, "reason": "paid_after_close"})
		}
		return nil
	}
	if !now.Before(run.EndsAt) {
		if !s.hasAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_SKIPPED") {
			s.writeAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_SKIPPED", "system", map[string]any{"run_id": run.ID, "reason": "callback_after_activity_end"})
		}
		return nil
	}
	if s.subscriptionSvc == nil || s.groupRepo == nil {
		return errors.New("activity subscription service unavailable")
	}
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	txCtx := dbent.NewTxContext(ctx, tx)
	alreadyClaimed, err := tx.ActivityClaim.Query().Where(activityclaim.RunIDEQ(run.ID), activityclaim.UserIDEQ(order.UserID)).Exist(txCtx)
	if err != nil {
		return err
	}
	if alreadyClaimed {
		if err = tx.Commit(); err != nil {
			return err
		}
		if !s.hasAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_ALREADY_CLAIMED") {
			s.writeAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_ALREADY_CLAIMED", "system", map[string]any{"run_id": run.ID})
		}
		return s.subscriptionSvc.invalidateSubscriptionCaches(order.UserID, run.GroupID)
	}
	group, err := s.groupRepo.GetByID(txCtx, run.GroupID)
	if err != nil || group == nil || !group.IsActive() || !group.IsSubscriptionType() {
		_ = tx.Rollback()
		s.writeAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_GRANT_FAILED", "system", map[string]any{"run_id": run.ID, "reason": "configured_group_unavailable"})
		return fmt.Errorf("%w: configured group unavailable", ErrTenYuanActivityInvalid)
	}
	sub, err := s.subscriptionSvc.assignTenYuanActivityUntilInTx(txCtx, order.UserID, run.GroupID, run.ID, run.EndsAt)
	if err != nil {
		_ = tx.Rollback()
		s.writeAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_GRANT_FAILED", "system", map[string]any{"run_id": run.ID, "reason": err.Error()})
		return err
	}
	_, err = tx.ActivityClaim.Create().SetRunID(run.ID).SetUserID(order.UserID).SetOrderID(order.ID).SetSubscriptionID(sub.ID).SetClaimedAt(now).Save(txCtx)
	if err != nil {
		_ = tx.Rollback()
		s.writeAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_GRANT_FAILED", "system", map[string]any{"run_id": run.ID, "reason": err.Error()})
		return err
	}
	if err = tx.Commit(); err != nil {
		s.writeAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_GRANT_FAILED", "system", map[string]any{"run_id": run.ID, "reason": err.Error()})
		return err
	}
	s.writeAuditLog(ctx, order.ID, "TEN_YUAN_ACTIVITY_GRANTED", "system", map[string]any{"run_id": run.ID, "subscription_id": sub.ID, "ends_at": run.EndsAt})
	return s.subscriptionSvc.invalidateSubscriptionCaches(order.UserID, run.GroupID)
}
