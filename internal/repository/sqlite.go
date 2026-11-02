// Package repository assure la persistance des véhicules et interventions dans SQLite.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/BouhassouneMariam/Tp-Go/internal/apperrors"
	"github.com/BouhassouneMariam/Tp-Go/internal/model"
	"modernc.org/sqlite"
	"modernc.org/sqlite/lib"
)

// Repository garde la connexion privée et expose les opérations du garage.
type Repository struct {
	db *sql.DB
}

func Open(path string) (*Repository, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("ouvrir SQLite : %w", err)
	}
	// Une seule connexion conserve les PRAGMA et évite les écritures simultanées.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	const schema = `
		PRAGMA foreign_keys = ON;
		PRAGMA busy_timeout = 5000;
		CREATE TABLE IF NOT EXISTS vehicules (
			plaque TEXT PRIMARY KEY,
			marque TEXT NOT NULL,
			modele TEXT NOT NULL,
			annee INTEGER NOT NULL,
			kilometrage INTEGER NOT NULL CHECK (kilometrage >= 0),
			proprietaire_nom TEXT NOT NULL,
			proprietaire_telephone TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS interventions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			plaque TEXT NOT NULL REFERENCES vehicules(plaque) ON DELETE RESTRICT,
			date TEXT NOT NULL,
			description TEXT NOT NULL,
			statut TEXT NOT NULL CHECK (statut IN ('en_cours', 'terminee', 'annulee'))
		);
		CREATE INDEX IF NOT EXISTS interventions_plaque ON interventions(plaque);
	`
	if _, err := db.Exec(schema); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			return nil, fmt.Errorf("initialiser SQLite : %w", errors.Join(err, closeErr))
		}
		return nil, fmt.Errorf("initialiser SQLite : %w", err)
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

const vehiculeColumns = `plaque, marque, modele, annee, kilometrage, proprietaire_nom, proprietaire_telephone`

// scanVehicule accepte aussi bien une ligne unique que les lignes d'une liste.
func scanVehicule(row interface{ Scan(...any) error }) (model.Vehicule, error) {
	var v model.Vehicule
	err := row.Scan(&v.Plaque, &v.Marque, &v.Modele, &v.Annee, &v.Kilometrage,
		&v.Proprietaire.Nom, &v.Proprietaire.Telephone)
	return v, err
}

func (r *Repository) ListVehicules(ctx context.Context) ([]model.Vehicule, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+vehiculeColumns+" FROM vehicules ORDER BY plaque")
	if err != nil {
		return nil, fmt.Errorf("lister les véhicules : %w", err)
	}
	defer rows.Close()
	vehicules := make([]model.Vehicule, 0)
	for rows.Next() {
		v, err := scanVehicule(rows)
		if err != nil {
			return nil, fmt.Errorf("lire un véhicule : %w", err)
		}
		vehicules = append(vehicules, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcourir les véhicules : %w", err)
	}
	return vehicules, nil
}

func (r *Repository) GetVehicule(ctx context.Context, plaque string) (model.Vehicule, error) {
	v, err := scanVehicule(r.db.QueryRowContext(ctx,
		"SELECT "+vehiculeColumns+" FROM vehicules WHERE plaque = ?", plaque))
	if errors.Is(err, sql.ErrNoRows) {
		return v, apperrors.ErrNotFound
	}
	if err != nil {
		return v, fmt.Errorf("chercher le véhicule : %w", err)
	}
	return v, nil
}

func (r *Repository) CreateVehicule(ctx context.Context, v model.Vehicule) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO vehicules (`+vehiculeColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		v.Plaque, v.Marque, v.Modele, v.Annee, v.Kilometrage, v.Proprietaire.Nom, v.Proprietaire.Telephone)
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && (sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY ||
		sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE) {
		return apperrors.ErrConflict
	}
	if err != nil {
		return fmt.Errorf("insérer le véhicule : %w", err)
	}
	return nil
}

func (r *Repository) DeleteVehicule(ctx context.Context, plaque string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM vehicules WHERE plaque = ?", plaque)
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && (sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY ||
		sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_TRIGGER) {
		return apperrors.ErrHasInterventions
	}
	if err != nil {
		return fmt.Errorf("supprimer le véhicule : %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("vérifier la suppression : %w", err)
	}
	if count == 0 {
		return apperrors.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateIntervention(ctx context.Context, i model.Intervention) (model.Intervention, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO interventions (plaque, date, description, statut) VALUES (?, ?, ?, ?)`,
		i.Plaque, i.Date, i.Description, i.Statut)
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY {
		return i, apperrors.ErrNotFound
	}
	if err != nil {
		return i, fmt.Errorf("insérer l'intervention : %w", err)
	}
	i.ID, err = result.LastInsertId()
	if err != nil {
		return i, fmt.Errorf("lire l'identifiant de l'intervention : %w", err)
	}
	return i, nil
}

func (r *Repository) ListInterventions(ctx context.Context, plaque string) ([]model.Intervention, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, plaque, date, description, statut FROM interventions WHERE plaque = ? ORDER BY date, id`, plaque)
	if err != nil {
		return nil, fmt.Errorf("lister les interventions : %w", err)
	}
	defer rows.Close()
	interventions := make([]model.Intervention, 0)
	for rows.Next() {
		var i model.Intervention
		if err := rows.Scan(&i.ID, &i.Plaque, &i.Date, &i.Description, &i.Statut); err != nil {
			return nil, fmt.Errorf("lire une intervention : %w", err)
		}
		interventions = append(interventions, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcourir les interventions : %w", err)
	}
	return interventions, nil
}

func (r *Repository) CountVehicules(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM vehicules").Scan(&count); err != nil {
		return 0, fmt.Errorf("compter les véhicules : %w", err)
	}
	return count, nil
}

func (r *Repository) CountInterventionsParStatut(ctx context.Context) (map[string]int, error) {
	counts := map[string]int{model.StatutEnCours: 0, model.StatutTerminee: 0, model.StatutAnnulee: 0}
	rows, err := r.db.QueryContext(ctx, "SELECT statut, COUNT(*) FROM interventions GROUP BY statut")
	if err != nil {
		return nil, fmt.Errorf("compter les interventions : %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var statut string
		var count int
		if err := rows.Scan(&statut, &count); err != nil {
			return nil, fmt.Errorf("lire le nombre d'interventions : %w", err)
		}
		counts[statut] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("parcourir les statistiques : %w", err)
	}
	return counts, nil
}
