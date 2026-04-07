package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

	// Start server in background.
	go func() {
		log.Printf("server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Block until SIGINT or SIGTERM is received.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutdown signal received, draining connections...")

	// Give in-flight requests up to 15 seconds to complete.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}

	log.Println("server stopped cleanly")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
