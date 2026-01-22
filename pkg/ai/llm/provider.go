// Package llm provides LLM (Large Language Model) provider implementations.
// Supports OpenAI, Google Gemini, Anthropic Claude, and Ollama.
package llm

import (
	"context"
	"errors"
	"os"
	"strings"
)

// Common errors.
var (
	ErrProviderNotConfigured = errors.New("LLM provider not configured")
	ErrInvalidAPIKey         = errors.New("invalid or missing API key")
	ErrModelNotSupported     = errors.New("model not supported")
	ErrRequestFailed         = errors.New("LLM request failed")
	ErrRateLimited           = errors.New("rate limited")
)

// ProviderType represents the LLM provider type.
type ProviderType string

const (
	ProviderOpenAI ProviderType = "openai"
	ProviderGemini ProviderType = "gemini"
	ProviderClaude ProviderType = "claude"
	ProviderOllama ProviderType = "ollama"
)

// Provider is the interface for LLM providers.
type Provider interface {
	// Name returns the provider name.
	Name() string

	// Complete sends a completion request and returns the response.
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Chat sends a chat request with message history.
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ListModels returns available models for this provider.
	ListModels(ctx context.Context) ([]string, error)

	// Close releases any resources.
	Close() error
}

// Config holds LLM provider configuration.
type Config struct {
	// Provider is the LLM provider type.
	Provider ProviderType `yaml:"provider" json:"provider"`

	// APIKey is the API key for the provider.
	APIKey string `yaml:"api_key" json:"api_key"`

	// Model is the model to use.
	Model string `yaml:"model" json:"model"`

	// BaseURL is the base URL for the API (optional, for custom endpoints).
	BaseURL string `yaml:"base_url" json:"base_url"`

	// OllamaURL is the Ollama server URL (for Ollama provider).
	OllamaURL string `yaml:"ollama_url" json:"ollama_url"`

	// Temperature controls randomness (0.0 - 1.0).
	Temperature float64 `yaml:"temperature" json:"temperature"`

	// MaxTokens is the maximum tokens to generate.
	MaxTokens int `yaml:"max_tokens" json:"max_tokens"`

	// Timeout is the request timeout in seconds.
	Timeout int `yaml:"timeout" json:"timeout"`
}

// DefaultConfig returns a default LLM configuration.
func DefaultConfig() Config {
	return Config{
		Provider:    ProviderOpenAI,
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   2048,
		Timeout:     60,
	}
}

// CompletionRequest represents a completion request.
type CompletionRequest struct {
	// Prompt is the input prompt.
	Prompt string `json:"prompt"`

	// SystemPrompt is the system instruction (optional).
	SystemPrompt string `json:"system_prompt,omitempty"`

	// MaxTokens overrides the default max tokens.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature overrides the default temperature.
	Temperature *float64 `json:"temperature,omitempty"`

	// Stop sequences to stop generation.
	Stop []string `json:"stop,omitempty"`
}

// CompletionResponse represents a completion response.
type CompletionResponse struct {
	// Text is the generated text.
	Text string `json:"text"`

	// FinishReason is why the generation stopped.
	FinishReason string `json:"finish_reason"`

	// Usage contains token usage information.
	Usage *Usage `json:"usage,omitempty"`

	// Model is the model used.
	Model string `json:"model"`
}

// ChatRequest represents a chat request.
type ChatRequest struct {
	// Messages is the conversation history.
	Messages []Message `json:"messages"`

	// SystemPrompt is the system instruction.
	SystemPrompt string `json:"system_prompt,omitempty"`

	// MaxTokens overrides the default max tokens.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature overrides the default temperature.
	Temperature *float64 `json:"temperature,omitempty"`

	// Stop sequences to stop generation.
	Stop []string `json:"stop,omitempty"`
}

// ChatResponse represents a chat response.
type ChatResponse struct {
	// Message is the assistant's response.
	Message Message `json:"message"`

	// FinishReason is why the generation stopped.
	FinishReason string `json:"finish_reason"`

	// Usage contains token usage information.
	Usage *Usage `json:"usage,omitempty"`

	// Model is the model used.
	Model string `json:"model"`
}

// Message represents a chat message.
type Message struct {
	// Role is the message role (system, user, assistant).
	Role string `json:"role"`

	// Content is the message content.
	Content string `json:"content"`
}

// Usage contains token usage information.
type Usage struct {
	// PromptTokens is the number of prompt tokens.
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of generated tokens.
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the total tokens used.
	TotalTokens int `json:"total_tokens"`
}

// NewProvider creates a new LLM provider based on configuration.
func NewProvider(config Config) (Provider, error) {
	// Resolve API key from environment variable if needed
	apiKey := resolveEnvVar(config.APIKey)

	switch config.Provider {
	case ProviderOpenAI:
		return NewOpenAIProvider(apiKey, config)
	case ProviderGemini:
		return NewGeminiProvider(apiKey, config)
	case ProviderClaude:
		return NewClaudeProvider(apiKey, config)
	case ProviderOllama:
		return NewOllamaProvider(config)
	default:
		return nil, ErrProviderNotConfigured
	}
}

// resolveEnvVar resolves environment variable references like ${VAR_NAME}.
func resolveEnvVar(value string) string {
	if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
		envName := strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")
		if envValue := os.Getenv(envName); envValue != "" {
			return envValue
		}
	}
	return value
}

// Role constants for chat messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)
