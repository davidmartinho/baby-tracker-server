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
	data     []server.Baby
	err      error
	createFn func(context.Context, server.Event) (server.Event, error)
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBabyStore) CreateEvent(ctx context.Context, event server.Event) (server.Event, error) {
	if s.createFn != nil {
		return s.createFn(ctx, event)
	}
	event.ID = 1
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

func TestCreateEventDiaper(t *testing.T) {
	t.Parallel()

	occurredAt := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	body := `{"type":"diaper","occurred_at":"2026-02-25T10:00:00Z"}`

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/12/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	store := stubBabyStore{
		createFn: func(_ context.Context, event server.Event) (server.Event, error) {
			if event.BabyID != 12 {
				t.Fatalf("expected baby_id 12, got %d", event.BabyID)
			}
			if event.Type != "diaper" {
				t.Fatalf("expected type diaper, got %q", event.Type)
			}
			if event.OccurredAt == nil || !event.OccurredAt.Equal(occurredAt) {
				t.Fatalf("expected occurred_at %v, got %v", occurredAt, event.OccurredAt)
			}
			if event.StartedAt != nil || event.EndedAt != nil {
				t.Fatalf("expected no range timestamps, got %v %v", event.StartedAt, event.EndedAt)
			}
			event.ID = 101
			return event, nil
		},
	}

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
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
	if got.Data.Type != "diaper" {
		t.Fatalf("expected type diaper, got %q", got.Data.Type)
	}
	if got.Data.OccurredAt == nil || !got.Data.OccurredAt.Equal(occurredAt) {
		t.Fatalf("expected occurred_at %v, got %v", occurredAt, got.Data.OccurredAt)
	}
}

func TestCreateEventNursing(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	endedAt := time.Date(2026, 2, 25, 10, 30, 0, 0, time.UTC)
	body := `{"type":"nursing","side":"left","started_at":"2026-02-25T10:00:00Z","ended_at":"2026-02-25T10:30:00Z"}`

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/5/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	store := stubBabyStore{
		createFn: func(_ context.Context, event server.Event) (server.Event, error) {
			if event.BabyID != 5 {
				t.Fatalf("expected baby_id 5, got %d", event.BabyID)
			}
			if event.Type != "nursing" {
				t.Fatalf("expected type nursing, got %q", event.Type)
			}
			if event.Side != "left" {
				t.Fatalf("expected side left, got %q", event.Side)
			}
			if event.StartedAt == nil || !event.StartedAt.Equal(startedAt) {
				t.Fatalf("expected started_at %v, got %v", startedAt, event.StartedAt)
			}
			if event.EndedAt == nil || !event.EndedAt.Equal(endedAt) {
				t.Fatalf("expected ended_at %v, got %v", endedAt, event.EndedAt)
			}
			event.ID = 202
			return event, nil
		},
	}

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
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
	if got.Data.Side != "left" {
		t.Fatalf("expected side left, got %q", got.Data.Side)
	}
	if got.Data.StartedAt == nil || !got.Data.StartedAt.Equal(startedAt) {
		t.Fatalf("expected started_at %v, got %v", startedAt, got.Data.StartedAt)
	}
	if got.Data.EndedAt == nil || !got.Data.EndedAt.Equal(endedAt) {
		t.Fatalf("expected ended_at %v, got %v", endedAt, got.Data.EndedAt)
	}
}

func TestCreateEventSleep(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	endedAt := time.Date(2026, 2, 25, 13, 15, 0, 0, time.UTC)
	body := `{"type":"sleep","started_at":"2026-02-25T12:00:00Z","ended_at":"2026-02-25T13:15:00Z"}`

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/9/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	store := stubBabyStore{
		createFn: func(_ context.Context, event server.Event) (server.Event, error) {
			if event.BabyID != 9 {
				t.Fatalf("expected baby_id 9, got %d", event.BabyID)
			}
			if event.Type != "sleep" {
				t.Fatalf("expected type sleep, got %q", event.Type)
			}
			if event.StartedAt == nil || !event.StartedAt.Equal(startedAt) {
				t.Fatalf("expected started_at %v, got %v", startedAt, event.StartedAt)
			}
			if event.EndedAt == nil || !event.EndedAt.Equal(endedAt) {
				t.Fatalf("expected ended_at %v, got %v", endedAt, event.EndedAt)
			}
			event.ID = 303
			return event, nil
		},
	}

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
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
}

func TestCreateEventValidationError(t *testing.T) {
	t.Parallel()

	body := `{"type":"nursing","side":"left"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateEventStoreError(t *testing.T) {
	t.Parallel()

	body := `{"type":"diaper","occurred_at":"2026-02-25T10:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		createFn: func(context.Context, server.Event) (server.Event, error) {
			return server.Event{}, errors.New("boom")
		},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

func TestCreateEventBadBabyID(t *testing.T) {
	t.Parallel()

	body := `{"type":"diaper","occurred_at":"2026-02-25T10:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/not-a-number/events", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
