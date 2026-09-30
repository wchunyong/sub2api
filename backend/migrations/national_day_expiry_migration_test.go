package migrations

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestNationalDayExpiryMigrationOnlyCapsCampaignDailyBenefits(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE groups (id INTEGER PRIMARY KEY, name TEXT, subscription_type TEXT);
		CREATE TABLE user_subscriptions (id INTEGER PRIMARY KEY, group_id INTEGER, notes TEXT, expires_at TEXT, updated_at TEXT, deleted_at TEXT, status TEXT, daily_usage_usd REAL);
		INSERT INTO groups VALUES (1, '10 元解锁国庆七天福利', 'subscription'), (2, '国庆老用户限时额度-20', 'subscription');
	`)
	require.NoError(t, err)
	const original = "2026-10-08 10:00:00+00:00"
	const deadline = "2026-10-07 16:00:00+00:00"
	const earlier = "2026-10-06 16:00:00+00:00"
	const marker = "national_day_2026_daily_benefit"
	cases := []struct {
		group                 int
		notes, status, expiry string
		deleted               any
		want                  string
	}{
		{1, marker, "active", original, nil, deadline},
		{1, marker, "suspended", original, nil, deadline},
		{1, marker, "expired", original, nil, deadline},
		{1, marker, "active", original, "2026-10-01", original},
		{1, "manual assignment", "active", original, nil, original},
		{2, "national_day_2026_old_user_limited_quota", "active", original, nil, original},
		{2, marker, "active", original, nil, original},
		{1, marker, "active", earlier, nil, earlier},
	}
	for i, tc := range cases {
		_, err = db.Exec("INSERT INTO user_subscriptions (id, group_id, notes, status, expires_at, deleted_at, daily_usage_usd) VALUES (?, ?, ?, ?, ?, ?, 6)", i, tc.group, tc.notes, tc.status, tc.expiry, tc.deleted)
		require.NoError(t, err)
	}
	content, err := FS.ReadFile("241_national_day_daily_benefit_expiry.sql")
	require.NoError(t, err)
	for range 2 {
		_, err = db.Exec(string(content))
		require.NoError(t, err)
		for i, tc := range cases {
			var expiry, status string
			var usage float64
			require.NoError(t, db.QueryRow("SELECT expires_at, status, daily_usage_usd FROM user_subscriptions WHERE id = ?", i).Scan(&expiry, &status, &usage))
			require.Equal(t, tc.want, expiry, "row %d", i)
			require.Equal(t, tc.status, status)
			require.Equal(t, 6.0, usage)
		}
	}
}
