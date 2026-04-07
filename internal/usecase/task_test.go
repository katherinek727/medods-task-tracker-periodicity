package usecase

import (
	"testing"
	"time"

	"github.com/katherinek727/medods-task-tracker-periodicity/internal/domain/task"
)

// fixed reference point: April 7 2026, 09:00 UTC
var base = time.Date(2026, 4, 7, 9, 0, 0, 0, time.UTC)

// horizon is exactly 1 year out
var horizon = base.AddDate(1, 0, 0)

// ── helpers ───────────────────────────────────────────────────────────────────

func dates(ts []time.Time) []int {
	days := make([]int, len(ts))
	for i, t := range ts {
		days[i] = t.Day()
	}
	return days
}

// ── daily ─────────────────────────────────────────────────────────────────────

func TestExpandRecurrence_Daily_Every1Day(t *testing.T) {
	r := &task.Recurrence{Type: task.RecurrenceDaily, Interval: 1}
	got := expandRecurrence(r, base, base.AddDate(0, 0, 5))

	if len(got) != 5 {
		t.Fatalf("expected 5 dates, got %d", len(got))
	}
	for i, d := range got {
		expected := base.AddDate(0, 0, i+1)
		if !d.Equal(expected) {
			t.Errorf("date[%d]: want %v, got %v", i, expected, d)
		}
	}
}

func TestExpandRecurrence_Daily_Every3Days(t *testing.T) {
	r := &task.Recurrence{Type: task.RecurrenceDaily, Interval: 3}
	got := expandRecurrence(r, base, base.AddDate(0, 0, 10))

	// days 3, 6, 9
	if len(got) != 3 {
		t.Fatalf("expected 3 dates, got %d: %v", len(got), got)
	}
	if !got[0].Equal(base.AddDate(0, 0, 3)) {
		t.Errorf("first date wrong: %v", got[0])
	}
}

func TestExpandRecurrence_Daily_NoneWithinWindow(t *testing.T) {
	r := &task.Recurrence{Type: task.RecurrenceDaily, Interval: 10}
	// window is only 5 days — no occurrence fits
	got := expandRecurrence(r, base, base.AddDate(0, 0, 5))
	if len(got) != 0 {
		t.Fatalf("expected 0 dates, got %d", len(got))
	}
}

// ── monthly ───────────────────────────────────────────────────────────────────

func TestExpandRecurrence_Monthly_FixedDay(t *testing.T) {
	// base is April 7; day_of_month=15 → first hit is April 15
	r := &task.Recurrence{Type: task.RecurrenceMonthly, DayOfMonth: 15}
	got := expandRecurrence(r, base, base.AddDate(0, 3, 0)) // 3-month window

	if len(got) != 3 {
		t.Fatalf("expected 3 dates, got %d: %v", len(got), got)
	}
	for _, d := range got {
		if d.Day() != 15 {
			t.Errorf("expected day 15, got %d (%v)", d.Day(), d)
		}
	}
}

func TestExpandRecurrence_Monthly_DayBeforeBase(t *testing.T) {
	// base is April 7; day_of_month=1 → first hit is May 1 (April 1 is before base)
	r := &task.Recurrence{Type: task.RecurrenceMonthly, DayOfMonth: 1}
	got := expandRecurrence(r, base, base.AddDate(0, 2, 0))

	if len(got) == 0 {
		t.Fatal("expected at least 1 date")
	}
	if got[0].Month() != time.May {
		t.Errorf("expected first hit in May, got %v", got[0].Month())
	}
}

// ── specific dates ────────────────────────────────────────────────────────────

func TestExpandRecurrence_SpecificDates_FiltersCorrectly(t *testing.T) {
	future1 := base.AddDate(0, 1, 0)  // in window
	future2 := base.AddDate(0, 6, 0)  // in window
	past := base.AddDate(0, -1, 0)    // before base — excluded
	beyond := base.AddDate(2, 0, 0)   // beyond horizon — excluded

	r := &task.Recurrence{
		Type:  task.RecurrenceSpecificDates,
		Dates: []time.Time{past, future1, future2, beyond},
	}
	got := expandRecurrence(r, base, horizon)

	if len(got) != 2 {
		t.Fatalf("expected 2 dates, got %d: %v", len(got), got)
	}
}

