package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Baby struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Child struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	BirthDate string `json:"birthDate"`
	Gender    string `json:"gender"`
}

type CreateChildInput struct {
	Name      string `json:"name"`
	BirthDate string `json:"birthDate"`
	Gender    string `json:"gender"`
}

type UpdateChildInput struct {
	Name      string `json:"name"`
	BirthDate string `json:"birthDate"`
	Gender    string `json:"gender"`
}

var ErrChildNotFound = errors.New("child not found")

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
	ListChildren(ctx context.Context) ([]Child, error)
	CreateChild(ctx context.Context, input CreateChildInput) (Child, error)
	UpdateChild(ctx context.Context, id int64, input UpdateChildInput) (Child, error)
	DeleteChild(ctx context.Context, id int64) error
}

// NewRouter creates the HTTP router for the Baby Tracker API.
func NewRouter(store BabyStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /v1/babies", listBabies(store))
	mux.HandleFunc("GET /v1/profile", getProfile)
	mux.HandleFunc("GET /v1/children", listChildren(store))
	mux.HandleFunc("POST /v1/children", createChild(store))
	mux.HandleFunc("PUT /v1/children/{id}", updateChild(store))
	mux.HandleFunc("DELETE /v1/children/{id}", deleteChild(store))

	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func listBabies(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := store.ListBabies(r.Context())
		if err != nil {
			log.Printf("list babies failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": data})
	}
}

func listChildren(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := store.ListChildren(r.Context())
		if err != nil {
			log.Printf("list children failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": data})
	}
}

func createChild(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input CreateChildInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		if err := validateChildInput(input.Name, input.BirthDate, input.Gender); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		created, err := store.CreateChild(r.Context(), input)
		if err != nil {
			log.Printf("create child failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": created})
	}
}

func updateChild(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var input UpdateChildInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		if err := validateChildInput(input.Name, input.BirthDate, input.Gender); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		updated, err := store.UpdateChild(r.Context(), id, input)
		if err != nil {
			if errors.Is(err, ErrChildNotFound) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			log.Printf("update child failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": updated})
	}
}

func deleteChild(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = store.DeleteChild(r.Context(), id)
		if err != nil {
			if errors.Is(err, ErrChildNotFound) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			log.Printf("delete child failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func getProfile(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"id":    "usr_mock_1",
		"name":  "Baby Tracker User",
		"email": "user@example.com",
	})
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid child id")
	}

	return id, nil
}

func validateChildInput(name, birthDate, gender string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}

	if _, err := time.Parse("2006-01-02", birthDate); err != nil {
		return errors.New("birthDate must be in YYYY-MM-DD format")
	}

	switch strings.ToLower(strings.TrimSpace(gender)) {
	case "male", "female", "other":
		return nil
	default:
		return errors.New("gender must be one of: male, female, other")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
