package ai

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/commatea/ComX-Bridge/pkg/ai/llm"
)

// LLMAnalyzer uses a Large Language Model to infer protocol structure.
type LLMAnalyzer struct {
	provider llm.Provider
}

func NewLLMAnalyzer(provider llm.Provider) *LLMAnalyzer {
	return &LLMAnalyzer{
		provider: provider,
	}
}

func (a *LLMAnalyzer) AnalyzeWithLLM(ctx context.Context, samples [][]byte) (*ProtocolAnalysis, error) {
	if a.provider == nil {
		return nil, fmt.Errorf("llm provider not configured")
	}

	// 1. Prepare Prompt with Hex Dump
	prompt := a.buildPrompt(samples)

	// 2. Query LLM
	req := &llm.CompletionRequest{
		Prompt: prompt,
	}
	response, err := a.provider.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	// 3. Parse JSON Response
	analysis, err := a.parseResponse(response.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse llm response: %w", err)
	}

	return analysis, nil
}

func (a *LLMAnalyzer) buildPrompt(samples [][]byte) string {
	var sb strings.Builder
	sb.WriteString("Analyze the following hex dump samples of a network protocol and infer its structure.\n")
	sb.WriteString("Return ONLY a valid JSON object matching this structure:\n")
	sb.WriteString(`{
  "packet_type": "binary|ascii",
  "encoding": "utf-8|hex|...",
  "fields": [
    {"name": "header", "offset": 0, "length": 2, "type": "uint16", "description": "packet start marker"},
    ...
  ],
  "confidence": 0.9,
  "suggestions": ["suggestion1"]
}`)
	sb.WriteString("\n\nSamples:\n")

	for i, s := range samples {
		if i > 5 {
			break // Limit samples
		}
		sb.WriteString(fmt.Sprintf("%d: %s | %s\n", i, hex.EncodeToString(s), toASCII(s)))
	}

	return sb.String()
}

func (a *LLMAnalyzer) parseResponse(response string) (*ProtocolAnalysis, error) {
	// Simple cleanup to handle potential markdown code blocks
	jsonStr := strings.TrimSpace(response)
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimPrefix(jsonStr, "```")
	jsonStr = strings.TrimSuffix(jsonStr, "```")

	var analysis ProtocolAnalysis
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, err
	}
	return &analysis, nil
}

func toASCII(data []byte) string {
	res := make([]byte, len(data))
	for i, b := range data {
		if b >= 32 && b <= 126 {
			res[i] = b
		} else {
			res[i] = '.'
		}
	}
	return string(res)
}
