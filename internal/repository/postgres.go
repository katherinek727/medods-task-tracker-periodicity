package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/katherinek727/medods-task-tracker-periodicity/internal/domain/task"
)

type postgresRepo struct {
	db *pgxpool.Pool
}

// NewPostgresRepository returns a task.Repository backed by PostgreSQL.
func NewPostgresRepository(db *pgxpool.Pool) task.Repository {
	return &postgresRepo{db: db}
}

// ── helpers ──────────────────────────────────────────────────────────────────

// rowToTask scans a full tasks row (including recurrence columns) into a Task.
func rowToTask(row pgx.Row) (*task.Task, error) {
	var (
		t            task.Task
		recType      *string
		recInterval  *int
		recDay       *int
		recDates     []time.Time
		parentTaskID *uuid.UUID
	)

	err := row.Scan(
		&t.ID,
		&t.Title,
		&t.Description,
		&t.Status,
		&t.ScheduledAt,
		&recType,
		&recInterval,
		&recDay,
		&recDates,
		&parentTaskID,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if recType != nil {
		t.Recurrence = &task.Recurrence{
			Type: task.RecurrenceType(*recType),
		}
		if recInterval != nil {
			t.Recurrence.Interval = *recInterval
		}
		if recDay != nil {
			t.Recurrence.DayOfMonth = *recDay
		}
		if len(recDates) > 0 {
			t.Recurrence.Dates = recDates
		}
	}

	t.ParentTaskID = parentTaskID

	return &t, nil
}

const selectCols = `
	id, title, description, status, scheduled_at,
	recurrence_type, recurrence_interval, recurrence_day, recurrence_dates,
	parent_task_id, created_at, updated_at`

// ── CRUD ─────────────────────────────────────────────────────────────────────

func (r *postgresRepo) Create(ctx context.Context, t *task.Task) error {
	t.ID = uuid.New()
	now := time.Now().UTC()
	t.CreatedAt = now
	t.UpdatedAt = now

	var recType *string
	var recInterval *int
	var recDay *int
	var recDates []time.Time

	if t.Recurrence != nil {
		rt := string(t.Recurrence.Type)
		recType = &rt
		if t.Recurrence.Interval > 0 {
			recInterval = &t.Recurrence.Interval
		}
		if t.Recurrence.DayOfMonth > 0 {
			recDay = &t.Recurrence.DayOfMonth
		}
		recDates = t.Recurrence.Dates
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO tasks
			(id, title, description, status, scheduled_at,
			 recurrence_type, recurrence_interval, recurrence_day, recurrence_dates,
			 parent_task_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		t.ID, t.Title, t.Description, t.Status, t.ScheduledAt,
		recType, recInterval, recDay, recDates,
		t.ParentTaskID,
		t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository: create task: %w", err)
	}
	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	row := r.db.QueryRow(ctx,
		`SELECT `+selectCols+` FROM tasks WHERE id = $1`, id)

	t, err := rowToTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, task.ErrNotFound
		}
		return nil, fmt.Errorf("repository: get task by id: %w", err)
	}
	return t, nil
}

func (r *postgresRepo) List(ctx context.Context, f task.ListFilter) ([]*task.Task, error) {
	// Build query dynamically based on which filters are set.
	query := `SELECT ` + selectCols + ` FROM tasks WHERE TRUE`
	args := []any{}
	n := 1

	if f.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", n)
		args = append(args, string(*f.Status))
		n++
	}
	if f.From != nil {
		query += fmt.Sprintf(" AND scheduled_at >= $%d", n)
		args = append(args, *f.From)
		n++
	}
	if f.To != nil {
		query += fmt.Sprintf(" AND scheduled_at <= $%d", n)
		args = append(args, *f.To)
		n++
	}
	if f.ParentID != nil {
		query += fmt.Sprintf(" AND parent_task_id = $%d", n)
		args = append(args, *f.ParentID)
		n++
	}
	_ = n // suppress unused warning after last increment

	query += " ORDER BY scheduled_at ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("repository: list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []*task.Task
	for rows.Next() {
		t, err := rowToTask(rows)
		if err != nil {
			return nil, fmt.Errorf("repository: scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

func (r *postgresRepo) Update(ctx context.Context, t *task.Task) error {
	t.UpdatedAt = time.Now().UTC()

	var recType *string
	var recInterval *int
	var recDay *int
	var recDates []time.Time

	if t.Recurrence != nil {
		rt := string(t.Recurrence.Type)
		recType = &rt
		if t.Recurrence.Interval > 0 {
			recInterval = &t.Recurrence.Interval
		}
		if t.Recurrence.DayOfMonth > 0 {
			recDay = &t.Recurrence.DayOfMonth
		}
		recDates = t.Recurrence.Dates
	}

	tag, err := r.db.Exec(ctx, `
		UPDATE tasks SET
			title               = $1,
			description         = $2,
			status              = $3,
			scheduled_at        = $4,
			recurrence_type     = $5,
			recurrence_interval = $6,
			recurrence_day      = $7,
			recurrence_dates    = $8,
			parent_task_id      = $9,
			updated_at          = $10
		WHERE id = $11`,
		t.Title, t.Description, t.Status, t.ScheduledAt,
		recType, recInterval, recDay, recDates,
		t.ParentTaskID,
		t.UpdatedAt, t.ID,
	)
	if err != nil {
		return fmt.Errorf("repository: update task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return task.ErrNotFound
	}
	return nil
}

func (r *postgresRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("repository: delete task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return task.ErrNotFound
	}
	return nil
}

func (r *postgresRepo) DeleteByParentID(ctx context.Context, parentID uuid.UUID) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE parent_task_id = $1`, parentID)
	if err != nil {
		return 0, fmt.Errorf("repository: delete by parent id: %w", err)
	}
	return tag.RowsAffected(), nil
}
