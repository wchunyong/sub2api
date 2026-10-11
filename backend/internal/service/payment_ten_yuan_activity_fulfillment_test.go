//go:build unit

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentauditlog"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func tenYuanFulfillmentFixture(t *testing.T) (*PaymentService, *subscriptionUserSubRepoStub, time.Time, time.Time) {
	t.Helper()
	svc := tenYuanScheduleService(t)
	groups := svc.groupRepo.(*nationalDayPromotionGroupRepoStub)
	repo := newSubscriptionUserSubRepoStub()
	subs := NewSubscriptionService(groups, repo, nil, nil, nil)
	svc.subscriptionSvc = subs
	start := time.Now().Add(-time.Hour).Truncate(time.Second)
	end := start.Add(168 * time.Hour)
	_, err := svc.entClient.ActivityRun.Create().SetCode(tenYuanActivityCode).SetCurrentMarker(tenYuanActivityCode).SetGroupID(10).SetStartsAt(start).SetEndsAt(end).SetCreatedBy(1).Save(context.Background())
	require.NoError(t, err)
	return svc, repo, start, end
}

func TestTenYuanGrantPaymentBoundariesAndReplay(t *testing.T) {
	ctx := context.Background()
	svc, repo, start, end := tenYuanFulfillmentFixture(t)
	below := &dbent.PaymentOrder{ID: 1, UserID: 41, Amount: 9, OrderType: payment.OrderTypeBalance, PaidAt: &start}
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, below))
	before := start.Add(-time.Second)
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, &dbent.PaymentOrder{ID: 2, UserID: 42, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &before}))
	require.Zero(t, repo.createCalls)
	first := &dbent.PaymentOrder{ID: 3, UserID: 42, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &start}
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, first))
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, first))
	got, err := repo.GetByUserIDAndGroupID(ctx, 42, 10)
	require.NoError(t, err)
	require.True(t, end.Equal(got.ExpiresAt))
	require.Equal(t, 1, repo.createCalls)
	claims, err := svc.entClient.ActivityClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, claims)
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, &dbent.PaymentOrder{ID: 4, UserID: 43, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &end}))
	claims, err = svc.entClient.ActivityClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, claims)
}

func TestTenYuanDelayedCallbackBeforeClose(t *testing.T) {
	ctx := context.Background()
	svc, _, start, _ := tenYuanFulfillmentFixture(t)
	closeAt := start.Add(30 * time.Minute)
	run, err := svc.entClient.ActivityRun.Query().Only(ctx)
	require.NoError(t, err)
	require.NoError(t, svc.CloseTenYuanActivity(ctx, run.ID, closeAt))
	paidAt := closeAt.Add(-time.Second)
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, &dbent.PaymentOrder{ID: 5, UserID: 44, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &paidAt}))
	claimed, err := svc.entClient.ActivityClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, claimed)
	paidAt = closeAt
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, &dbent.PaymentOrder{ID: 6, UserID: 45, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &paidAt}))
	claimed, err = svc.entClient.ActivityClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, claimed)
	audit, err := svc.entClient.PaymentAuditLog.Query().Where(paymentauditlog.ActionEQ("TEN_YUAN_ACTIVITY_SKIPPED")).Only(ctx)
	require.NoError(t, err)
	require.Contains(t, audit.Detail, "paid_after_close")
}

func TestTenYuanGrantFailureRetryable(t *testing.T) {
	ctx := context.Background()
	svc, _, start, _ := tenYuanFulfillmentFixture(t)
	groups := svc.groupRepo.(*nationalDayPromotionGroupRepoStub)
	groups.byName[tenYuanActivityGroupName].Status = "inactive"
	err := svc.applyTenYuanActivityForOrder(ctx, &dbent.PaymentOrder{ID: 7, UserID: 46, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &start})
	require.Error(t, err)
	claimed, err := svc.entClient.ActivityClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, claimed)
}

func TestTenYuanAlreadyClaimedOrderCompletesWhenGroupDisabled(t *testing.T) {
	ctx := context.Background()
	svc, repo, start, _ := tenYuanFulfillmentFixture(t)
	first := &dbent.PaymentOrder{ID: 31, UserID: 47, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &start}
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, first))
	svc.groupRepo.(*nationalDayPromotionGroupRepoStub).byName[tenYuanActivityGroupName].Status = "inactive"
	second := &dbent.PaymentOrder{ID: 32, UserID: 47, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &start}
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, second))
	require.Equal(t, 1, repo.createCalls)
}

