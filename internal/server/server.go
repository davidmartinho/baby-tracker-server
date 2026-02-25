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

type EventType string

const (
	EventTypeDiaper  EventType = "diaper"
	EventTypeNursing EventType = "nursing"
	EventTypeSleep   EventType = "sleep"
)

type Event struct {
	ID         int64           `json:"id"`
	BabyID     int64           `json:"baby_id"`
	Type       EventType       `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Details    json.RawMessage `json:"details"`
}

type CreateEventInput struct {
	BabyID     int64
	Type       EventType
	OccurredAt time.Time
	Details    json.RawMessage
}

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
	CreateEvent(ctx context.Context, input CreateEventInput) (Event, error)
}

var ErrNotFound = errors.New("not found")

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
	Type       EventType       `json:"type"`
	OccurredAt *time.Time      `json:"occurred_at"`
	Diaper     *DiaperDetails  `json:"diaper,omitempty"`
	Nursing    *NursingDetails `json:"nursing,omitempty"`
	Sleep      *SleepDetails   `json:"sleep,omitempty"`
}

type DiaperDetails struct {
	Kind string `json:"kind"`
}

type NursingDetails struct {
	Side            string `json:"side"`
	DurationMinutes int    `json:"duration_minutes"`
}

type SleepDetails struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := strconv.ParseInt(r.PathValue("babyID"), 10, 64)
		if err != nil || babyID <= 0 {
			writeError(w, http.StatusBadRequest, "invalid baby id")
			return
		}

		var req createEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json body")
			return
		}
		if req.OccurredAt == nil || req.OccurredAt.IsZero() {
			writeError(w, http.StatusBadRequest, "occurred_at is required")
			return
		}

		var details any
		switch req.Type {
		case EventTypeDiaper:
			if req.Diaper == nil {
				writeError(w, http.StatusBadRequest, "diaper details are required")
				return
			}
			switch req.Diaper.Kind {
			case "wet", "dirty", "mixed":
			default:
				writeError(w, http.StatusBadRequest, "diaper.kind must be wet, dirty, or mixed")
				return
			}
			details = req.Diaper
		case EventTypeNursing:
			if req.Nursing == nil {
				writeError(w, http.StatusBadRequest, "nursing details are required")
				return
			}
			if req.Nursing.DurationMinutes <= 0 {
				writeError(w, http.StatusBadRequest, "nursing.duration_minutes must be positive")
				return
			}
			switch req.Nursing.Side {
			case "left", "right":
			default:
				writeError(w, http.StatusBadRequest, "nursing.side must be left or right")
				return
			}
			details = req.Nursing
		case EventTypeSleep:
			if req.Sleep == nil {
				writeError(w, http.StatusBadRequest, "sleep details are required")
				return
			}
			if req.Sleep.StartTime.IsZero() || req.Sleep.EndTime.IsZero() {
				writeError(w, http.StatusBadRequest, "sleep.start_time and sleep.end_time are required")
				return
			}
			if !req.Sleep.EndTime.After(req.Sleep.StartTime) {
				writeError(w, http.StatusBadRequest, "sleep.end_time must be after sleep.start_time")
				return
			}
			details = req.Sleep
		default:
			writeError(w, http.StatusBadRequest, "invalid event type")
			return
		}

		detailsJSON, err := json.Marshal(details)
		if err != nil {
			log.Printf("marshal event details failed: %v", err)
			writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}

		event, err := store.CreateEvent(r.Context(), CreateEventInput{
			BabyID:     babyID,
			Type:       req.Type,
			OccurredAt: *req.OccurredAt,
			Details:    detailsJSON,
		})
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				writeError(w, http.StatusNotFound, "baby not found")
				return
			}
			log.Printf("create event failed: %v", err)
			writeError(w, http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": event})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
