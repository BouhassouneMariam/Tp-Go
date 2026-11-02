package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BouhassouneMariam/Tp-Go/internal/apperrors"
	"github.com/BouhassouneMariam/Tp-Go/internal/model"
	"github.com/BouhassouneMariam/Tp-Go/internal/repository"
	"github.com/BouhassouneMariam/Tp-Go/internal/service"
)

const vehiculeJSON = `{"plaque":"AA-123-BB","marque":"Renault","modele":"Clio","annee":2020,"kilometrage":42000,"proprietaire":{"nom":"Camille Martin","telephone":"0612345678"}}`
const interventionJSON = `{"date":"2024-03-01","description":"Vidange","statut":"en_cours"}`

func newTestHandler(t *testing.T) http.Handler {
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
	return New(service.New(repo))
}

func request(t *testing.T, h http.Handler, method, path, body string, wantStatus int) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != wantStatus {
		t.Fatalf("%s %s : code %d attendu %d, corps %s", method, path, w.Code, wantStatus, w.Body.String())
	}
	if wantStatus == http.StatusNoContent {
		if w.Body.Len() != 0 {
			t.Fatalf("corps inattendu pour 204 : %s", w.Body.String())
		}
		return w
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") || !json.Valid(w.Body.Bytes()) {
		t.Fatalf("réponse JSON attendue : %v, %s", w.Header(), w.Body.String())
	}
	if wantStatus >= 400 {
		var data struct{ Erreur string }
		decode(t, w, &data)
		if data.Erreur == "" {
			t.Fatal("message d'erreur absent")
		}
	}
	return w
}

func decode(t *testing.T, w *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), dest); err != nil {
		t.Fatal(err)
	}
}

func TestCycleGarage(t *testing.T) {
	h := newTestHandler(t)
	w := request(t, h, "GET", "/vehicules", "", 200)
	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("liste vide attendue : %s", w.Body.String())
	}
	w = request(t, h, "POST", "/vehicules", vehiculeJSON, 201)
	if w.Header().Get("Location") != "/vehicules/AA-123-BB" {
		t.Fatalf("Location absente : %v", w.Header())
	}
	var v model.Vehicule
	decode(t, w, &v)
	if v.Plaque != "AA-123-BB" || v.Proprietaire.Nom != "Camille Martin" {
		t.Fatalf("véhicule créé incorrect : %+v", v)
	}
	request(t, h, "POST", "/vehicules", vehiculeJSON, 409)
	w = request(t, h, "GET", "/vehicules/AA-123-BB", "", 200)
	var got model.Vehicule
	decode(t, w, &got)
	if got != v {
		t.Fatalf("véhicule relu différent : %+v", got)
	}
	w = request(t, h, "GET", "/vehicules", "", 200)
	var vehicules []model.Vehicule
	decode(t, w, &vehicules)
	if len(vehicules) != 1 || vehicules[0] != v {
		t.Fatalf("liste incorrecte : %+v", vehicules)
	}
	w = request(t, h, "GET", "/vehicules/AA-123-BB/interventions", "", 200)
	if strings.TrimSpace(w.Body.String()) != "[]" {
		t.Fatalf("liste vide attendue : %s", w.Body.String())
	}
	w = request(t, h, "POST", "/vehicules/AA-123-BB/interventions", interventionJSON, 201)
	var intervention model.Intervention
	decode(t, w, &intervention)
	if intervention.ID <= 0 || intervention.Plaque != v.Plaque || intervention.Statut != "en_cours" {
		t.Fatalf("intervention incorrecte : %+v", intervention)
	}
	w = request(t, h, "GET", "/vehicules/AA-123-BB/interventions", "", 200)
	var interventions []model.Intervention
	decode(t, w, &interventions)
	if len(interventions) != 1 || interventions[0] != intervention {
		t.Fatalf("liste incorrecte : %+v", interventions)
	}
	request(t, h, "DELETE", "/vehicules/AA-123-BB", "", 409)
	w = request(t, h, "GET", "/stats", "", 200)
	var stats model.Stats
	decode(t, w, &stats)
	if stats.NombreVehicules != 1 || stats.InterventionsParStatut["en_cours"] != 1 ||
		stats.InterventionsParStatut["terminee"] != 0 || stats.InterventionsParStatut["annulee"] != 0 ||
		len(stats.InterventionsParStatut) != 3 {
		t.Fatalf("statistiques incorrectes : %+v", stats)
	}
	second := strings.Replace(vehiculeJSON, "AA-123-BB", "CC-456-DD", 1)
	request(t, h, "POST", "/vehicules", second, 201)
	request(t, h, "DELETE", "/vehicules/CC-456-DD", "", 204)
	request(t, h, "GET", "/vehicules/CC-456-DD", "", 404)
}

