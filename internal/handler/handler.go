// Package handler expose les routes HTTP du garage et les réponses JSON.
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"

	"github.com/BouhassouneMariam/Tp-Go/internal/apperrors"
	"github.com/BouhassouneMariam/Tp-Go/internal/model"
	"github.com/BouhassouneMariam/Tp-Go/internal/service"
)

const maxBodySize = 1 << 20 // 1 Mio

type Handler struct {
	garage *service.Garage
}

func New(garage *service.Garage) http.Handler {
	h := &Handler{garage: garage}
	mux := http.NewServeMux()
	mux.HandleFunc("/vehicules", h.vehicules)
	mux.HandleFunc("/vehicules/{plaque}", h.vehicule)
	mux.HandleFunc("/vehicules/{plaque}/interventions", h.interventions)
	mux.HandleFunc("/stats", h.stats)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "route introuvable")
	})
	return mux
}

func (h *Handler) vehicules(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		vehicules, err := h.garage.ListVehicules(r.Context())
		if err != nil {
			handleError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, vehicules)
	case http.MethodPost:
		var v model.Vehicule
		if !readJSON(w, r, &v) {
			return
		}
		if err := h.garage.CreateVehicule(r.Context(), v); err != nil {
			handleError(w, err)
			return
		}
		w.Header().Set("Location", "/vehicules/"+v.Plaque)
		writeJSON(w, http.StatusCreated, v)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func (h *Handler) vehicule(w http.ResponseWriter, r *http.Request) {
	plaque := r.PathValue("plaque")
	switch r.Method {
	case http.MethodGet:
		v, err := h.garage.GetVehicule(r.Context(), plaque)
		if err != nil {
			handleError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, v)
	case http.MethodDelete:
		if err := h.garage.DeleteVehicule(r.Context(), plaque); err != nil {
			handleError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w, "GET, DELETE")
	}
}

func (h *Handler) interventions(w http.ResponseWriter, r *http.Request) {
	plaque := r.PathValue("plaque")
	switch r.Method {
	case http.MethodGet:
		interventions, err := h.garage.ListInterventions(r.Context(), plaque)
		if err != nil {
			handleError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, interventions)
	case http.MethodPost:
		// L'identifiant vient de SQLite et la plaque vient de l'URL.
		var input struct {
			Date        string `json:"date"`
			Description string `json:"description"`
			Statut      string `json:"statut"`
		}
		if !readJSON(w, r, &input) {
			return
		}
		i, err := h.garage.CreateIntervention(r.Context(), model.Intervention{
			Plaque: plaque, Date: input.Date, Description: input.Description, Statut: input.Statut,
		})
		if err != nil {
			handleError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, i)
	default:
		methodNotAllowed(w, "GET, POST")
	}
}

func (h *Handler) stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w, "GET")
		return
	}
	stats, err := h.garage.Stats(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func readJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type attendu : application/json")
		return false
	}
	// Lire le corps borné vérifie aussi la taille des espaces après l'objet JSON.
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodySize))
	if err != nil {
		var sizeErr *http.MaxBytesError
		if errors.As(err, &sizeErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "le corps de la requête dépasse 1 Mio")
		} else {
			writeError(w, http.StatusBadRequest, "impossible de lire le corps de la requête")
		}
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		writeError(w, http.StatusBadRequest, "corps JSON invalide ou champ inconnu")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "le corps doit contenir un seul objet JSON")
		return false
	}
	return true
}

func handleError(w http.ResponseWriter, err error) {
	var validationErr apperrors.ValidationError
	switch {
	case errors.As(err, &validationErr):
		writeError(w, http.StatusBadRequest, validationErr.Error())
	case errors.Is(err, apperrors.ErrNotFound):
		writeError(w, http.StatusNotFound, apperrors.ErrNotFound.Error())
	case errors.Is(err, apperrors.ErrConflict):
		writeError(w, http.StatusConflict, apperrors.ErrConflict.Error())
	case errors.Is(err, apperrors.ErrHasInterventions):
		writeError(w, http.StatusConflict, apperrors.ErrHasInterventions.Error())
	default:
		log.Printf("erreur interne : %v", err)
		writeError(w, http.StatusInternalServerError, "erreur interne du serveur")
	}
}

func methodNotAllowed(w http.ResponseWriter, allow string) {
	w.Header().Set("Allow", allow)
	writeError(w, http.StatusMethodNotAllowed, "méthode non autorisée")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, struct {
		Erreur string `json:"erreur"`
	}{Erreur: message})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("écrire la réponse JSON : %v", err)
	}
}
