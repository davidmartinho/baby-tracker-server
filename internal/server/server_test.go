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
	data []server.Baby
	err  error

	createEventFn func(ctx context.Context, babyID int64, input server.EventInput) (server.Event, error)
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBabyStore) CreateEvent(ctx context.Context, babyID int64, input server.EventInput) (server.Event, error) {
	if s.createEventFn == nil {
		return server.Event{}, errors.New("not implemented")
	}
	return s.createEventFn(ctx, babyID, input)
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

func TestCreateDiaperChangeEvent(t *testing.T) {
	t.Parallel()

	reqBody := `{"type":"diaper_change","occurred_at":"2026-02-25T10:00:00Z","details":{"kind":"wet"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/12/events", strings.NewReader(reqBody))
	rr := httptest.NewRecorder()

	store := stubBabyStore{
		createEventFn: func(_ context.Context, babyID int64, input server.EventInput) (server.Event, error) {
			if babyID != 12 {
				t.Fatalf("expected babyID 12, got %d", babyID)
			}
			if input.Type != "diaper_change" {
				t.Fatalf("expected type diaper_change, got %q", input.Type)
			}
			if input.Details["kind"] != "wet" {
				t.Fatalf("expected kind wet, got %v", input.Details["kind"])
			}

			return server.Event{
				ID:         101,
				BabyID:     babyID,
				Type:       input.Type,
				OccurredAt: input.OccurredAt,
				Details:    input.Details,
				CreatedAt:  time.Date(2026, 2, 25, 10, 1, 0, 0, time.UTC),
			}, nil
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

	if got.Data.Type != "diaper_change" {
		t.Fatalf("expected event type diaper_change, got %q", got.Data.Type)
	}
	if got.Data.Details["kind"] != "wet" {
		t.Fatalf("expected details.kind wet, got %v", got.Data.Details["kind"])
	}
}

func TestCreateNursingEvent(t *testing.T) {
	t.Parallel()

	reqBody := `{"type":"nursing","occurred_at":"2026-02-25T11:00:00Z","details":{"side":"left","duration_minutes":15}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/5/events", strings.NewReader(reqBody))
	rr := httptest.NewRecorder()

	store := stubBabyStore{
		createEventFn: func(_ context.Context, babyID int64, input server.EventInput) (server.Event, error) {
			if babyID != 5 {
				t.Fatalf("expected babyID 5, got %d", babyID)
			}
			if input.Type != "nursing" {
				t.Fatalf("expected type nursing, got %q", input.Type)
			}
			if input.Details["side"] != "left" {
				t.Fatalf("expected side left, got %v", input.Details["side"])
			}
			if input.Details["duration_minutes"] != 15 {
				t.Fatalf("expected duration 15, got %v", input.Details["duration_minutes"])
			}

			return server.Event{
				ID:         202,
				BabyID:     babyID,
				Type:       input.Type,
				OccurredAt: input.OccurredAt,
				Details:    input.Details,
				CreatedAt:  time.Date(2026, 2, 25, 11, 2, 0, 0, time.UTC),
			}, nil
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

	if got.Data.Type != "nursing" {
		t.Fatalf("expected event type nursing, got %q", got.Data.Type)
	}
}

func TestCreateSleepEvent(t *testing.T) {
	t.Parallel()

	reqBody := `{"type":"sleep","occurred_at":"2026-02-25T12:00:00Z","details":{"duration_minutes":45}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/7/events", strings.NewReader(reqBody))
	rr := httptest.NewRecorder()

	store := stubBabyStore{
		createEventFn: func(_ context.Context, babyID int64, input server.EventInput) (server.Event, error) {
			if babyID != 7 {
				t.Fatalf("expected babyID 7, got %d", babyID)
			}
			if input.Type != "sleep" {
				t.Fatalf("expected type sleep, got %q", input.Type)
			}
			if input.Details["duration_minutes"] != 45 {
				t.Fatalf("expected duration 45, got %v", input.Details["duration_minutes"])
			}

			return server.Event{
				ID:         303,
				BabyID:     babyID,
				Type:       input.Type,
				OccurredAt: input.OccurredAt,
				Details:    input.Details,
				CreatedAt:  time.Date(2026, 2, 25, 12, 4, 0, 0, time.UTC),
			}, nil
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

	if got.Data.Type != "sleep" {
		t.Fatalf("expected event type sleep, got %q", got.Data.Type)
	}
}
