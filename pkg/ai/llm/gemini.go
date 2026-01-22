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

// Gemini API constants.
const (
	geminiDefaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"
)

// GeminiProvider implements the Provider interface for Google Gemini.
type GeminiProvider struct {
	apiKey  string
	baseURL string
	model   string
	config  Config
	client  *http.Client
}

// NewGeminiProvider creates a new Gemini provider.
func NewGeminiProvider(apiKey string, config Config) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, ErrInvalidAPIKey
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = geminiDefaultBaseURL
	}

	model := config.Model
	if model == "" {
		model = "gemini-1.5-flash"
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = 60
	}

	return &GeminiProvider{
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
func (p *GeminiProvider) Name() string {
	return string(ProviderGemini)
}

// Complete sends a completion request.
func (p *GeminiProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	messages := []Message{}
	if req.SystemPrompt != "" {
		messages = append(messages, Message{Role: RoleUser, Content: req.SystemPrompt + "\n\n" + req.Prompt})
	} else {
		messages = append(messages, Message{Role: RoleUser, Content: req.Prompt})
	}

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
func (p *GeminiProvider) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Build Gemini request
	contents := make([]geminiContent, 0, len(req.Messages))

	for _, msg := range req.Messages {
		role := "user"
		if msg.Role == RoleAssistant {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role: role,
			Parts: []geminiPart{
				{Text: msg.Content},
			},
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

	body := geminiRequest{
		Contents: contents,
		GenerationConfig: geminiGenerationConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     temperature,
			StopSequences:   req.Stop,
		},
	}

	// Add system instruction if provided
	if req.SystemPrompt != "" {
		body.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

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
	var geminiResp geminiResponse
	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := geminiResp.Candidates[0]
	content := ""
	if len(candidate.Content.Parts) > 0 {
		content = candidate.Content.Parts[0].Text
	}

	var usage *Usage
	if geminiResp.UsageMetadata != nil {
		usage = &Usage{
			PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
			CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
			TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
		}
	}

	return &ChatResponse{
		Message: Message{
			Role:    RoleAssistant,
			Content: content,
		},
		FinishReason: candidate.FinishReason,
		Usage:        usage,
		Model:        p.model,
	}, nil
}

// ListModels returns available models.
func (p *GeminiProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-2.0-flash-exp",
		"gemini-1.0-pro",
	}, nil
}

// Close releases resources.
func (p *GeminiProvider) Close() error {
	return nil
}

// Gemini API types.
type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	Temperature     float64  `json:"temperature,omitempty"`
	TopP            float64  `json:"topP,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiResponse struct {
	Candidates    []geminiCandidate    `json:"candidates"`
	UsageMetadata *geminiUsageMetadata `json:"usageMetadata,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
	Index        int           `json:"index"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}
