package task

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListFilter holds optional query parameters for listing tasks.
// Zero values mean "no filter applied" for that field.
type ListFilter struct {
	Status   *Status    // filter by exact status
	From     *time.Time // scheduled_at >= From
	To       *time.Time // scheduled_at <= To
	ParentID *uuid.UUID // filter by parent_task_id (recurring instances of a template)
}

// Repository defines the persistence contract for tasks.
type Repository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*Task, error)
	List(ctx context.Context, f ListFilter) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id uuid.UUID) error
	// DeleteByParentID removes all recurring instances that belong to the
	// given parent (template) task. Returns the number of rows deleted.
	DeleteByParentID(ctx context.Context, parentID uuid.UUID) (int64, error)
}
