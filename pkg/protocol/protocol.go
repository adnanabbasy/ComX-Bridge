// Package protocol defines the abstract interface for communication protocols.
// It provides a unified API for encoding/decoding messages and handling
// protocol-specific logic like Modbus, custom serial protocols, etc.
package protocol

import (
	"context"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/parser"
)

// Protocol is the core interface for all communication protocols.
// A protocol defines how to encode requests and decode responses.
type Protocol interface {
	// Name returns the protocol name.
	Name() string

	// Version returns the protocol version.
	Version() string

	// Encode converts a request into bytes for transmission.
	Encode(request *Request) ([]byte, error)

	// Decode converts received bytes into a response.
	Decode(data []byte) (*Response, error)

	// Parser returns the packet parser for this protocol.
	Parser() parser.Parser

	// Validate checks if the data is valid for this protocol.
	Validate(data []byte) error

	// Configure configures the protocol with given options.
	Configure(config Config) error
}

// Config holds the configuration for a protocol.
type Config struct {
	// Type is the protocol type (modbus-rtu, modbus-tcp, raw, etc.)
	Type string `yaml:"type" json:"type"`

	// Options contains protocol-specific options.
	Options map[string]interface{} `yaml:"options" json:"options"`

	// Timeout is the default timeout for protocol operations.
	Timeout time.Duration `yaml:"timeout" json:"timeout"`
}

// Request represents a protocol request.
type Request struct {
	// ID is a unique request identifier.
	ID string `json:"id"`

	// Command is the command/function to execute.
	Command string `json:"command"`

	// Address is the target address (device, register, etc.)
	Address interface{} `json:"address,omitempty"`

	// Data is the request payload.
	Data interface{} `json:"data,omitempty"`

	// Metadata contains additional request metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Timeout is the request-specific timeout.
	Timeout time.Duration `json:"timeout,omitempty"`
}

// Response represents a protocol response.
type Response struct {
	// RequestID is the ID of the request this responds to.
	RequestID string `json:"request_id"`

	// Success indicates if the request was successful.
	Success bool `json:"success"`

	// Data is the response payload.
	Data interface{} `json:"data,omitempty"`

	// Error is the error message if not successful.
	Error string `json:"error,omitempty"`

	// ErrorCode is the protocol-specific error code.
	ErrorCode int `json:"error_code,omitempty"`

	// RawData is the raw response bytes.
	RawData []byte `json:"raw_data,omitempty"`

	// Timestamp is when the response was received.
	Timestamp time.Time `json:"timestamp"`

	// Latency is the request-response latency.
	Latency time.Duration `json:"latency"`
}

// Command represents an executable protocol command.
type Command struct {
	// Name is the command name.
	Name string `json:"name"`

	// Type is the command type (read, write, execute, etc.)
	Type CommandType `json:"type"`

	// Parameters are the command parameters.
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// CommandType represents the type of command.
type CommandType int

const (
	// CommandRead is a read command.
	CommandRead CommandType = iota
	// CommandWrite is a write command.
	CommandWrite
	// CommandExecute is an execute/action command.
	CommandExecute
	// CommandSubscribe is a subscription command.
	CommandSubscribe
)

// Result represents the result of a command execution.
type Result struct {
	// Success indicates if the command was successful.
	Success bool `json:"success"`

	// Data is the result data.
	Data interface{} `json:"data,omitempty"`

	// Error is the error if not successful.
	Error error `json:"error,omitempty"`

	// Responses are the protocol responses.
	Responses []*Response `json:"responses,omitempty"`
}

// Handler handles protocol-level operations.
type Handler interface {
	Protocol

	// Execute executes a command using the protocol.
	Execute(ctx context.Context, cmd *Command) (*Result, error)

	// HandleReceive processes received data.
	HandleReceive(ctx context.Context, data []byte) (*Response, error)
}

// Factory creates protocol instances.
type Factory interface {
	// Type returns the protocol type this factory creates.
	Type() string

	// Create creates a new protocol instance with the given config.
	Create(config Config) (Protocol, error)

	// Validate validates the configuration for this protocol type.
	Validate(config Config) error
}

// Registry manages protocol factories.
type Registry interface {
	// Register adds a factory to the registry.
	Register(factory Factory) error

	// Get retrieves a factory by type.
	Get(protocolType string) (Factory, error)

	// List returns all registered protocol types.
	List() []string

	// Create creates a protocol using the appropriate factory.
	Create(config Config) (Protocol, error)
}
