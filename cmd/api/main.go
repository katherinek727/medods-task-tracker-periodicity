package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/katherinek727/medods-task-tracker-periodicity/internal/infrastructure"
	"github.com/katherinek727/medods-task-tracker-periodicity/internal/repository"
	transportHTTP "github.com/katherinek727/medods-task-tracker-periodicity/internal/transport/http"
	"github.com/katherinek727/medods-task-tracker-periodicity/internal/usecase"
)

func main() {
	ctx := context.Background()

	pool, err := infrastructure.NewPostgresPool(ctx, infrastructure.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		DBName:   getEnv("DB_NAME", "taskdb"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	})
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}
	defer pool.Close()

	repo := repository.NewPostgresRepository(pool)
	uc := usecase.New(repo)
	handler := transportHTTP.NewHandler(uc)
	router := transportHTTP.NewRouter(handler)

	addr := fmt.Sprintf(":%s", getEnv("PORT", "8080"))
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("server listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
