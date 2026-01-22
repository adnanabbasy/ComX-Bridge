// Package ai provides AI-powered features for protocol analysis,
// anomaly detection, code generation, and natural language processing.
package ai

import (
	"context"
	"time"
)

// Engine is the main AI engine interface.
type Engine interface {
	// Protocol Analysis
	ProtocolAnalyzer

	// Anomaly Detection
	AnomalyDetector

	// Code Generation
	CodeGenerator

	// Natural Language Processing
	NLProcessor

	// Lifecycle
	Start(ctx context.Context) error
	Stop() error
	Health() HealthStatus
}

// HealthStatus represents AI engine health.
type HealthStatus struct {
	Status    string            `json:"status"` // healthy, degraded, unhealthy
	Modules   map[string]string `json:"modules"`
	LastCheck time.Time         `json:"last_check"`
}

// ProtocolAnalyzer analyzes packet patterns and infers protocol structure.
type ProtocolAnalyzer interface {
	// AnalyzePackets analyzes a set of packet samples.
	AnalyzePackets(ctx context.Context, samples [][]byte) (*ProtocolAnalysis, error)

	// InferStructure infers the structure of a single packet.
	InferStructure(ctx context.Context, data []byte) (*PacketStructure, error)

	// DetectCRC detects and identifies CRC/checksum algorithms.
	DetectCRC(ctx context.Context, samples [][]byte) (*CRCAnalysis, error)

	// LearnProtocol learns from labeled packet samples.
	LearnProtocol(ctx context.Context, samples []LabeledSample) error
}

// ProtocolAnalysis contains the result of protocol analysis.
type ProtocolAnalysis struct {
	// PacketType is the detected packet type (binary, ascii, mixed).
	PacketType string `json:"packet_type"`

	// Encoding is the detected encoding.
	Encoding string `json:"encoding"`

	// HasDelimiter indicates if packets have delimiters.
	HasDelimiter bool `json:"has_delimiter"`

	// Delimiter is the detected delimiter bytes.
	Delimiter []byte `json:"delimiter,omitempty"`

	// HasLengthField indicates if packets have a length field.
	HasLengthField bool `json:"has_length_field"`

	// LengthFieldInfo contains length field details.
	LengthFieldInfo *LengthFieldInfo `json:"length_field_info,omitempty"`

	// HasCRC indicates if packets have CRC/checksum.
	HasCRC bool `json:"has_crc"`

	// CRCInfo contains CRC details.
	CRCInfo *CRCAnalysis `json:"crc_info,omitempty"`

	// Fields are the detected packet fields.
	Fields []FieldAnalysis `json:"fields"`

	// Confidence is the analysis confidence (0-1).
	Confidence float64 `json:"confidence"`

	// Suggestions are improvement suggestions.
	Suggestions []string `json:"suggestions,omitempty"`
}

// PacketStructure represents the inferred structure of a packet.
type PacketStructure struct {
	// TotalLength is the total packet length.
	TotalLength int `json:"total_length"`

	// Fields are the packet fields.
	Fields []FieldInfo `json:"fields"`

	// Checksum is the checksum information.
	Checksum *ChecksumInfo `json:"checksum,omitempty"`
}

// FieldInfo describes a packet field.
type FieldInfo struct {
	// Name is the inferred field name.
	Name string `json:"name"`

	// Offset is the byte offset.
	Offset int `json:"offset"`

	// Length is the field length in bytes.
	Length int `json:"length"`

	// Type is the inferred type (uint8, uint16, string, etc.).
	Type string `json:"type"`

	// Endian is the byte order (big, little).
	Endian string `json:"endian,omitempty"`

	// Value is the parsed value.
	Value interface{} `json:"value,omitempty"`

	// Description is a human-readable description.
	Description string `json:"description,omitempty"`
}

// LengthFieldInfo describes a length field.
type LengthFieldInfo struct {
	// Offset is the byte offset of the length field.
	Offset int `json:"offset"`

	// Size is the length field size in bytes.
	Size int `json:"size"`

	// Endian is the byte order.
	Endian string `json:"endian"`

	// Adjust is the adjustment to get actual data length.
	Adjust int `json:"adjust"`
}

// CRCAnalysis contains CRC detection results.
type CRCAnalysis struct {
	// Type is the CRC type (crc16, crc32, checksum, etc.).
	Type string `json:"type"`

	// Algorithm is the specific algorithm (CRC-16-MODBUS, etc.).
	Algorithm string `json:"algorithm"`

	// Offset is the CRC byte offset (negative from end).
	Offset int `json:"offset"`

	// Size is the CRC size in bytes.
	Size int `json:"size"`

	// Polynomial is the CRC polynomial (if detected).
	Polynomial uint64 `json:"polynomial,omitempty"`

	// Confidence is the detection confidence.
	Confidence float64 `json:"confidence"`
}

// FieldAnalysis contains field analysis results.
type FieldAnalysis struct {
	FieldInfo

	// Variance is the value variance across samples.
	Variance float64 `json:"variance"`

	// IsConstant indicates if the field is constant.
	IsConstant bool `json:"is_constant"`

	// PossibleValues are observed values.
	PossibleValues []interface{} `json:"possible_values,omitempty"`
}

// ChecksumInfo describes checksum information.
type ChecksumInfo struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	Valid  bool   `json:"valid"`
}

// LabeledSample is a packet sample with labels.
type LabeledSample struct {
	Data   []byte            `json:"data"`
	Labels map[string]string `json:"labels"`
}

