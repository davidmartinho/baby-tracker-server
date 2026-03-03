package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"baby-tracker-server/internal/server"
)

type stubBabyStore struct {
	data          []server.Baby
	err           error
	children      []server.Child
	createChildFn func(context.Context, server.CreateChildInput) (server.Child, error)
	updateChildFn func(context.Context, int64, server.UpdateChildInput) (server.Child, error)
	deleteChildFn func(context.Context, int64) error
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s stubBabyStore) ListChildren(_ context.Context) ([]server.Child, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.children, nil
}

func (s stubBabyStore) CreateChild(ctx context.Context, input server.CreateChildInput) (server.Child, error) {
	if s.createChildFn != nil {
		return s.createChildFn(ctx, input)
	}
	if s.err != nil {
		return server.Child{}, s.err
	}
	return server.Child{ID: 1, Name: input.Name, BirthDate: input.BirthDate, Gender: input.Gender}, nil
}

func (s stubBabyStore) UpdateChild(ctx context.Context, id int64, input server.UpdateChildInput) (server.Child, error) {
	if s.updateChildFn != nil {
		return s.updateChildFn(ctx, id, input)
	}
	if s.err != nil {
		return server.Child{}, s.err
	}
	return server.Child{ID: id, Name: input.Name, BirthDate: input.BirthDate, Gender: input.Gender}, nil
}

func (s stubBabyStore) DeleteChild(ctx context.Context, id int64) error {
	if s.deleteChildFn != nil {
		return s.deleteChildFn(ctx, id)
	}
	return s.err
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

func TestListChildren(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/children", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		children: []server.Child{{ID: 1, Name: "Alice", BirthDate: "2024-01-01", Gender: "female"}},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var got struct {
		Data []server.Child `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(got.Data) != 1 {
		t.Fatalf("expected 1 child, got %d", len(got.Data))
	}
	if got.Data[0].BirthDate != "2024-01-01" {
		t.Fatalf("expected birth date 2024-01-01, got %q", got.Data[0].BirthDate)
	}
}

func TestCreateChild(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/v1/children", bytes.NewBufferString(`{"name":"Alice","birthDate":"2024-01-01","gender":"female"}`))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	var got struct {
		Data server.Child `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if got.Data.Name != "Alice" {
		t.Fatalf("expected child name Alice, got %q", got.Data.Name)
	}
}

func TestCreateChildValidationError(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/v1/children", bytes.NewBufferString(`{"name":"Alice","birthDate":"01-01-2024","gender":"female"}`))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestUpdateChildNotFound(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPut, "/v1/children/99", bytes.NewBufferString(`{"name":"Alice","birthDate":"2024-01-01","gender":"female"}`))
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{
		updateChildFn: func(_ context.Context, _ int64, _ server.UpdateChildInput) (server.Child, error) {
			return server.Child{}, server.ErrNotFound
		},
	}).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestDeleteChild(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodDelete, "/v1/children/1", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
}