func TestExpandRecurrence_SpecificDates_Empty(t *testing.T) {
	r := &task.Recurrence{
		Type:  task.RecurrenceSpecificDates,
		Dates: []time.Time{base.AddDate(-1, 0, 0)}, // all in the past
	}
	got := expandRecurrence(r, base, horizon)
	if len(got) != 0 {
		t.Fatalf("expected 0 dates, got %d", len(got))
	}
}

// ── even days ─────────────────────────────────────────────────────────────────

func TestExpandRecurrence_EvenDays_AllEven(t *testing.T) {
	r := &task.Recurrence{Type: task.RecurrenceEvenDays}
	// use a small 10-day window for determinism
	got := expandRecurrence(r, base, base.AddDate(0, 0, 10))

	for _, d := range got {
		if d.Day()%2 != 0 {
			t.Errorf("odd day found: %d (%v)", d.Day(), d)
		}
	}
	if len(got) == 0 {
		t.Fatal("expected at least one even day")
	}
}

// ── odd days ──────────────────────────────────────────────────────────────────

func TestExpandRecurrence_OddDays_AllOdd(t *testing.T) {
	r := &task.Recurrence{Type: task.RecurrenceOddDays}
	got := expandRecurrence(r, base, base.AddDate(0, 0, 10))

	for _, d := range got {
		if d.Day()%2 == 0 {
			t.Errorf("even day found: %d (%v)", d.Day(), d)
		}
	}
	if len(got) == 0 {
		t.Fatal("expected at least one odd day")
	}
}

func TestExpandRecurrence_EvenOdd_NoOverlap(t *testing.T) {
	window := base.AddDate(0, 1, 0)
	even := expandRecurrence(&task.Recurrence{Type: task.RecurrenceEvenDays}, base, window)
	odd := expandRecurrence(&task.Recurrence{Type: task.RecurrenceOddDays}, base, window)

	evenSet := make(map[time.Time]bool, len(even))
	for _, d := range even {
		evenSet[d] = true
	}
	for _, d := range odd {
		if evenSet[d] {
			t.Errorf("date %v appears in both even and odd sets", d)
		}
	}
}

// ── domain validation ─────────────────────────────────────────────────────────

func TestRecurrenceValidate_Daily_InvalidInterval(t *testing.T) {
	r := &task.Recurrence{Type: task.RecurrenceDaily, Interval: 0}
	if err := r.Validate(); err == nil {
		t.Error("expected error for interval=0, got nil")
	}
}

func TestRecurrenceValidate_Monthly_DayOutOfRange(t *testing.T) {
	cases := []int{0, 31, -1, 100}
	for _, day := range cases {
		r := &task.Recurrence{Type: task.RecurrenceMonthly, DayOfMonth: day}
		if err := r.Validate(); err == nil {
			t.Errorf("expected error for day_of_month=%d, got nil", day)
		}
	}
}

func TestRecurrenceValidate_SpecificDates_EmptyList(t *testing.T) {
	r := &task.Recurrence{Type: task.RecurrenceSpecificDates, Dates: nil}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty dates, got nil")
	}
}

func TestRecurrenceValidate_EvenOdd_NoExtraFields(t *testing.T) {
	for _, rt := range []task.RecurrenceType{task.RecurrenceEvenDays, task.RecurrenceOddDays} {
		r := &task.Recurrence{Type: rt}
		if err := r.Validate(); err != nil {
			t.Errorf("unexpected error for %s: %v", rt, err)
		}
	}
}

func TestRecurrenceValidate_UnknownType(t *testing.T) {
	r := &task.Recurrence{Type: "weekly"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}
