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
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Event struct {
	ID                     int64      `json:"id"`
	BabyID                 int64      `json:"baby_id"`
	Type                   string     `json:"type"`
	OccurredAt             time.Time  `json:"occurred_at"`
	NursingSide            *string    `json:"nursing_side,omitempty"`
	NursingDurationMinutes *int       `json:"nursing_duration_minutes,omitempty"`
	SleepStart             *time.Time `json:"sleep_start,omitempty"`
	SleepEnd               *time.Time `json:"sleep_end,omitempty"`
	DiaperKind             *string    `json:"diaper_kind,omitempty"`
}

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
	CreateEvent(ctx context.Context, babyID int64, event Event) (Event, error)
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

type createEventRequest struct {
	Type                   string     `json:"type"`
	OccurredAt             *time.Time `json:"occurred_at"`
	NursingSide            string     `json:"nursing_side"`
	NursingDurationMinutes *int       `json:"nursing_duration_minutes"`
	SleepStart             *time.Time `json:"sleep_start"`
	SleepEnd               *time.Time `json:"sleep_end"`
	DiaperKind             string     `json:"diaper_kind"`
}

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := strconv.ParseInt(r.PathValue("babyID"), 10, 64)
		if err != nil || babyID <= 0 {
			http.Error(w, "invalid baby id", http.StatusBadRequest)
			return
		}

		var req createEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		event, err := buildEventFromRequest(babyID, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		created, err := store.CreateEvent(r.Context(), babyID, event)
		if err != nil {
			log.Printf("create event failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": created})
	}
}

func buildEventFromRequest(babyID int64, req createEventRequest) (Event, error) {
	eventType := strings.TrimSpace(strings.ToLower(req.Type))
	if eventType == "" {
		return Event{}, errors.New("type is required")
	}

	switch eventType {
	case "diaper":
		if req.OccurredAt == nil {
			return Event{}, errors.New("occurred_at is required for diaper events")
		}
		if req.DiaperKind != "" {
			kind := strings.ToLower(strings.TrimSpace(req.DiaperKind))
			if kind != "wet" && kind != "dirty" && kind != "mixed" {
				return Event{}, errors.New("diaper_kind must be wet, dirty, or mixed")
			}
			return Event{
				BabyID:     babyID,
				Type:       eventType,
				OccurredAt: req.OccurredAt.UTC(),
				DiaperKind: &kind,
			}, nil
		}
		return Event{
			BabyID:     babyID,
			Type:       eventType,
			OccurredAt: req.OccurredAt.UTC(),
		}, nil
	case "nursing":
		if req.OccurredAt == nil {
			return Event{}, errors.New("occurred_at is required for nursing events")
		}
		side := strings.ToLower(strings.TrimSpace(req.NursingSide))
		if side != "left" && side != "right" {
			return Event{}, errors.New("nursing_side must be left or right")
		}
		return Event{
			BabyID:                 babyID,
			Type:                   eventType,
			OccurredAt:             req.OccurredAt.UTC(),
			NursingSide:            &side,
			NursingDurationMinutes: req.NursingDurationMinutes,
		}, nil
	case "sleep":
		if req.SleepStart == nil || req.SleepEnd == nil {
			return Event{}, errors.New("sleep_start and sleep_end are required for sleep events")
		}
		if req.SleepEnd.Before(*req.SleepStart) {
			return Event{}, errors.New("sleep_end must be after sleep_start")
		}
		start := req.SleepStart.UTC()
		end := req.SleepEnd.UTC()
		return Event{
			BabyID:     babyID,
			Type:       eventType,
			OccurredAt: start,
			SleepStart: &start,
			SleepEnd:   &end,
		}, nil
	default:
		return Event{}, errors.New("unsupported event type")
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
