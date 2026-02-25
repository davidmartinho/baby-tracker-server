package server_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"baby-tracker-server/internal/server"
)

type stubBabyStore struct {
	data          []server.Baby
	err           error
	createEventFn func(context.Context, int64, server.CreateEventInput) (server.Event, error)
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBabyStore) CreateEvent(ctx context.Context, babyID int64, input server.CreateEventInput) (server.Event, error) {
	if s.err != nil {
		return server.Event{}, s.err
	}
	if s.createEventFn == nil {
		return server.Event{}, errors.New("createEventFn not set")
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

func TestCreateEventDiaperChange(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 2, 25, 10, 0, 0, 0, time.UTC)
	body := strings.NewReader(fmt.Sprintf(`{"type":"diaper_change","start_time":"%s"}`, start.Format(time.RFC3339Nano)))

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", body)
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		createEventFn: func(_ context.Context, babyID int64, input server.CreateEventInput) (server.Event, error) {
			if babyID != 1 {
				t.Fatalf("expected babyID 1, got %d", babyID)
			}
			if input.Type != "diaper_change" {
				t.Fatalf("expected type diaper_change, got %q", input.Type)
			}
			if input.StartTime == nil || !input.StartTime.Equal(start) {
				t.Fatalf("expected start_time %v, got %v", start, input.StartTime)
			}
			return server.Event{
				ID:        10,
				BabyID:    babyID,
				Type:      input.Type,
				StartTime: *input.StartTime,
			}, nil
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

	if got.Data.Type != "diaper_change" {
		t.Fatalf("expected diaper_change event, got %q", got.Data.Type)
	}
}

func TestCreateEventNursing(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 2, 25, 11, 0, 0, 0, time.UTC)
	end := start.Add(15 * time.Minute)
	body := strings.NewReader(fmt.Sprintf(`{"type":"nursing","start_time":"%s","end_time":"%s","side":"left"}`, start.Format(time.RFC3339Nano), end.Format(time.RFC3339Nano)))

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/2/events", body)
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		createEventFn: func(_ context.Context, babyID int64, input server.CreateEventInput) (server.Event, error) {
			if babyID != 2 {
				t.Fatalf("expected babyID 2, got %d", babyID)
			}
			return server.Event{
				ID:        11,
				BabyID:    babyID,
				Type:      input.Type,
				StartTime: *input.StartTime,
				EndTime:   input.EndTime,
				Side:      input.Side,
			}, nil
		},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestCreateEventSleep(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 2, 25, 12, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	body := strings.NewReader(fmt.Sprintf(`{"type":"sleep","start_time":"%s","end_time":"%s"}`, start.Format(time.RFC3339Nano), end.Format(time.RFC3339Nano)))

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/3/events", body)
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		createEventFn: func(_ context.Context, babyID int64, input server.CreateEventInput) (server.Event, error) {
			if babyID != 3 {
				t.Fatalf("expected babyID 3, got %d", babyID)
			}
			return server.Event{
				ID:        12,
				BabyID:    babyID,
				Type:      input.Type,
				StartTime: *input.StartTime,
				EndTime:   input.EndTime,
			}, nil
		},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
}

func TestCreateEventValidation(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/v1/babies/1/events", strings.NewReader(`{"type":"nursing","start_time":"2026-02-25T10:00:00Z"}`))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
