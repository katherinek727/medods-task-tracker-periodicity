package http

import (
	"encoding/json"
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
// @Success      200 {array}  task.Task
// @Failure      500 {object} errorResponse
// @Router       /tasks [get]
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.uc.ListTasks(r.Context())
	if err != nil {
		respondError(w, err)
		return
	}
	if tasks == nil {
		tasks = []*task.Task{}
	}
	writeJSON(w, http.StatusOK, tasks)
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
