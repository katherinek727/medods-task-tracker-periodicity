package task

import (
	"context"

	"github.com/google/uuid"
)

// Repository defines the persistence contract for tasks.
type Repository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*Task, error)
	List(ctx context.Context) ([]*Task, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id uuid.UUID) error
}
