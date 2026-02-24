package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"baby-tracker-server/internal/server"
)

type stubBabyStore struct {
	data         []server.Baby
	err          error
	createErr    error
	createdEvent server.Event
}

func (s *stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s *stubBabyStore) CreateEvent(_ context.Context, event server.Event) (server.Event, error) {
	if s.createErr != nil {
		return server.Event{}, s.createErr
	}
	event.ID = 99
	s.createdEvent = event
	return event, nil
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

func TestCreateDiaperChangeEvent(t *testing.T) {
	t.Parallel()

	store := &stubBabyStore{}
	startedAt := time.Date(2026, time.February, 24, 10, 0, 0, 0, time.UTC)
	payload := map[string]any{
		"type":       server.EventTypeDiaperChange,
		"started_at": startedAt.Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/42/events", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	if store.createdEvent.BabyID != 42 {
		t.Fatalf("expected baby id 42, got %d", store.createdEvent.BabyID)
	}
	if store.createdEvent.Type != server.EventTypeDiaperChange {
		t.Fatalf("expected type diaper_change, got %s", store.createdEvent.Type)
	}
	if !store.createdEvent.StartedAt.Equal(startedAt) {
		t.Fatalf("expected started_at %s, got %s", startedAt, store.createdEvent.StartedAt)
	}
	if store.createdEvent.EndedAt != nil {
		t.Fatalf("expected ended_at to be nil")
	}
}

func TestCreateNursingEventRequiresEnd(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.February, 24, 11, 0, 0, 0, time.UTC)
	payload := map[string]any{
		"type":       server.EventTypeNursingLeft,
		"started_at": startedAt.Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(&stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateSleepEvent(t *testing.T) {
	t.Parallel()

	store := &stubBabyStore{}
	startedAt := time.Date(2026, time.February, 24, 12, 0, 0, 0, time.UTC)
	endedAt := time.Date(2026, time.February, 24, 13, 0, 0, 0, time.UTC)
	payload := map[string]any{
		"type":       server.EventTypeSleep,
		"started_at": startedAt.Format(time.RFC3339),
		"ended_at":   endedAt.Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/7/events", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
	if store.createdEvent.EndedAt == nil {
		t.Fatalf("expected ended_at to be set")
	}
	if !store.createdEvent.EndedAt.Equal(endedAt) {
		t.Fatalf("expected ended_at %s, got %s", endedAt, store.createdEvent.EndedAt)
	}
}
