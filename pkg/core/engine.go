// Package core provides the main engine that orchestrates all components
// of the ComX-Bridge system including transports, protocols, parsers, and AI.
package core

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/ai"
	"github.com/commatea/ComX-Bridge/pkg/cluster"
	"github.com/commatea/ComX-Bridge/pkg/logger"
	"github.com/commatea/ComX-Bridge/pkg/parser"
	"github.com/commatea/ComX-Bridge/pkg/persistence"
	"github.com/commatea/ComX-Bridge/pkg/persistence/sqlite"
	"github.com/commatea/ComX-Bridge/pkg/plugin"
	"github.com/commatea/ComX-Bridge/pkg/protocol"
	"github.com/commatea/ComX-Bridge/pkg/rules"
	"github.com/commatea/ComX-Bridge/pkg/transport"
)

// Common errors.
var (
	ErrEngineNotStarted = errors.New("engine not started")
	ErrEngineStopped    = errors.New("engine stopped")
	ErrGatewayNotFound  = errors.New("gateway not found")
	ErrGatewayExists    = errors.New("gateway already exists")
	ErrInvalidConfig    = errors.New("invalid configuration")
)

// Engine is the main orchestrator of the ComX-Bridge system.
type Engine struct {
	mu sync.RWMutex

	// Registries
	transportRegistry transport.Registry
	protocolRegistry  protocol.Registry

	// Plugin Manager
	pluginManager plugin.Manager

	// Gateways
	gateways map[string]*Gateway

	// Sub-Engines
	aiEngine ai.Engine

	// Configuration
	config *Config

	// Persistence
	store persistence.Store

	// Cluster
	cluster *cluster.Manager

	// Logger
	logger *logger.Logger

	// State
	started bool
	ctx     context.Context
	cancel  context.CancelFunc

	// Event handling
	eventChan chan Event
	handlers  []EventHandler
}

// Config holds the engine configuration.
type Config struct {
	// Gateways defines the gateway configurations.
	Gateways []GatewayConfig `yaml:"gateways" json:"gateways"`

	// Plugins defines plugin directories and settings.
	Plugins PluginConfig `yaml:"plugins" json:"plugins"`

	// API configuration
	API APIConfig `yaml:"api" json:"api"`

	// AI defines AI engine settings.
	AI AIConfig `yaml:"ai" json:"ai"`

	// Logging defines logging settings.
	Logging LoggingConfig `yaml:"logging" json:"logging"`

	// Metrics defines metrics settings.
	Metrics MetricsConfig `yaml:"metrics" json:"metrics"`

	// Persistence defines data buffering settings.
	Persistence PersistenceConfig `yaml:"persistence" json:"persistence"`

	// Cluster defines high availability settings.
	Cluster ClusterConfig `yaml:"cluster" json:"cluster"`

	// Bridges defines the gateway bridging configuration.
	Bridges []BridgeConfig `yaml:"bridges" json:"bridges"`
}

// ClusterConfig holds high availability settings.
type ClusterConfig struct {
	Enabled  bool          `yaml:"enabled" json:"enabled"`
	Role     string        `yaml:"role" json:"role"` // primary, secondary
	PeerIP   string        `yaml:"peer_ip" json:"peer_ip"`
	Port     int           `yaml:"port" json:"port"`
	Interval time.Duration `yaml:"interval" json:"interval"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
}

// PersistenceConfig holds persistence settings.
type PersistenceConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path" json:"path"` // Path to SQLite DB
}

// BridgeConfig defines a bridge between two gateways.
type BridgeConfig struct {
	Source      string `yaml:"source" json:"source"`
	Destination string `yaml:"destination" json:"destination"`
	UseAI       bool   `yaml:"use_ai" json:"use_ai"`
	Prompt      string `yaml:"prompt" json:"prompt"` // If using AI, what is the instruction?
}

// APIConfig holds API settings.
type APIConfig struct {
	Enabled bool                `yaml:"enabled" json:"enabled"`
	Port    int                 `yaml:"port" json:"port" validate:"min=1,max=65535"`
	Auth    AuthConfig          `yaml:"auth" json:"auth"`
	TLS     transport.TLSConfig `yaml:"tls" json:"tls"`
}

