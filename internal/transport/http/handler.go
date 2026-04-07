package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/katherinek727/medods-task-tracker-periodicity/internal/domain/task"
	"github.com/katherinek727/medods-task-tracker-periodicity/internal/usecase"
)

// Handler holds all HTTP handler methods for the task resource.
type Handler struct {
	uc *usecase.TaskUseCase
}

// NewHandler constructs a Handler with the given use case.
func NewHandler(uc *usecase.TaskUseCase) *Handler {
	return &Handler{uc: uc}
}

// ── DTOs ─────────────────────────────────────────────────────────────────────

type recurrenceInput struct {
	Type       task.RecurrenceType `json:"type"`
	Interval   int                 `json:"interval,omitempty"`
	DayOfMonth int                 `json:"day_of_month,omitempty"`
	Dates      []time.Time         `json:"dates,omitempty"`
}

type createTaskRequest struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Status      task.Status      `json:"status"`
	ScheduledAt time.Time        `json:"scheduled_at"`
	Recurrence  *recurrenceInput `json:"recurrence,omitempty"`
}

type updateTaskRequest struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Status      task.Status      `json:"status"`
	ScheduledAt time.Time        `json:"scheduled_at"`
	Recurrence  *recurrenceInput `json:"recurrence,omitempty"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func recurrenceFromInput(r *recurrenceInput) *task.Recurrence {
	if r == nil {
		return nil
	}
	return &task.Recurrence{
		Type:       r.Type,
		Interval:   r.Interval,
		DayOfMonth: r.DayOfMonth,
		Dates:      r.Dates,
	}
}

func parseUUID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id: must be a valid UUID")
		return uuid.UUID{}, false
	}
	return id, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request body: "+err.Error())
		return false
	}
	return true
}

// ── handlers ──────────────────────────────────────────────────────────────────

// CreateTask godoc
// @Summary      Create a task
// @Description  Creates a new task. If recurrence is provided, all instances for 1 year are generated and returned.
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        body body createTaskRequest true "Task payload"
// @Success      201  {array}  task.Task
// @Failure      400  {object} errorResponse
// @Failure      500  {object} errorResponse
// @Router       /tasks [post]
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.ScheduledAt.IsZero() {
		writeError(w, http.StatusBadRequest, "scheduled_at is required")
		return
	}

	t := &task.Task{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		ScheduledAt: req.ScheduledAt,
		Recurrence:  recurrenceFromInput(req.Recurrence),
	}

	tasks, err := h.uc.CreateTask(r.Context(), t)
	if err != nil {
		respondValidationError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, tasks)
}

// GetTask godoc
// @Summary      Get a task by ID
// @Tags         tasks
// @Produce      json
// @Param        id path string true "Task UUID"
// @Success      200 {object} task.Task
// @Failure      400 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Router       /tasks/{id} [get]
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	t, err := h.uc.GetTask(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// ListTasks godoc
// @Summary      List all tasks
// @Tags         tasks
// @Produce      json
// @Param        status    query string false "Filter by status (new|in_progress|done|cancelled)"
// @Param        from      query string false "Filter scheduled_at >= from (RFC3339)"
// @Param        to        query string false "Filter scheduled_at <= to (RFC3339)"
// @Param        parent_id query string false "Filter by parent_task_id (UUID)"
// @Success      200 {array}  task.Task
// @Failure      400 {object} errorResponse
// @Failure      500 {object} errorResponse
// @Router       /tasks [get]
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	f, err := parseListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tasks, err := h.uc.ListTasks(r.Context(), f)
	if err != nil {
		respondError(w, err)
		return
	}
	if tasks == nil {
		tasks = []*task.Task{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

// parseListFilter reads optional query parameters and builds a ListFilter.
func parseListFilter(r *http.Request) (task.ListFilter, error) {
	var f task.ListFilter

	if s := r.URL.Query().Get("status"); s != "" {
		st := task.Status(s)
		switch st {
		case task.StatusNew, task.StatusInProgress, task.StatusDone, task.StatusCancelled:
		default:
			return f, fmt.Errorf("invalid status %q: must be one of new, in_progress, done, cancelled", s)
		}
		f.Status = &st
	}

	if s := r.URL.Query().Get("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return f, fmt.Errorf("invalid from: %w", err)
		}
		f.From = &t
	}

	if s := r.URL.Query().Get("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return f, fmt.Errorf("invalid to: %w", err)
		}
		f.To = &t
	}

	if s := r.URL.Query().Get("parent_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return f, fmt.Errorf("invalid parent_id: must be a valid UUID")
		}
		f.ParentID = &id
	}

	return f, nil
}

// UpdateTask godoc
// @Summary      Update a task
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id   path string          true "Task UUID"
// @Param        body body updateTaskRequest true "Task payload"
// @Success      200 {object} task.Task
// @Failure      400 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Router       /tasks/{id} [put]
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	var req updateTaskRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	t := &task.Task{
		ID:          id,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		ScheduledAt: req.ScheduledAt,
		Recurrence:  recurrenceFromInput(req.Recurrence),
	}

	if err := h.uc.UpdateTask(r.Context(), t); err != nil {
		respondError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// DeleteTask godoc
// @Summary      Delete a task
// @Tags         tasks
// @Param        id path string true "Task UUID"
// @Success      204
// @Failure      400 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Router       /tasks/{id} [delete]
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	if err := h.uc.DeleteTask(r.Context(), id); err != nil {
		respondError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteRecurrences godoc
// @Summary      Delete all recurring instances of a task
// @Description  Deletes every instance whose parent_task_id matches the given template ID. The template itself is kept.
// @Tags         tasks
// @Param        id path string true "Template task UUID"
// @Success      200 {object} deleteRecurrencesResponse
// @Failure      400 {object} errorResponse
// @Failure      404 {object} errorResponse
// @Router       /tasks/{id}/recurrences [delete]
func (h *Handler) DeleteRecurrences(w http.ResponseWriter, r *http.Request) {
	id, ok := parseUUID(w, r)
	if !ok {
		return
	}

	count, err := h.uc.DeleteRecurrences(r.Context(), id)
	if err != nil {
		respondError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, deleteRecurrencesResponse{Deleted: count})
}

type deleteRecurrencesResponse struct {
	Deleted int64 `json:"deleted"`
}
