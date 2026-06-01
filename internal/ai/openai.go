package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs.
// Works with: OpenAI, Azure OpenAI, Ollama, vLLM, LiteLLM, and any server
// that speaks the OpenAI chat completions protocol.
type OpenAIProvider struct {
	apiKey      string
	baseURL     string
	model       string
	maxTokens   int
	temperature float64
	client      *http.Client
}

// NewOpenAIProvider creates a provider configured from the given Config.
func NewOpenAIProvider(cfg Config) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:      cfg.APIKey,
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
		client: &http.Client{
			Timeout: 120 * time.Second, // LLM calls can be slow
		},
	}
}

func (p *OpenAIProvider) ModelName() string { return p.model }

// ── OpenAI API types ─────────────────────────────────────────────────────────

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// ── Complete ─────────────────────────────────────────────────────────────────

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.maxTokens
	}
	temp := req.Temperature
	if temp == 0 {
		temp = p.temperature
	}

	oaiReq := openAIRequest{
		Model:       model,
		Messages:    toOpenAIMessages(req.Messages),
		MaxTokens:   maxTokens,
		Temperature: temp,
		Stream:      false,
	}

	body, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var oaiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return &CompletionResponse{
		Content:      oaiResp.Choices[0].Message.Content,
		Model:        oaiResp.Model,
		PromptTokens: oaiResp.Usage.PromptTokens,
		OutputTokens: oaiResp.Usage.CompletionTokens,
		FinishReason: oaiResp.Choices[0].FinishReason,
	}, nil
}

// ── Stream ───────────────────────────────────────────────────────────────────

func (p *OpenAIProvider) Stream(ctx context.Context, req CompletionRequest, onChunk func(StreamChunk) error) error {
	model := req.Model
	if model == "" {
		model = p.model
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.maxTokens
	}
	temp := req.Temperature
	if temp == 0 {
		temp = p.temperature
	}

	oaiReq := openAIRequest{
		Model:       model,
		Messages:    toOpenAIMessages(req.Messages),
		MaxTokens:   maxTokens,
		Temperature: temp,
		Stream:      true,
	}

	body, err := json.Marshal(oaiReq)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments.
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE data field.
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		// [DONE] signals end of stream.
		if data == "[DONE]" {
			return onChunk(StreamChunk{Done: true})
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue // skip malformed chunks
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		sc := StreamChunk{
			Content: chunk.Choices[0].Delta.Content,
		}
		if chunk.Choices[0].FinishReason != nil {
			sc.FinishReason = *chunk.Choices[0].FinishReason
			sc.Done = true
		}

		if err := onChunk(sc); err != nil {
			return err
		}
	}

	return scanner.Err()
}

func toOpenAIMessages(msgs []Message) []openAIMessage {
	out := make([]openAIMessage, len(msgs))
	for i, m := range msgs {
		out[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}
	return out
}
