package main

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNextMondayWindowFromSundayNightWaitsUntilMonday(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, 9, 13, 23, 59, 59, 0, loc)

	window := nextMondayWindow(now, loc)

	wantStart := time.Date(2026, 9, 14, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 9, 15, 0, 0, 0, 0, loc)
	if !window.Start.Equal(wantStart) {
		t.Fatalf("start = %s, want %s", window.Start, wantStart)
	}
	if !window.End.Equal(wantEnd) {
		t.Fatalf("end = %s, want %s", window.End, wantEnd)
	}
	if got := window.DelayUntilStart(now); got != time.Second {
		t.Fatalf("delay = %s, want 1s", got)
	}
}

func TestLoadActiveUserIDsRequiresCompletedCNYSpendingAboveTen(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectQuery("(?s)FROM users.*FROM payment_orders.*status = 'completed'.*provider_snapshot.*> 10").WithArgs(int64(0), 500).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(42)))
	ids, err := loadActiveUserIDs(context.Background(), db, 0, 500)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 42 {
		t.Fatalf("ids = %v", ids)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNextMondayWindowDuringMondayUsesCurrentMonday(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	now := time.Date(2026, 9, 14, 9, 30, 0, 0, loc)

	window := nextMondayWindow(now, loc)

	wantStart := time.Date(2026, 9, 14, 0, 0, 0, 0, loc)
	wantEnd := time.Date(2026, 9, 15, 0, 0, 0, 0, loc)
	if !window.Start.Equal(wantStart) {
		t.Fatalf("start = %s, want %s", window.Start, wantStart)
	}
	if !window.End.Equal(wantEnd) {
		t.Fatalf("end = %s, want %s", window.End, wantEnd)
	}
	if got := window.DelayUntilStart(now); got != 0 {
		t.Fatalf("delay = %s, want 0", got)
	}
}

func TestActivityKeyUsesMondayDate(t *testing.T) {
	loc := time.FixedZone("CST", 8*60*60)
	window := mondayWindow{
		Start: time.Date(2026, 9, 14, 0, 0, 0, 0, loc),
		End:   time.Date(2026, 9, 15, 0, 0, 0, 0, loc),
	}

	if got, want := window.ActivityKey(), "crazy-monday:2026-09-14"; got != want {
		t.Fatalf("activity key = %q, want %q", got, want)
	}
}
