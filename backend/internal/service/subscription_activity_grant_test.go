//go:build unit

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type wrappedNotFoundActivitySubRepo struct {
	*subscriptionUserSubRepoStub
}

func (r *wrappedNotFoundActivitySubRepo) GetByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	sub, err := r.subscriptionUserSubRepoStub.GetByUserIDAndGroupID(ctx, userID, groupID)
	if errors.Is(err, ErrSubscriptionNotFound) {
		return nil, ErrSubscriptionNotFound.WithCause(errors.New("ent: user_subscription not found"))
	}
	return sub, err
}

func TestTenYuanGrantCreatesWhenRepositoryWrapsNotFound(t *testing.T) {
	ctx := context.Background()
	group := &Group{ID: 10, Name: tenYuanActivityGroupName, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	groups := &nationalDayPromotionGroupRepoStub{byName: map[string]*Group{group.Name: group}}
	repo := &wrappedNotFoundActivitySubRepo{subscriptionUserSubRepoStub: newSubscriptionUserSubRepoStub()}
	subs := NewSubscriptionService(groups, repo, nil, nil, nil)
	now := time.Now().Truncate(time.Second)
	subs.now = func() time.Time { return now }

	sub, err := subs.assignTenYuanActivityUntilInTx(ctx, 42, 10, 1, now.Add(168*time.Hour))
	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusActive, sub.Status)
	require.Equal(t, 1, repo.createCalls)
}

func TestTenYuanGrantRenewsExpiredNonActivitySubscription(t *testing.T) {
	ctx := context.Background()
	group := &Group{ID: 10, Name: tenYuanActivityGroupName, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	groups := &nationalDayPromotionGroupRepoStub{byName: map[string]*Group{group.Name: group}}
	repo := newSubscriptionUserSubRepoStub()
	subs := NewSubscriptionService(groups, repo, nil, nil, nil)
	now := time.Now().Truncate(time.Second)
	subs.now = func() time.Time { return now }
	repo.seed(&UserSubscription{ID: 5, UserID: 42, GroupID: 10, StartsAt: now.Add(-8 * 24 * time.Hour), ExpiresAt: now.Add(-3 * 24 * time.Hour), Status: SubscriptionStatusExpired, Notes: "national_day_2026_daily_benefit", DailyUsageUSD: 3})

	sub, err := subs.assignTenYuanActivityUntilInTx(ctx, 42, 10, 1, now.Add(168*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(5), sub.ID)
	require.Equal(t, SubscriptionStatusActive, sub.Status)
	require.Contains(t, sub.Notes, "national_day_2026_daily_benefit")
	require.Contains(t, sub.Notes, "ten_yuan_seven_days:1")
	require.Zero(t, sub.DailyUsageUSD)
	require.Zero(t, repo.createCalls)
}

func TestTenYuanGrantDoesNotReactivateSuspendedSubscription(t *testing.T) {
	ctx := context.Background()
	group := &Group{ID: 10, Name: tenYuanActivityGroupName, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	repo := newSubscriptionUserSubRepoStub()
	subs := NewSubscriptionService(&nationalDayPromotionGroupRepoStub{byName: map[string]*Group{group.Name: group}}, repo, nil, nil, nil)
	now := time.Now().Truncate(time.Second)
	subs.now = func() time.Time { return now }
	repo.seed(&UserSubscription{ID: 6, UserID: 42, GroupID: 10, StartsAt: now.Add(-8 * 24 * time.Hour), ExpiresAt: now.Add(-3 * 24 * time.Hour), Status: SubscriptionStatusSuspended, Notes: "manual suspension"})

	_, err := subs.assignTenYuanActivityUntilInTx(ctx, 42, 10, 1, now.Add(168*time.Hour))
	require.ErrorIs(t, err, ErrSubscriptionAssignConflict)
	got, err := repo.GetByUserIDAndGroupID(ctx, 42, 10)
	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusSuspended, got.Status)
}

func TestTenYuanGrantPreservesUsageAndRenewsNextRound(t *testing.T) {
	ctx := context.Background()
	group := &Group{ID: 10, Name: tenYuanActivityGroupName, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	groups := &nationalDayPromotionGroupRepoStub{byName: map[string]*Group{group.Name: group}}
	repo := newSubscriptionUserSubRepoStub()
	subs := NewSubscriptionService(groups, repo, nil, nil, nil)
	now := time.Now().Truncate(time.Second)
	subs.now = func() time.Time { return now }
	expiry := now.Add(168 * time.Hour)
	first, err := subs.assignTenYuanActivityUntilInTx(ctx, 42, 10, 1, expiry)
	require.NoError(t, err)
	require.Equal(t, expiry, first.ExpiresAt)
	first.DailyUsageUSD = 3
	require.NoError(t, repo.Update(ctx, first))
	second, err := subs.assignTenYuanActivityUntilInTx(ctx, 42, 10, 1, expiry)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, 3.0, second.DailyUsageUSD)
	subs.now = func() time.Time { return expiry.Add(time.Second) }
	third, err := subs.assignTenYuanActivityUntilInTx(ctx, 42, 10, 2, expiry.Add(168*time.Hour))
	require.NoError(t, err)
	require.Equal(t, first.ID, third.ID)
	require.Zero(t, third.DailyUsageUSD)
	require.Equal(t, expiry.Add(168*time.Hour), third.ExpiresAt)
}

func TestTenYuanGrantRejectsManualSubscription(t *testing.T) {
	ctx := context.Background()
	group := &Group{ID: 10, Name: tenYuanActivityGroupName, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	groups := &nationalDayPromotionGroupRepoStub{byName: map[string]*Group{group.Name: group}}
	repo := newSubscriptionUserSubRepoStub()
	subs := NewSubscriptionService(groups, repo, nil, nil, nil)
	now := time.Now().Truncate(time.Second)
	subs.now = func() time.Time { return now }
	require.NoError(t, repo.Create(ctx, &UserSubscription{UserID: 42, GroupID: 10, StartsAt: now, ExpiresAt: now.Add(30 * time.Hour), Status: SubscriptionStatusActive, Notes: "manual"}))
	_, err := subs.assignTenYuanActivityUntilInTx(ctx, 42, 10, 1, now.Add(168*time.Hour))
	require.ErrorIs(t, err, ErrSubscriptionAssignConflict)
	got, err := repo.GetByUserIDAndGroupID(ctx, 42, 10)
	require.NoError(t, err)
	require.Equal(t, "manual", got.Notes)
}
