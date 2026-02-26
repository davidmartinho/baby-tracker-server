package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Baby struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Event struct {
	ID         int64      `json:"id"`
	BabyID     int64      `json:"baby_id"`
	Type       string     `json:"type"`
	OccurredAt *time.Time `json:"occurred_at,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	Side       *string    `json:"side,omitempty"`
}

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
	CreateEvent(ctx context.Context, event Event) (Event, error)
}

// NewRouter creates the HTTP router for the Baby Tracker API.
func NewRouter(store BabyStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /v1/babies", listBabies(store))
	mux.HandleFunc("GET /v1/profile", getProfile)
	mux.HandleFunc("POST /v1/events", createEvent(store))

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

type createEventRequest struct {
	BabyID     int64      `json:"baby_id"`
	Type       string     `json:"type"`
	OccurredAt *time.Time `json:"occurred_at"`
	StartedAt  *time.Time `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at"`
	Side       *string    `json:"side"`
}

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		if req.BabyID == 0 {
			http.Error(w, "baby_id is required", http.StatusBadRequest)
			return
		}

		if err := validateEventRequest(req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		created, err := store.CreateEvent(r.Context(), Event{
			BabyID:     req.BabyID,
			Type:       req.Type,
			OccurredAt: req.OccurredAt,
			StartedAt:  req.StartedAt,
			EndedAt:    req.EndedAt,
			Side:       req.Side,
		})
		if err != nil {
			log.Printf("create event failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": created})
	}
}

func validateEventRequest(req createEventRequest) error {
	switch req.Type {
	case "diaper_change":
		if req.OccurredAt == nil {
			return errInvalidEvent("occurred_at is required for diaper_change")
		}
		if req.StartedAt != nil || req.EndedAt != nil || req.Side != nil {
			return errInvalidEvent("diaper_change only supports occurred_at")
		}
	case "nursing":
		if req.StartedAt == nil || req.EndedAt == nil || req.Side == nil {
			return errInvalidEvent("nursing requires started_at, ended_at, and side")
		}
		if *req.Side != "left" && *req.Side != "right" {
			return errInvalidEvent("nursing side must be left or right")
		}
		if req.OccurredAt != nil {
			return errInvalidEvent("nursing does not support occurred_at")
		}
	case "sleep":
		if req.StartedAt == nil || req.EndedAt == nil {
			return errInvalidEvent("sleep requires started_at and ended_at")
		}
		if req.Side != nil || req.OccurredAt != nil {
			return errInvalidEvent("sleep only supports started_at and ended_at")
		}
	default:
		return errInvalidEvent("unsupported event type")
	}

	if req.StartedAt != nil && req.EndedAt != nil && req.EndedAt.Before(*req.StartedAt) {
		return errInvalidEvent("ended_at must be after started_at")
	}

	return nil
}

type errInvalidEvent string

func (e errInvalidEvent) Error() string {
	return string(e)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
