package ai

import (
	"context"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/ai/llm"
)

// StandardEngine is the default AI engine implementation.
type StandardEngine struct {
	config Config

	// Components
	analyzer  ProtocolAnalyzer
	anomaly   AnomalyDetector
	codegen   CodeGenerator
	configGen *ConfigGenerator // New component
	nlp       NLProcessor

	// LLM Provider
	llmProvider llm.Provider

	// Auto Optimizer
	optimizer *AutoOptimizer

	// Digital Twin
	digitalTwin *DigitalTwin

	started bool
	ctx     context.Context
	cancel  context.CancelFunc
}

// Config holds AI engine configuration.
type Config struct {
	Enabled bool

	// LLM holds LLM provider configuration.
	LLM llm.Config

	// Optimizer holds auto optimizer configuration.
	Optimizer OptimizerConfig

	// Twin holds digital twin configuration.
	Twin TwinConfig
}

// NewEngine creates a new AI engine.
func NewEngine(config Config) (Engine, error) {
	engine := &StandardEngine{
		config:      config,
		analyzer:    NewHeuristicAnalyzer(),
		anomaly:     NewStatisticalDetector(),
		codegen:     NewTemplateGenerator(),
		configGen:   NewConfigGenerator(),
		nlp:         NewKeywordProcessor(),
		optimizer:   NewAutoOptimizer(config.Optimizer),
		digitalTwin: NewDigitalTwin(config.Twin),
	}

	// Initialize LLM provider if configured
	if config.LLM.Provider != "" && config.LLM.APIKey != "" {
		provider, err := llm.NewProvider(config.LLM)
		if err != nil {
			// Log warning but don't fail - LLM is optional
			// Fall back to heuristic methods
		} else {
			engine.llmProvider = provider
		}
	}

	return engine, nil
}

