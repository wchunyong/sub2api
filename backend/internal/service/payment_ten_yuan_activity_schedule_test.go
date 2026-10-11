//go:build unit

package service

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newTenYuanActivityTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	db, err := sql.Open("sqlite", "file:ten_yuan_activity?mode=memory&cache=shared&_pragma=foreign_keys(1)&_pragma=busy_timeout(3000)&_time_format=sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(entsql.OpenDB(dialect.SQLite, db))))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func tenYuanScheduleService(t *testing.T) *PaymentService {
	t.Helper()
	group := &Group{ID: 10, Name: "十元七天乐", Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	return &PaymentService{entClient: newTenYuanActivityTestClient(t), groupRepo: &nationalDayPromotionGroupRepoStub{byName: map[string]*Group{group.Name: group}}}
}

func TestTenYuanScheduleLifecycle(t *testing.T) {
	ctx := context.Background()
	svc := tenYuanScheduleService(t)
	now := time.Date(2026, 10, 11, 8, 0, 0, 0, nationalDayPromotionLocation())
	start := now.Add(2*time.Hour + 13*time.Second)
	state, err := svc.GetTenYuanActivityAdmin(ctx, now)
	require.NoError(t, err)
	require.Equal(t, int64(10), state.DefaultGroupID)
	require.Nil(t, state.Current)
	run, err := svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 10, StartsAt: start}, 1, now)
	require.NoError(t, err)
	require.Equal(t, start.Add(168*time.Hour), run.EndsAt)
	require.Equal(t, "scheduled", tenYuanActivityStatus(run, start.Add(-time.Second)))
	require.Equal(t, "active", tenYuanActivityStatus(run, start))
	require.Equal(t, "ended", tenYuanActivityStatus(run, run.EndsAt))
	newStart := start.Add(time.Hour)
	edited, err := svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 10, StartsAt: newStart}, 1, now)
	require.NoError(t, err)
	require.Equal(t, run.ID, edited.ID)
	require.Equal(t, newStart.Add(168*time.Hour), edited.EndsAt)
	_, err = svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 10, StartsAt: newStart.Add(time.Hour)}, 1, newStart)
	require.Error(t, err)
	require.NoError(t, svc.CloseTenYuanActivity(ctx, run.ID, newStart.Add(time.Hour)))
	state, err = svc.GetTenYuanActivityAdmin(ctx, newStart.Add(time.Hour))
	require.NoError(t, err)
	require.True(t, edited.EndsAt.Equal(state.Current.EndsAt))
	require.NotNil(t, state.Current.DisabledAt)
	_, err = svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 10, StartsAt: newStart.Add(2 * time.Hour)}, 1, newStart.Add(time.Hour))
	require.Error(t, err)
	_, err = svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 10, StartsAt: edited.EndsAt.Add(time.Second)}, 1, edited.EndsAt)
	require.NoError(t, err)
}

func TestTenYuanScheduleRejectsInvalidGroupAndTime(t *testing.T) {
	ctx := context.Background()
	svc := tenYuanScheduleService(t)
	now := time.Now().Truncate(time.Second)
	_, err := svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 999, StartsAt: now.Add(time.Hour)}, 1, now)
	require.Error(t, err)
	_, err = svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 10, StartsAt: now.Add(-time.Second)}, 1, now)
	require.Error(t, err)
	_, err = svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 10, StartsAt: now.Add(time.Hour + time.Nanosecond)}, 1, now)
	require.Error(t, err)
	svc.groupRepo = &nationalDayPromotionGroupRepoStub{byName: map[string]*Group{"十元七天乐": {ID: 10, Name: "十元七天乐", Status: "inactive", SubscriptionType: SubscriptionTypeSubscription}}}
	_, err = svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 10, StartsAt: now.Add(time.Hour)}, 1, now)
	require.Error(t, err)
	svc.groupRepo = &nationalDayPromotionGroupRepoStub{byName: map[string]*Group{"十元七天乐 Plus": {ID: 11, Name: "十元七天乐 Plus", Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}}}
	state, err := svc.GetTenYuanActivityAdmin(ctx, now)
	require.NoError(t, err)
	require.Zero(t, state.DefaultGroupID)
}

func TestTenYuanScheduleConcurrent(t *testing.T) {
	ctx := context.Background()
	svc := tenYuanScheduleService(t)
	now := time.Now().Truncate(time.Second)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = svc.ScheduleTenYuanActivity(ctx, TenYuanActivityScheduleInput{GroupID: 10, StartsAt: now.Add(time.Hour)}, 1, now)
		}()
	}
	wg.Wait()
	state, err := svc.GetTenYuanActivityAdmin(ctx, now)
	require.NoError(t, err)
	require.NotNil(t, state.Current)
	require.Len(t, state.Recent, 1)
}
