package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// ContentTypeJSON enforces application/json on mutating requests.
func ContentTypeJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			ct := r.Header.Get("Content-Type")
			if ct != "application/json" {
				writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// RequestLogger returns a structured request logger using chi's built-in formatter.
func RequestLogger(next http.Handler) http.Handler {
	return middleware.RequestLogger(&middleware.DefaultLogFormatter{
		Logger:  nil, // uses log.Printf
		NoColor: false,
	})(next)
}

// Timeout wraps each request with a 30-second deadline.
func Timeout(next http.Handler) http.Handler {
	return http.TimeoutHandler(next, 30*time.Second, `{"error":"request timeout"}`)
}
