// Package plugin provides the plugin system for extending ComX-Bridge
// with custom transports, protocols, and AI modules.
package plugin

import (
	"context"
	"errors"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/protocol"
	"github.com/commatea/ComX-Bridge/pkg/transport"
)

// Common errors.
var (
	ErrPluginNotFound     = errors.New("plugin not found")
	ErrPluginExists       = errors.New("plugin already registered")
	ErrPluginLoadFailed   = errors.New("failed to load plugin")
	ErrPluginInvalid      = errors.New("invalid plugin")
	ErrPluginTypeMismatch = errors.New("plugin type mismatch")
)

// Type represents the plugin type.
type Type string

const (
	// TypeTransport is a transport plugin.
	TypeTransport Type = "transport"

	// TypeProtocol is a protocol plugin.
	TypeProtocol Type = "protocol"

	// TypeParser is a parser plugin.
	TypeParser Type = "parser"

	// TypeAI is an AI module plugin.
	TypeAI Type = "ai"

	// TypeMiddleware is a middleware plugin.
	TypeMiddleware Type = "middleware"
)

// HealthStatus represents plugin health.
type HealthStatus int

const (
	HealthUnknown HealthStatus = iota
	HealthHealthy
	HealthDegraded
	HealthUnhealthy
)

func (h HealthStatus) String() string {
	switch h {
	case HealthHealthy:
		return "healthy"
	case HealthDegraded:
		return "degraded"
	case HealthUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// Info contains plugin metadata.
type Info struct {
	// Name is the plugin name.
	Name string `json:"name"`

	// Version is the plugin version.
	Version string `json:"version"`

	// Type is the plugin type.
	Type Type `json:"type"`

	// Description is the plugin description.
	Description string `json:"description"`

	// Author is the plugin author.
	Author string `json:"author"`

	// License is the plugin license.
	License string `json:"license"`

	// Homepage is the plugin homepage URL.
	Homepage string `json:"homepage"`

	// Requires lists plugin dependencies.
	Requires []string `json:"requires,omitempty"`

	// Tags are searchable tags.
	Tags []string `json:"tags,omitempty"`
}

// Config holds plugin configuration.
type Config struct {
	// Name is the plugin name.
	Name string `yaml:"name" json:"name"`

	// Enabled indicates if the plugin is enabled.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Options are plugin-specific options.
	Options map[string]interface{} `yaml:"options" json:"options"`
}

// Plugin is the base interface for all plugins.
type Plugin interface {
	// Info returns plugin metadata.
	Info() Info

	// Init initializes the plugin.
	Init(ctx context.Context, config Config) error

	// Start starts the plugin.
	Start() error

	// Stop stops the plugin.
	Stop() error

	// Health returns the plugin health status.
	Health() HealthStatus
}

// TransportPlugin extends Plugin with transport creation capabilities.
type TransportPlugin interface {
	Plugin

	// CreateTransport creates a new transport instance.
	CreateTransport(config transport.Config) (transport.Transport, error)

	// SupportedTypes returns the transport types this plugin supports.
	SupportedTypes() []string
}

// ProtocolPlugin extends Plugin with protocol creation capabilities.
type ProtocolPlugin interface {
	Plugin

	// CreateProtocol creates a new protocol instance.
	CreateProtocol(config protocol.Config) (protocol.Protocol, error)

	// SupportedTypes returns the protocol types this plugin supports.
	SupportedTypes() []string
}

// Lifecycle provides plugin lifecycle hooks.
type Lifecycle interface {
	// OnLoad is called when the plugin is loaded.
	OnLoad() error

	// OnUnload is called when the plugin is unloaded.
	OnUnload() error

	// OnConfigChange is called when configuration changes.
	OnConfigChange(config Config) error
}

// Manager manages plugin lifecycle and registration.
type Manager interface {
	// Load loads a plugin from path.
	Load(path string) (Plugin, error)

	// Unload unloads a plugin.
	Unload(name string) error

	// Get retrieves a plugin by name.
	Get(name string) (Plugin, error)

	// List returns all loaded plugins.
	List() []Info

	// ListByType returns plugins of a specific type.
	ListByType(pluginType Type) []Info

	// Start starts all plugins.
	Start() error

	// Stop stops all plugins.
	Stop() error

	// Health returns the health of all plugins.
	Health() map[string]HealthStatus
}

// Loader loads plugins from disk.
type Loader interface {
	// LoadDir loads all plugins from a directory.
	LoadDir(dir string) ([]Plugin, error)

	// LoadFile loads a plugin from a file.
	LoadFile(path string) (Plugin, error)

	// SupportedExtensions returns supported file extensions.
	SupportedExtensions() []string
}

// ScriptLoader loads script-based plugins (Lua, JavaScript).
type ScriptLoader interface {
	Loader

	// LoadScript loads a plugin from script content.
	LoadScript(name string, content []byte) (Plugin, error)

	// Language returns the scripting language.
	Language() string
}

// Sandbox provides plugin isolation.
type Sandbox interface {
	// Run runs a function in the sandbox.
	Run(fn func() error) error

	// SetTimeout sets the execution timeout.
	SetTimeout(timeout time.Duration)

	// SetMemoryLimit sets the memory limit.
	SetMemoryLimit(bytes int64)

	// SetCPULimit sets the CPU limit.
	SetCPULimit(percent float64)
}
