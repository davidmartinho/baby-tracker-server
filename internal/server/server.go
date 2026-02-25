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
	ID        int64      `json:"id"`
	BabyID    int64      `json:"baby_id"`
	Type      string     `json:"type"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Side      *string    `json:"side,omitempty"`
}

type CreateEventInput struct {
	Type      string     `json:"type"`
	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Side      *string    `json:"side,omitempty"`
}

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
	CreateEvent(ctx context.Context, babyID int64, input CreateEventInput) (Event, error)
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

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := parsePathID(r.PathValue("babyID"))
		if err != nil {
			http.Error(w, "invalid baby id", http.StatusBadRequest)
			return
		}

		var input CreateEventInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if err := validateCreateEventInput(input); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		created, err := store.CreateEvent(r.Context(), babyID, input)
		if err != nil {
			log.Printf("create event failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": created})
	}
}

func parsePathID(raw string) (int64, error) {
	if raw == "" {
		return 0, errors.New("missing id")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func validateCreateEventInput(input CreateEventInput) error {
	switch input.Type {
	case "diaper_change":
		if input.StartTime == nil {
			return errors.New("start_time is required")
		}
		if input.EndTime != nil {
			return errors.New("end_time must be empty for diaper_change")
		}
		if input.Side != nil {
			return errors.New("side must be empty for diaper_change")
		}
		return nil
	case "nursing":
		if input.StartTime == nil || input.EndTime == nil {
			return errors.New("start_time and end_time are required")
		}
		if input.Side == nil {
			return errors.New("side is required for nursing")
		}
		if *input.Side != "left" && *input.Side != "right" {
			return errors.New("side must be left or right")
		}
		if input.EndTime.Before(*input.StartTime) {
			return errors.New("end_time must be after start_time")
		}
		return nil
	case "sleep":
		if input.StartTime == nil || input.EndTime == nil {
			return errors.New("start_time and end_time are required")
		}
		if input.Side != nil {
			return errors.New("side must be empty for sleep")
		}
		if input.EndTime.Before(*input.StartTime) {
			return errors.New("end_time must be after start_time")
		}
		return nil
	default:
		return errors.New("type must be diaper_change, nursing, or sleep")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
