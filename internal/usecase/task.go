package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/katherinek727/medods-task-tracker-periodicity/internal/domain/task"
)

// TaskUseCase contains all business logic for the task tracker.
type TaskUseCase struct {
	repo task.Repository
}

// New returns a new TaskUseCase.
func New(repo task.Repository) *TaskUseCase {
	return &TaskUseCase{repo: repo}
}

// CreateTask validates and persists a new task.
// If the task has a Recurrence, it also generates and stores all
// recurring instances within a 1-year horizon from ScheduledAt.
func (uc *TaskUseCase) CreateTask(ctx context.Context, t *task.Task) ([]*task.Task, error) {
	if err := t.Validate(); err != nil {
		return nil, fmt.Errorf("usecase: %w", err)
	}

	if t.Status == "" {
		t.Status = task.StatusNew
	}

	// Persist the "template" task first.
	if err := uc.repo.Create(ctx, t); err != nil {
		return nil, err
	}

	if t.Recurrence == nil {
		return []*task.Task{t}, nil
	}

	// Generate recurring instances.
	instances, err := uc.generateInstances(ctx, t)
	if err != nil {
		return nil, err
	}

	result := append([]*task.Task{t}, instances...)
	return result, nil
}

// generateInstances creates and stores all recurring task copies
// within a 1-year window starting the day after the template's ScheduledAt.
func (uc *TaskUseCase) generateInstances(ctx context.Context, template *task.Task) ([]*task.Task, error) {
	horizon := template.ScheduledAt.AddDate(1, 0, 0)
	dates := expandRecurrence(template.Recurrence, template.ScheduledAt, horizon)

	var created []*task.Task
	for _, d := range dates {
		instance := &task.Task{
			Title:        template.Title,
			Description:  template.Description,
			Status:       task.StatusNew,
			ScheduledAt:  d,
			Recurrence:   template.Recurrence,
			ParentTaskID: &template.ID,
		}
		if err := uc.repo.Create(ctx, instance); err != nil {
			return nil, fmt.Errorf("usecase: create recurrence instance: %w", err)
		}
		created = append(created, instance)
	}
	return created, nil
}

// expandRecurrence returns all dates matching the recurrence rule
// strictly after `from` and up to (inclusive) `until`.
func expandRecurrence(r *task.Recurrence, from, until time.Time) []time.Time {
	var dates []time.Time

	switch r.Type {
	case task.RecurrenceDaily:
		d := from.AddDate(0, 0, r.Interval)
		for !d.After(until) {
			dates = append(dates, d)
			d = d.AddDate(0, 0, r.Interval)
		}

	case task.RecurrenceMonthly:
		// Advance month by month, keeping the fixed day.
		d := nextMonthlyDate(from, r.DayOfMonth)
		for !d.After(until) {
			dates = append(dates, d)
			d = nextMonthlyDate(d, r.DayOfMonth)
		}

	case task.RecurrenceSpecificDates:
		for _, sd := range r.Dates {
			if sd.After(from) && !sd.After(until) {
				dates = append(dates, sd)
			}
		}

	case task.RecurrenceEvenDays:
		d := from.AddDate(0, 0, 1)
		for !d.After(until) {
			if d.Day()%2 == 0 {
				dates = append(dates, d)
			}
			d = d.AddDate(0, 0, 1)
		}

	case task.RecurrenceOddDays:
		d := from.AddDate(0, 0, 1)
		for !d.After(until) {
			if d.Day()%2 != 0 {
				dates = append(dates, d)
			}
			d = d.AddDate(0, 0, 1)
		}
	}

	return dates
}

// nextMonthlyDate returns the next occurrence of dayOfMonth after `after`.
func nextMonthlyDate(after time.Time, dayOfMonth int) time.Time {
	y, m, _ := after.Date()
	loc := after.Location()
	h, min, s := after.Clock()

	candidate := time.Date(y, m, dayOfMonth, h, min, s, 0, loc)
	if !candidate.After(after) {
		// Move to next month.
		candidate = time.Date(y, m+1, dayOfMonth, h, min, s, 0, loc)
	}
	return candidate
}

// GetTask retrieves a single task by ID.
func (uc *TaskUseCase) GetTask(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	return uc.repo.GetByID(ctx, id)
}

// ListTasks returns all tasks ordered by scheduled_at.
func (uc *TaskUseCase) ListTasks(ctx context.Context) ([]*task.Task, error) {
	return uc.repo.List(ctx)
}

// UpdateTask validates and updates an existing task.
func (uc *TaskUseCase) UpdateTask(ctx context.Context, t *task.Task) error {
	if err := t.Validate(); err != nil {
		return fmt.Errorf("usecase: %w", err)
	}
	return uc.repo.Update(ctx, t)
}

// DeleteTask removes a task by ID.
func (uc *TaskUseCase) DeleteTask(ctx context.Context, id uuid.UUID) error {
	return uc.repo.Delete(ctx, id)
}

// DeleteRecurrences removes all recurring instances that belong to the given
// template task. It verifies the template exists first, then deletes its
// children. The template itself is NOT deleted.
// Returns the count of deleted instances.
func (uc *TaskUseCase) DeleteRecurrences(ctx context.Context, templateID uuid.UUID) (int64, error) {
	if _, err := uc.repo.GetByID(ctx, templateID); err != nil {
		return 0, err // propagates task.ErrNotFound
	}
	return uc.repo.DeleteByParentID(ctx, templateID)
}
