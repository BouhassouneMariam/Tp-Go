package apperrors

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelles(t *testing.T) {
	for _, target := range []error{ErrNotFound, ErrConflict, ErrHasInterventions} {
		t.Run(target.Error(), func(t *testing.T) {
			wrapped := fmt.Errorf("opération : %w", target)
			if !errors.Is(wrapped, target) {
				t.Fatalf("errors.Is(%v, %v) = false", wrapped, target)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := fmt.Errorf("ajout : %w", ValidationError{Field: "plaque", Message: "invalide"})
	var validationErr ValidationError
	if !errors.As(err, &validationErr) || validationErr.Field != "plaque" {
		t.Fatalf("erreur de validation perdue : %v", err)
	}
	if validationErr.Error() != "plaque : invalide" {
		t.Fatalf("message inattendu : %s", validationErr.Error())
	}
}
