package main

import (
	"testing"
	"time"
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
