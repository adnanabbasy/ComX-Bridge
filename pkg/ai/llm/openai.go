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

// OpenAI API constants.
const (
	openAIDefaultBaseURL = "https://api.openai.com/v1"
)

// OpenAIProvider implements the Provider interface for OpenAI.
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	config  Config
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(apiKey string, config Config) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = openAIDefaultBaseURL
	}

	model := config.Model
	if model == "" {
		model = "gpt-4"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60
	}

	return &OpenAIProvider{
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
func (p *OpenAIProvider) Name() string {
	return string(ProviderOpenAI)
}

// Complete sends a completion request.
func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// Convert to chat format for modern OpenAI API
	messages := []Message{}
	if req.SystemPrompt != "" {
		messages = append(messages, Message{Role: RoleSystem, Content: req.SystemPrompt})
	}
	messages = append(messages, Message{Role: RoleUser, Content: req.Prompt})

	chatReq := &ChatRequest{
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stop:        req.Stop,
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
func (p *OpenAIProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Build request body
	messages := make([]map[string]string, 0, len(req.Messages)+1)

	if req.SystemPrompt != "" {
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
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

	body := map[string]interface{}{
		"model":       p.model,
		"messages":    messages,
		"max_tokens":  maxTokens,
		"temperature": temperature,
	}

	if len(req.Stop) > 0 {
		body["stop"] = req.Stop
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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
	var openAIResp openAIChatResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	choice := openAIResp.Choices[0]

	return &ChatResponse{
		Message: Message{
			Role:    choice.Message.Role,
			Content: choice.Message.Content,
		},
		FinishReason: choice.FinishReason,
		Usage: &Usage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
		Model: openAIResp.Model,
	}, nil
}

// ListModels returns available models.
func (p *OpenAIProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{
		"gpt-4",
		"gpt-4-turbo",
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-3.5-turbo",
	}, nil
}

// Close releases resources.
func (p *OpenAIProvider) Close() error {
	return nil
}

// OpenAI API response types.
type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}