// AuthConfig holds API authentication settings.
type AuthConfig struct {
	Enabled   bool         `yaml:"enabled" json:"enabled"`
	JWTSecret string       `yaml:"jwt_secret" json:"jwt_secret"`
	Users     []UserConfig `yaml:"users" json:"users"`
}

// UserConfig holds user credentials and role.
type UserConfig struct {
	Name string `yaml:"name" json:"name"`
	Key  string `yaml:"key" json:"key"`
	Role string `yaml:"role" json:"role"` // "admin", "viewer"
}

type GatewayConfig struct {
	// Name is the unique gateway name.
	Name string `yaml:"name" json:"name" validate:"required,alphanum"`

	// Enabled indicates if the gateway is enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Transport defines the transport configuration.
	Transport transport.Config `yaml:"transport" json:"transport" validate:"required"`

	// Protocol defines the protocol configuration.
	Protocol protocol.Config `yaml:"protocol" json:"protocol" validate:"required"`

	// Parser defines the parser configuration.
	Parser parser.Config `yaml:"parser" json:"parser"`

	// AutoReconnect enables automatic reconnection.
	AutoReconnect bool `yaml:"auto_reconnect" json:"auto_reconnect"`

	// RuleScript is the path to the Lua script for edge processing.
	RuleScript string `yaml:"rule_script" json:"rule_script"`
}

// PluginConfig holds plugin system configuration.
type PluginConfig struct {
	// Directory is the plugin directory path.
	Directory string `yaml:"directory" json:"directory"`

	// AutoLoad enables automatic plugin loading.
	AutoLoad bool `yaml:"auto_load" json:"auto_load"`

	// Sandbox enables plugin sandboxing.
	Sandbox bool `yaml:"sandbox" json:"sandbox"`
}

// AIConfig holds AI engine configuration.
type AIConfig struct {
	// Enabled enables the AI engine.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// LLM defines LLM provider settings.
	LLM LLMConfig `yaml:"llm" json:"llm"`

	// Sidecar defines AI sidecar settings.
	Sidecar SidecarConfig `yaml:"sidecar" json:"sidecar"`

	// Features defines enabled AI features.
	Features AIFeatures `yaml:"features" json:"features"`
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	// Provider is the LLM provider (openai, gemini, claude, ollama).
	Provider string `yaml:"provider" json:"provider"`

	// APIKey is the API key for the provider.
	APIKey string `yaml:"api_key" json:"api_key"`

	// Model is the model to use.
	Model string `yaml:"model" json:"model"`

	// BaseURL is the custom API endpoint (optional).
	BaseURL string `yaml:"base_url" json:"base_url"`

	// OllamaURL is the Ollama server URL.
	OllamaURL string `yaml:"ollama_url" json:"ollama_url"`

	// Temperature controls randomness (0.0 - 1.0).
	Temperature float64 `yaml:"temperature" json:"temperature"`

	// MaxTokens is the maximum tokens to generate.
	MaxTokens int `yaml:"max_tokens" json:"max_tokens"`

	// Timeout is the request timeout in seconds.
	Timeout int `yaml:"timeout" json:"timeout"`
}

