// Command workshop-api starts the Kfz-Werkstatt customer portal API: it loads
// its configuration, opens the PostgreSQL pool, applies the migrations,
// connects the Valkey publisher and serves the HTTP routes on PORT.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"workshop-api/internal/auth"
	"workshop-api/internal/config"
	"workshop-api/internal/db"
	"workshop-api/internal/httpapi"
	"workshop-api/internal/queue"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool, ""); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	if err := auth.Seed(ctx, auth.NewStore(pool), cfg.EmployeeEmail, cfg.EmployeePassword); err != nil {
		log.Printf("seed employee: %v", err)
	}

	publisher := queue.NewPublisher(cfg.ValkeyURL, cfg.QueueName)
	router := httpapi.NewRouter(cfg, pool, publisher)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("workshop-api listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("http server: %v", err)
	}
}
