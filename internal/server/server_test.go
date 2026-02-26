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
	data         []server.Baby
	err          error
	createdEvent server.Event
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBabyStore) CreateEvent(_ context.Context, _ int64, event server.Event) (server.Event, error) {
	if s.err != nil {
		return server.Event{}, s.err
	}
	if s.createdEvent.ID != 0 {
		return s.createdEvent, nil
	}
	event.ID = 99
	return event, nil
}

func TestHealthz(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

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

	server.NewRouter(stubBabyStore{
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

	server.NewRouter(stubBabyStore{err: errors.New("boom")}).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestGetProfile(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

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

	occurredAt := time.Date(2026, 2, 25, 9, 0, 0, 0, time.UTC)
	body := `{"type":"diaper","occurred_at":"` + occurredAt.Format(time.RFC3339) + `","diaper_kind":"wet"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/10/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var got struct {
		Data server.Event `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if got.Data.Type != "diaper" {
		t.Fatalf("expected diaper event, got %q", got.Data.Type)
	}
	if got.Data.DiaperKind == nil || *got.Data.DiaperKind != "wet" {
		t.Fatalf("expected diaper_kind wet, got %#v", got.Data.DiaperKind)
	}
}

func TestCreateNursingEvent(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	body := `{"type":"nursing","occurred_at":"` + occurredAt.Format(time.RFC3339) + `","nursing_side":"left","nursing_duration_minutes":15}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/3/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var got struct {
		Data server.Event `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if got.Data.NursingSide == nil || *got.Data.NursingSide != "left" {
		t.Fatalf("expected nursing_side left, got %#v", got.Data.NursingSide)
	}
	if got.Data.NursingDurationMinutes == nil || *got.Data.NursingDurationMinutes != 15 {
		t.Fatalf("expected duration 15, got %#v", got.Data.NursingDurationMinutes)
	}
}

func TestCreateSleepEvent(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 2, 25, 20, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 25, 22, 0, 0, 0, time.UTC)
	body := `{"type":"sleep","sleep_start":"` + start.Format(time.RFC3339) + `","sleep_end":"` + end.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/5/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var got struct {
		Data server.Event `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if got.Data.SleepStart == nil || !got.Data.SleepStart.Equal(start) {
		t.Fatalf("expected sleep_start %v, got %#v", start, got.Data.SleepStart)
	}
	if got.Data.SleepEnd == nil || !got.Data.SleepEnd.Equal(end) {
		t.Fatalf("expected sleep_end %v, got %#v", end, got.Data.SleepEnd)
	}
}

func TestCreateEventBadRequest(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/0/events", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateEventStoreError(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, 2, 25, 9, 0, 0, 0, time.UTC)
	body := `{"type":"diaper","occurred_at":"` + occurredAt.Format(time.RFC3339) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/2/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{err: errors.New("fail")}).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
