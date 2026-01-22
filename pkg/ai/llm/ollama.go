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

// Ollama API constants.
const (
	ollamaDefaultURL = "http://localhost:11434"
)

// OllamaProvider implements the Provider interface for Ollama (local LLM).
type OllamaProvider struct {
	baseURL string
	model   string
	config  Config
	client  *http.Client
}

// NewOllamaProvider creates a new Ollama provider.
func NewOllamaProvider(config Config) (*OllamaProvider, error) {
	baseURL := config.OllamaURL
	if baseURL == "" {
		baseURL = ollamaDefaultURL
	}

	model := config.Model
	if model == "" {
		model = "llama2"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 120 // Longer timeout for local models
	}

	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		config:  config,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}, nil
}

// Name returns the provider name.
func (p *OllamaProvider) Name() string {
	return string(ProviderOllama)
}

// Complete sends a completion request.
func (p *OllamaProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	prompt := req.Prompt
	if req.SystemPrompt != "" {
		prompt = req.SystemPrompt + "\n\n" + prompt
	}

	body := ollamaGenerateRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
		Options: ollamaOptions{
			Temperature: p.config.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	if req.Temperature != nil {
		body.Options.Temperature = *req.Temperature
	}

	if len(req.Stop) > 0 {
		body.Options.Stop = req.Stop
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("%w: %s", ErrRequestFailed, string(respBody))
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	finishReason := "stop"
	if !ollamaResp.Done {
		finishReason = "length"
	}

	return &CompletionResponse{
		Text:         ollamaResp.Response,
		FinishReason: finishReason,
		Usage: &Usage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
		Model: ollamaResp.Model,
	}, nil
}

// Chat sends a chat request.
func (p *OllamaProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	messages := make([]ollamaChatMessage, 0, len(req.Messages)+1)

	if req.SystemPrompt != "" {
		messages = append(messages, ollamaChatMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	for _, msg := range req.Messages {
		messages = append(messages, ollamaChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	body := ollamaChatRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   false,
		Options: ollamaOptions{
			Temperature: p.config.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	if req.Temperature != nil {
		body.Options.Temperature = *req.Temperature
	}

	if len(req.Stop) > 0 {
		body.Options.Stop = req.Stop
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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
		return nil, fmt.Errorf("%w: %s", ErrRequestFailed, string(respBody))
	}

	var ollamaResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	finishReason := "stop"
	if !ollamaResp.Done {
		finishReason = "length"
	}

	return &ChatResponse{
		Message: Message{
			Role:    ollamaResp.Message.Role,
			Content: ollamaResp.Message.Content,
		},
		FinishReason: finishReason,
		Usage: &Usage{
			PromptTokens:     ollamaResp.PromptEvalCount,
			CompletionTokens: ollamaResp.EvalCount,
			TotalTokens:      ollamaResp.PromptEvalCount + ollamaResp.EvalCount,
		},
		Model: ollamaResp.Model,
	}, nil
}

// ListModels returns available models from Ollama.
func (p *OllamaProvider) ListModels(ctx context.Context) ([]string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list models")
	}

	var tagsResp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	models := make([]string, len(tagsResp.Models))
	for i, m := range tagsResp.Models {
		models[i] = m.Name
	}

	return models, nil
}

// Close releases resources.
func (p *OllamaProvider) Close() error {
	return nil
}

// Ollama API types.
type ollamaGenerateRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Stream  bool          `json:"stream"`
	Options ollamaOptions `json:"options,omitempty"`
}

type ollamaOptions struct {
	Temperature float64  `json:"temperature,omitempty"`
	NumPredict  int      `json:"num_predict,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

type ollamaGenerateResponse struct {
	Model           string `json:"model"`
	Response        string `json:"response"`
	Done            bool   `json:"done"`
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []ollamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
	Options  ollamaOptions       `json:"options,omitempty"`
}

type ollamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Model           string            `json:"model"`
	Message         ollamaChatMessage `json:"message"`
	Done            bool              `json:"done"`
	PromptEvalCount int               `json:"prompt_eval_count"`
	EvalCount       int               `json:"eval_count"`
}