func TestTenYuanCallbackAfterEndSkipsWithAudit(t *testing.T) {
	ctx := context.Background()
	svc := tenYuanScheduleService(t)
	start := time.Now().Add(-8 * 24 * time.Hour).Truncate(time.Second)
	end := start.Add(168 * time.Hour)
	oldRun, err := svc.entClient.ActivityRun.Create().SetCode(tenYuanActivityCode).SetGroupID(10).SetStartsAt(start).SetEndsAt(end).SetCreatedBy(1).Save(ctx)
	require.NoError(t, err)
	newStart := time.Now().Add(time.Hour).Truncate(time.Second)
	_, err = svc.entClient.ActivityRun.Create().SetCode(tenYuanActivityCode).SetCurrentMarker(tenYuanActivityCode).SetGroupID(10).SetStartsAt(newStart).SetEndsAt(newStart.Add(168 * time.Hour)).SetCreatedBy(1).Save(ctx)
	require.NoError(t, err)
	paidAt := start.Add(time.Hour)
	require.NoError(t, svc.applyTenYuanActivityForOrder(ctx, &dbent.PaymentOrder{ID: 101, UserID: 42, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &paidAt}))
	claims, err := svc.entClient.ActivityClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, claims)
	audit, err := svc.entClient.PaymentAuditLog.Query().Where(paymentauditlog.ActionEQ("TEN_YUAN_ACTIVITY_SKIPPED")).Only(ctx)
	require.NoError(t, err)
	require.Contains(t, audit.Detail, "callback_after_activity_end")
	require.Contains(t, audit.Detail, "run_id")
	require.NotZero(t, oldRun.ID)
}

type tenYuanEntSubRepo struct {
	userSubRepoNoop
	client *dbent.Client
}

func (r *tenYuanEntSubRepo) queryClient(ctx context.Context) *dbent.Client {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.client
}

func (r *tenYuanEntSubRepo) GetByUserIDAndGroupID(ctx context.Context, userID, groupID int64) (*UserSubscription, error) {
	row, err := r.queryClient(ctx).UserSubscription.Query().Where(usersubscription.UserIDEQ(userID), usersubscription.GroupIDEQ(groupID)).Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, ErrSubscriptionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &UserSubscription{ID: row.ID, UserID: row.UserID, GroupID: row.GroupID, StartsAt: row.StartsAt, ExpiresAt: row.ExpiresAt, Status: row.Status, DailyUsageUSD: row.DailyUsageUsd, Notes: *row.Notes}, nil
}

func (r *tenYuanEntSubRepo) GetByIDForUpdate(ctx context.Context, id int64) (*UserSubscription, error) {
	row, err := r.queryClient(ctx).UserSubscription.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &UserSubscription{ID: row.ID, UserID: row.UserID, GroupID: row.GroupID, StartsAt: row.StartsAt, ExpiresAt: row.ExpiresAt, Status: row.Status, DailyUsageUSD: row.DailyUsageUsd, Notes: *row.Notes}, nil
}

func (r *tenYuanEntSubRepo) Create(ctx context.Context, sub *UserSubscription) error {
	row, err := r.queryClient(ctx).UserSubscription.Create().SetUserID(sub.UserID).SetGroupID(sub.GroupID).SetStartsAt(sub.StartsAt).SetExpiresAt(sub.ExpiresAt).SetStatus(sub.Status).SetAssignedAt(sub.AssignedAt).SetNotes(sub.Notes).Save(ctx)
	if err == nil {
		sub.ID = row.ID
	}
	return err
}

func TestTenYuanConcurrentClaim(t *testing.T) {
	ctx := context.Background()
	client := newTenYuanActivityTestClient(t)
	user, err := client.User.Create().SetEmail("ten-yuan-concurrent@example.com").SetPasswordHash("hash").SetUsername("ten-yuan-concurrent").Save(ctx)
	require.NoError(t, err)
	group, err := client.Group.Create().SetName(tenYuanActivityGroupName).SetSubscriptionType(SubscriptionTypeSubscription).Save(ctx)
	require.NoError(t, err)
	groups := &nationalDayPromotionGroupRepoStub{byName: map[string]*Group{tenYuanActivityGroupName: {ID: group.ID, Name: tenYuanActivityGroupName, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}}}
	subs := NewSubscriptionService(groups, &tenYuanEntSubRepo{client: client}, nil, nil, nil)
	svc := &PaymentService{entClient: client, groupRepo: groups, subscriptionSvc: subs}
	paidAt := time.Now().Add(-time.Minute).Truncate(time.Second)
	end := paidAt.Add(168 * time.Hour)
	_, err = client.ActivityRun.Create().SetCode(tenYuanActivityCode).SetCurrentMarker(tenYuanActivityCode).SetGroupID(group.ID).SetStartsAt(paidAt).SetEndsAt(end).SetCreatedBy(user.ID).Save(ctx)
	require.NoError(t, err)
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := int64(1); i <= 2; i++ {
		wg.Add(1)
		go func(orderID int64) {
			defer wg.Done()
			results <- svc.applyTenYuanActivityForOrder(ctx, &dbent.PaymentOrder{ID: orderID, UserID: user.ID, Amount: 10, OrderType: payment.OrderTypeBalance, PaidAt: &paidAt})
		}(i)
	}
	wg.Wait()
	close(results)
	success := 0
	for result := range results {
		if result == nil {
			success++
		}
	}
	require.GreaterOrEqual(t, success, 1)
	claims, err := client.ActivityClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, claims)
	subscriptions, err := client.UserSubscription.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, subscriptions)
}
