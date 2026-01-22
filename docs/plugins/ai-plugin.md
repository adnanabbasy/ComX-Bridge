# AI Plugin Development Guide

AI plugins extend the intelligence capabilities of ComX-Bridge, enabling custom analysis, anomaly detection, and automation.

## Interface Definition

```go
package plugin

import (
	"context"
	"github.com/commatea/ComX-Bridge/pkg/ai"
)

// AIPlugin is the interface for AI modules.
type AIPlugin interface {
	Plugin
	GetAnalyzer() (ai.ProtocolAnalyzer, error)
	GetDetector() (ai.AnomalyDetector, error)
}
```

## Implementation Example

```go
package main

import (
	"context"
	"github.com/commatea/ComX-Bridge/pkg/ai"
	"github.com/commatea/ComX-Bridge/pkg/plugin"
)

type MyAIPlugin struct {
	analyzer *MyAnalyzer
	detector *MyDetector
}

func (p *MyAIPlugin) GetAnalyzer() (ai.ProtocolAnalyzer, error) {
	return p.analyzer, nil
}

func (p *MyAIPlugin) GetDetector() (ai.AnomalyDetector, error) {
	return p.detector, nil
}

// Implement ai.ProtocolAnalyzer
type MyAnalyzer struct{}

func (a *MyAnalyzer) AnalyzePackets(ctx context.Context, samples [][]byte) (*ai.ProtocolAnalysis, error) {
	// Custom analysis logic
	return &ai.ProtocolAnalysis{
		PacketType: "custom",
		Confidence: 0.95,
	}, nil
}

// ... Implement other methods
```

## Considerations
- AI plugins often require heavy computation. Run intensive tasks in background goroutines.
- Use `context` to handle cancellation and timeouts properly.
