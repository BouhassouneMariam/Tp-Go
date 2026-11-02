// Command api démarre le serveur HTTP du garage sur le port 8080.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/BouhassouneMariam/Tp-Go/internal/handler"
	"github.com/BouhassouneMariam/Tp-Go/internal/repository"
	"github.com/BouhassouneMariam/Tp-Go/internal/service"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "garage.db"
	}
	if err := run(dbPath); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(dbPath string) error {
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
	log.Print("Démarrage de l'API du garage sur http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("servir l'API : %w", err)
	}
	return nil
}
