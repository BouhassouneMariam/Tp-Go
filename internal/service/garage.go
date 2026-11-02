// Package service applique les règles métier avant d'accéder au stockage.
package service

import (
	"context"
	"fmt"

	"github.com/BouhassouneMariam/Tp-Go/internal/model"
	"github.com/BouhassouneMariam/Tp-Go/internal/repository"
)

type Garage struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Garage {
	return &Garage{repo: repo}
}

func (g *Garage) ListVehicules(ctx context.Context) ([]model.Vehicule, error) {
	return g.repo.ListVehicules(ctx)
}

func (g *Garage) GetVehicule(ctx context.Context, plaque string) (model.Vehicule, error) {
	if err := model.ValidatePlaque(plaque); err != nil {
		return model.Vehicule{}, err
	}
	return g.repo.GetVehicule(ctx, plaque)
}

func (g *Garage) CreateVehicule(ctx context.Context, v model.Vehicule) error {
	if err := v.Validate(); err != nil {
		return err
	}
	if err := g.repo.CreateVehicule(ctx, v); err != nil {
		return fmt.Errorf("ajouter un véhicule : %w", err)
	}
	return nil
}

func (g *Garage) DeleteVehicule(ctx context.Context, plaque string) error {
	if err := model.ValidatePlaque(plaque); err != nil {
		return err
	}
	if err := g.repo.DeleteVehicule(ctx, plaque); err != nil {
		return fmt.Errorf("supprimer un véhicule : %w", err)
	}
	return nil
}

func (g *Garage) CreateIntervention(ctx context.Context, i model.Intervention) (model.Intervention, error) {
	if err := i.Validate(); err != nil {
		return i, err
	}
	created, err := g.repo.CreateIntervention(ctx, i)
	if err != nil {
		return i, fmt.Errorf("ajouter une intervention : %w", err)
	}
	return created, nil
}

func (g *Garage) ListInterventions(ctx context.Context, plaque string) ([]model.Intervention, error) {
	if _, err := g.GetVehicule(ctx, plaque); err != nil {
		return nil, err
	}
	return g.repo.ListInterventions(ctx, plaque)
}

// Stats lance les deux comptages dans des goroutines et collecte leurs résultats.
func (g *Garage) Stats(ctx context.Context) (model.Stats, error) {
	type result struct {
		vehicules int
		statuts   map[string]int
		err       error
	}
	results := make(chan result, 2)
	go func() {
		count, err := g.repo.CountVehicules(ctx)
		results <- result{vehicules: count, err: err}
	}()
	go func() {
		counts, err := g.repo.CountInterventionsParStatut(ctx)
		results <- result{statuts: counts, err: err}
	}()
	var stats model.Stats
	for range 2 {
		r := <-results
		if r.err != nil {
			return stats, fmt.Errorf("calculer les statistiques : %w", r.err)
		}
		if r.statuts != nil {
			stats.InterventionsParStatut = r.statuts
		} else {
			stats.NombreVehicules = r.vehicules
		}
	}
	return stats, nil
}
