package service

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"github.com/BouhassouneMariam/Tp-Go/internal/apperrors"
	"github.com/BouhassouneMariam/Tp-Go/internal/model"
	"github.com/BouhassouneMariam/Tp-Go/internal/repository"
)

func newTestGarage(t *testing.T) *Garage {
	t.Helper()
	repo, err := repository.Open(filepath.Join(t.TempDir(), "garage.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := repo.Close(); err != nil {
			t.Error(err)
		}
	})
	return New(repo)
}

func TestValidationAvantStockage(t *testing.T) {
	g := newTestGarage(t)
	ctx := context.Background()
	tests := []struct {
		name string
		call func() error
	}{
		{"ajout véhicule", func() error { return g.CreateVehicule(ctx, model.Vehicule{}) }},
		{"lecture", func() error { _, err := g.GetVehicule(ctx, "invalide"); return err }},
		{"suppression", func() error { return g.DeleteVehicule(ctx, "invalide") }},
		{"liste interventions", func() error { _, err := g.ListInterventions(ctx, "invalide"); return err }},
		{"ajout intervention", func() error { _, err := g.CreateIntervention(ctx, model.Intervention{}); return err }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var validationErr apperrors.ValidationError
			if err := tt.call(); !errors.As(err, &validationErr) {
				t.Fatalf("erreur de validation attendue : %v", err)
			}
		})
	}
	stats, err := g.Stats(ctx)
	if err != nil || stats.NombreVehicules != 0 || len(stats.InterventionsParStatut) != 3 {
		t.Fatalf("base vide : %+v, %v", stats, err)
	}
	if _, err := g.ListInterventions(ctx, "ZZ-999-ZZ"); !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("véhicule absent : %v", err)
	}
}

func TestStatsConcurrentes(t *testing.T) {
	g := newTestGarage(t)
	ctx := context.Background()
	v := model.Vehicule{
		Plaque: "AA-123-BB", Marque: "Renault", Modele: "Clio", Annee: 2020,
		Proprietaire: model.Proprietaire{Nom: "Camille", Telephone: "0612345678"},
	}
	if err := g.CreateVehicule(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err := g.CreateVehicule(ctx, v); !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("doublon perdu dans la propagation : %v", err)
	}
	for _, statut := range []string{model.StatutEnCours, model.StatutEnCours, model.StatutTerminee} {
		if _, err := g.CreateIntervention(ctx, model.Intervention{
			Plaque: v.Plaque, Date: "2024-03-01", Description: "Révision", Statut: statut,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.DeleteVehicule(ctx, v.Plaque); !errors.Is(err, apperrors.ErrHasInterventions) {
		t.Fatalf("contrainte perdue dans la propagation : %v", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stats, err := g.Stats(ctx)
			if err != nil || stats.NombreVehicules != 1 || stats.InterventionsParStatut[model.StatutEnCours] != 2 ||
				stats.InterventionsParStatut[model.StatutTerminee] != 1 || stats.InterventionsParStatut[model.StatutAnnulee] != 0 {
				t.Errorf("stats = %+v, erreur = %v", stats, err)
			}
		}()
	}
	wg.Wait()
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := g.Stats(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("annulation du calcul : %v", err)
	}
}
