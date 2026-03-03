package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	Name      string
	BirthDate string
	Gender    string
}

type UpdateChildInput struct {
	Name      string
	BirthDate string
	Gender    string
}

var ErrNotFound = errors.New("resource not found")

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
	mux.HandleFunc("GET /v1/children", listChildren(store))
	mux.HandleFunc("POST /v1/children", createChild(store))
	mux.HandleFunc("PUT /v1/children/{id}", updateChild(store))
	mux.HandleFunc("DELETE /v1/children/{id}", deleteChild(store))
	mux.HandleFunc("GET /v1/profile", getProfile)

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
		var payload struct {
			Name      string `json:"name"`
			BirthDate string `json:"birthDate"`
			Gender    string `json:"gender"`
		}

		if err := decodeJSON(r.Body, &payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		input, err := normalizeChildInput(payload.Name, payload.BirthDate, payload.Gender)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		created, err := store.CreateChild(r.Context(), CreateChildInput(input))
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
		id, err := parseIDPathValue(r, "id")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		var payload struct {
			Name      string `json:"name"`
			BirthDate string `json:"birthDate"`
			Gender    string `json:"gender"`
		}

		if err := decodeJSON(r.Body, &payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		input, err := normalizeChildInput(payload.Name, payload.BirthDate, payload.Gender)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		updated, err := store.UpdateChild(r.Context(), id, UpdateChildInput(input))
		if err != nil {
			if errors.Is(err, ErrNotFound) {
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
		id, err := parseIDPathValue(r, "id")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := store.DeleteChild(r.Context(), id); err != nil {
			if errors.Is(err, ErrNotFound) {
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

func parseIDPathValue(r *http.Request, key string) (int64, error) {
	rawID := strings.TrimSpace(r.PathValue(key))
	if rawID == "" {
		return 0, fmt.Errorf("missing %s path parameter", key)
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s path parameter", key)
	}

	return id, nil
}

func decodeJSON(body io.ReadCloser, dst any) error {
	defer body.Close()

	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON payload")
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("invalid JSON payload")
	}

	return nil
}

func normalizeChildInput(name, birthDate, gender string) (struct {
	Name      string
	BirthDate string
	Gender    string
}, error) {
	normalizedName := strings.TrimSpace(name)
	if normalizedName == "" {
		return struct {
			Name      string
			BirthDate string
			Gender    string
		}{}, fmt.Errorf("name is required")
	}

	normalizedBirthDate := strings.TrimSpace(birthDate)
	if normalizedBirthDate == "" {
		return struct {
			Name      string
			BirthDate string
			Gender    string
		}{}, fmt.Errorf("birthDate is required")
	}

	parsedBirthDate, err := time.Parse("2006-01-02", normalizedBirthDate)
	if err != nil {
		return struct {
			Name      string
			BirthDate string
			Gender    string
		}{}, fmt.Errorf("birthDate must use YYYY-MM-DD format")
	}

	normalizedGender := strings.TrimSpace(gender)
	if normalizedGender == "" {
		return struct {
			Name      string
			BirthDate string
			Gender    string
		}{}, fmt.Errorf("gender is required")
	}

	return struct {
		Name      string
		BirthDate string
		Gender    string
	}{
		Name:      normalizedName,
		BirthDate: parsedBirthDate.Format("2006-01-02"),
		Gender:    normalizedGender,
	}, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
