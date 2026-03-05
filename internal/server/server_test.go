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
	data        []server.Baby
	err         error
	createData  server.Baby
	updateData  server.Baby
	deleteErr   error
	createInput server.CreateBabyInput
	updateID    int64
	updateInput server.UpdateBabyInput
	deleteID    int64
}

func (s stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.data, nil
}

func (s *stubBabyStore) CreateBaby(_ context.Context, input server.CreateBabyInput) (server.Baby, error) {
	if s.err != nil {
		return server.Baby{}, s.err
	}
	s.createInput = input
	return s.createData, nil
}

func (s *stubBabyStore) UpdateBaby(_ context.Context, id int64, input server.UpdateBabyInput) (server.Baby, error) {
	if s.err != nil {
		return server.Baby{}, s.err
	}
	s.updateID = id
	s.updateInput = input
	return s.updateData, nil
}

func (s *stubBabyStore) DeleteBaby(_ context.Context, id int64) error {
	s.deleteID = id
	return s.deleteErr
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

func TestCreateBaby(t *testing.T) {
	t.Parallel()

	store := &stubBabyStore{
		createData: server.Baby{
			ID:        1,
			Name:      "Alice",
			BirthDate: "2024-01-12",
			Gender:    "female",
		},
	}

	body := bytes.NewBufferString(`{"name":"Alice","birthDate":"2024-01-12","gender":"female"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/babies", body)
	rr := httptest.NewRecorder()

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}
	if store.createInput.Name != "Alice" {
		t.Fatalf("expected store to receive name Alice, got %q", store.createInput.Name)
	}
}

func TestCreateBabyValidation(t *testing.T) {
	t.Parallel()

	body := bytes.NewBufferString(`{"name":"Alice","birthDate":"invalid","gender":"female"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/babies", body)
	rr := httptest.NewRecorder()

	server.NewRouter(&stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestUpdateBaby(t *testing.T) {
	t.Parallel()

	store := &stubBabyStore{
		updateData: server.Baby{
			ID:        5,
			Name:      "Bob",
			BirthDate: "2023-06-10",
			Gender:    "male",
		},
	}

	body := bytes.NewBufferString(`{"name":"Bob","birthDate":"2023-06-10","gender":"male"}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/babies/5", body)
	rr := httptest.NewRecorder()

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if store.updateID != 5 {
		t.Fatalf("expected id 5, got %d", store.updateID)
	}
}

func TestUpdateBabyNotFound(t *testing.T) {
	t.Parallel()

	store := &stubBabyStore{err: server.ErrNotFound}
	body := bytes.NewBufferString(`{"name":"Bob","birthDate":"2023-06-10","gender":"male"}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/babies/99", body)
	rr := httptest.NewRecorder()

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestDeleteBaby(t *testing.T) {
	t.Parallel()

	store := &stubBabyStore{}
	req := httptest.NewRequest(http.MethodDelete, "/v1/babies/7", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}
	if store.deleteID != 7 {
		t.Fatalf("expected id 7, got %d", store.deleteID)
	}
}

func TestDeleteBabyNotFound(t *testing.T) {
	t.Parallel()

	store := &stubBabyStore{deleteErr: server.ErrNotFound}
	req := httptest.NewRequest(http.MethodDelete, "/v1/babies/7", nil)
	rr := httptest.NewRecorder()

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}
