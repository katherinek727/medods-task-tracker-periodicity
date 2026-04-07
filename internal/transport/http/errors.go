package http

import (
	"errors"
	"net/http"

	"github.com/katherinek727/medods-task-tracker-periodicity/internal/domain/task"
)

// errorResponse is the canonical JSON error envelope.
type errorResponse struct {
	Error string `json:"error"`
}

// domainStatusCode maps known domain errors to HTTP status codes.
func domainStatusCode(err error) int {
	switch {
	case errors.Is(err, task.ErrNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// respondError writes a JSON error response, choosing the right status code
// based on the error type.
func respondError(w http.ResponseWriter, err error) {
	code := domainStatusCode(err)
	writeJSON(w, code, errorResponse{Error: err.Error()})
}

// respondValidationError writes a 400 JSON error for validation/input failures.
func respondValidationError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
}
