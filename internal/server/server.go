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
	EventTypeDiaper  EventType = "diaper"
	EventTypeNursing EventType = "nursing"
	EventTypeSleep   EventType = "sleep"
)

type DiaperEvent struct {
	Kind string `json:"kind,omitempty"`
	Note string `json:"note,omitempty"`
}

type NursingEvent struct {
	Side            string `json:"side"`
	DurationMinutes int    `json:"duration_minutes,omitempty"`
}

type SleepEvent struct {
	StartAt time.Time `json:"start_at"`
	EndAt   time.Time `json:"end_at"`
}

type NewEvent struct {
	BabyID     int64
	Type       EventType
	OccurredAt time.Time
	Details    json.RawMessage
}

type Event struct {
	ID         int64           `json:"id"`
	BabyID     int64           `json:"baby_id"`
	Type       EventType       `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Details    json.RawMessage `json:"details"`
}

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
	CreateEvent(ctx context.Context, event NewEvent) (Event, error)
}

// NewRouter creates the HTTP router for the Baby Tracker API.
func NewRouter(store BabyStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /v1/babies", listBabies(store))
	mux.HandleFunc("POST /v1/babies/{id}/events", createEvent(store))
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
	Type       EventType     `json:"type"`
	OccurredAt *time.Time    `json:"occurred_at,omitempty"`
	Diaper     *DiaperEvent  `json:"diaper,omitempty"`
	Nursing    *NursingEvent `json:"nursing,omitempty"`
	Sleep      *SleepEvent   `json:"sleep,omitempty"`
}

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil || babyID <= 0 {
			http.Error(w, "invalid baby id", http.StatusBadRequest)
			return
		}

		var req createEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		occurredAt := time.Now().UTC()
		if req.OccurredAt != nil {
			occurredAt = req.OccurredAt.UTC()
		}

		details, err := buildEventDetails(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		event, err := store.CreateEvent(r.Context(), NewEvent{
			BabyID:     babyID,
			Type:       req.Type,
			OccurredAt: occurredAt,
			Details:    details,
		})
		if err != nil {
			log.Printf("create event failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": event})
	}
}

func buildEventDetails(req createEventRequest) (json.RawMessage, error) {
	switch req.Type {
	case EventTypeDiaper:
		if req.Diaper == nil {
			return nil, errMissingEventDetails("diaper")
		}
		return marshalDetails(req.Diaper)
	case EventTypeNursing:
		if req.Nursing == nil {
			return nil, errMissingEventDetails("nursing")
		}
		side := req.Nursing.Side
		if side != "left" && side != "right" {
			return nil, errInvalidField("nursing.side", "must be left or right")
		}
		return marshalDetails(req.Nursing)
	case EventTypeSleep:
		if req.Sleep == nil {
			return nil, errMissingEventDetails("sleep")
		}
		if req.Sleep.StartAt.IsZero() || req.Sleep.EndAt.IsZero() {
			return nil, errInvalidField("sleep", "start_at and end_at are required")
		}
		if !req.Sleep.EndAt.After(req.Sleep.StartAt) {
			return nil, errInvalidField("sleep", "end_at must be after start_at")
		}
		return marshalDetails(req.Sleep)
	default:
		return nil, errInvalidField("type", "must be diaper, nursing, or sleep")
	}
}

func marshalDetails(payload any) (json.RawMessage, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

func errMissingEventDetails(name string) error {
	return errInvalidField(name, "details are required")
}

func errInvalidField(field, message string) error {
	return &invalidFieldError{field: field, message: message}
}

type invalidFieldError struct {
	field   string
	message string
}

func (e *invalidFieldError) Error() string {
	return "invalid " + e.field + ": " + e.message
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
