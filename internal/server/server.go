package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
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
	Side       string     `json:"side,omitempty"`
	OccurredAt *time.Time `json:"occurred_at,omitempty"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
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
	mux.HandleFunc("POST /v1/babies/{babyID}/events", createEvent(store))
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

type createEventRequest struct {
	Type       string `json:"type"`
	Side       string `json:"side,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
	StartedAt  string `json:"started_at,omitempty"`
	EndedAt    string `json:"ended_at,omitempty"`
}

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := parsePathInt64(r, "babyID")
		if err != nil {
			http.Error(w, "invalid baby id", http.StatusBadRequest)
			return
		}

		var req createEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}

		event, err := buildEventRequest(babyID, req)
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

func buildEventRequest(babyID int64, req createEventRequest) (Event, error) {
	if req.Type == "" {
		return Event{}, errors.New("type is required")
	}

	event := Event{BabyID: babyID, Type: req.Type}
	switch req.Type {
	case "diaper":
		occurredAt, err := parseTimestamp(req.OccurredAt)
		if err != nil {
			return Event{}, errors.New("occurred_at is required and must be RFC3339")
		}
		event.OccurredAt = occurredAt
	case "nursing":
		startedAt, endedAt, err := parseRange(req.StartedAt, req.EndedAt)
		if err != nil {
			return Event{}, err
		}
		if req.Side != "left" && req.Side != "right" {
			return Event{}, errors.New("side must be 'left' or 'right'")
		}
		event.StartedAt = startedAt
		event.EndedAt = endedAt
		event.Side = req.Side
	case "sleep":
		startedAt, endedAt, err := parseRange(req.StartedAt, req.EndedAt)
		if err != nil {
			return Event{}, err
		}
		event.StartedAt = startedAt
		event.EndedAt = endedAt
	default:
		return Event{}, errors.New("type must be one of: diaper, nursing, sleep")
	}

	return event, nil
}

func parseRange(start, end string) (*time.Time, *time.Time, error) {
	startedAt, err := parseTimestamp(start)
	if err != nil {
		return nil, nil, errors.New("started_at is required and must be RFC3339")
	}
	endedAt, err := parseTimestamp(end)
	if err != nil {
		return nil, nil, errors.New("ended_at is required and must be RFC3339")
	}
	if startedAt.After(*endedAt) {
		return nil, nil, errors.New("started_at must be before ended_at")
	}
	return startedAt, endedAt, nil
}

func parseTimestamp(value string) (*time.Time, error) {
	if value == "" {
		return nil, errors.New("missing timestamp")
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parsePathInt64(r *http.Request, name string) (int64, error) {
	value := r.PathValue(name)
	if value == "" {
		return 0, errors.New("missing path value")
	}
	return strconv.ParseInt(value, 10, 64)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
