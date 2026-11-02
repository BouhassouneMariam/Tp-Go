// Package apperrors définit les erreurs métier partagées par les couches de l'API.
package apperrors

import "errors"

var (
	ErrNotFound         = errors.New("véhicule introuvable")
	ErrConflict         = errors.New("un véhicule possède déjà cette plaque")
	ErrHasInterventions = errors.New("ce véhicule possède des interventions et ne peut pas être supprimé")
)

// ValidationError précise quel champ ne respecte pas les règles du garage.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return e.Field + " : " + e.Message
}
