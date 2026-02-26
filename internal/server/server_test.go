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
	data        []server.Baby
	err         error
	createEvent func(event server.Event) (server.Event, error)
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBabyStore) CreateEvent(_ context.Context, event server.Event) (server.Event, error) {
	if s.createEvent == nil {
		return server.Event{}, errors.New("create event not configured")
	}
	return s.createEvent(event)
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

	occurredAt := time.Date(2026, 2, 25, 10, 30, 0, 0, time.UTC)
	body := map[string]any{
		"type":        "diaper",
		"occurred_at": occurredAt.Format(time.RFC3339),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/events", mustJSONBody(t, body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		createEvent: func(event server.Event) (server.Event, error) {
			event.ID = 42
			return event, nil
		},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var got struct {
		Data server.Event `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if got.Data.ID != 42 {
		t.Fatalf("expected event id 42, got %d", got.Data.ID)
	}
	if got.Data.Type != "diaper" {
		t.Fatalf("expected event type diaper, got %q", got.Data.Type)
	}
	if got.Data.OccurredAt == nil || !got.Data.OccurredAt.Equal(occurredAt) {
		t.Fatalf("expected occurred_at %v, got %v", occurredAt, got.Data.OccurredAt)
	}
}

func TestCreateNursingEvent(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 2, 25, 11, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(10 * time.Minute)
	side := "left"
	body := map[string]any{
		"type":       "nursing",
		"started_at": startedAt.Format(time.RFC3339),
		"ended_at":   endedAt.Format(time.RFC3339),
		"side":       side,
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/events", mustJSONBody(t, body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		createEvent: func(event server.Event) (server.Event, error) {
			event.ID = 7
			return event, nil
		},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var got struct {
		Data server.Event `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if got.Data.Type != "nursing" {
		t.Fatalf("expected event type nursing, got %q", got.Data.Type)
	}
	if got.Data.StartedAt == nil || !got.Data.StartedAt.Equal(startedAt) {
		t.Fatalf("expected started_at %v, got %v", startedAt, got.Data.StartedAt)
	}
	if got.Data.EndedAt == nil || !got.Data.EndedAt.Equal(endedAt) {
		t.Fatalf("expected ended_at %v, got %v", endedAt, got.Data.EndedAt)
	}
	if got.Data.Side == nil || *got.Data.Side != side {
		t.Fatalf("expected side %q, got %v", side, got.Data.Side)
	}
}

func TestCreateSleepEvent(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 2, 25, 20, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(2 * time.Hour)
	body := map[string]any{
		"type":       "sleep",
		"started_at": startedAt.Format(time.RFC3339),
		"ended_at":   endedAt.Format(time.RFC3339),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/events", mustJSONBody(t, body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		createEvent: func(event server.Event) (server.Event, error) {
			event.ID = 11
			return event, nil
		},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var got struct {
		Data server.Event `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if got.Data.Type != "sleep" {
		t.Fatalf("expected event type sleep, got %q", got.Data.Type)
	}
	if got.Data.StartedAt == nil || !got.Data.StartedAt.Equal(startedAt) {
		t.Fatalf("expected started_at %v, got %v", startedAt, got.Data.StartedAt)
	}
	if got.Data.EndedAt == nil || !got.Data.EndedAt.Equal(endedAt) {
		t.Fatalf("expected ended_at %v, got %v", endedAt, got.Data.EndedAt)
	}
	if got.Data.Side != nil {
		t.Fatalf("expected no side for sleep event, got %v", got.Data.Side)
	}
}

func TestCreateEventValidation(t *testing.T) {
	t.Parallel()

	body := map[string]any{
		"type": "nursing",
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/events", mustJSONBody(t, body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		createEvent: func(event server.Event) (server.Event, error) {
			return event, nil
		},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateEventStoreError(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, 2, 25, 10, 30, 0, 0, time.UTC)
	body := map[string]any{
		"type":        "diaper",
		"occurred_at": occurredAt.Format(time.RFC3339),
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/events", mustJSONBody(t, body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		createEvent: func(event server.Event) (server.Event, error) {
			return server.Event{}, errors.New("boom")
		},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func mustJSONBody(t *testing.T, payload any) *bytes.Reader {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}

	return bytes.NewReader(data)
}
