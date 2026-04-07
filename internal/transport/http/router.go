package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// NewRouter wires all routes and middleware.
func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)

	// Serve the OpenAPI spec file.
	r.Get("/swagger/openapi.json", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "docs/swagger.json")
	})

	// Swagger UI — reads spec from /swagger/openapi.json.
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/openapi.json"),
	))

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/tasks", func(r chi.Router) {
			r.Post("/", h.CreateTask)
			r.Get("/", h.ListTasks)
			r.Get("/{id}", h.GetTask)
			r.Put("/{id}", h.UpdateTask)
			r.Delete("/{id}", h.DeleteTask)
		})
	})

	return r
}
