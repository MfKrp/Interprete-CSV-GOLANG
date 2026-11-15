package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/repository"
	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/service"
)

func TestDatasetUploadMetadataAndAllQueriesUseSameDataset(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode Ollama request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		var prompt struct {
			Question string `json:"question"`
		}
		if len(request.Messages) != 2 || json.Unmarshal([]byte(request.Messages[1].Content), &prompt) != nil {
			t.Errorf("invalid Ollama prompt: %#v", request.Messages)
			http.Error(w, "bad prompt", http.StatusBadRequest)
			return
		}
		instructions := map[string]any{
			"¿Cuántos vendedores hay?": map[string]any{"operation": "count_distinct", "column": "vendedor"},
			"row count":                map[string]any{"operation": "count_rows"},
			"distinct sellers":         map[string]any{"operation": "count_distinct", "column": "vendedor"},
			"distinct regions":         map[string]any{"operation": "count_distinct", "column": "region"},
			"north count":              map[string]any{"operation": "count_filtered", "column": "region", "operator": "eq", "value": "norte"},
			"sales above 25":           map[string]any{"operation": "count_filtered", "column": "total_venta", "operator": "gt", "value": 25},
			"not north":                map[string]any{"operation": "count_filtered", "column": "region", "operator": "neq", "value": "Norte"},
			"sales sum":                map[string]any{"operation": "sum", "column": "total_venta"},
			"units average":            map[string]any{"operation": "average", "column": "unidades"},
			"sales minimum":            map[string]any{"operation": "min", "column": "total_venta"},
			"sales maximum":            map[string]any{"operation": "max", "column": "total_venta"},
			"count per region":         map[string]any{"operation": "group_by", "group_column": "region", "aggregation": "count"},
			"sum per region":           map[string]any{"operation": "group_by", "group_column": "region", "aggregation": "sum", "target_column": "total_venta"},
		}
		instruction, ok := instructions[prompt.Question]
		if !ok {
			t.Errorf("unexpected question sent to Ollama: %q", prompt.Question)
			http.Error(w, "unknown test question", http.StatusBadRequest)
			return
		}
		content, err := json.Marshal(instruction)
		if err != nil {
			t.Errorf("encode mock instruction: %v", err)
			http.Error(w, "bad instruction", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"role": "assistant", "content": string(content)}})
	}))
	defer ollama.Close()

	repo := repository.NewDatasetRepository()
	mux := http.NewServeMux()
	NewCSVHandler(repo, 10*1024*1024).Register(mux)
	NewQueryHandler(repo, service.NewOllamaClient(ollama.URL, "mock-qwen", ollama.Client()), service.NewAnalysisService(repo)).Register(mux)

	csvData := "vendedor,region,unidades,total_venta\nAna,Norte,2,20\nBruno,Sur,1,5\nAna,Norte,3,30\nCarla,Centro,0,0\n"
	upload := httptest.NewRecorder()
	mux.ServeHTTP(upload, uploadRequest(t, csvData))
	if upload.Code != http.StatusCreated {
		t.Fatalf("upload status = %d: %s", upload.Code, upload.Body)
	}
	var uploaded struct {
		ID       string `json:"id"`
		Filename string `json:"filename"`
		Rows     int    `json:"row_count"`
	}
	if err := json.Unmarshal(upload.Body.Bytes(), &uploaded); err != nil {
		t.Fatal(err)
	}
	if uploaded.ID == "" || uploaded.Filename != "sample.csv" || uploaded.Rows != 4 {
		t.Fatalf("upload metadata = %#v", uploaded)
	}

	metadataResponse := httptest.NewRecorder()
	mux.ServeHTTP(metadataResponse, httptest.NewRequest(http.MethodGet, "/api/datasets/"+uploaded.ID, nil))
	if metadataResponse.Code != http.StatusOK {
		t.Fatalf("metadata status = %d: %s", metadataResponse.Code, metadataResponse.Body)
	}
	var metadata struct {
		ID      string           `json:"id"`
		Rows    int              `json:"row_count"`
		Columns []service.Column `json:"columns"`
	}
	if err := json.Unmarshal(metadataResponse.Body.Bytes(), &metadata); err != nil {
		t.Fatal(err)
	}
	if metadata.ID != uploaded.ID || metadata.Rows != 4 || len(metadata.Columns) != 4 {
		t.Fatalf("metadata for uploaded ID = %#v, uploaded = %#v", metadata, uploaded)
	}

	tests := []struct {
		question string
		value    float64
		groups   map[string]float64
		counts   map[string]int
	}{
		{question: "row count", value: 4},
		{question: "¿Cuántos vendedores hay?", value: 3},
		{question: "distinct sellers", value: 3},
		{question: "distinct regions", value: 3},
		{question: "north count", value: 2},
		{question: "sales above 25", value: 1},
		{question: "not north", value: 2},
		{question: "sales sum", value: 55},
		{question: "units average", value: 1.5},
		{question: "sales minimum", value: 0},
		{question: "sales maximum", value: 30},
		{question: "count per region", counts: map[string]int{"Centro": 1, "Norte": 2, "Sur": 1}},
		{question: "sum per region", groups: map[string]float64{"Centro": 0, "Norte": 50, "Sur": 5}},
	}
	for _, test := range tests {
		t.Run(test.question, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{"question": test.question})
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/datasets/"+uploaded.ID+"/query", strings.NewReader(string(body)))
			request.Header.Set("Content-Type", "application/json")
			mux.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("query status = %d: %s", response.Code, response.Body)
			}
			var result struct {
				Question string `json:"question"`
				Result   struct {
					Value  *float64 `json:"value"`
					Groups []struct {
						Key   string   `json:"key"`
						Count *int     `json:"count"`
						Sum   *float64 `json:"sum"`
					} `json:"groups"`
				} `json:"result"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Question != test.question {
				t.Errorf("question = %q, want %q", result.Question, test.question)
			}
			if test.value != 0 || test.question == "sales minimum" {
				if result.Result.Value == nil || *result.Result.Value != test.value {
					t.Errorf("value = %v, want %v", result.Result.Value, test.value)
				}
			}
			if test.groups != nil {
				if len(result.Result.Groups) != len(test.groups) {
					t.Fatalf("groups = %#v", result.Result.Groups)
				}
				for _, group := range result.Result.Groups {
					want, exists := test.groups[group.Key]
					if !exists || group.Sum == nil || *group.Sum != want {
						t.Errorf("sum for %q = %v, want %v", group.Key, group.Sum, want)
					}
				}
			}
			if test.counts != nil {
				if len(result.Result.Groups) != len(test.counts) {
					t.Fatalf("groups = %#v", result.Result.Groups)
				}
				for _, group := range result.Result.Groups {
					want, exists := test.counts[group.Key]
					if !exists || group.Count == nil || *group.Count != want {
						t.Errorf("count for %q = %v, want %d", group.Key, group.Count, want)
					}
				}
			}
		})
	}
}

func TestHTTPErrorCasesForUploadAndQuery(t *testing.T) {
	repo := repository.NewDatasetRepository()
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"content":"{\"operation\":\"uninterpretable\"}"}}`))
	}))
	defer ollama.Close()
	mux := http.NewServeMux()
	NewCSVHandler(repo, 16).Register(mux)
	NewQueryHandler(repo, service.NewOllamaClient(ollama.URL, "mock", ollama.Client()), service.NewAnalysisService(repo)).Register(mux)

	for _, test := range []struct {
		name string
		req  *http.Request
		want int
	}{
		{"empty CSV file", uploadRequest(t, ""), http.StatusBadRequest},
		{"malformed CSV rows", uploadRequest(t, "a,b\n1\n"), http.StatusBadRequest},
		{"oversized file", uploadRequest(t, "a,b\n12345678901234567890,1\n"), http.StatusRequestEntityTooLarge},
		{"invalid dataset ID", httptest.NewRequest(http.MethodGet, "/api/datasets/not-an-id", nil), http.StatusNotFound},
		{"query with invalid dataset ID", httptest.NewRequest(http.MethodPost, "/api/datasets/not-an-id/query", strings.NewReader(`{"question":"test"}`)), http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, test.req)
			if response.Code != test.want {
				t.Fatalf("status = %d, want %d: %s", response.Code, test.want, response.Body)
			}
		})
	}
}
