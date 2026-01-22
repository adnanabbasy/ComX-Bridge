package ai

import (
	"context"
	"unicode/utf8"
)

// HeuristicAnalyzer implements basic protocol analysis using heuristics.
type HeuristicAnalyzer struct{}

func NewHeuristicAnalyzer() *HeuristicAnalyzer {
	return &HeuristicAnalyzer{}
}

func (a *HeuristicAnalyzer) AnalyzePackets(ctx context.Context, samples [][]byte) (*ProtocolAnalysis, error) {
	if len(samples) == 0 {
		return nil, nil
	}

	analysis := &ProtocolAnalysis{
		PacketType: "unknown",
		Confidence: 0.5,
	}

	// 1. Detect ASCII vs Binary
	asciiCount := 0
	for _, s := range samples {
		if isASCII(s) {
			asciiCount++
		}
	}

	isAsciiProto := float64(asciiCount)/float64(len(samples)) > 0.8
	if isAsciiProto {
		analysis.PacketType = "ascii"
		analysis.Encoding = "utf-8"
	} else {
		analysis.PacketType = "binary"
	}

	// 2. Specific Protocol Detection
	if a.detectModbus(samples) {
		analysis.Suggestions = append(analysis.Suggestions, "Modbus Protocol Detected (RTU/TCP)")
		analysis.PacketType = "modbus"
	} else if isAsciiProto {
		if a.detectJSON(samples) {
			analysis.Suggestions = append(analysis.Suggestions, "JSON Format Detected")
			analysis.Encoding = "json"
		} else if a.detectHTTP(samples) {
			analysis.Suggestions = append(analysis.Suggestions, "HTTP Protocol Detected")
			analysis.Encoding = "http"
		}
	}

	// 3. Detect Fixed Length
	if isFixedLength(samples) && analysis.PacketType == "binary" {
		analysis.Suggestions = append(analysis.Suggestions, "Fixed length protocol detected")
		analysis.HasLengthField = false
	}

	return analysis, nil
}

func (a *HeuristicAnalyzer) InferStructure(ctx context.Context, data []byte) (*PacketStructure, error) {
	return &PacketStructure{
		TotalLength: len(data),
		Fields: []FieldInfo{
			{Name: "data", Offset: 0, Length: len(data), Type: "bytes"},
		},
	}, nil
}

func (a *HeuristicAnalyzer) DetectCRC(ctx context.Context, samples [][]byte) (*CRCAnalysis, error) {
	return nil, nil
}

func (a *HeuristicAnalyzer) LearnProtocol(ctx context.Context, samples []LabeledSample) error {
	return nil
}

// detectModbus checks for Modbus RTU or TCP patterns
func (a *HeuristicAnalyzer) detectModbus(samples [][]byte) bool {
	// Simple heuristic:
	// Modbus TCP: header is 7 bytes, 3rd & 4th byte usually 0x00 0x00 (Protocol ID)
	// Modbus RTU: check for valid CRC16 (function code 01, 02, 03, 04, 05, 06, 15, 16)
	score := 0
	for _, s := range samples {
		if len(s) < 4 {
			continue
		}

		// Modbus TCP Check
		if len(s) >= 7 && s[2] == 0x00 && s[3] == 0x00 {
			score++
			continue
		}

		// Modbus RTU Check (Function Code)
		if len(s) >= 4 {
			fc := s[1]
			if fc == 0x01 || fc == 0x02 || fc == 0x03 || fc == 0x04 || fc == 0x05 || fc == 0x06 || fc == 0x0F || fc == 0x10 {
				score++
			}
		}
	}
	return float64(score)/float64(len(samples)) > 0.5
}

// detectJSON checks if samples look like JSON
func (a *HeuristicAnalyzer) detectJSON(samples [][]byte) bool {
	score := 0
	for _, s := range samples {
		str := string(s)
		// Trim whitespace
		// Simple check for start/end
		if len(str) > 2 && ((str[0] == '{' && str[len(str)-1] == '}') || (str[0] == '[' && str[len(str)-1] == ']')) {
			score++
		}
	}
	return float64(score)/float64(len(samples)) > 0.6
}

// detectHTTP checks for HTTP methods
func (a *HeuristicAnalyzer) detectHTTP(samples [][]byte) bool {
	methods := []string{"GET ", "POST ", "PUT ", "DELETE ", "HTTP/"}
	score := 0
	for _, s := range samples {
		if len(s) < 5 {
			continue
		}
		prefix := string(s[:5])
		for _, m := range methods {
			if len(prefix) >= len(m) && prefix[:len(m)] == m {
				score++
				break
			}
		}
	}
	return float64(score)/float64(len(samples)) > 0.5
}

func isASCII(data []byte) bool {
	return utf8.Valid(data)
}

func isFixedLength(samples [][]byte) bool {
	if len(samples) == 0 {
		return false
	}
	l := len(samples[0])
	for _, s := range samples {
		if len(s) != l {
			return false
		}
	}
	return true
}
