package ai

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds AI provider configuration loaded from environment variables.
type Config struct {
	Enabled     bool    // AI_ENABLED
	ProviderName string // AI_PROVIDER (openai, anthropic, ollama)
	APIKey      string  // AI_API_KEY
	BaseURL     string  // AI_API_BASE_URL
	Model       string  // AI_MODEL
	MaxTokens   int     // AI_MAX_TOKENS
	Temperature float64 // AI_TEMPERATURE
}

// LoadConfig reads AI configuration from environment variables.
func LoadConfig() Config {
	temp, _ := strconv.ParseFloat(os.Getenv("AI_TEMPERATURE"), 64)
	if temp == 0 {
		temp = 0.7
	}
	maxTokens, _ := strconv.Atoi(os.Getenv("AI_MAX_TOKENS"))
	if maxTokens == 0 {
		maxTokens = 4096
	}

	return Config{
		Enabled:      os.Getenv("AI_ENABLED") == "true",
		ProviderName: envOrDefault("AI_PROVIDER", "openai"),
		APIKey:       os.Getenv("AI_API_KEY"),
		BaseURL:      envOrDefault("AI_API_BASE_URL", "https://api.openai.com/v1"),
		Model:        envOrDefault("AI_MODEL", "gpt-4o"),
		MaxTokens:    maxTokens,
		Temperature:  temp,
	}
}

// NewProvider creates a Provider from the given configuration.
// Currently supports OpenAI-compatible APIs. Additional providers can be
// added by extending this factory function.
func NewProvider(cfg Config) (Provider, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("AI is disabled (AI_ENABLED != true)")
	}

	switch cfg.ProviderName {
	case "openai", "azure", "ollama", "vllm":
		return NewOpenAIProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", cfg.ProviderName)
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
