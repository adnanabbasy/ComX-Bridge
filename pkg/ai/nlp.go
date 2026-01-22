// Package ai provides AI powered features.
package ai

import (
	"context"
	"fmt"
	"strings"
)

// KeywordProcessor implements NLProcessor using keyword matching.
type KeywordProcessor struct{}

// NewKeywordProcessor creates a new keyword processor.
func NewKeywordProcessor() *KeywordProcessor {
	return &KeywordProcessor{}
}

func (p *KeywordProcessor) ParseCommand(ctx context.Context, text string) (*ParsedCommand, error) {
	text = strings.ToLower(text)

	cmd := &ParsedCommand{
		Original:   text,
		Parameters: make(map[string]interface{}),
		Confidence: 0.5, // Base confidence
	}

	if strings.Contains(text, "connect") {
		cmd.Intent = "connect"
		cmd.Action = "connect"
		// Extract target? e.g. "connect to serial"
		parts := strings.Fields(text)
		for i, word := range parts {
			if word == "to" && i+1 < len(parts) {
				cmd.Target = parts[i+1]
			}
		}
	} else if strings.Contains(text, "send") {
		cmd.Intent = "send"
		cmd.Action = "send"
		// "send hello to serial"
	} else if strings.Contains(text, "status") {
		cmd.Intent = "status"
		cmd.Action = "get_status"
	} else {
		return nil, fmt.Errorf("unknown command")
	}

	return cmd, nil
}

func (p *KeywordProcessor) ExplainPacket(ctx context.Context, data []byte, protocol string) (string, error) {
	return fmt.Sprintf("Packet of %d bytes (Protocol: %s)", len(data), protocol), nil
}

func (p *KeywordProcessor) GenerateQuery(ctx context.Context, text string) (*Query, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *KeywordProcessor) Suggest(ctx context.Context, partial string) ([]string, error) {
	suggestions := []string{"connect", "send", "status", "disconnect"}
	var matches []string
	for _, s := range suggestions {
		if strings.HasPrefix(s, strings.ToLower(partial)) {
			matches = append(matches, s)
		}
	}
	return matches, nil
}
