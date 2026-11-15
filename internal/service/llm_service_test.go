package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var llmTestColumns = []Column{{Name: "vendedor", Type: "string"}, {Name: "total_venta", Type: "number"}}

func ollamaResponse(content string) string {
	encoded, _ := json.Marshal(content)
	return `{"message":{"role":"assistant","content":` + string(encoded) + `}}`
}

func TestOllamaInterpretSuccessAndStructuredRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" || r.Method != http.MethodPost {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Model    string          `json:"model"`
			Stream   bool            `json:"stream"`
			Format   json.RawMessage `json:"format"`
			Messages []ollamaMessage `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.Model != "test-model" || body.Stream || len(body.Format) == 0 {
			t.Errorf("request settings: %#v", body)
		}
		if !strings.Contains(string(body.Messages[1].Content), "total_venta") || !strings.Contains(string(body.Messages[1].Content), "¿Cuál fue el total?") {
			t.Errorf("missing question/schema: %s", body.Messages[1].Content)
		}
		if !strings.Contains(body.Messages[0].Content, "count_distinct") || !strings.Contains(body.Messages[0].Content, "count_filtered") {
			t.Errorf("system prompt omits new operations: %s", body.Messages[0].Content)
		}
		if !strings.Contains(body.Messages[0].Content, `"operation":"count_distinct","column":"vendedor"`) {
			t.Errorf("system prompt omits distinct-count example: %s", body.Messages[0].Content)
		}
		var prompt struct {
			Operations []string `json:"allowed_operations"`
			Operators  []string `json:"allowed_filter_operators"`
		}
		if err := json.Unmarshal([]byte(body.Messages[1].Content), &prompt); err != nil {
			t.Fatal(err)
		}
		if !containsString(prompt.Operations, "count_distinct") || !containsString(prompt.Operations, "count_filtered") || !containsString(prompt.Operators, "contains") {
			t.Errorf("prompt operation/operator lists = %#v / %#v", prompt.Operations, prompt.Operators)
		}
		if !strings.Contains(body.Messages[0].Content, "filtros combinados") || !strings.Contains(body.Messages[0].Content, "una condición única") {
			t.Errorf("system prompt must explain filter limits: %s", body.Messages[0].Content)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(ollamaResponse(`{"operation":"sum","column":"total_venta"}`)))
	}))
	defer server.Close()
	client := NewOllamaClient(server.URL, "test-model", server.Client())
	got, err := client.Interpret(context.Background(), "¿Cuál fue el total?", llmTestColumns)
	if err != nil {
		t.Fatal(err)
	}
	if got.Operation != "sum" || got.Column != "total_venta" {
		t.Fatalf("instruction = %#v", got)
	}
}

func TestOllamaUnavailableAndTimeout(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	endpoint := closed.URL
	closed.Close()
	_, err := NewOllamaClient(endpoint, "test", closed.Client()).Interpret(context.Background(), "count rows", llmTestColumns)
	if !errors.Is(err, ErrOllamaUnavailable) {
		t.Fatalf("closed server error = %v", err)
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(50 * time.Millisecond) }))
	defer slow.Close()
	_, err = NewOllamaClient(slow.URL, "test", &http.Client{Timeout: 5 * time.Millisecond}).Interpret(context.Background(), "count rows", llmTestColumns)
	if !errors.Is(err, ErrOllamaUnavailable) {
		t.Fatalf("timeout error = %v", err)
	}
}

func TestOllamaRejectsInvalidResponsesAndInstructions(t *testing.T) {
	tests := []struct {
		name, content string
		want          error
	}{
		{"malformed JSON", `{operation:sum}`, ErrInvalidOllamaResponse},
		{"unknown field", `{"operation":"sum","column":"total_venta","sql":"drop"}`, ErrInvalidOllamaResponse},
		{"unsupported operation", `{"operation":"delete"}`, ErrInvalidInstruction},
		{"missing column", `{"operation":"sum","column":"absent"}`, ErrInvalidInstruction},
		{"nonnumeric column", `{"operation":"average","column":"vendedor"}`, ErrInvalidInstruction},
		{"missing parameter", `{"operation":"group_by","group_column":"vendedor","aggregation":"sum"}`, ErrInvalidInstruction},
		{"ambiguous", `{"operation":"uninterpretable"}`, ErrUninterpretable},
		{"distinct", `{"operation":"count_distinct","column":"vendedor"}`, nil},
		{"filtered text", `{"operation":"count_filtered","column":"vendedor","operator":"eq","value":"Ana"}`, nil},
		{"filtered number", `{"operation":"count_filtered","column":"total_venta","operator":"gte","value":12.5}`, nil},
		{"missing filter value", `{"operation":"count_filtered","column":"vendedor","operator":"eq"}`, ErrInvalidInstruction},
		{"invalid filter operator", `{"operation":"count_filtered","column":"vendedor","operator":"regex","value":"Ana"}`, ErrInvalidInstruction},
		{"missing filter column", `{"operation":"count_filtered","column":"absent","operator":"eq","value":"Ana"}`, ErrInvalidInstruction},
		{"text numeric comparison", `{"operation":"count_filtered","column":"vendedor","operator":"gt","value":4}`, ErrInvalidInstruction},
		{"numeric string value is normalized", `{"operation":"count_filtered","column":"total_venta","operator":"gt","value":"12"}`, nil},
		{"invalid numeric string value", `{"operation":"count_filtered","column":"total_venta","operator":"gt","value":"doce"}`, ErrInvalidInstruction},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(ollamaResponse(test.content))) }))
			defer server.Close()
			got, err := NewOllamaClient(server.URL, "test", server.Client()).Interpret(context.Background(), "question", llmTestColumns)
			if test.want == nil {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if test.name == "numeric string value is normalized" && got.Value != float64(12) {
					t.Fatalf("normalized numeric filter value = %#v, want float64(12)", got.Value)
				}
				return
			}
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func TestOllamaRejectsEmptyQuestion(t *testing.T) {
	_, err := NewOllamaClient("http://localhost:11434", "test", nil).Interpret(context.Background(), "  ", nil)
	if !errors.Is(err, ErrEmptyQuestion) {
		t.Fatalf("error = %v", err)
	}
}

func TestOllamaRejectsTrailingDataInChatResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(ollamaResponse(`{"operation":"count_rows"}`) + " trailing"))
	}))
	defer server.Close()
	_, err := NewOllamaClient(server.URL, "test", server.Client()).Interpret(context.Background(), "count rows", llmTestColumns)
	if !errors.Is(err, ErrInvalidOllamaResponse) {
		t.Fatalf("error = %v, want invalid Ollama response", err)
	}
}
