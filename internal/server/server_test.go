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
	babies   []server.Baby
	children []server.Child

	listBabiesErr   error
	listChildrenErr error
	createChildErr  error
	updateChildErr  error
	deleteChildErr  error

	createChildInput server.CreateChildInput
	updateChildID    int64
	updateChildInput server.UpdateChildInput
	deleteChildID    int64
}

func (s *stubBabyStore) ListBabies(_ context.Context) ([]server.Baby, error) {
	if s.listBabiesErr != nil {
		return nil, s.listBabiesErr
	}
	return s.babies, nil
}

func (s *stubBabyStore) ListChildren(_ context.Context) ([]server.Child, error) {
	if s.listChildrenErr != nil {
		return nil, s.listChildrenErr
	}
	return s.children, nil
}

func (s *stubBabyStore) CreateChild(_ context.Context, input server.CreateChildInput) (server.Child, error) {
	s.createChildInput = input
	if s.createChildErr != nil {
		return server.Child{}, s.createChildErr
	}

	if len(s.children) > 0 {
		return s.children[0], nil
	}

	return server.Child{ID: 1, Name: input.Name, BirthDate: input.BirthDate, Gender: input.Gender}, nil
}

func (s *stubBabyStore) UpdateChild(_ context.Context, id int64, input server.UpdateChildInput) (server.Child, error) {
	s.updateChildID = id
	s.updateChildInput = input
	if s.updateChildErr != nil {
		return server.Child{}, s.updateChildErr
	}

	if len(s.children) > 0 {
		return s.children[0], nil
	}

	return server.Child{ID: id, Name: input.Name, BirthDate: input.BirthDate, Gender: input.Gender}, nil
}

func (s *stubBabyStore) DeleteChild(_ context.Context, id int64) error {
	s.deleteChildID = id
	if s.deleteChildErr != nil {
		return s.deleteChildErr
	}
	return nil
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

	server.NewRouter(&stubBabyStore{babies: []server.Baby{{ID: 1, Name: "Alice"}}}).ServeHTTP(rr, req)

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

	server.NewRouter(&stubBabyStore{listBabiesErr: errors.New("boom")}).ServeHTTP(rr, req)

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

func TestListChildren(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/v1/children", nil)
	rr := httptest.NewRecorder()
	store := &stubBabyStore{children: []server.Child{{ID: 1, Name: "Eve", BirthDate: "2024-01-02", Gender: "female"}}}

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var got struct {
		Data []server.Child `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(got.Data) != 1 || got.Data[0].Name != "Eve" {
		t.Fatalf("unexpected response data: %+v", got.Data)
	}
}

func TestCreateChild(t *testing.T) {
	t.Parallel()

	body := `{"name":"Eve","birthDate":"2024-01-02","gender":"female"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/children", strings.NewReader(body))
	rr := httptest.NewRecorder()
	store := &stubBabyStore{}

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rr.Code)
	}

	if store.createChildInput.Name != "Eve" || store.createChildInput.BirthDate != "2024-01-02" || store.createChildInput.Gender != "female" {
		t.Fatalf("unexpected create payload: %+v", store.createChildInput)
	}
}

func TestCreateChildValidationError(t *testing.T) {
	t.Parallel()

	body := `{"name":"","birthDate":"bad-date","gender":"unknown"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/children", strings.NewReader(body))
	rr := httptest.NewRecorder()

	server.NewRouter(&stubBabyStore{}).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestUpdateChildNotFound(t *testing.T) {
	t.Parallel()

	body := `{"name":"Eve","birthDate":"2024-01-02","gender":"female"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/children/42", strings.NewReader(body))
	rr := httptest.NewRecorder()
	store := &stubBabyStore{updateChildErr: server.ErrChildNotFound}

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}

func TestUpdateChild(t *testing.T) {
	t.Parallel()

	body := `{"name":"Eve","birthDate":"2024-01-02","gender":"female"}`
	req := httptest.NewRequest(http.MethodPut, "/v1/children/7", strings.NewReader(body))
	rr := httptest.NewRecorder()
	store := &stubBabyStore{}

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if store.updateChildID != 7 {
		t.Fatalf("expected update id 7, got %d", store.updateChildID)
	}
}

func TestDeleteChild(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodDelete, "/v1/children/7", nil)
	rr := httptest.NewRecorder()
	store := &stubBabyStore{}

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, rr.Code)
	}

	if store.deleteChildID != 7 {
		t.Fatalf("expected delete id 7, got %d", store.deleteChildID)
	}
}

func TestDeleteChildNotFound(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodDelete, "/v1/children/7", nil)
	rr := httptest.NewRecorder()
	store := &stubBabyStore{deleteChildErr: server.ErrChildNotFound}

	server.NewRouter(store).ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rr.Code)
	}
}
