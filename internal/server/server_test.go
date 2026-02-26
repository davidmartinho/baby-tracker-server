package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"baby-tracker-server/internal/server"
)

type stubBabyStore struct {
	data []server.Baby
	err  error

	createdEvent server.Event
	createErr    error
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBabyStore) CreateEvent(_ context.Context, event server.Event) (server.Event, error) {
	if s.createErr != nil {
		return server.Event{}, s.createErr
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

func TestCreateEventDiaperChange(t *testing.T) {
	t.Parallel()

	body := `{"baby_id":1,"type":"diaper_change","occurred_at":"2026-02-25T10:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(body))
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

	if got.Data.Type != "diaper_change" {
		t.Fatalf("expected diaper_change, got %q", got.Data.Type)
	}
	if got.Data.OccurredAt == nil {
		t.Fatal("expected occurred_at to be set")
	}
	if got.Data.ID == 0 {
		t.Fatal("expected id to be set")
	}
}

func TestCreateEventNursing(t *testing.T) {
	t.Parallel()

	body := `{"baby_id":2,"type":"nursing","started_at":"2026-02-25T10:00:00Z","ended_at":"2026-02-25T10:20:00Z","side":"left"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(body))
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

	if got.Data.Side == nil || *got.Data.Side != "left" {
		t.Fatalf("expected side left, got %v", got.Data.Side)
	}
}

func TestCreateEventSleep(t *testing.T) {
	t.Parallel()

	body := `{"baby_id":3,"type":"sleep","started_at":"2026-02-25T21:00:00Z","ended_at":"2026-02-25T23:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(body))
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

	if got.Data.StartedAt == nil || got.Data.EndedAt == nil {
		t.Fatal("expected started_at and ended_at to be set")
	}
}

func TestCreateEventValidation(t *testing.T) {
	t.Parallel()

	body := `{"baby_id":1,"type":"nursing","started_at":"2026-02-25T10:00:00Z","ended_at":"2026-02-25T10:20:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateEventStoreError(t *testing.T) {
	t.Parallel()

	body := `{"baby_id":1,"type":"diaper_change","occurred_at":"2026-02-25T10:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{createErr: errors.New("boom")}).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}
