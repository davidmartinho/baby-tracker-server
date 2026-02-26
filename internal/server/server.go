package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"time"
)

type Baby struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Event struct {
	ID         int64          `json:"id"`
	BabyID     int64          `json:"baby_id"`
	Type       string         `json:"type"`
	OccurredAt time.Time      `json:"occurred_at"`
	Details    map[string]any `json:"details"`
	CreatedAt  time.Time      `json:"created_at"`
}

type EventInput struct {
	Type       string
	OccurredAt time.Time
	Details    map[string]any
}

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
	CreateEvent(ctx context.Context, babyID int64, input EventInput) (Event, error)
}

// NewRouter creates the HTTP router for the Baby Tracker API.
func NewRouter(store BabyStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /v1/babies", listBabies(store))
	mux.HandleFunc("GET /v1/profile", getProfile)
	mux.HandleFunc("POST /v1/babies/{babyID}/events", createEvent(store))

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

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := parseBabyID(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		input, err := parseEventRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		event, err := store.CreateEvent(r.Context(), babyID, input)
		if err != nil {
			log.Printf("create event failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": event})
	}
}

func parseBabyID(r *http.Request) (int64, error) {
	babyIDRaw := r.PathValue("babyID")
	if babyIDRaw == "" {
		return 0, errors.New("baby_id is required")
	}

	var babyID int64
	if _, err := fmt.Sscanf(babyIDRaw, "%d", &babyID); err != nil || babyID <= 0 {
		return 0, errors.New("baby_id must be a positive integer")
	}

	return babyID, nil
}

type eventRequest struct {
	Type       string         `json:"type"`
	OccurredAt string         `json:"occurred_at"`
	Details    map[string]any `json:"details"`
}

func parseEventRequest(r *http.Request) (EventInput, error) {
	var req eventRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		return EventInput{}, errors.New("invalid JSON payload")
	}

	if req.Type == "" {
		return EventInput{}, errors.New("type is required")
	}
	if req.OccurredAt == "" {
		return EventInput{}, errors.New("occurred_at is required")
	}

	occurredAt, err := time.Parse(time.RFC3339, req.OccurredAt)
	if err != nil {
		return EventInput{}, errors.New("occurred_at must be RFC3339")
	}

	details := req.Details
	if details == nil {
		details = map[string]any{}
	}

	validatedDetails, err := validateEventDetails(req.Type, details)
	if err != nil {
		return EventInput{}, err
	}

	return EventInput{
		Type:       req.Type,
		OccurredAt: occurredAt,
		Details:    validatedDetails,
	}, nil
}

func validateEventDetails(eventType string, details map[string]any) (map[string]any, error) {
	switch eventType {
	case "diaper_change":
		return validateDiaperChange(details)
	case "nursing":
		return validateNursing(details)
	case "sleep":
		return validateSleep(details)
	default:
		return nil, errors.New("type must be one of diaper_change, nursing, sleep")
	}
}

func validateDiaperChange(details map[string]any) (map[string]any, error) {
	kind := getString(details, "kind")
	if kind == "" {
		return nil, errors.New("details.kind is required for diaper_change")
	}
	switch kind {
	case "wet", "dirty", "both":
		return map[string]any{"kind": kind}, nil
	default:
		return nil, errors.New("details.kind must be wet, dirty, or both")
	}
}

func validateNursing(details map[string]any) (map[string]any, error) {
	side := getString(details, "side")
	if side == "" {
		return nil, errors.New("details.side is required for nursing")
	}
	if side != "left" && side != "right" {
		return nil, errors.New("details.side must be left or right")
	}

	duration, ok := getPositiveInt(details, "duration_minutes")
	if !ok {
		return nil, errors.New("details.duration_minutes is required for nursing")
	}

	return map[string]any{
		"side":             side,
		"duration_minutes": duration,
	}, nil
}

func validateSleep(details map[string]any) (map[string]any, error) {
	duration, ok := getPositiveInt(details, "duration_minutes")
	if !ok {
		return nil, errors.New("details.duration_minutes is required for sleep")
	}

	return map[string]any{
		"duration_minutes": duration,
	}, nil
}

func getString(details map[string]any, key string) string {
	value, ok := details[key]
	if !ok {
		return ""
	}

	str, ok := value.(string)
	if !ok {
		return ""
	}

	return str
}

func getPositiveInt(details map[string]any, key string) (int, bool) {
	value, ok := details[key]
	if !ok {
		return 0, false
	}

	switch v := value.(type) {
	case float64:
		if v <= 0 || math.Trunc(v) != v {
			return 0, false
		}
		return int(v), true
	case int:
		if v <= 0 {
			return 0, false
		}
		return v, true
	default:
		return 0, false
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
