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
	data     []server.Baby
	err      error
	event    server.Event
	eventErr error
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBabyStore) CreateEvent(_ context.Context, event server.NewEvent) (server.Event, error) {
	if s.eventErr != nil {
		return server.Event{}, s.eventErr
	}
	return s.event, nil
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

	store := stubBabyStore{event: server.Event{ID: 12, BabyID: 1, Type: server.EventTypeDiaper}}
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", strings.NewReader(`{"type":"diaper","diaper":{"kind":"wet","note":"quick"}}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

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
	if got.Data.Type != server.EventTypeDiaper {
		t.Fatalf("expected event type diaper, got %q", got.Data.Type)
	}
}

func TestCreateEventNursing(t *testing.T) {
	t.Parallel()

	store := stubBabyStore{event: server.Event{ID: 13, BabyID: 1, Type: server.EventTypeNursing}}
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", strings.NewReader(`{"type":"nursing","nursing":{"side":"left","duration_minutes":12}}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestCreateEventSleep(t *testing.T) {
	t.Parallel()

	store := stubBabyStore{event: server.Event{ID: 14, BabyID: 1, Type: server.EventTypeSleep}}
	req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", strings.NewReader(`{"type":"sleep","sleep":{"start_at":"2026-02-23T10:00:00Z","end_at":"2026-02-23T11:00:00Z"}}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestCreateEventValidation(t *testing.T) {
	t.Parallel()

	store := stubBabyStore{}

	t.Run("invalid id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/babies/0/events", strings.NewReader(`{"type":"diaper","diaper":{"kind":"wet"}}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		server.NewRouter(store).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("missing details", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", strings.NewReader(`{"type":"diaper"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		server.NewRouter(store).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("invalid nursing side", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", strings.NewReader(`{"type":"nursing","nursing":{"side":"middle"}}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		server.NewRouter(store).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})

	t.Run("invalid sleep window", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", strings.NewReader(`{"type":"sleep","sleep":{"start_at":"2026-02-23T11:00:00Z","end_at":"2026-02-23T10:00:00Z"}}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		server.NewRouter(store).ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})
}
