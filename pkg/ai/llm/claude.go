package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Claude API constants.
const (
	claudeDefaultBaseURL = "https://api.anthropic.com/v1"
	claudeAPIVersion     = "2023-06-01"
)

// ClaudeProvider implements the Provider interface for Anthropic Claude.
type ClaudeProvider struct {
	apiKey  string
	baseURL string
	model   string
	config  Config
	client  *http.Client
}

// NewClaudeProvider creates a new Claude provider.
func NewClaudeProvider(apiKey string, config Config) (*ClaudeProvider, error) {
	if apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = claudeDefaultBaseURL
	}

	model := config.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60
	}

	return &ClaudeProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		config:  config,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}, nil
}

// Name returns the provider name.
func (p *ClaudeProvider) Name() string {
	return string(ProviderClaude)
}

// Complete sends a completion request.
func (p *ClaudeProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	messages := []Message{
		{Role: RoleUser, Content: req.Prompt},
	}

	chatReq := &ChatRequest{
		Messages:     messages,
		SystemPrompt: req.SystemPrompt,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		Stop:         req.Stop,
	}

	chatResp, err := p.Chat(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	return &CompletionResponse{
		Text:         chatResp.Message.Content,
		FinishReason: chatResp.FinishReason,
		Usage:        chatResp.Usage,
		Model:        chatResp.Model,
	}, nil
}

// Chat sends a chat request.
func (p *ClaudeProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Build Claude request
	messages := make([]claudeMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			continue // System prompt is handled separately
		}
		messages = append(messages, claudeMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.config.MaxTokens
	}

	temperature := p.config.Temperature
	if req.Temperature != nil {
		temperature = *req.Temperature
	}

	body := claudeRequest{
		Model:       p.model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: temperature,
		StopSeq:     req.Stop,
	}

	if req.SystemPrompt != "" {
		body.System = req.SystemPrompt
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", claudeAPIVersion)

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == 429 {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("%w: %s", ErrRequestFailed, string(respBody))
	}

	// Parse response
	var claudeResp claudeResponse
	if err := json.Unmarshal(respBody, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract content
	content := ""
	for _, block := range claudeResp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &ChatResponse{
		Message: Message{
			Role:    RoleAssistant,
			Content: content,
		},
		FinishReason: claudeResp.StopReason,
		Usage: &Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		},
		Model: claudeResp.Model,
	}, nil
}

// ListModels returns available models.
func (p *ClaudeProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
		"claude-3-sonnet-20240229",
		"claude-3-haiku-20240307",
	}, nil
}

// Close releases resources.
func (p *ClaudeProvider) Close() error {
	return nil
}

// Claude API types.
type claudeRequest struct {
	Model       string          `json:"model"`
	Messages    []claudeMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	System      string          `json:"system,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	StopSeq     []string        `json:"stop_sequences,omitempty"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	Role       string `json:"role"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
