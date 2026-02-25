package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"baby-tracker-server/internal/server"
)

type stubBabyStore struct {
	data            []server.Baby
	err             error
	createEventErr  error
	createEventID   int64
	createEventArgs *server.CreateEventInput
}

func (s *stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s *stubBabyStore) CreateEvent(_ context.Context, input server.CreateEventInput) (server.Event, error) {
	s.createEventArgs = &input
	if s.createEventErr != nil {
		return server.Event{}, s.createEventErr
	}
	return server.Event{
		ID:         s.createEventID,
		BabyID:     input.BabyID,
		Type:       input.Type,
		OccurredAt: input.OccurredAt,
		Details:    input.Details,
	}, nil
}

func TestHealthz(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(&stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var got map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if got["status"] != "ok" {
		t.Fatalf("expected health status to be 'ok', got %q", got["status"])
	}
}

func TestListBabies(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/babies", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(&stubBabyStore{
		data: []server.Baby{{ID: 1, Name: "Alice"}},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var got struct {
		Data []server.Baby `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(got.Data) != 1 {
		t.Fatalf("expected 1 baby, got %d", len(got.Data))
	}
	if got.Data[0].Name != "Alice" {
		t.Fatalf("expected baby name Alice, got %q", got.Data[0].Name)
	}
}

func TestListBabiesStoreError(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/babies", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(&stubBabyStore{err: errors.New("boom")}).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestGetProfile(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(&stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var got struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if got.ID == "" {
		t.Fatal("expected id to be present")
	}
	if got.Name == "" {
		t.Fatal("expected name to be present")
	}
	if got.Email == "" {
		t.Fatal("expected email to be present")
	}
}

func TestCreateDiaperEvent(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.February, 25, 9, 30, 0, 0, time.UTC)
	body := `{"type":"diaper","occurred_at":"` + occurredAt.Format(time.RFC3339) + `","diaper":{"kind":"wet"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/42/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	store := &stubBabyStore{createEventID: 101}
	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
	if store.createEventArgs == nil {
		t.Fatal("expected CreateEvent to be called")
	}
	if store.createEventArgs.BabyID != 42 {
		t.Fatalf("expected baby id 42, got %d", store.createEventArgs.BabyID)
	}
	if store.createEventArgs.Type != server.EventTypeDiaper {
		t.Fatalf("expected diaper event, got %s", store.createEventArgs.Type)
	}

	var got struct {
		Data server.Event `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got.Data.ID != 101 {
		t.Fatalf("expected event id 101, got %d", got.Data.ID)
	}

	var details server.DiaperDetails
	if err := json.Unmarshal(got.Data.Details, &details); err != nil {
		t.Fatalf("failed to unmarshal diaper details: %v", err)
	}
	if details.Kind != "wet" {
		t.Fatalf("expected diaper kind wet, got %q", details.Kind)
	}
}

func TestCreateNursingEvent(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.February, 25, 10, 0, 0, 0, time.UTC)
	body := `{"type":"nursing","occurred_at":"` + occurredAt.Format(time.RFC3339) + `","nursing":{"side":"left","duration_minutes":18}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/7/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	store := &stubBabyStore{createEventID: 202}
	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
	if store.createEventArgs == nil {
		t.Fatal("expected CreateEvent to be called")
	}
	if store.createEventArgs.BabyID != 7 {
		t.Fatalf("expected baby id 7, got %d", store.createEventArgs.BabyID)
	}
	if store.createEventArgs.Type != server.EventTypeNursing {
		t.Fatalf("expected nursing event, got %s", store.createEventArgs.Type)
	}

	var got struct {
		Data server.Event `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got.Data.ID != 202 {
		t.Fatalf("expected event id 202, got %d", got.Data.ID)
	}

	var details server.NursingDetails
	if err := json.Unmarshal(got.Data.Details, &details); err != nil {
		t.Fatalf("failed to unmarshal nursing details: %v", err)
	}
	if details.Side != "left" {
		t.Fatalf("expected nursing side left, got %q", details.Side)
	}
	if details.DurationMinutes != 18 {
		t.Fatalf("expected nursing duration 18, got %d", details.DurationMinutes)
	}
}

func TestCreateSleepEvent(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, time.February, 25, 12, 30, 0, 0, time.UTC)
	startTime := time.Date(2026, time.February, 25, 12, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, time.February, 25, 12, 30, 0, 0, time.UTC)
	body := `{"type":"sleep","occurred_at":"` + occurredAt.Format(time.RFC3339) + `","sleep":{"start_time":"` + startTime.Format(time.RFC3339) + `","end_time":"` + endTime.Format(time.RFC3339) + `"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/9/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	store := &stubBabyStore{createEventID: 303}
	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
	if store.createEventArgs == nil {
		t.Fatal("expected CreateEvent to be called")
	}
	if store.createEventArgs.BabyID != 9 {
		t.Fatalf("expected baby id 9, got %d", store.createEventArgs.BabyID)
	}
	if store.createEventArgs.Type != server.EventTypeSleep {
		t.Fatalf("expected sleep event, got %s", store.createEventArgs.Type)
	}

	var got struct {
		Data server.Event `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got.Data.ID != 303 {
		t.Fatalf("expected event id 303, got %d", got.Data.ID)
	}

	var details server.SleepDetails
	if err := json.Unmarshal(got.Data.Details, &details); err != nil {
		t.Fatalf("failed to unmarshal sleep details: %v", err)
	}
	if !details.StartTime.Equal(startTime) {
		t.Fatalf("expected sleep start time %s, got %s", startTime.Format(time.RFC3339), details.StartTime.Format(time.RFC3339))
	}
	if !details.EndTime.Equal(endTime) {
		t.Fatalf("expected sleep end time %s, got %s", endTime.Format(time.RFC3339), details.EndTime.Format(time.RFC3339))
	}
}
