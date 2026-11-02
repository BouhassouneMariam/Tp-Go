// Package model décrit les données du garage et leurs règles de validation.
package model

import (
	"regexp"
	"strings"
	"time"

	"github.com/BouhassouneMariam/Tp-Go/internal/apperrors"
)

var plaquePattern = regexp.MustCompile(`^[A-Z]{2}-[0-9]{3}-[A-Z]{2}$`)

// Proprietaire est composé dans Vehicule pour regrouper ses coordonnées.
type Proprietaire struct {
	Nom       string `json:"nom"`
	Telephone string `json:"telephone"`
}

type Vehicule struct {
	Plaque       string       `json:"plaque"`
	Marque       string       `json:"marque"`
	Modele       string       `json:"modele"`
	Annee        int          `json:"annee"`
	Kilometrage  int          `json:"kilometrage"`
	Proprietaire Proprietaire `json:"proprietaire"`
}

const (
	StatutEnCours  = "en_cours"
	StatutTerminee = "terminee"
	StatutAnnulee  = "annulee"
)

type Intervention struct {
	ID          int64  `json:"id"`
	Plaque      string `json:"plaque"`
	Date        string `json:"date"`
	Description string `json:"description"`
	Statut      string `json:"statut"`
}

type Stats struct {
	NombreVehicules        int            `json:"nombre_vehicules"`
	InterventionsParStatut map[string]int `json:"interventions_par_statut"`
}

func ValidatePlaque(plaque string) error {
	if !plaquePattern.MatchString(plaque) {
		return apperrors.ValidationError{Field: "plaque", Message: "format attendu AA-123-AA"}
	}
	return nil
}

// Validate utilise un récepteur valeur : la validation ne modifie pas le véhicule.
func (v Vehicule) Validate() error {
	if err := ValidatePlaque(v.Plaque); err != nil {
		return err
	}
	for _, field := range []struct {
		name  string
		value string
	}{
		{"marque", v.Marque},
		{"modele", v.Modele},
		{"proprietaire.nom", v.Proprietaire.Nom},
		{"proprietaire.telephone", v.Proprietaire.Telephone},
	} {
		if strings.TrimSpace(field.value) == "" {
			return apperrors.ValidationError{Field: field.name, Message: "ce champ est obligatoire"}
		}
	}
	if v.Annee < 1886 || v.Annee > time.Now().Year() {
		return apperrors.ValidationError{Field: "annee", Message: "doit être comprise entre 1886 et l'année actuelle"}
	}
	if v.Kilometrage < 0 {
		return apperrors.ValidationError{Field: "kilometrage", Message: "doit être positif ou nul"}
	}
	return nil
}

func (i Intervention) Validate() error {
	if err := ValidatePlaque(i.Plaque); err != nil {
		return err
	}
	if _, err := time.Parse(time.DateOnly, i.Date); err != nil {
		return apperrors.ValidationError{Field: "date", Message: "date valide attendue au format YYYY-MM-DD"}
	}
	if strings.TrimSpace(i.Description) == "" {
		return apperrors.ValidationError{Field: "description", Message: "ce champ est obligatoire"}
	}
	switch i.Statut {
	case StatutEnCours, StatutTerminee, StatutAnnulee:
		return nil
	default:
		return apperrors.ValidationError{Field: "statut", Message: "valeurs acceptées : en_cours, terminee, annulee"}
	}
}
