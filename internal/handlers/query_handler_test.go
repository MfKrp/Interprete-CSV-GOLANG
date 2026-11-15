package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/repository"
	"github.com/MfKrp/Interprete-CSV-GOLANG/internal/service"
)

func TestQueryEndpointRunsInterpretedInstruction(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("Ollama path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"{\"operation\":\"group_by\",\"group_column\":\"vendedor\",\"aggregation\":\"sum\",\"target_column\":\"total_venta\"}"}}`))
	}))
	defer ollama.Close()

	repo := repository.NewDatasetRepository()
	stored, err := repo.Save(repository.Dataset{Filename: "sales.csv", Dataset: service.Dataset{
		Columns: []service.Column{{Name: "vendedor", Type: "string"}, {Name: "total_venta", Type: "number"}},
		Records: [][]string{{"Ana", "10"}, {"Bruno", "7"}, {"Ana", "5"}, {"Carla", "0"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	llm := service.NewOllamaClient(ollama.URL, "mock-model", ollama.Client())
	analysisService := service.NewAnalysisService(repo)
	NewQueryHandler(repo, llm, analysisService).Register(mux)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/datasets/"+stored.ID+"/query", strings.NewReader(`{"question":"sales by seller"}`))
	request.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", response.Code, response.Body)
	}
	var body struct {
		Question    string                  `json:"question"`
		Instruction service.AnalysisRequest `json:"instruction"`
		Result      struct {
			Groups []struct {
				Key string   `json:"key"`
				Sum *float64 `json:"sum"`
			} `json:"groups"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Question != "sales by seller" || body.Instruction.Operation != "group_by" {
		t.Fatalf("response = %#v", body)
	}
	if len(body.Result.Groups) != 3 || body.Result.Groups[0].Key != "Ana" || body.Result.Groups[0].Sum == nil || *body.Result.Groups[0].Sum != 15 {
		t.Fatalf("groups = %#v", body.Result.Groups)
	}
	if body.Result.Groups[2].Key != "Carla" || body.Result.Groups[2].Sum == nil || *body.Result.Groups[2].Sum != 0 {
		t.Fatalf("zero sum was not returned: %#v", body.Result.Groups[2])
	}
}

func TestQueryEndpointHidesInvalidModelDetails(t *testing.T) {
	instructions := map[string]string{
		"unsupported":        `{"operation":"delete"}`,
		"missing column":     `{"operation":"sum","column":"absent"}`,
		"wrong type":         `{"operation":"sum","column":"vendedor"}`,
		"incomplete group":   `{"operation":"group_by","group_column":"vendedor","aggregation":"sum"}`,
		"malformed response": `not-json`,
	}
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		var prompt struct {
			Question string `json:"question"`
		}
		if err := json.Unmarshal([]byte(request.Messages[1].Content), &prompt); err != nil {
			t.Errorf("decode prompt: %v", err)
			return
		}
		content, _ := json.Marshal(instructions[prompt.Question])
		_ = json.NewEncoder(w).Encode(map[string]any{"message": map[string]string{"content": string(content)}})
	}))
	defer ollama.Close()
	repo := repository.NewDatasetRepository()
	stored, err := repo.Save(repository.Dataset{Dataset: service.Dataset{
		Columns: []service.Column{{Name: "vendedor", Type: "string"}, {Name: "ventas", Type: "number"}},
		Records: [][]string{{"Ana", "10"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewQueryHandler(repo, service.NewOllamaClient(ollama.URL, "mock", ollama.Client()), service.NewAnalysisService(repo)).Register(mux)
	for question := range instructions {
		body, _ := json.Marshal(map[string]string{"question": question})
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/datasets/"+stored.ID+"/query", strings.NewReader(string(body))))
		if response.Code != http.StatusBadGateway {
			t.Errorf("question %q status = %d, want 502", question, response.Code)
		}
		if strings.Contains(response.Body.String(), "decode instruction") || strings.Contains(response.Body.String(), "does not exist") {
			t.Errorf("response leaked internal detail: %s", response.Body)
		}
	}
}

func TestQueryEndpointMapsOllamaUnavailableAndTimeout(t *testing.T) {
	repo := repository.NewDatasetRepository()
	stored, err := repo.Save(repository.Dataset{Dataset: service.Dataset{Columns: []service.Column{{Name: "sales", Type: "number"}}}})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		server *httptest.Server
		client *http.Client
	}{
		{name: "disconnected", server: httptest.NewServer(http.NotFoundHandler())},
		{name: "timeout", server: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(50 * time.Millisecond) })), client: &http.Client{Timeout: 5 * time.Millisecond}},
	}
	for _, test := range tests {
		if test.client == nil {
			test.client = test.server.Client()
		}
		endpoint := test.server.URL
		if test.name == "disconnected" {
			test.server.Close()
		} else {
			defer test.server.Close()
		}
		mux := http.NewServeMux()
		NewQueryHandler(repo, service.NewOllamaClient(endpoint, "mock", test.client), service.NewAnalysisService(repo)).Register(mux)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/datasets/"+stored.ID+"/query", strings.NewReader(`{"question":"sum sales"}`)))
		if response.Code != http.StatusBadGateway || !strings.Contains(response.Body.String(), "unavailable or timed out") {
			t.Errorf("%s response = %d %s", test.name, response.Code, response.Body)
		}
	}
}

func TestQueryEndpointErrors(t *testing.T) {
	repo := repository.NewDatasetRepository()
	stored, err := repo.Save(repository.Dataset{Dataset: service.Dataset{Columns: []service.Column{{Name: "sales", Type: "number"}}, Records: [][]string{{"3"}}}})
	if err != nil {
		t.Fatal(err)
	}
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"message":{"content":"{\"operation\":\"uninterpretable\"}"}}`))
	}))
	defer ollama.Close()
	mux := http.NewServeMux()
	NewQueryHandler(repo, service.NewOllamaClient(ollama.URL, "mock", ollama.Client()), service.NewAnalysisService(repo)).Register(mux)

	for _, test := range []struct {
		id, body string
		want     int
	}{
		{stored.ID, `{"question":"  "}`, http.StatusBadRequest},
		{"missing", `{"question":"how many"}`, http.StatusNotFound},
		{stored.ID, `{"question":"unclear"}`, http.StatusUnprocessableEntity},
		{stored.ID, `{"question":"ok","operation":"sum"}`, http.StatusBadRequest},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/datasets/"+test.id+"/query", strings.NewReader(test.body))
		mux.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Errorf("status for %q = %d, want %d: %s", test.body, response.Code, test.want, response.Body)
		}
	}
}
