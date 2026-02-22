package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"baby-tracker-server/internal/postgres"
	"baby-tracker-server/internal/server"
)

func main() {
	addr := os.Getenv("PORT")
	if addr == "" {
		addr = "8080"
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store, err := postgres.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("failed to initialize postgres store: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("failed to close postgres store: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:              ":" + addr,
		Handler:           server.NewRouter(store),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("baby-tracker-server listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}
}
