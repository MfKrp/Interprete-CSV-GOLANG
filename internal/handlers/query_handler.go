package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/service"
)

const maxQueryBodyBytes = 32 * 1024

type QueryInterpreter interface {
	Interpret(ctx context.Context, question string, columns []service.Column) (service.AnalysisRequest, error)
}

type QueryExecutor interface {
	Execute(datasetID string, request service.AnalysisRequest) (service.AnalysisResult, error)
}

type QueryHandler struct {
	datasets service.DatasetProvider
	llm      QueryInterpreter
	analysis QueryExecutor
}

type queryRequest struct {
	Question string `json:"question"`
}

type queryResponse struct {
	Question    string                  `json:"question"`
	Instruction service.AnalysisRequest `json:"instruction"`
	Result      queryResult             `json:"result"`
}

type queryResult struct {
	Value  any                `json:"value,omitempty"`
	Groups []queryGroupResult `json:"groups,omitempty"`
}

type queryGroupResult struct {
	Key   string   `json:"key"`
	Count *int     `json:"count,omitempty"`
	Sum   *float64 `json:"sum,omitempty"`
}

func NewQueryHandler(datasets service.DatasetProvider, llm QueryInterpreter, analysisService QueryExecutor) *QueryHandler {
	return &QueryHandler{datasets: datasets, llm: llm, analysis: analysisService}
}

func (h *QueryHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/datasets/{id}/query", h.query)
}

func (h *QueryHandler) query(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxQueryBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input queryRequest
	if err := decoder.Decode(&input); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body is too large")
		} else {
			writeError(w, http.StatusBadRequest, "body must be JSON with a question field")
		}
		return
	}
	if err := ensureRequestJSONEnd(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "body must contain one JSON object")
		return
	}
	if strings.TrimSpace(input.Question) == "" {
		writeError(w, http.StatusBadRequest, "question must not be empty")
		return
	}
	datasetID := r.PathValue("id")
	dataset, ok := h.datasets.GetDataset(datasetID)
	if !ok {
		writeError(w, http.StatusNotFound, "dataset not found")
		return
	}
	instruction, err := h.llm.Interpret(r.Context(), input.Question, dataset.Columns)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUninterpretable):
			writeError(w, http.StatusUnprocessableEntity, "question could not be interpreted with available operations")
		case errors.Is(err, service.ErrEmptyQuestion):
			writeError(w, http.StatusBadRequest, "question must not be empty")
		case errors.Is(err, service.ErrOllamaUnavailable):
			writeError(w, http.StatusBadGateway, "Ollama is unavailable or timed out")
		case errors.Is(err, service.ErrInvalidOllamaResponse), errors.Is(err, service.ErrInvalidInstruction):
			writeError(w, http.StatusBadGateway, "Ollama returned an invalid analysis instruction")
		default:
			writeError(w, http.StatusBadGateway, "query interpretation failed")
		}
		return
	}
	result, err := h.analysis.Execute(datasetID, instruction)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	groups := make([]queryGroupResult, 0, len(result.Groups))
	for _, group := range result.Groups {
		item := queryGroupResult{Key: group.Key}
		if instruction.Aggregation == "count" {
			count := group.Count
			item.Count = &count
		} else if instruction.Aggregation == "sum" {
			sum := group.Sum
			item.Sum = &sum
		}
		groups = append(groups, item)
	}
	writeJSON(w, http.StatusOK, queryResponse{
		Question:    input.Question,
		Instruction: instruction,
		Result:      queryResult{Value: result.Value, Groups: groups},
	})
}

func ensureRequestJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("unexpected data after JSON object")
	}
	return nil
}
