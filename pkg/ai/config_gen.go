// pkg/ai/config_gen.go
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/commatea/ComX-Bridge/pkg/protocol/dynamic"
)

// ConfigSnippet represents a generated configuration snippet.
type ConfigSnippet struct {
	Type    string `json:"type"`    // protocol type
	Content string `json:"content"` // yaml content
}

// ConfigGenerator converts natural language or analysis to dynamic protocol configuration.
type ConfigGenerator struct {
	// In a real scenario, this would hold an LLM client.
}

func NewConfigGenerator() *ConfigGenerator {
	return &ConfigGenerator{}
}

// GenerateConfig generates a configuration snippet from protocol analysis.
func (cg *ConfigGenerator) GenerateConfig(ctx context.Context, analysis *ProtocolAnalysis) (*ConfigSnippet, error) {
	if analysis == nil {
		return nil, fmt.Errorf("analysis is nil")
	}

	snippet := &ConfigSnippet{
		Type: analysis.PacketType,
	}

	var sb strings.Builder

	switch analysis.PacketType {
	case "modbus":
		sb.WriteString("# Generated Modbus Configuration\n")
		sb.WriteString("gateways:\n")
		sb.WriteString("  - name: modbus-device\n")
		sb.WriteString("    transport:\n")
		sb.WriteString("      type: tcp\n")
		sb.WriteString("      address: \"192.168.0.10:502\"\n")
		sb.WriteString("    protocol:\n")
		sb.WriteString("      type: modbus\n")
		sb.WriteString("      options:\n")
		sb.WriteString("        slave_id: 1\n")
		sb.WriteString("        timeout: 1s\n")

	case "http":
		sb.WriteString("# Generated HTTP Configuration\n")
		sb.WriteString("gateways:\n")
		sb.WriteString("  - name: http-service\n")
		sb.WriteString("    transport:\n")
		sb.WriteString("      type: http\n")
		sb.WriteString("      url: \"http://example.com\"\n")

	default:
		// Generic Binary/Dynamic Protocol
		sb.WriteString("# Generated Dynamic Protocol Configuration\n")
		sb.WriteString("gateways:\n")
		sb.WriteString("  - name: custom-device\n")
		sb.WriteString("    transport:\n")
		sb.WriteString("      type: tcp # or serial\n")
		sb.WriteString("      address: \"127.0.0.1:9000\"\n")
		sb.WriteString("    protocol:\n")
		sb.WriteString("      type: dynamic\n")
		sb.WriteString("      parser:\n")

		// Heuristic Parser Config
		if analysis.HasDelimiter && len(analysis.Delimiter) > 0 {
			sb.WriteString("        type: delimiter\n")
			sb.WriteString(fmt.Sprintf("        start: \"%x\"\n", analysis.Delimiter[0])) // simplified
		} else if analysis.HasLengthField {
			sb.WriteString("        type: length_field\n")
			// ... length field details
		} else {
			sb.WriteString("        type: fixed # Default fallback\n")
		}

		sb.WriteString("      fields:\n")
		for _, f := range analysis.Fields {
			sb.WriteString(fmt.Sprintf("        - name: %s\n", f.Name))
			sb.WriteString(fmt.Sprintf("          type: %s\n", f.Type))
			sb.WriteString(fmt.Sprintf("          offset: %d\n", f.Offset))
			sb.WriteString(fmt.Sprintf("          length: %d\n", f.Length))
		}
	}

	snippet.Content = sb.String()
	return snippet, nil
}

// GenerateConfigFromText creates a dynamic protocol config from text description.
// Example Text: "Protocol starts with STX(02) and ends with ETX(03). First byte is cmd, next 2 bytes are temp (uint16)."
func (cg *ConfigGenerator) GenerateConfigFromText(ctx context.Context, text string) (*dynamic.Config, error) {
	// Simple Heuristic Implementation for MVP
	// This simulates what an LLM would yield.

	config := &dynamic.Config{
		Name:      "generated-protocol",
		ByteOrder: "big", // Default
		Parser: dynamic.ParserConfig{
			Type:    "fixed", // Default fallback
			Options: make(map[string]string),
		},
		Fields: []dynamic.Field{},
	}

	// 1. Detect Parsers (Regex heuristics)
	if match, _ := regexp.MatchString(`(?i)STX.*02.*ETX.*03`, text); match {
		config.Parser.Type = "delimiter"
		config.Parser.Options["start"] = "0x02"
		config.Parser.Options["end"] = "0x03"
	}

	// 2. Detect Fields (Regex heuristics)
	// Looks for patterns like "cmd (byte)", "temp (uint16)"
	// This is very basic; real NLP would be better.

	currentOffset := 1 // Assume offset 0 is STX if delimiter detected? Let's just append sequentially.
	if config.Parser.Type == "delimiter" {
		currentOffset = 1 // Start after STX
	} else {
		currentOffset = 0
	}

	// Example pattern: "first byte is cmd"
	if match, _ := regexp.MatchString(`(?i)first byte is cmd`, text); match {
		config.Fields = append(config.Fields, dynamic.Field{
			Name: "cmd", Offset: currentOffset, Length: 1, Type: "byte",
		})
		currentOffset += 1
	}

	// Example pattern: "next 2 bytes are temp"
	if match, _ := regexp.MatchString(`(?i)next 2 bytes are temp`, text); match {
		config.Fields = append(config.Fields, dynamic.Field{
			Name: "temp", Offset: currentOffset, Length: 2, Type: "uint16",
		})
		currentOffset += 2
	}

	// Example pattern: "next 4 bytes are id"
	if match, _ := regexp.MatchString(`(?i)next 4 bytes are id`, text); match {
		config.Fields = append(config.Fields, dynamic.Field{
			Name: "id", Offset: currentOffset, Length: 4, Type: "uint32",
		})
		currentOffset += 4
	}

	return config, nil
}

// GenerateSpecJSON returns the JSON string representation for storage.
func (cg *ConfigGenerator) GenerateSpecJSON(ctx context.Context, text string) (string, error) {
	cfg, err := cg.GenerateConfigFromText(ctx, text)
	if err != nil {
		return "", err
	}

	bytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
