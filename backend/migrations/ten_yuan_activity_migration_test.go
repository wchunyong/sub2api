package migrations

import (
	sqlpkg "database/sql"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestTenYuanActivityMigration(t *testing.T) {
	content, err := FS.ReadFile("243_ten_yuan_activity.sql")
	require.NoError(t, err)
	sql := strings.ToLower(string(content))
	for _, fragment := range []string{"activity_runs", "activity_claims", "current_marker", "unique", "run_id", "user_id", "order_id", "status"} {
		require.Contains(t, sql, fragment)
	}
	db, err := sqlpkg.Open("sqlite", ":memory:")
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`CREATE TABLE activity_runs (id INTEGER PRIMARY KEY, code TEXT, current_marker TEXT, starts_at TEXT, ends_at TEXT);
		CREATE TABLE activity_claims (id INTEGER PRIMARY KEY, run_id INTEGER, user_id INTEGER, order_id INTEGER);`)
	require.NoError(t, err)
	indexes := regexp.MustCompile(`(?i)CREATE (?:UNIQUE )?INDEX IF NOT EXISTS [^;]+;`).FindAllString(string(content), -1)
	require.Len(t, indexes, 6)
	for _, statement := range indexes {
		_, err = db.Exec(statement)
		require.NoError(t, err)
	}
	_, err = db.Exec(`INSERT INTO activity_runs(id,code,current_marker) VALUES (1,'ten_yuan_seven_days','ten_yuan_seven_days')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO activity_runs(id,code,current_marker) VALUES (2,'ten_yuan_seven_days','ten_yuan_seven_days')`)
	require.Error(t, err)
	_, err = db.Exec(`INSERT INTO activity_claims(id,run_id,user_id,order_id) VALUES (1,1,42,101)`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO activity_claims(id,run_id,user_id,order_id) VALUES (2,1,42,102)`)
	require.Error(t, err)
	_, err = db.Exec(`INSERT INTO activity_claims(id,run_id,user_id,order_id) VALUES (3,1,43,101)`)
	require.Error(t, err)
}
