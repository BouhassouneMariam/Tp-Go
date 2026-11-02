package model

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/BouhassouneMariam/Tp-Go/internal/apperrors"
)

func TestVehiculeValidate(t *testing.T) {
	valid := Vehicule{
		Plaque: "AA-123-BB", Marque: "Renault", Modele: "Clio", Annee: 2020, Kilometrage: 42000,
		Proprietaire: Proprietaire{Nom: "Camille Martin", Telephone: "0612345678"},
	}
	tests := []struct {
		name   string
		change func(*Vehicule)
		field  string
	}{
		{"valide", func(v *Vehicule) {}, ""},
		{"plaque en minuscules", func(v *Vehicule) { v.Plaque = "aa-123-bb" }, "plaque"},
		{"plaque sans tirets", func(v *Vehicule) { v.Plaque = "AA123BB" }, "plaque"},
		{"marque vide", func(v *Vehicule) { v.Marque = " " }, "marque"},
		{"modèle vide", func(v *Vehicule) { v.Modele = "" }, "modele"},
		{"nom vide", func(v *Vehicule) { v.Proprietaire.Nom = "" }, "proprietaire.nom"},
		{"téléphone vide", func(v *Vehicule) { v.Proprietaire.Telephone = " " }, "proprietaire.telephone"},
		{"année trop ancienne", func(v *Vehicule) { v.Annee = 1800 }, "annee"},
		{"année future", func(v *Vehicule) { v.Annee = time.Now().Year() + 1 }, "annee"},
		{"kilométrage négatif", func(v *Vehicule) { v.Kilometrage = -1 }, "kilometrage"},
		{"véhicule neuf", func(v *Vehicule) { v.Kilometrage = 0 }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := valid
			tt.change(&v)
			assertValidation(t, v.Validate(), tt.field)
		})
	}
}

func TestInterventionValidate(t *testing.T) {
	tests := []struct {
		name, plaque, date, description, statut, field string
	}{
		{"en cours", "AA-123-BB", "2024-02-29", "Vidange", StatutEnCours, ""},
		{"terminée", "AA-123-BB", "2024-03-01", "Vidange", StatutTerminee, ""},
		{"annulée", "AA-123-BB", "2024-03-01", "Vidange", StatutAnnulee, ""},
		{"plaque invalide", "ABC", "2024-03-01", "Vidange", StatutEnCours, "plaque"},
		{"date impossible", "AA-123-BB", "2023-02-29", "Vidange", StatutEnCours, "date"},
		{"format de date", "AA-123-BB", "01/03/2024", "Vidange", StatutEnCours, "date"},
		{"description vide", "AA-123-BB", "2024-03-01", " ", StatutEnCours, "description"},
		{"statut inconnu", "AA-123-BB", "2024-03-01", "Vidange", "finie", "statut"},
		{"statut manquant", "AA-123-BB", "2024-03-01", "Vidange", "", "statut"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := Intervention{Plaque: tt.plaque, Date: tt.date, Description: tt.description, Statut: tt.statut}
			assertValidation(t, i.Validate(), tt.field)
		})
	}
}

func assertValidation(t *testing.T, err error, field string) {
	t.Helper()
	if field == "" {
		if err != nil {
			t.Fatal(err)
		}
		return
	}
	var validationErr apperrors.ValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != field {
		t.Fatalf("erreur = %v, champ attendu = %s", err, field)
	}
}

func TestCompositionJSON(t *testing.T) {
	var v Vehicule
	if err := json.Unmarshal([]byte(`{"plaque":"AA-123-BB","proprietaire":{"nom":"Camille","telephone":"0612345678"}}`), &v); err != nil {
		t.Fatal(err)
	}
	if v.Plaque != "AA-123-BB" || v.Proprietaire.Nom != "Camille" || v.Proprietaire.Telephone != "0612345678" {
		t.Fatalf("composition mal décodée : %+v", v)
	}
}
