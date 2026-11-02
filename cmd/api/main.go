// Command api démarre le serveur HTTP du garage sur le port 8080.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/BouhassouneMariam/Tp-Go/internal/handler"
	"github.com/BouhassouneMariam/Tp-Go/internal/repository"
	"github.com/BouhassouneMariam/Tp-Go/internal/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "garage.db"
	}
	err := run(ctx, dbPath)
	stop()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, dbPath string) error {
	repo, err := repository.Open(dbPath)
	if err != nil {
		return fmt.Errorf("préparer la base : %w", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			log.Printf("fermer la base : %v", err)
		}
	}()
	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler.New(service.New(repo)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErrors := make(chan error, 1)
	go func() {
		log.Print("API du garage disponible sur http://localhost:8080")
		serverErrors <- server.ListenAndServe()
	}()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("servir l'API : %w", err)
		}
		return nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			if closeErr := server.Close(); closeErr != nil {
				return fmt.Errorf("arrêter le serveur : %w", errors.Join(err, closeErr))
			}
			return fmt.Errorf("arrêter le serveur : %w", err)
		}
		return nil
	}
}
