package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/repository"
	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/service"
)

type CSVHandler struct {
	repository *repository.DatasetRepository
	maxBytes   int64
}

func NewCSVHandler(repo *repository.DatasetRepository, maxBytes int64) *CSVHandler {
	return &CSVHandler{repository: repo, maxBytes: maxBytes}
}

func (h *CSVHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/datasets", h.create)
	mux.HandleFunc("GET /api/datasets/{id}", h.get)
}

func (h *CSVHandler) create(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes+(1<<20)) // multipart overhead allowance
	parseErr := r.ParseMultipartForm(h.maxBytes)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if parseErr != nil {
		var maxErr *http.MaxBytesError
		if errors.As(parseErr, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "file exceeds 10 MB limit")
		} else {
			writeError(w, http.StatusBadRequest, "invalid multipart form")
		}
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "multipart field 'file' is required")
		return
	}
	defer file.Close()
	if header.Size == 0 {
		writeError(w, http.StatusBadRequest, "file must not be empty")
		return
	}
	if header.Size > h.maxBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file exceeds 10 MB limit")
		return
	}
	parsed, err := service.ReadCSV(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid CSV: %v", err))
		return
	}
	dataset, err := h.repository.Save(repository.Dataset{Filename: filepath.Base(header.Filename), Dataset: parsed})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not store dataset")
		return
	}
	writeJSON(w, http.StatusCreated, dataset)
}

func (h *CSVHandler) get(w http.ResponseWriter, r *http.Request) {
	dataset, ok := h.repository.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "dataset not found")
		return
	}
	writeJSON(w, http.StatusOK, dataset)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
