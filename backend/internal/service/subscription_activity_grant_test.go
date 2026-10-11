//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
