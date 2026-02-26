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
	data            []server.Baby
	err             error
	createEventFunc func(ctx context.Context, input server.CreateEventInput) (server.Event, error)
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBabyStore) CreateEvent(ctx context.Context, input server.CreateEventInput) (server.Event, error) {
	if s.createEventFunc == nil {
		return server.Event{}, errors.New("create event not implemented")
	}
	return s.createEventFunc(ctx, input)
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

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/42/events", strings.NewReader(`{
		"type": "diaper",
		"occurred_at": "2026-02-26T10:00:00Z",
		"notes": "quick change"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	store := stubBabyStore{
		createEventFunc: func(_ context.Context, input server.CreateEventInput) (server.Event, error) {
			if input.BabyID != 42 {
				t.Fatalf("expected baby id 42, got %d", input.BabyID)
			}
			if input.Type != "diaper" {
				t.Fatalf("expected type diaper, got %q", input.Type)
			}
			return server.Event{
				ID:         100,
				BabyID:     input.BabyID,
				Type:       input.Type,
				OccurredAt: input.OccurredAt,
				Details:    input.Details,
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
	if got.Data.Type != "diaper" {
		t.Fatalf("expected type diaper, got %q", got.Data.Type)
	}
}

func TestCreateEventNursing(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/7/events", strings.NewReader(`{
		"type": "nursing",
		"occurred_at": "2026-02-26T11:15:00Z",
		"side": "left",
		"duration_minutes": 18
	}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	store := stubBabyStore{
		createEventFunc: func(_ context.Context, input server.CreateEventInput) (server.Event, error) {
			if input.Type != "nursing" {
				t.Fatalf("expected type nursing, got %q", input.Type)
			}
			return server.Event{
				ID:         101,
				BabyID:     input.BabyID,
				Type:       input.Type,
				OccurredAt: input.OccurredAt,
				Details:    input.Details,
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
		t.Fatalf("expected type nursing, got %q", got.Data.Type)
	}
}

func TestCreateEventSleep(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/9/events", strings.NewReader(`{
		"type": "sleep",
		"start_at": "2026-02-26T12:00:00Z",
		"end_at": "2026-02-26T13:30:00Z"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	store := stubBabyStore{
		createEventFunc: func(_ context.Context, input server.CreateEventInput) (server.Event, error) {
			if input.Type != "sleep" {
				t.Fatalf("expected type sleep, got %q", input.Type)
			}
			return server.Event{
				ID:         102,
				BabyID:     input.BabyID,
				Type:       input.Type,
				OccurredAt: input.OccurredAt,
				Details:    input.Details,
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
		t.Fatalf("expected type sleep, got %q", got.Data.Type)
	}
}
