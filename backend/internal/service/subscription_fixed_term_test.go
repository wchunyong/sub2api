package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fixedTermSubscriptionRepo struct {
	*subscriptionUserSubRepoStub
	expiryUpdates int
}

func (r *fixedTermSubscriptionRepo) ExtendExpiry(ctx context.Context, id int64, expiresAt time.Time) error {
	sub, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	r.expiryUpdates++
	sub.ExpiresAt = expiresAt
	return r.Update(ctx, sub)
}

func TestAssignSubscriptionUntilPreservesExistingUsageAndStatus(t *testing.T) {
	for _, status := range []string{SubscriptionStatusActive, SubscriptionStatusSuspended, SubscriptionStatusExpired} {
		t.Run(status, func(t *testing.T) {
			ctx := context.Background()
			now := nationalDayPromotionStart().Add(time.Hour)
			window := nationalDayPromotionStart()
			repo := &fixedTermSubscriptionRepo{subscriptionUserSubRepoStub: newSubscriptionUserSubRepoStub()}
			repo.seed(&UserSubscription{
				ID: 1, UserID: 42, GroupID: 10, StartsAt: now.Add(-time.Minute),
				ExpiresAt: nationalDayPromotionEnd().Add(18 * time.Hour), Status: status,
				Notes: "national_day_2026_daily_benefit", DailyUsageUSD: 6, WeeklyUsageUSD: 8, MonthlyUsageUSD: 9,
				DailyWindowStart: &window,
			})
			groups := &subscriptionGroupRepoStub{group: &Group{ID: 10, SubscriptionType: SubscriptionTypeSubscription}}
			svc := NewSubscriptionService(groups, repo, nil, nil, nil)
			svc.now = func() time.Time { return now }
			input := &AssignSubscriptionInput{UserID: 42, GroupID: 10, Notes: "national_day_2026_daily_benefit"}
			for range 2 {
				sub, err := svc.assignSubscriptionUntil(ctx, input, nationalDayPromotionEnd())
				require.NoError(t, err)
				require.Equal(t, nationalDayPromotionEnd(), sub.ExpiresAt)
				require.Equal(t, status, sub.Status)
				require.Equal(t, 6.0, sub.DailyUsageUSD)
				require.Equal(t, 8.0, sub.WeeklyUsageUSD)
				require.Equal(t, 9.0, sub.MonthlyUsageUSD)
				require.Equal(t, window, *sub.DailyWindowStart)
			}
			require.Equal(t, 0, repo.createCalls)
			require.Equal(t, 1, repo.expiryUpdates)
		})
	}
}

func TestAssignSubscriptionUntilDoesNotExtendShorterGrantOrReplaceManualGrant(t *testing.T) {
	ctx := context.Background()
	now := nationalDayPromotionStart()
	repo := newSubscriptionUserSubRepoStub()
	groups := &subscriptionGroupRepoStub{group: &Group{ID: 10, SubscriptionType: SubscriptionTypeSubscription}}
	svc := NewSubscriptionService(groups, repo, nil, nil, nil)
	svc.now = func() time.Time { return now }
	input := &AssignSubscriptionInput{UserID: 42, GroupID: 10, Notes: "national_day_2026_daily_benefit"}
	earlier := nationalDayPromotionEnd().Add(-time.Hour)
	repo.seed(&UserSubscription{ID: 1, UserID: 42, GroupID: 10, StartsAt: now, ExpiresAt: earlier, Status: SubscriptionStatusActive, Notes: input.Notes})
	sub, err := svc.assignSubscriptionUntil(ctx, input, nationalDayPromotionEnd())
	require.NoError(t, err)
	require.Equal(t, earlier, sub.ExpiresAt)
	sub.Notes = "manual grant"
	require.NoError(t, repo.Update(ctx, sub))
	_, err = svc.assignSubscriptionUntil(ctx, input, nationalDayPromotionEnd())
	require.ErrorIs(t, err, ErrSubscriptionAssignConflict)
}

func TestAssignSubscriptionUntilRejectsEndedCampaign(t *testing.T) {
	for _, now := range []time.Time{nationalDayPromotionEnd(), nationalDayPromotionEnd().Add(time.Second)} {
		svc := NewSubscriptionService(groupRepoNoop{}, userSubRepoNoop{}, nil, nil, nil)
		svc.now = func() time.Time { return now }
		_, err := svc.assignSubscriptionUntil(context.Background(), &AssignSubscriptionInput{UserID: 42, GroupID: 10}, nationalDayPromotionEnd())
		require.ErrorIs(t, err, ErrSubscriptionExpired)
	}
}
