package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repository"
)

const (
	defaultGroupName = "疯狂星期一"
	defaultQuotaUSD  = 10
	defaultBatchSize = 500
)

type mondayWindow struct {
	Start time.Time
	End   time.Time
}

func (w mondayWindow) ActivityKey() string {
	return "crazy-monday:" + w.Start.Format("2006-01-02")
}

func (w mondayWindow) DelayUntilStart(now time.Time) time.Duration {
	if !now.Before(w.Start) {
		return 0
	}
	return w.Start.Sub(now)
}

func nextMondayWindow(now time.Time, loc *time.Location) mondayWindow {
	now = now.In(loc)
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	daysUntilMonday := (int(time.Monday) - int(now.Weekday()) + 7) % 7
	if now.Weekday() == time.Monday {
		daysUntilMonday = 0
	}
	start := startOfToday.AddDate(0, 0, daysUntilMonday)
	return mondayWindow{
		Start: start,
		End:   start.AddDate(0, 0, 1),
	}
}

type options struct {
	groupName string
	quotaUSD  float64
	execute   bool
	batchSize int
	wait      bool
}

type groupConfig struct {
	ID                 int64
	Name               string
	DailyLimitUSD      sql.NullFloat64
	SubscriptionType   string
	Status             string
	DefaultValidityDay int
}

type stats struct {
	scanned   int64
	created   int64
	refreshed int64
	skipped   int64
	failed    int64
}

func main() {
	opts := parseFlags()
	if err := run(context.Background(), opts, time.Now); err != nil {
		log.Fatal(err)
	}
}

func parseFlags() options {
	opts := options{}
	flag.StringVar(&opts.groupName, "group-name", defaultGroupName, "subscription group name for the campaign")
	flag.Float64Var(&opts.quotaUSD, "quota-usd", defaultQuotaUSD, "required daily quota in USD")
	flag.BoolVar(&opts.execute, "execute", false, "write changes; default is dry-run")
	flag.IntVar(&opts.batchSize, "batch-size", defaultBatchSize, "user scan batch size")
	flag.BoolVar(&opts.wait, "wait", true, "wait until Monday 00:00 before writing when started early")
	flag.Parse()
	if opts.batchSize <= 0 {
		opts.batchSize = defaultBatchSize
	}
	return opts
}

func run(ctx context.Context, opts options, nowFunc func() time.Time) error {
	cfg, err := config.LoadForBootstrap()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return fmt.Errorf("load timezone %q: %w", cfg.Timezone, err)
	}
	window := nextMondayWindow(nowFunc(), loc)
	delay := window.DelayUntilStart(nowFunc().In(loc))
	if opts.execute && opts.wait && delay > 0 {
		log.Printf("waiting %s until activity window starts at %s", delay, window.Start.Format(time.RFC3339))
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	client, db, err := repository.InitEnt(cfg)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer func() { _ = client.Close() }()

	group, err := loadAndValidateGroup(ctx, db, opts.groupName, opts.quotaUSD)
	if err != nil {
		return err
	}

	mode := "dry-run"
	if opts.execute {
		mode = "execute"
	}
	log.Printf("mode=%s group_id=%d group_name=%q quota_usd=%.2f starts_at=%s expires_at=%s activity_key=%s",
		mode, group.ID, group.Name, opts.quotaUSD, window.Start.Format(time.RFC3339), window.End.Format(time.RFC3339), window.ActivityKey())

	result, err := applyCampaign(ctx, db, group.ID, window, opts)
	if err != nil {
		return err
	}
	log.Printf("scanned=%d created=%d refreshed=%d skipped=%d failed=%d",
		result.scanned, result.created, result.refreshed, result.skipped, result.failed)
	if !opts.execute {
		log.Println("dry-run complete; rerun with --execute to write changes")
	}
	return nil
}

func loadAndValidateGroup(ctx context.Context, db *sql.DB, groupName string, quotaUSD float64) (*groupConfig, error) {
	var g groupConfig
	err := db.QueryRowContext(ctx, `
		SELECT id, name, daily_limit_usd, subscription_type, status, default_validity_days
		FROM groups
		WHERE name = $1 AND deleted_at IS NULL
	`, groupName).Scan(&g.ID, &g.Name, &g.DailyLimitUSD, &g.SubscriptionType, &g.Status, &g.DefaultValidityDay)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group %q not found", groupName)
	}
	if err != nil {
		return nil, fmt.Errorf("load group: %w", err)
	}
	if g.Status != "active" {
		return nil, fmt.Errorf("group %q is not active: %s", groupName, g.Status)
	}
	if g.SubscriptionType != "subscription" {
		return nil, fmt.Errorf("group %q is not subscription type: %s", groupName, g.SubscriptionType)
	}
	if !g.DailyLimitUSD.Valid || g.DailyLimitUSD.Float64 != quotaUSD {
		return nil, fmt.Errorf("group %q daily_limit_usd = %v, want %.2f", groupName, g.DailyLimitUSD, quotaUSD)
	}
	return &g, nil
}

