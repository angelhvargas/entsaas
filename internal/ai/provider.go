// Package ai provides a generic LLM provider interface and adapters for
// integrating AI capabilities into EntSaaS applications. The design is
// provider-agnostic — any OpenAI-compatible API (OpenAI, Azure, Ollama,
// vLLM, Anthropic via proxy) can be used by swapping the base URL.
package ai

import (
	"context"
	"io"
)

// Role constants for chat messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message represents a single chat message in a conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest is the input to a chat completion call.
type CompletionRequest struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// CompletionResponse is the output of a non-streaming completion.
type CompletionResponse struct {
	Content      string `json:"content"`
	Model        string `json:"model"`
	PromptTokens int   `json:"prompt_tokens"`
	OutputTokens int   `json:"output_tokens"`
	FinishReason string `json:"finish_reason"`
}

// StreamChunk represents a single chunk from a streaming completion.
type StreamChunk struct {
	Content      string `json:"content"`
	Done         bool   `json:"done"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Provider defines the interface for LLM interactions.
// All providers must implement Complete; Stream is optional but recommended.
type Provider interface {
	// Complete performs a single chat completion and returns the full response.
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)

	// Stream performs a streaming chat completion, writing chunks to the writer.
	// The callback is invoked for each chunk. Return an error to abort the stream.
	Stream(ctx context.Context, req CompletionRequest, onChunk func(StreamChunk) error) error

	// ModelName returns the configured model identifier.
	ModelName() string
}

// StreamWriter adapts a Provider.Stream call to an io.Writer for SSE output.
type StreamWriter struct {
	Writer io.Writer
}

// WriteChunk formats a StreamChunk as an SSE event and writes it.
func (sw *StreamWriter) WriteChunk(chunk StreamChunk) error {
	var data string
	if chunk.Done {
		data = "data: [DONE]\n\n"
	} else {
		data = "data: " + chunk.Content + "\n\n"
	}
	_, err := sw.Writer.Write([]byte(data))
	return err
}
