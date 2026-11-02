package repository

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/BouhassouneMariam/Tp-Go/internal/apperrors"
	"github.com/BouhassouneMariam/Tp-Go/internal/model"
)

func openTestRepo(t *testing.T) *Repository {
	t.Helper()
	r, err := Open(filepath.Join(t.TempDir(), "garage.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	return r
}

func testVehicule() model.Vehicule {
	return model.Vehicule{
		Plaque: "AA-123-BB", Marque: "Renault", Modele: "Clio", Annee: 2020, Kilometrage: 42000,
		Proprietaire: model.Proprietaire{Nom: "Camille Martin", Telephone: "0612345678"},
	}
}

func TestVehicules(t *testing.T) {
	r := openTestRepo(t)
	ctx := context.Background()
	v := testVehicule()
	list, err := r.ListVehicules(ctx)
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("liste vide = %v, erreur = %v", list, err)
	}
	if err := r.CreateVehicule(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err := r.CreateVehicule(ctx, v); !errors.Is(err, apperrors.ErrConflict) {
		t.Fatalf("doublon : %v", err)
	}
	got, err := r.GetVehicule(ctx, v.Plaque)
	if err != nil || got != v {
		t.Fatalf("véhicule = %+v, erreur = %v", got, err)
	}
	list, err = r.ListVehicules(ctx)
	if err != nil || !reflect.DeepEqual(list, []model.Vehicule{v}) {
		t.Fatalf("liste = %+v, erreur = %v", list, err)
	}
	if err := r.DeleteVehicule(ctx, v.Plaque); err != nil {
		t.Fatal(err)
	}
	for _, plaque := range []string{v.Plaque, "ZZ-999-ZZ"} {
		t.Run(plaque, func(t *testing.T) {
			if _, err := r.GetVehicule(ctx, plaque); !errors.Is(err, apperrors.ErrNotFound) {
				t.Fatalf("lecture absente : %v", err)
			}
			if err := r.DeleteVehicule(ctx, plaque); !errors.Is(err, apperrors.ErrNotFound) {
				t.Fatalf("suppression absente : %v", err)
			}
		})
	}
}

func TestInterventionsEtContraintes(t *testing.T) {
	r := openTestRepo(t)
	ctx := context.Background()
	v := testVehicule()
	i := model.Intervention{Plaque: v.Plaque, Date: "2024-03-01", Description: "Vidange", Statut: model.StatutEnCours}
	if _, err := r.CreateIntervention(ctx, i); !errors.Is(err, apperrors.ErrNotFound) {
		t.Fatalf("intervention orpheline acceptée : %v", err)
	}
	if err := r.CreateVehicule(ctx, v); err != nil {
		t.Fatal(err)
	}
	list, err := r.ListInterventions(ctx, v.Plaque)
	if err != nil || list == nil || len(list) != 0 {
		t.Fatalf("liste vide = %v, erreur = %v", list, err)
	}
	var previousID int64
	for _, statut := range []string{model.StatutEnCours, model.StatutTerminee, model.StatutAnnulee} {
		t.Run(statut, func(t *testing.T) {
			i.Statut = statut
			created, err := r.CreateIntervention(ctx, i)
			if err != nil || created.ID <= previousID || created.Plaque != v.Plaque {
				t.Fatalf("intervention = %+v, erreur = %v", created, err)
			}
			previousID = created.ID
			if err := r.DeleteVehicule(ctx, v.Plaque); !errors.Is(err, apperrors.ErrHasInterventions) {
				t.Fatalf("suppression avec intervention : %v", err)
			}
		})
	}
	list, err = r.ListInterventions(ctx, v.Plaque)
	if err != nil || len(list) != 3 {
		t.Fatalf("interventions = %+v, erreur = %v", list, err)
	}
	counts, err := r.CountInterventionsParStatut(ctx)
	if err != nil || !reflect.DeepEqual(counts, map[string]int{"en_cours": 1, "terminee": 1, "annulee": 1}) {
		t.Fatalf("statistiques = %v, erreur = %v", counts, err)
	}
	count, err := r.CountVehicules(ctx)
	if err != nil || count != 1 {
		t.Fatalf("nombre de véhicules = %d, erreur = %v", count, err)
	}
}

func TestPersistance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "garage.db")
	r, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	v := testVehicule()
	createErr := r.CreateVehicule(context.Background(), v)
	closeErr := r.Close()
	if err := errors.Join(createErr, closeErr); err != nil {
		t.Fatal(err)
	}
	r, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	}()
	got, err := r.GetVehicule(context.Background(), v.Plaque)
	if err != nil || got != v {
		t.Fatalf("données perdues après réouverture : %+v, %v", got, err)
	}
}

func TestErreursStockage(t *testing.T) {
	r := openTestRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.ListVehicules(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("annulation perdue : %v", err)
	}
	if _, err := Open(filepath.Join(t.TempDir(), "absent", "garage.db")); err == nil {
		t.Fatal("ouverture impossible acceptée")
	}
}