func applyCampaign(ctx context.Context, db *sql.DB, groupID int64, window mondayWindow, opts options) (stats, error) {
	var result stats
	var cursor int64
	for {
		userIDs, err := loadActiveUserIDs(ctx, db, cursor, opts.batchSize)
		if err != nil {
			return result, err
		}
		if len(userIDs) == 0 {
			return result, nil
		}
		for _, userID := range userIDs {
			result.scanned++
			status, err := applyUserSubscription(ctx, db, userID, groupID, window, opts.execute)
			if err != nil {
				result.failed++
				log.Printf("user_id=%d failed: %v", userID, err)
				continue
			}
			switch status {
			case "created":
				result.created++
			case "refreshed":
				result.refreshed++
			case "skipped":
				result.skipped++
			}
			cursor = userID
		}
	}
}

func loadActiveUserIDs(ctx context.Context, db *sql.DB, cursor int64, limit int) ([]int64, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id
		FROM users
		WHERE id > $1
		  AND status = 'active'
		  AND deleted_at IS NULL
		ORDER BY id ASC
		LIMIT $2
	`, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("load active users: %w", err)
	}
	defer rows.Close()

	userIDs := make([]int64, 0, limit)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user id: %w", err)
		}
		userIDs = append(userIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return userIDs, nil
}

func applyUserSubscription(ctx context.Context, db *sql.DB, userID, groupID int64, window mondayWindow, execute bool) (string, error) {
	activityKey := window.ActivityKey()
	if !execute {
		return dryRunUserSubscription(ctx, db, userID, groupID, activityKey)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	var subID int64
	var notes string
	err = tx.QueryRowContext(ctx, `
		SELECT id, COALESCE(notes, '')
		FROM user_subscriptions
		WHERE user_id = $1
		  AND group_id = $2
		  AND deleted_at IS NULL
		FOR UPDATE
	`, userID, groupID).Scan(&subID, &notes)
	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_subscriptions (
				user_id, group_id, starts_at, expires_at, status,
				daily_window_start, weekly_window_start, monthly_window_start,
				daily_usage_usd, weekly_usage_usd, monthly_usage_usd,
				assigned_at, notes, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, 'active',
				$3, $3, $3,
				0, 0, 0,
				NOW(), $5, NOW(), NOW()
			)
		`, userID, groupID, window.Start, window.End, activityKey)
		if err != nil {
			return "", fmt.Errorf("insert subscription: %w", err)
		}
		if err = tx.Commit(); err != nil {
			return "", fmt.Errorf("commit insert: %w", err)
		}
		committed = true
		return "created", nil
	}
	if err != nil {
		return "", fmt.Errorf("lock subscription: %w", err)
	}
	if strings.Contains(notes, activityKey) {
		if err = tx.Commit(); err != nil {
			return "", fmt.Errorf("commit skip: %w", err)
		}
		committed = true
		return "skipped", nil
	}

	updatedNotes := appendNote(notes, activityKey)
	_, err = tx.ExecContext(ctx, `
		UPDATE user_subscriptions
		SET starts_at = $1,
			expires_at = $2,
			status = 'active',
			daily_window_start = $1,
			weekly_window_start = $1,
			monthly_window_start = $1,
			daily_usage_usd = 0,
			weekly_usage_usd = 0,
			monthly_usage_usd = 0,
			assigned_at = NOW(),
			notes = $3,
			updated_at = NOW()
		WHERE id = $4
	`, window.Start, window.End, updatedNotes, subID)
	if err != nil {
		return "", fmt.Errorf("update subscription: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return "", fmt.Errorf("commit update: %w", err)
	}
	committed = true
	return "refreshed", nil
}

func dryRunUserSubscription(ctx context.Context, db *sql.DB, userID, groupID int64, activityKey string) (string, error) {
	var notes string
	err := db.QueryRowContext(ctx, `
		SELECT COALESCE(notes, '')
		FROM user_subscriptions
		WHERE user_id = $1
		  AND group_id = $2
		  AND deleted_at IS NULL
	`, userID, groupID).Scan(&notes)
	if err == sql.ErrNoRows {
		return "created", nil
	}
	if err != nil {
		return "", fmt.Errorf("load subscription: %w", err)
	}
	if strings.Contains(notes, activityKey) {
		return "skipped", nil
	}
	return "refreshed", nil
}

func appendNote(existing, note string) string {
	existing = strings.TrimSpace(existing)
	if existing == "" {
		return note
	}
	return existing + "\n" + note
}
