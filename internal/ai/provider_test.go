package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	os.Unsetenv("AI_ENABLED")
	os.Unsetenv("AI_PROVIDER")
	os.Unsetenv("AI_API_KEY")
	os.Unsetenv("AI_API_BASE_URL")
	os.Unsetenv("AI_MODEL")
	os.Unsetenv("AI_MAX_TOKENS")
	os.Unsetenv("AI_TEMPERATURE")

	cfg := LoadConfig()
	if cfg.Enabled {
		t.Error("expected AI_ENABLED to default to false")
	}
	if cfg.ProviderName != "openai" {
		t.Errorf("expected default provider openai, got %q", cfg.ProviderName)
	}
	if cfg.Model != "gpt-4o" {
		t.Errorf("expected default model gpt-4o, got %q", cfg.Model)
	}
	if cfg.MaxTokens != 4096 {
		t.Errorf("expected default max tokens 4096, got %d", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.7 {
		t.Errorf("expected default temperature 0.7, got %v", cfg.Temperature)
	}
}

func TestLoadConfig_Custom(t *testing.T) {
	os.Setenv("AI_ENABLED", "true")
	os.Setenv("AI_PROVIDER", "ollama")
	os.Setenv("AI_API_KEY", "ollama-key")
	os.Setenv("AI_API_BASE_URL", "http://localhost:11434/v1")
	os.Setenv("AI_MODEL", "llama3")
	os.Setenv("AI_MAX_TOKENS", "1024")
	os.Setenv("AI_TEMPERATURE", "0.2")

	defer func() {
		os.Unsetenv("AI_ENABLED")
		os.Unsetenv("AI_PROVIDER")
		os.Unsetenv("AI_API_KEY")
		os.Unsetenv("AI_API_BASE_URL")
		os.Unsetenv("AI_MODEL")
		os.Unsetenv("AI_MAX_TOKENS")
		os.Unsetenv("AI_TEMPERATURE")
	}()

	cfg := LoadConfig()
	if !cfg.Enabled {
		t.Error("expected AI_ENABLED to be true")
	}
	if cfg.ProviderName != "ollama" {
		t.Errorf("expected provider ollama, got %q", cfg.ProviderName)
	}
	if cfg.APIKey != "ollama-key" {
		t.Errorf("expected api key ollama-key, got %q", cfg.APIKey)
	}
	if cfg.BaseURL != "http://localhost:11434/v1" {
		t.Errorf("expected base URL, got %q", cfg.BaseURL)
	}
	if cfg.Model != "llama3" {
		t.Errorf("expected model llama3, got %q", cfg.Model)
	}
	if cfg.MaxTokens != 1024 {
		t.Errorf("expected max tokens 1024, got %d", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.2 {
		t.Errorf("expected temperature 0.2, got %v", cfg.Temperature)
	}
}

func TestNewProvider_Disabled(t *testing.T) {
	cfg := Config{Enabled: false}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("expected error when building provider with AI disabled")
	}
}

func TestNewProvider_Unsupported(t *testing.T) {
	cfg := Config{Enabled: true, ProviderName: "unsupported"}
	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("expected error for unsupported provider")
	}
}

func TestStreamWriter_WriteChunk(t *testing.T) {
	var buf bytes.Buffer
	sw := StreamWriter{Writer: &buf}

	// 1. Content chunk
	err := sw.WriteChunk(StreamChunk{Content: "hello"})
	if err != nil {
		t.Fatalf("failed to write chunk: %v", err)
	}
	if buf.String() != "data: hello\n\n" {
		t.Errorf("expected SSE formatted string, got %q", buf.String())
	}

	buf.Reset()

	// 2. Done chunk
	err = sw.WriteChunk(StreamChunk{Done: true})
	if err != nil {
		t.Fatalf("failed to write done chunk: %v", err)
	}
	if buf.String() != "data: [DONE]\n\n" {
		t.Errorf("expected [DONE] format, got %q", buf.String())
	}
}

func TestOpenAIProvider_Complete(t *testing.T) {
	// Set up mock completions server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify endpoint path
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected path /chat/completions, got %q", r.URL.Path)
		}
		// Verify headers
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected JSON content-type header, got %q", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("Authorization") != "Bearer secret-key" {
			t.Errorf("expected authorization header, got %q", r.Header.Get("Authorization"))
		}

		// Decode request
		var req openAIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Validate mapped model
		if req.Model != "gpt-4-custom" {
			t.Errorf("expected model gpt-4-custom, got %q", req.Model)
		}

		// Send mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-123",
			"object": "chat.completion",
			"created": 1677652288,
			"model": "gpt-4-custom",
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 15,
				"total_tokens": 25
			},
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "Hi! I am EntSaaS assistant."
					},
					"finish_reason": "stop",
					"index": 0
				}
			]
		}`))
	}))
	defer mockServer.Close()

	cfg := Config{
		Enabled:      true,
		ProviderName: "openai",
		APIKey:       "secret-key",
		BaseURL:      mockServer.URL,
		Model:        "gpt-4-custom",
		MaxTokens:    2048,
		Temperature:  0.8,
	}

	prov, err := NewProvider(cfg)
	if err != nil {
		t.Fatalf("failed to create provider: %v", err)
	}

	resp, err := prov.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hello!"},
		},
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Content != "Hi! I am EntSaaS assistant." {
		t.Errorf("expected response content, got %q", resp.Content)
	}
	if resp.Model != "gpt-4-custom" {
		t.Errorf("expected model, got %q", resp.Model)
	}
	if resp.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", resp.PromptTokens)
	}
	if resp.OutputTokens != 15 {
		t.Errorf("expected 15 output tokens, got %d", resp.OutputTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected stop finish reason, got %q", resp.FinishReason)
	}
}

func TestOpenAIProvider_Stream(t *testing.T) {
	// Set up mock stream completions server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		// Send chunks
		_, _ = fmt.Fprint(w, "data: {\"choices\": [{\"delta\": {\"content\": \"Part 1\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\": [{\"delta\": {\"content\": \" Part 2\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\": [{\"delta\": {\"content\": \"\"}, \"finish_reason\": \"stop\"}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer mockServer.Close()

	p := NewOpenAIProvider(Config{
		Enabled: true,
		BaseURL: mockServer.URL,
		Model:   "gpt-4",
	})

	var outputs []string
	var wasCompleted bool

	err := p.Stream(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "stream me"}},
	}, func(c StreamChunk) error {
		if c.Done {
			wasCompleted = true
		} else {
			outputs = append(outputs, c.Content)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	combined := strings.Join(outputs, "")
	if combined != "Part 1 Part 2" {
		t.Errorf("expected compiled stream output to be 'Part 1 Part 2', got %q", combined)
	}

	if !wasCompleted {
		t.Error("expected Stream completion callback to trigger Done = true")
	}
}
