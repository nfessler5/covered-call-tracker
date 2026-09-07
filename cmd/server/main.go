package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"covered-call-tracker/internal/poller"
	"covered-call-tracker/internal/repository"
	"covered-call-tracker/pkg/marketdata"
)

func main() {
	// 1. Initialize Postgres connection
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		dbDSN = "postgres://tracker_user:tracker_password@localhost:5432/covered_call_tracker?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbDSN)
	if err != nil {
		log.Fatalf("Failed connecting to Postgres: %v", err)
	}
	defer db.Close()

	// 2. Initialize Repositories and Market Data Client
	repo := repository.NewPostgresRepository(db)
	mdClient := marketdata.NewMockClient() // Easily swap with NewIBKRClient() later!

	// 3. Instantiate and Start Background Poller (Polls every 30 seconds)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bgPoller := poller.NewPoller(repo, mdClient, 30*time.Second)
	bgPoller.Start(ctx)

	// 4. Handle Graceful Shutdown (Listen for Ctrl+C or SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Covered Call Tracker service running. Press Ctrl+C to exit.")
	<-sigChan

	log.Println("Shutting down service...")
	bgPoller.Stop()
}