func (e *StandardEngine) Start(ctx context.Context) error {
	e.ctx, e.cancel = context.WithCancel(ctx)
	e.started = true

	// Start Auto Optimizer if enabled
	if e.config.Optimizer.Enabled {
		if err := e.optimizer.Start(ctx); err != nil {
			return err
		}
	}

	// Start Digital Twin if enabled
	if e.config.Twin.Enabled {
		if err := e.digitalTwin.Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (e *StandardEngine) Stop() error {
	// Stop optimizer
	if e.optimizer != nil {
		e.optimizer.Stop()
	}

	// Stop digital twin
	if e.digitalTwin != nil {
		e.digitalTwin.Stop()
	}

	if e.cancel != nil {
		e.cancel()
	}
	e.started = false
	return nil
}

func (e *StandardEngine) Health() HealthStatus {
	status := "healthy"
	if !e.started {
		status = "stopped"
	}

	modules := map[string]string{
		"analyzer": "active",
	}

	if e.config.Optimizer.Enabled {
		modules["optimizer"] = "active"
	}
	if e.config.Twin.Enabled {
		modules["digital_twin"] = "active"
	}
	if e.llmProvider != nil {
		modules["llm"] = e.llmProvider.Name()
	}

	return HealthStatus{
		Status:    status,
		LastCheck: time.Now(),
		Modules:   modules,
	}
}

// Delegate ProtocolAnalyzer methods
func (e *StandardEngine) AnalyzePackets(ctx context.Context, samples [][]byte) (*ProtocolAnalysis, error) {
	return e.analyzer.AnalyzePackets(ctx, samples)
}

func (e *StandardEngine) InferStructure(ctx context.Context, data []byte) (*PacketStructure, error) {
	return e.analyzer.InferStructure(ctx, data)
}

func (e *StandardEngine) DetectCRC(ctx context.Context, samples [][]byte) (*CRCAnalysis, error) {
	return e.analyzer.DetectCRC(ctx, samples)
}

func (e *StandardEngine) LearnProtocol(ctx context.Context, samples []LabeledSample) error {
	return e.analyzer.LearnProtocol(ctx, samples)
}

// Delegate AnomalyDetector methods
func (e *StandardEngine) DetectAnomaly(ctx context.Context, stream <-chan []byte) (<-chan Anomaly, error) {
	return e.anomaly.DetectAnomaly(ctx, stream)
}
func (e *StandardEngine) AnalyzePacket(ctx context.Context, data []byte) (*AnomalyResult, error) {
	return e.anomaly.AnalyzePacket(ctx, data)
}
func (e *StandardEngine) LearnNormalPattern(ctx context.Context, samples [][]byte) error {
	return e.anomaly.LearnNormalPattern(ctx, samples)
}
func (e *StandardEngine) SetThreshold(threshold float64) {
	e.anomaly.SetThreshold(threshold)
}

func (e *StandardEngine) GenerateParser(ctx context.Context, structure *PacketStructure, lang string) (*GeneratedCode, error) {
	return e.codegen.GenerateParser(ctx, structure, lang)
}
func (e *StandardEngine) GenerateProtocol(ctx context.Context, analysis *ProtocolAnalysis, lang string) (*GeneratedCode, error) {
	return e.codegen.GenerateProtocol(ctx, analysis, lang)
}
func (e *StandardEngine) GeneratePlugin(ctx context.Context, spec *PluginSpec, lang string) (*GeneratedCode, error) {
	return e.codegen.GeneratePlugin(ctx, spec, lang)
}
func (e *StandardEngine) SupportedLanguages() []string { return e.codegen.SupportedLanguages() }

func (e *StandardEngine) ParseCommand(ctx context.Context, text string) (*ParsedCommand, error) {
	return e.nlp.ParseCommand(ctx, text)
}
func (e *StandardEngine) ExplainPacket(ctx context.Context, data []byte, protocol string) (string, error) {
	return e.nlp.ExplainPacket(ctx, data, protocol)
}
func (e *StandardEngine) GenerateQuery(ctx context.Context, text string) (*Query, error) {
	return e.nlp.GenerateQuery(ctx, text)
}
func (e *StandardEngine) Suggest(ctx context.Context, partial string) ([]string, error) {
	return e.nlp.Suggest(ctx, partial)
}

// GenerateConfigFromText delegates to ConfigGenerator
func (e *StandardEngine) GenerateConfigFromText(ctx context.Context, text string) (string, error) {
	return e.configGen.GenerateSpecJSON(ctx, text)
}

// LLMProvider returns the LLM provider if available.
func (e *StandardEngine) LLMProvider() llm.Provider {
	return e.llmProvider
}

// HasLLM returns true if an LLM provider is configured.
func (e *StandardEngine) HasLLM() bool {
	return e.llmProvider != nil
}

// LLMComplete sends a completion request to the LLM provider.
func (e *StandardEngine) LLMComplete(ctx context.Context, prompt, systemPrompt string) (string, error) {
	if e.llmProvider == nil {
		return "", llm.ErrProviderNotConfigured
	}

	resp, err := e.llmProvider.Complete(ctx, &llm.CompletionRequest{
		Prompt:       prompt,
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		return "", err
	}

	return resp.Text, nil
}

// LLMChat sends a chat request to the LLM provider.
func (e *StandardEngine) LLMChat(ctx context.Context, messages []llm.Message, systemPrompt string) (string, error) {
	if e.llmProvider == nil {
		return "", llm.ErrProviderNotConfigured
	}

	resp, err := e.llmProvider.Chat(ctx, &llm.ChatRequest{
		Messages:     messages,
		SystemPrompt: systemPrompt,
	})
	if err != nil {
		return "", err
	}

	return resp.Message.Content, nil
}

// Optimizer returns the auto optimizer.
func (e *StandardEngine) Optimizer() *AutoOptimizer {
	return e.optimizer
}

// DigitalTwin returns the digital twin.
func (e *StandardEngine) DigitalTwin() *DigitalTwin {
	return e.digitalTwin
}

// GetOptimizerMetrics returns current optimizer metrics.
func (e *StandardEngine) GetOptimizerMetrics() MetricsSummary {
	if e.optimizer == nil {
		return MetricsSummary{}
	}
	return e.optimizer.GetMetrics()
}

// GetDigitalTwinStats returns digital twin statistics.
func (e *StandardEngine) GetDigitalTwinStats() TwinStats {
	if e.digitalTwin == nil {
		return TwinStats{}
	}
	return e.digitalTwin.GetStats()
}
