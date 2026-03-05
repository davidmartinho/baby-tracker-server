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
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	BirthDate string `json:"birthDate"`
	Gender    string `json:"gender"`
}

type CreateBabyInput struct {
	Name      string
	BirthDate string
	Gender    string
}

type UpdateBabyInput struct {
	Name      string
	BirthDate string
	Gender    string
}

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
	CreateBaby(ctx context.Context, input CreateBabyInput) (Baby, error)
	UpdateBaby(ctx context.Context, id int64, input UpdateBabyInput) (Baby, error)
	DeleteBaby(ctx context.Context, id int64) error
}

var ErrNotFound = errors.New("not found")

// NewRouter creates the HTTP router for the Baby Tracker API.
func NewRouter(store BabyStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /v1/babies", listBabies(store))
	mux.HandleFunc("POST /v1/babies", createBaby(store))
	mux.HandleFunc("PUT /v1/babies/{id}", updateBaby(store))
	mux.HandleFunc("DELETE /v1/babies/{id}", deleteBaby(store))
	mux.HandleFunc("GET /v1/profile", getProfile)

	return mux
}

type babyPayload struct {
	Name      string `json:"name"`
	BirthDate string `json:"birthDate"`
	Gender    string `json:"gender"`
}

func createBaby(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		payload, err := decodeBabyPayload(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		data, err := store.CreateBaby(r.Context(), CreateBabyInput{
			Name:      payload.Name,
			BirthDate: payload.BirthDate,
			Gender:    payload.Gender,
		})
		if err != nil {
			log.Printf("create baby failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": data})
	}
}

func updateBaby(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		payload, err := decodeBabyPayload(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		data, err := store.UpdateBaby(r.Context(), id, UpdateBabyInput{
			Name:      payload.Name,
			BirthDate: payload.BirthDate,
			Gender:    payload.Gender,
		})
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			log.Printf("update baby failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": data})
	}
}

func deleteBaby(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r.PathValue("id"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if err := store.DeleteBaby(r.Context(), id); err != nil {
			if errors.Is(err, ErrNotFound) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			log.Printf("delete baby failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
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

func getProfile(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"id":    "usr_mock_1",
		"name":  "Baby Tracker User",
		"email": "user@example.com",
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}

func decodeBabyPayload(r *http.Request) (babyPayload, error) {
	defer r.Body.Close()

	var payload babyPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return babyPayload{}, errors.New("invalid JSON body")
	}

	payload.Name = strings.TrimSpace(payload.Name)
	payload.BirthDate = strings.TrimSpace(payload.BirthDate)
	payload.Gender = strings.TrimSpace(payload.Gender)

	if payload.Name == "" {
		return babyPayload{}, errors.New("name is required")
	}
	if payload.BirthDate == "" {
		return babyPayload{}, errors.New("birthDate is required")
	}
	if _, err := time.Parse("2006-01-02", payload.BirthDate); err != nil {
		return babyPayload{}, errors.New("birthDate must be in YYYY-MM-DD format")
	}
	if payload.Gender == "" {
		return babyPayload{}, errors.New("gender is required")
	}

	return payload, nil
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
