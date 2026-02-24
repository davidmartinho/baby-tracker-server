package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Baby struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type EventType string

const (
	EventTypeDiaperChange EventType = "diaper_change"
	EventTypeNursingLeft  EventType = "nursing_left"
	EventTypeNursingRight EventType = "nursing_right"
	EventTypeSleep        EventType = "sleep"
)

type Event struct {
	ID        int64      `json:"id"`
	BabyID    int64      `json:"baby_id"`
	Type      EventType  `json:"type"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
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
	mux.HandleFunc("POST /v1/babies/{id}/events", createEvent(store))

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
	Type      EventType  `json:"type"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`
}

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || babyID <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid baby id"})
			return
		}

		var req createEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}

		if req.StartedAt.IsZero() {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "started_at is required"})
			return
		}

		if err := validateEventRequest(req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		event := Event{
			BabyID:    babyID,
			Type:      req.Type,
			StartedAt: req.StartedAt,
			EndedAt:   req.EndedAt,
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

func validateEventRequest(req createEventRequest) error {
	switch req.Type {
	case EventTypeDiaperChange:
		if req.EndedAt != nil {
			return errValidation("ended_at must be omitted for diaper_change")
		}
	case EventTypeNursingLeft, EventTypeNursingRight, EventTypeSleep:
		if req.EndedAt == nil {
			return errValidation("ended_at is required for this event type")
		}
	default:
		return errValidation("unsupported event type")
	}

	if req.EndedAt != nil && req.EndedAt.Before(req.StartedAt) {
		return errValidation("ended_at must be after started_at")
	}

	return nil
}

type errValidation string

func (e errValidation) Error() string {
	return string(e)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