// AnomalyDetector detects communication anomalies.
type AnomalyDetector interface {
	// DetectAnomaly starts real-time anomaly detection.
	DetectAnomaly(ctx context.Context, stream <-chan []byte) (<-chan Anomaly, error)

	// AnalyzePacket analyzes a single packet for anomalies.
	AnalyzePacket(ctx context.Context, data []byte) (*AnomalyResult, error)

	// LearnNormalPattern learns normal patterns from samples.
	LearnNormalPattern(ctx context.Context, samples [][]byte) error

	// SetThreshold sets the anomaly detection threshold.
	SetThreshold(threshold float64)
}

// Anomaly represents a detected anomaly.
type Anomaly struct {
	// ID is the anomaly identifier.
	ID string `json:"id"`

	// Type is the anomaly type.
	Type AnomalyType `json:"type"`

	// Severity is the anomaly severity.
	Severity Severity `json:"severity"`

	// Description is a human-readable description.
	Description string `json:"description"`

	// Data is the anomalous data.
	Data []byte `json:"data"`

	// Score is the anomaly score (0-1).
	Score float64 `json:"score"`

	// Suggestion is the recommended action.
	Suggestion string `json:"suggestion"`

	// Timestamp is when the anomaly was detected.
	Timestamp time.Time `json:"timestamp"`
}

// AnomalyType represents the type of anomaly.
type AnomalyType int

const (
	AnomalyUnknown AnomalyType = iota
	AnomalyLatency         // Unusual response time
	AnomalyError           // Error pattern
	AnomalyFormat          // Invalid packet format
	AnomalySequence        // Sequence violation
	AnomalyValue           // Unusual value
	AnomalyFrequency       // Unusual frequency
	AnomalySecurity        // Security-related
)

func (t AnomalyType) String() string {
	switch t {
	case AnomalyLatency:
		return "latency"
	case AnomalyError:
		return "error"
	case AnomalyFormat:
		return "format"
	case AnomalySequence:
		return "sequence"
	case AnomalyValue:
		return "value"
	case AnomalyFrequency:
		return "frequency"
	case AnomalySecurity:
		return "security"
	default:
		return "unknown"
	}
}

// Severity represents anomaly severity.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "info"
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// AnomalyResult contains the result of anomaly analysis.
type AnomalyResult struct {
	IsAnomaly  bool      `json:"is_anomaly"`
	Score      float64   `json:"score"`
	Anomalies  []Anomaly `json:"anomalies,omitempty"`
}

// CodeGenerator generates code from protocol specifications.
type CodeGenerator interface {
	// GenerateParser generates parser code from structure.
	GenerateParser(ctx context.Context, structure *PacketStructure, lang string) (*GeneratedCode, error)

	// GenerateProtocol generates protocol implementation.
	GenerateProtocol(ctx context.Context, analysis *ProtocolAnalysis, lang string) (*GeneratedCode, error)

	// GeneratePlugin generates a plugin from specification.
	GeneratePlugin(ctx context.Context, spec *PluginSpec, lang string) (*GeneratedCode, error)

	// SupportedLanguages returns supported target languages.
	SupportedLanguages() []string
}

// GeneratedCode contains generated code and metadata.
type GeneratedCode struct {
	// Language is the target language.
	Language string `json:"language"`

	// Files are the generated files.
	Files []GeneratedFile `json:"files"`

	// Dependencies are required dependencies.
	Dependencies []string `json:"dependencies,omitempty"`

	// Instructions are usage instructions.
	Instructions string `json:"instructions,omitempty"`
}

// GeneratedFile represents a generated file.
type GeneratedFile struct {
	// Path is the file path.
	Path string `json:"path"`

	// Content is the file content.
	Content string `json:"content"`

	// Type is the file type (source, test, config).
	Type string `json:"type"`
}

// PluginSpec specifies a plugin to generate.
type PluginSpec struct {
	// Name is the plugin name.
	Name string `json:"name"`

	// Type is the plugin type.
	Type string `json:"type"`

	// Protocol is the protocol analysis.
	Protocol *ProtocolAnalysis `json:"protocol,omitempty"`

	// Description is the plugin description.
	Description string `json:"description"`
}

// NLProcessor processes natural language commands.
type NLProcessor interface {
	// ParseCommand parses a natural language command.
	ParseCommand(ctx context.Context, text string) (*ParsedCommand, error)

	// ExplainPacket explains a packet in natural language.
	ExplainPacket(ctx context.Context, data []byte, protocol string) (string, error)

	// GenerateQuery generates a query from natural language.
	GenerateQuery(ctx context.Context, text string) (*Query, error)

	// Suggest provides command suggestions.
	Suggest(ctx context.Context, partial string) ([]string, error)
}

// ParsedCommand represents a parsed natural language command.
type ParsedCommand struct {
	// Intent is the detected intent.
	Intent string `json:"intent"`

	// Action is the action to perform.
	Action string `json:"action"`

	// Target is the target of the action.
	Target string `json:"target"`

	// Parameters are extracted parameters.
	Parameters map[string]interface{} `json:"parameters"`

	// Confidence is the parsing confidence.
	Confidence float64 `json:"confidence"`

	// Original is the original text.
	Original string `json:"original"`
}

// Query represents a generated query.
type Query struct {
	// Type is the query type.
	Type string `json:"type"`

	// Target is the query target.
	Target string `json:"target"`

	// Filters are query filters.
	Filters map[string]interface{} `json:"filters"`

	// Original is the original natural language.
	Original string `json:"original"`
}
