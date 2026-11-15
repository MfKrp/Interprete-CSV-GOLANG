package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/repository"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testServer() *http.ServeMux {
	mux := http.NewServeMux()
	NewCSVHandler(repository.NewDatasetRepository(), 10*1024*1024).Register(mux)
	return mux
}

func uploadRequest(t *testing.T, content string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	f, err := w.CreateFormFile("file", "sample.csv")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte(content))
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/datasets", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestDatasetEndpoints(t *testing.T) {
	mux := testServer()
	created := httptest.NewRecorder()
	mux.ServeHTTP(created, uploadRequest(t, "name,age\nAna,30\n"))
	if created.Code != http.StatusCreated {
		t.Fatalf("POST status = %d: %s", created.Code, created.Body)
	}
	var metadata struct {
		ID   string `json:"id"`
		Rows int    `json:"row_count"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.ID == "" || metadata.Rows != 1 {
		t.Fatalf("metadata = %#v", metadata)
	}
	got := httptest.NewRecorder()
	mux.ServeHTTP(got, httptest.NewRequest(http.MethodGet, "/api/datasets/"+metadata.ID, nil))
	if got.Code != http.StatusOK {
		t.Fatalf("GET status = %d", got.Code)
	}
	missing := httptest.NewRecorder()
	mux.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/datasets/missing", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d", missing.Code)
	}
}

func TestDatasetUploadRejectsInvalidCSV(t *testing.T) {
	response := httptest.NewRecorder()
	testServer().ServeHTTP(response, uploadRequest(t, "a,b\n1\n"))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	if !strings.Contains(response.Body.String(), "invalid CSV") {
		t.Fatalf("body = %s", response.Body)
	}
}
