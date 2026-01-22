package main

import (
	"context"
	"fmt"
	"log"

	"github.com/commatea/ComX-Bridge/pkg/ai"
)

func main() {
	// Create Analyzer
	analyzer := ai.NewHeuristicAnalyzer()
	ctx := context.Background()

	// 1. Test Modbus TCP
	// Transaction ID (2), Protocol ID (2, always 00 00), Length (2), Unit ID (1), FC (1), Data...
	modbusSample := [][]byte{
		{0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0x00, 0x00, 0x00, 0x01},
		{0x00, 0x02, 0x00, 0x00, 0x00, 0x06, 0x01, 0x03, 0x00, 0x00, 0x00, 0x0A},
	}

	fmt.Println("--- Testing Modbus TCP ---")
	result, err := analyzer.AnalyzePackets(ctx, modbusSample)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	printResult(result)

	// 2. Test HTTP
	httpSample := [][]byte{
		[]byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n"),
		[]byte("POST /api/v1/data HTTP/1.1\r\nContent-Type: application/json\r\n\r\n{}"),
	}

	fmt.Println("\n--- Testing HTTP ---")
	result, err = analyzer.AnalyzePackets(ctx, httpSample)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	printResult(result)

	// 3. Test JSON
	jsonSample := [][]byte{
		[]byte(`{"status": "ok", "value": 123}`),
		[]byte(`[1, 2, 3, 4, 5]`),
	}

	fmt.Println("\n--- Testing JSON ---")
	result, err = analyzer.AnalyzePackets(ctx, jsonSample)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	printResult(result)

	// 4. Test Generation (New Feature)
	fmt.Println("\n--- Testing Generation Features ---")

	// Config Gen
	configGen := ai.NewConfigGenerator()
	if result.PacketType == "ascii" { // JSON is detected as ascii/json in heuristic
		// Mock a binary analysis for generation test
		binaryAnalysis := &ai.ProtocolAnalysis{
			PacketType: "binary",
			Fields: []ai.FieldAnalysis{
				{FieldInfo: ai.FieldInfo{Name: "header", Offset: 0, Length: 2, Type: "uint16"}},
				{FieldInfo: ai.FieldInfo{Name: "payload", Offset: 2, Length: 8, Type: "string"}},
			},
		}
		cfg, err := configGen.GenerateConfig(ctx, binaryAnalysis)
		if err != nil {
			log.Printf("Config Gen Error: %v", err)
		} else {
			fmt.Printf("[Generated Config]\n%s\n", cfg.Content)
		}

		// Code Gen (Lua)
		codeGen := ai.NewTemplateGenerator()
		structure := &ai.PacketStructure{
			Fields: []ai.FieldInfo{
				{Name: "header", Offset: 0, Length: 2, Type: "uint16"},
				{Name: "payload", Offset: 2, Length: 8, Type: "string"},
			},
		}
		code, err := codeGen.GenerateParser(ctx, structure, "lua")
		if err != nil {
			log.Printf("Code Gen Error: %v", err)
		} else {
			fmt.Printf("[Generated Lua Parser]\n%s\n", code.Files[0].Content)
		}
	}
}

func printResult(r *ai.ProtocolAnalysis) {
	fmt.Printf("Packet Type: %s\n", r.PacketType)
	if r.Encoding != "" {
		fmt.Printf("Encoding: %s\n", r.Encoding)
	}
	if len(r.Suggestions) > 0 {
		fmt.Printf("Suggestions: %v\n", r.Suggestions)
	}
	fmt.Printf("Confidence: %.2f\n", r.Confidence)
}