// SidecarConfig holds AI sidecar configuration.
type SidecarConfig struct {
	// Address is the sidecar gRPC address.
	Address string `yaml:"address" json:"address"`

	// Timeout is the sidecar call timeout.
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// AIFeatures defines enabled AI features.
type AIFeatures struct {
	// AnomalyDetection enables anomaly detection.
	AnomalyDetection bool `yaml:"anomaly_detection" json:"anomaly_detection"`

	// ProtocolAnalysis enables protocol analysis.
	ProtocolAnalysis bool `yaml:"protocol_analysis" json:"protocol_analysis"`

	// AutoOptimize enables automatic optimization.
	AutoOptimize bool `yaml:"auto_optimize" json:"auto_optimize"`

	// CodeGeneration enables code generation.
	CodeGeneration bool `yaml:"code_generation" json:"code_generation"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	// Level is the log level (debug, info, warn, error).
	Level string `yaml:"level" json:"level"`

	// Format is the log format (json, text).
	Format string `yaml:"format" json:"format"`

	// Output is the log output (stdout, file).
	Output string `yaml:"output" json:"output"`

	// File is the log file path.
	File string `yaml:"file" json:"file"`
}

// MetricsConfig holds metrics configuration.
type MetricsConfig struct {
	// Enabled enables metrics collection.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Endpoint is the metrics HTTP endpoint.
	Endpoint string `yaml:"endpoint" json:"endpoint"`

	// Interval is the metrics collection interval.
	Interval time.Duration `yaml:"interval" json:"interval"`
}

// NewEngine creates a new engine instance.
func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		config = &Config{}
	}

	// Initialize Logger
	logConfig := logger.Config{
		Level:  config.Logging.Level,
		Format: config.Logging.Format,
		Output: config.Logging.Output,
		File:   config.Logging.File,
	}
	// Defaults
	if logConfig.Level == "" {
		logConfig.Level = "info"
	}
	if logConfig.Format == "" {
		logConfig.Format = "text"
	}

	l := logger.New(logConfig)
	logger.SetGlobal(l) // Set as global for legacy compatibility if needed

	engine := &Engine{
		gateways:  make(map[string]*Gateway),
		config:    config,
		logger:    l,
		eventChan: make(chan Event, 1000),
	}

	// Initialize Plugin System
	loader := plugin.NewFileLoader()
	pluginReg := plugin.NewRegistry()
	engine.pluginManager = plugin.NewManager(loader, pluginReg)

	// Initialize Persistence
	if config.Persistence.Enabled {
		storePath := config.Persistence.Path
		if storePath == "" {
			storePath = "./comx.db"
		}
		store, err := sqlite.NewStore(storePath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize persistence: %w", err)
		}
		engine.store = store
		l.Info("Persistence enabled", "path", storePath)
	}

	// Initialize AI Engine
	if config.AI.Enabled {
		aiConfig := ai.Config{Enabled: true}
		aiEng, err := ai.NewEngine(aiConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create AI engine: %w", err)
		}
		engine.aiEngine = aiEng
	}

	return engine, nil
}

// SetTransportRegistry sets the transport registry.
func (e *Engine) SetTransportRegistry(registry transport.Registry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.transportRegistry = registry
}

// SetProtocolRegistry sets the protocol registry.
func (e *Engine) SetProtocolRegistry(registry protocol.Registry) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.protocolRegistry = registry
}

// Start starts the engine and all configured gateways.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Panic Recovery
	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("Panic recovered in Engine.Start", "error", r, "stack", string(debug.Stack()))
		}
	}()

	if e.started {
		return nil
	}

	e.ctx, e.cancel = context.WithCancel(ctx)

	e.logger.Info("Starting Engine", "gateways", len(e.config.Gateways))

	// Start AI Engine
	if e.aiEngine != nil {
		if err := e.aiEngine.Start(e.ctx); err != nil {
			return fmt.Errorf("failed to start AI engine: %w", err)
		}
	}

	// Start event dispatcher
	go e.dispatchEvents()

	// Initialize Cluster
	shouldStartGateways := true
	if e.config.Cluster.Enabled {
		mgr, err := cluster.NewManager(cluster.Config{
			Enabled:  true,
			Role:     e.config.Cluster.Role,
			PeerIP:   e.config.Cluster.PeerIP,
			Port:     e.config.Cluster.Port,
			Interval: e.config.Cluster.Interval,
			Timeout:  e.config.Cluster.Timeout,
		})
		if err != nil {
			e.logger.Error("Failed to initialize cluster", "error", err)
			return err
		}

		e.cluster = mgr

		// Set callbacks
		e.cluster.SetCallbacks(func() {
			e.logger.Warn("Cluster state changed: Promoted to Active")
			e.startGateways()
		}, func() {
			e.logger.Warn("Cluster state changed: Demoted to Standby")
			e.stopGateways()
		})

		if err := e.cluster.Start(e.ctx); err != nil {
			return err
		}

		if !e.cluster.IsActive() {
			shouldStartGateways = false
			e.logger.Info("Engine starting in Standby mode")
		}
	}

	if shouldStartGateways {
		if err := e.startGateways(); err != nil {
			return err
		}
	}

	// Initialize Bridges
	for _, bridgeCfg := range e.config.Bridges {
		if err := e.Link(bridgeCfg.Source, bridgeCfg.Destination, bridgeCfg); err != nil {
			// Log error but continue
			e.logger.Error("Failed to create bridge",
				"source", bridgeCfg.Source,
				"destination", bridgeCfg.Destination,
				"error", err)
		} else {
			e.logger.Info("Bridge created",
				"source", bridgeCfg.Source,
				"destination", bridgeCfg.Destination,
				"use_ai", bridgeCfg.UseAI)
		}
	}

	e.started = true
	e.emit(Event{Type: EventEngineStarted, Timestamp: time.Now()})

	return nil
}

// Stop stops the engine and all gateways.
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.started {
		return nil
	}

	e.logger.Info("Stopping Engine...")

	// Stop AI Engine
	if e.aiEngine != nil {
		if err := e.aiEngine.Stop(); err != nil {
			e.logger.Warn("Error stopping AI engine", "error", err)
		}
	}

	// Stop all gateways
	for name, gw := range e.gateways {
		if err := gw.Stop(); err != nil {
			e.logger.Warn("Error stopping gateway", "name", name, "error", err)
		}
	}

	// Close persistence
	if e.store != nil {
		if err := e.store.Close(); err != nil {
			e.logger.Warn("Error closing persistence", "error", err)
		}
	}

	// Cancel context
	if e.cancel != nil {
		e.cancel()
	}

	e.started = false
	e.emit(Event{Type: EventEngineStopped, Timestamp: time.Now()})

	return nil
}

// startGateways starts all gateways.
func (e *Engine) startGateways() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, gwConfig := range e.config.Gateways {
		if !gwConfig.Enabled {
			continue
		}

		// Check if exists
		var gw *Gateway
		if existing, ok := e.gateways[gwConfig.Name]; ok {
			gw = existing
		} else {
			// Create
			newGw, err := e.createGateway(gwConfig)
			if err != nil {
				e.logger.Error("Failed to create gateway", "name", gwConfig.Name, "error", err)
				return err
			}
			e.gateways[gwConfig.Name] = newGw
			gw = newGw
		}

		if err := gw.Start(e.ctx); err != nil {
			e.logger.Error("Failed to start gateway", "name", gwConfig.Name, "error", err)
			return err
		}
		e.logger.Info("Gateway started", "name", gwConfig.Name)
	}
	return nil
}

// stopGateways stops all gateways.
func (e *Engine) stopGateways() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for name, gw := range e.gateways {
		if err := gw.Stop(); err != nil {
			e.logger.Warn("Error stopping gateway", "name", name, "error", err)
		}
	}
}

// GetGateway returns a gateway by name.
func (e *Engine) GetGateway(name string) (*Gateway, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	gw, ok := e.gateways[name]
	if !ok {
		return nil, ErrGatewayNotFound
	}
	return gw, nil
}

// AddGateway adds a new gateway at runtime.
func (e *Engine) AddGateway(config GatewayConfig) (*Gateway, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.gateways[config.Name]; exists {
		return nil, ErrGatewayExists
	}

	gw, err := e.createGateway(config)
	if err != nil {
		return nil, err
	}

	e.gateways[config.Name] = gw

	if e.started {
		if err := gw.Start(e.ctx); err != nil {
			delete(e.gateways, config.Name)
			return nil, err
		}
	}

	e.logger.Info("Gateway added", "name", config.Name)
	return gw, nil
}

// RemoveGateway removes a gateway.
func (e *Engine) RemoveGateway(name string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	gw, ok := e.gateways[name]
	if !ok {
		return ErrGatewayNotFound
	}

	if err := gw.Stop(); err != nil {
		return err
	}

	delete(e.gateways, name)
	e.logger.Info("Gateway removed", "name", name)
	return nil
}

// ListGateways returns all gateway names.
func (e *Engine) ListGateways() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	names := make([]string, 0, len(e.gateways))
	for name := range e.gateways {
		names = append(names, name)
	}
	return names
}

// Status returns the engine status.
func (e *Engine) Status() EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := EngineStatus{
		Started:  e.started,
		Gateways: make(map[string]GatewayStatus),
	}

	for name, gw := range e.gateways {
		status.Gateways[name] = gw.Status()
	}

	if e.aiEngine != nil {
		status.AI = e.aiEngine.Health()
	}

	return status
}

// Config returns the engine configuration.
func (e *Engine) Config() *Config {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// OnEvent registers an event handler.
func (e *Engine) OnEvent(handler EventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, handler)
}

// createGateway creates a gateway from config.
func (e *Engine) createGateway(config GatewayConfig) (*Gateway, error) {
	// Create transport
	var tr transport.Transport
	if e.transportRegistry != nil {
		var err error
		tr, err = e.transportRegistry.Create(config.Transport)
		if err != nil {
			return nil, err
		}
	}

	// Create protocol
	var proto protocol.Protocol
	if e.protocolRegistry != nil {
		var err error
		proto, err = e.protocolRegistry.Create(config.Protocol)
		if err != nil {
			return nil, err
		}
	}

	// Create Rule Engine
	var ruleEngine rules.Engine
	if config.RuleScript != "" {
		re, err := rules.NewLuaEngine(config.RuleScript)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule engine: %w", err)
		}
		ruleEngine = re
		e.logger.Info("Rule engine initialized", "gateway", config.Name, "script", config.RuleScript)
	}

	return &Gateway{
		name:       config.Name,
		transport:  tr,
		protocol:   proto,
		config:     config,
		store:      e.store,
		ruleEngine: ruleEngine,
	}, nil
}

// emit sends an event to handlers.
func (e *Engine) emit(event Event) {
	select {
	case e.eventChan <- event:
	default:
		// Channel full, drop event
	}
}

// dispatchEvents dispatches events to handlers.
func (e *Engine) dispatchEvents() {
	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("Panic in event dispatcher", "error", r)
		}
	}()

	for event := range e.eventChan {
		e.mu.RLock()
		handlers := make([]EventHandler, len(e.handlers))
		copy(handlers, e.handlers)
		e.mu.RUnlock()

		for _, handler := range handlers {
			// Protect individual handlers
			func() {
				defer func() {
					if r := recover(); r != nil {
						e.logger.Error("Panic in event handler", "error", r)
					}
				}()
				handler.OnEvent(event)
			}()
		}
	}
}

// EngineStatus represents the engine status.
type EngineStatus struct {
	Started  bool                     `json:"started"`
	Gateways map[string]GatewayStatus `json:"gateways"`
	AI       ai.HealthStatus          `json:"ai,omitempty"`
}

// EventType represents engine event types.
type EventType int

const (
	EventEngineStarted EventType = iota
	EventEngineStopped
	EventGatewayAdded
	EventGatewayRemoved
	EventGatewayConnected
	EventGatewayDisconnected
	EventGatewayError
	EventMessageReceived
	EventMessageSent
)

// Event represents an engine event.
type Event struct {
	Type      EventType
	Gateway   string
	Message   interface{}
	Error     error
	Timestamp time.Time
}

// EventHandler handles engine events.
type EventHandler interface {
	OnEvent(event Event)
}

// EventHandlerFunc is a function adapter for EventHandler.
type EventHandlerFunc func(event Event)

func (f EventHandlerFunc) OnEvent(event Event) {
	f(event)
}

// Link creates a bridge between two gateways.
func (e *Engine) Link(sourceName, destName string, config BridgeConfig) error {
	e.mu.RLock()
	source, ok1 := e.gateways[sourceName]
	dest, ok2 := e.gateways[destName]
	e.mu.RUnlock()

	if !ok1 || !ok2 {
		return fmt.Errorf("source or destination gateway not found")
	}

	// Create subscription
	// Note: We need to verify if Subscribe() handles channel closing gracefully on Stop()
	ch := source.Subscribe()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				e.logger.Error("Panic recovered in Bridge loop",
					"bridge", fmt.Sprintf("%s->%s", sourceName, destName),
					"error", r,
					"stack", string(debug.Stack()))
			}
		}()

		for {
			select {
			case <-e.ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}

				// Skip outbound messages to prevent loops if bidirectional
				if msg.Direction == MessageOutbound {
					continue
				}

				var dataToSend []byte
				var err error

				if config.UseAI && e.aiEngine != nil {
					// AI Transformation Logic (Simulated)
					dataToSend = msg.RawData
				} else {
					dataToSend = msg.RawData
				}

				// Send to destination
				_, err = dest.SendRaw(context.Background(), dataToSend)
				if err != nil {
					e.logger.Error("Bridge send failed",
						"source", sourceName,
						"dest", destName,
						"error", err)
				}
			}
		}
	}()

	return nil
}
