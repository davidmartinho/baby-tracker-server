package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
)

type Baby struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
}

// NewRouter creates the HTTP router for the Baby Tracker API.
func NewRouter(store BabyStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /v1/babies", listBabies(store))
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