func TestRequetesInvalides(t *testing.T) {
	h := newTestHandler(t)
	request(t, h, "POST", "/vehicules", vehiculeJSON, 201)
	tests := []struct {
		name, method, path, body string
		status                   int
	}{
		{"route inconnue", "GET", "/inconnue", "", 404},
		{"route en trop", "GET", "/vehicules/AA-123-BB/interventions/1", "", 404},
		{"plaque invalide", "GET", "/vehicules/invalide", "", 400},
		{"véhicule absent", "GET", "/vehicules/ZZ-999-ZZ", "", 404},
		{"suppression absente", "DELETE", "/vehicules/ZZ-999-ZZ", "", 404},
		{"interventions absentes", "GET", "/vehicules/ZZ-999-ZZ/interventions", "", 404},
		{"intervention orpheline", "POST", "/vehicules/ZZ-999-ZZ/interventions", interventionJSON, 404},
		{"JSON cassé", "POST", "/vehicules", "{", 400},
		{"JSON vide", "POST", "/vehicules", "", 400},
		{"JSON null", "POST", "/vehicules", "null", 400},
		{"JSON tableau", "POST", "/vehicules", "[]", 400},
		{"JSON multiple", "POST", "/vehicules", vehiculeJSON + " {}", 400},
		{"champ inconnu", "POST", "/vehicules", `{"autre":1}`, 400},
		{"mauvais type", "POST", "/vehicules", strings.Replace(vehiculeJSON, "42000", `"abc"`, 1), 400},
		{"kilométrage négatif", "POST", "/vehicules", strings.Replace(vehiculeJSON, "42000", "-1", 1), 400},
		{"statut invalide", "POST", "/vehicules/AA-123-BB/interventions", strings.Replace(interventionJSON, "en_cours", "finie", 1), 400},
		{"date impossible", "POST", "/vehicules/AA-123-BB/interventions", strings.Replace(interventionJSON, "2024-03-01", "2024-02-30", 1), 400},
		{"identifiant fourni", "POST", "/vehicules/AA-123-BB/interventions", `{"id":1,"date":"2024-03-01","description":"Vidange","statut":"en_cours"}`, 400},
		{"corps trop grand", "POST", "/vehicules", strings.Repeat("x", maxBodySize+1), 413},
		{"espaces trop grands", "POST", "/vehicules", vehiculeJSON + strings.Repeat(" ", maxBodySize), 413},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request(t, h, tt.method, tt.path, tt.body, tt.status)
		})
	}
}

func TestMethodesEtContentType(t *testing.T) {
	h := newTestHandler(t)
	for _, tt := range []struct{ path, allow string }{
		{"/vehicules", "GET, POST"},
		{"/vehicules/AA-123-BB", "GET, DELETE"},
		{"/vehicules/AA-123-BB/interventions", "GET, POST"},
		{"/stats", "GET"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			w := request(t, h, "PUT", tt.path, "", 405)
			if w.Header().Get("Allow") != tt.allow {
				t.Fatalf("Allow incorrect : %q", w.Header().Get("Allow"))
			}
		})
	}
	for _, contentType := range []string{"", "text/plain", "application/json; charset=utf-8"} {
		t.Run("Content-Type="+contentType, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/vehicules", strings.NewReader(vehiculeJSON))
			r.Header.Set("Content-Type", contentType)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			want := 415
			if strings.HasPrefix(contentType, "application/json") {
				want = 201
			}
			if w.Code != want || !json.Valid(w.Body.Bytes()) {
				t.Fatalf("code %d attendu %d, corps %s", w.Code, want, w.Body.String())
			}
		})
	}
}

func TestMappingErreurs(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
	}{
		{"validation", apperrors.ValidationError{Field: "date", Message: "invalide"}, 400},
		{"absent", fmt.Errorf("chercher : %w", apperrors.ErrNotFound), 404},
		{"doublon", fmt.Errorf("ajouter : %w", apperrors.ErrConflict), 409},
		{"suppression", fmt.Errorf("supprimer : %w", apperrors.ErrHasInterventions), 409},
		{"interne", io.ErrUnexpectedEOF, 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleError(w, tt.err)
			var body struct{ Erreur string }
			decode(t, w, &body)
			if w.Code != tt.status || body.Erreur == "" {
				t.Fatalf("réponse incorrecte : %d %s", w.Code, w.Body.String())
			}
			if tt.status == 500 && strings.Contains(body.Erreur, tt.err.Error()) {
				t.Fatal("l'erreur interne est exposée au client")
			}
		})
	}
}
