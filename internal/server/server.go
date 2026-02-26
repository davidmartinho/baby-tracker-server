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
	Type       string     `json:"type"`
	OccurredAt *time.Time `json:"occurred_at,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	Side       *string    `json:"side,omitempty"`
	CreatedAt  *time.Time `json:"created_at,omitempty"`
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

type eventRequest struct {
	Type       string     `json:"type"`
	OccurredAt *time.Time `json:"occurred_at"`
	StartedAt  *time.Time `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at"`
	Side       *string    `json:"side"`
}

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req eventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON payload", http.StatusBadRequest)
			return
		}

		event, err := validateEventRequest(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		created, err := store.CreateEvent(r.Context(), event)
		if err != nil {
			log.Printf("create event failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": created})
	}
}

func validateEventRequest(req eventRequest) (Event, error) {
	event := Event{
		Type:       req.Type,
		OccurredAt: req.OccurredAt,
		StartedAt:  req.StartedAt,
		EndedAt:    req.EndedAt,
		Side:       req.Side,
	}

	switch req.Type {
	case "diaper":
		if req.OccurredAt == nil {
			return Event{}, errBadRequest("diaper events require occurred_at")
		}
		if req.StartedAt != nil || req.EndedAt != nil || req.Side != nil {
			return Event{}, errBadRequest("diaper events only allow occurred_at")
		}
	case "nursing":
		if req.StartedAt == nil || req.EndedAt == nil {
			return Event{}, errBadRequest("nursing events require started_at and ended_at")
		}
		if req.Side == nil {
			return Event{}, errBadRequest("nursing events require side")
		}
		if req.OccurredAt != nil {
			return Event{}, errBadRequest("nursing events do not allow occurred_at")
		}
		if req.EndedAt.Before(*req.StartedAt) {
			return Event{}, errBadRequest("nursing events require ended_at after started_at")
		}
		if *req.Side != "left" && *req.Side != "right" {
			return Event{}, errBadRequest("nursing events require side to be left or right")
		}
	case "sleep":
		if req.StartedAt == nil || req.EndedAt == nil {
			return Event{}, errBadRequest("sleep events require started_at and ended_at")
		}
		if req.OccurredAt != nil || req.Side != nil {
			return Event{}, errBadRequest("sleep events only allow started_at and ended_at")
		}
		if req.EndedAt.Before(*req.StartedAt) {
			return Event{}, errBadRequest("sleep events require ended_at after started_at")
		}
	default:
		return Event{}, errBadRequest("unsupported event type")
	}

	return event, nil
}

type errBadRequest string

func (e errBadRequest) Error() string {
	return string(e)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
