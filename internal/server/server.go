package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	ID         int64           `json:"id"`
	BabyID     int64           `json:"baby_id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Details    json.RawMessage `json:"details"`
}

type CreateEventInput struct {
	BabyID     int64
	Type       string
	OccurredAt time.Time
	Details    json.RawMessage
}

type WeightEntry struct {
	OccurredAt time.Time `json:"occurred_at"`
	WeightKg   float64   `json:"weight_kg"`
}

type BabyStore interface {
	ListBabies(ctx context.Context) ([]Baby, error)
	CreateEvent(ctx context.Context, input CreateEventInput) (Event, error)
	ListWeightEntries(ctx context.Context, babyID int64) ([]WeightEntry, error)
}

// NewRouter creates the HTTP router for the Baby Tracker API.
func NewRouter(store BabyStore) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /v1/babies", listBabies(store))
	mux.HandleFunc("GET /v1/babies/{id}/weights", listWeightEntries(store))
	mux.HandleFunc("GET /v1/babies/{id}/report.pdf", getBabyReportPDF(store))
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

func listWeightEntries(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := parseID(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid baby id", http.StatusBadRequest)
			return
		}

		data, err := store.ListWeightEntries(r.Context(), babyID)
		if err != nil {
			log.Printf("list weight entries failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": data})
	}
}

func getBabyReportPDF(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := parseID(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid baby id", http.StatusBadRequest)
			return
		}

		babies, err := store.ListBabies(r.Context())
		if err != nil {
			log.Printf("list babies for report failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		var baby *Baby
		for i := range babies {
			if babies[i].ID == babyID {
				baby = &babies[i]
				break
			}
		}
		if baby == nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		weights, err := store.ListWeightEntries(r.Context(), babyID)
		if err != nil {
			log.Printf("list weight entries for report failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		pdf, err := buildBabyReportPDF(*baby, weights)
		if err != nil {
			log.Printf("build baby report pdf failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("baby-report-%d.pdf", babyID)
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pdf)
	}
}

func buildBabyReportPDF(baby Baby, entries []WeightEntry) ([]byte, error) {
	lines := make([]string, 0, len(entries)+5)
	lines = append(lines, "Baby Tracker Report")
	lines = append(lines, fmt.Sprintf("Baby: %s (ID %d)", baby.Name, baby.ID))
	lines = append(lines, fmt.Sprintf("Generated at: %s", time.Now().UTC().Format(time.RFC3339)))
	lines = append(lines, "Weight entries:")
	if len(entries) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, entry := range entries {
			lines = append(lines, fmt.Sprintf("- %s: %.2f kg", entry.OccurredAt.UTC().Format(time.RFC3339), entry.WeightKg))
		}
	}

	return renderSimplePDF(lines)
}

func renderSimplePDF(lines []string) ([]byte, error) {
	var content strings.Builder
	content.WriteString("BT\n/F1 12 Tf\n72 760 Td\n")
	for i, line := range lines {
		escaped := escapePDFText(line)
		if i == 0 {
			content.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
		} else {
			content.WriteString(fmt.Sprintf("0 -18 Td (%s) Tj\n", escaped))
		}
	}
	content.WriteString("ET\n")

	contentBody := content.String()
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Count 1 /Kids [3 0 R] >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(contentBody), contentBody),
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, 0, len(objects)+1)
	offsets = append(offsets, 0)
	for i, obj := range objects {
		offsets = append(offsets, buf.Len())
		buf.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", i+1, obj))
	}

	xrefOffset := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objects)+1))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefOffset))

	return buf.Bytes(), nil
}

func escapePDFText(input string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)")
	return replacer.Replace(input)
}

type createEventRequest struct {
	Type            string `json:"type"`
	OccurredAt      string `json:"occurred_at"`
	StartAt         string `json:"start_at"`
	EndAt           string `json:"end_at"`
	Side            string `json:"side"`
	DurationMinutes int    `json:"duration_minutes"`
	Notes           string `json:"notes"`
}

func createEvent(store BabyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		babyID, err := parseID(r.PathValue("id"))
		if err != nil {
			http.Error(w, "invalid baby id", http.StatusBadRequest)
			return
		}

		var req createEventRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		input, err := buildCreateEventInput(babyID, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		event, err := store.CreateEvent(r.Context(), input)
		if err != nil {
			log.Printf("create event failed: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{"data": event})
	}
}

func buildCreateEventInput(babyID int64, req createEventRequest) (CreateEventInput, error) {
	switch strings.ToLower(strings.TrimSpace(req.Type)) {
	case "diaper":
		occurredAt, err := parseTimestamp(req.OccurredAt)
		if err != nil {
			return CreateEventInput{}, errors.New("occurred_at is required for diaper events")
		}

		details := map[string]any{}
		if strings.TrimSpace(req.Notes) != "" {
			details["notes"] = req.Notes
		}

		payload, err := json.Marshal(details)
		if err != nil {
			return CreateEventInput{}, errors.New("failed to encode details")
		}

		return CreateEventInput{
			BabyID:     babyID,
			Type:       "diaper",
			OccurredAt: occurredAt,
			Details:    payload,
		}, nil
	case "nursing":
		occurredAt, err := parseTimestamp(req.OccurredAt)
		if err != nil {
			return CreateEventInput{}, errors.New("occurred_at is required for nursing events")
		}

		side := strings.ToLower(strings.TrimSpace(req.Side))
		if side != "left" && side != "right" {
			return CreateEventInput{}, errors.New("side must be left or right for nursing events")
		}
		if req.DurationMinutes <= 0 {
			return CreateEventInput{}, errors.New("duration_minutes must be greater than 0 for nursing events")
		}

		payload, err := json.Marshal(map[string]any{
			"side":             side,
			"duration_minutes": req.DurationMinutes,
		})
		if err != nil {
			return CreateEventInput{}, errors.New("failed to encode details")
		}

		return CreateEventInput{
			BabyID:     babyID,
			Type:       "nursing",
			OccurredAt: occurredAt,
			Details:    payload,
		}, nil
	case "sleep":
		startAt, err := parseTimestamp(req.StartAt)
		if err != nil {
			return CreateEventInput{}, errors.New("start_at is required for sleep events")
		}
		endAt, err := parseTimestamp(req.EndAt)
		if err != nil {
			return CreateEventInput{}, errors.New("end_at is required for sleep events")
		}
		if !endAt.After(startAt) {
			return CreateEventInput{}, errors.New("end_at must be after start_at for sleep events")
		}

		payload, err := json.Marshal(map[string]any{
			"start_at": startAt.Format(time.RFC3339),
			"end_at":   endAt.Format(time.RFC3339),
		})
		if err != nil {
			return CreateEventInput{}, errors.New("failed to encode details")
		}

		return CreateEventInput{
			BabyID:     babyID,
			Type:       "sleep",
			OccurredAt: startAt,
			Details:    payload,
		}, nil
	default:
		return CreateEventInput{}, errors.New("type must be diaper, nursing, or sleep")
	}
}

func parseTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("timestamp required")
	}
	return time.Parse(time.RFC3339, value)
}

func parseID(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("id required")
	}
	return strconv.ParseInt(value, 10, 64)
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
