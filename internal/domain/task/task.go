package task

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when a task does not exist.
var ErrNotFound = errors.New("task: not found")

// Status represents the lifecycle state of a task.
type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

// RecurrenceType defines how a task repeats.
type RecurrenceType string

const (
	// RecurrenceDaily repeats every N days (interval >= 1).
	RecurrenceDaily RecurrenceType = "daily"
	// RecurrenceMonthly repeats on a fixed day-of-month (1–30).
	RecurrenceMonthly RecurrenceType = "monthly"
	// RecurrenceSpecificDates repeats only on the listed dates.
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	// RecurrenceEvenDays repeats on even calendar days of the month.
	RecurrenceEvenDays RecurrenceType = "even_days"
	// RecurrenceOddDays repeats on odd calendar days of the month.
	RecurrenceOddDays RecurrenceType = "odd_days"
)

// Recurrence holds the periodicity settings for a task.
// Only the fields relevant to the chosen Type should be populated.
type Recurrence struct {
	// Type is the recurrence strategy.
	Type RecurrenceType `json:"type"`

	// Interval is used by RecurrenceDaily: repeat every Interval days (>= 1).
	Interval int `json:"interval,omitempty"`

	// DayOfMonth is used by RecurrenceMonthly: the calendar day (1–30).
	DayOfMonth int `json:"day_of_month,omitempty"`

	// Dates is used by RecurrenceSpecificDates: explicit list of dates (RFC3339 date part).
	Dates []time.Time `json:"dates,omitempty"`
}

// Validate checks that the Recurrence fields are consistent with its Type.
func (r *Recurrence) Validate() error {
	switch r.Type {
	case RecurrenceDaily:
		if r.Interval < 1 {
			return errors.New("recurrence: interval must be >= 1 for daily recurrence")
		}
	case RecurrenceMonthly:
		if r.DayOfMonth < 1 || r.DayOfMonth > 30 {
			return errors.New("recurrence: day_of_month must be between 1 and 30")
		}
	case RecurrenceSpecificDates:
		if len(r.Dates) == 0 {
			return errors.New("recurrence: dates must not be empty for specific_dates recurrence")
		}
	case RecurrenceEvenDays, RecurrenceOddDays:
		// no extra fields required
	default:
		return errors.New("recurrence: unknown recurrence type")
	}
	return nil
}

// Task is the core domain entity.
type Task struct {
	ID          uuid.UUID   `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Status      Status      `json:"status"`
	ScheduledAt time.Time   `json:"scheduled_at"`
	Recurrence  *Recurrence `json:"recurrence,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Validate performs basic domain validation on a Task.
func (t *Task) Validate() error {
	if t.Title == "" {
		return errors.New("task: title is required")
	}
	if t.Recurrence != nil {
		return t.Recurrence.Validate()
	}
	return nil
}
