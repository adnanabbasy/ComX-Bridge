// pkg/protocol/dynamic/protocol.go
package dynamic

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/parser"
	"github.com/commatea/ComX-Bridge/pkg/protocol"
)

// Config defines the dynamic protocol structure.
// This struct will be populated from YAML/JSON.
type Config struct {
	Name      string       `json:"name" yaml:"name"`
	Parser    ParserConfig `json:"parser" yaml:"parser"`
	Fields    []Field      `json:"fields" yaml:"fields"`
	ByteOrder string       `json:"byte_order" yaml:"byte_order"` // "big" or "little"
}

type ParserConfig struct {
	Type    string            `json:"type" yaml:"type"` // "delimiter", "fixed", "length"
	Options map[string]string `json:"options" yaml:"options"`
}

type Field struct {
	Name   string `json:"name" yaml:"name"`
	Offset int    `json:"offset" yaml:"offset"`
	Length int    `json:"length" yaml:"length"`
	Type   string `json:"type" yaml:"type"` // "byte", "uint16", "uint32", "string"
}

// DynamicProtocol implements protocol.Protocol interface driven by runtime config.
type DynamicProtocol struct {
	config    Config
	parser    parser.Parser
	byteOrder binary.ByteOrder
}

func New(config protocol.Config) (*DynamicProtocol, error) {
	// 1. Parse dynamic options from the generic config
	// The user provides the full dynamic spec in config.Options["spec"]
	// or we could load it from a file. For now, let's assume 'spec' contains the JSON string.

	var dynConfig Config
	if spec, ok := config.Options["spec"].(string); ok {
		if err := json.Unmarshal([]byte(spec), &dynConfig); err != nil {
			return nil, fmt.Errorf("failed to parse dynamic spec: %w", err)
		}
	} else {
		// Fallback: try to map map[string]interface{} to Config if possible
		// Or just return error for now as we expect a simpler usage flow
		return nil, errors.New("missing 'spec' in options for dynamic protocol")
	}

	p := &DynamicProtocol{
		config: dynConfig,
	}

	// 2. Setup ByteOrder
	if dynConfig.ByteOrder == "little" {
		p.byteOrder = binary.LittleEndian
	} else {
		p.byteOrder = binary.BigEndian
	}

	// 3. Setup Parser
	if err := p.setupParser(); err != nil {
		return nil, err
	}

	return p, nil
}

func (p *DynamicProtocol) setupParser() error {
	switch p.config.Parser.Type {
	case "delimiter":
		return errors.New("delimiter parser factory not fully implemented for dynamic yet, use existing packages")
		// In a real implementation, we would instantiate parser.NewDelimiterParser(...) here.
		// For MVP, we can mock or reuse existing if available in parser package.

	case "fixed":
		// Simple fixed length parser
		// For now, let's assume we implement a custom one or wait for pkg/parser to support it generic.
		return nil

	default:
		// Default to generic pass-through for now if parser logic is complex
		// or strictly assume "custom" is handled elsewhere.
		return nil
	}
	// TODO: Wire up real parsers using p.config.Parser.Options
}

// Name returns the protocol name.
func (p *DynamicProtocol) Name() string {
	return "dynamic-" + p.config.Name
}

// Version returns the protocol version.
func (p *DynamicProtocol) Version() string {
	return "1.0.0"
}

// Parser returns the parser.
func (p *DynamicProtocol) Parser() parser.Parser {
	return p.parser
}

// Configure updates configuration.
func (p *DynamicProtocol) Configure(config protocol.Config) error {
	return nil // Not supported for now
}

// Encode converts a request to bytes based on field definitions.
func (p *DynamicProtocol) Encode(req *protocol.Request) ([]byte, error) {
	// Determine total size from fields
	// This is a simplified implementation. Real-world would need more layout logic.
	maxOffset := 0
	for _, f := range p.config.Fields {
		end := f.Offset + f.Length
		if end > maxOffset {
			maxOffset = end
		}
	}

	buffer := make([]byte, maxOffset)

	// Map data to fields
	// req.Data is expected to be map[string]interface{}
	dataMap, ok := req.Data.(map[string]interface{})
	if !ok {
		return nil, errors.New("request data must be a map for dynamic protocol")
	}

	for _, f := range p.config.Fields {
		val, exists := dataMap[f.Name]
		if !exists {
			continue // Skip or error?
		}

		if err := p.writeField(buffer, f, val); err != nil {
			return nil, err
		}
	}

	return buffer, nil
}

func (p *DynamicProtocol) writeField(buf []byte, f Field, val interface{}) error {
	if f.Offset+f.Length > len(buf) {
		return errors.New("buffer overflow")
	}

	target := buf[f.Offset : f.Offset+f.Length]

	switch f.Type {
	case "byte":
		if v, ok := val.(float64); ok { // JSON often unmarshals numbers as float64
			target[0] = byte(v)
		} else if v, ok := val.(int); ok {
			target[0] = byte(v)
		}
	case "uint16":
		v_int := 0
		if v, ok := val.(float64); ok {
			v_int = int(v)
		} else if v, ok := val.(int); ok {
			v_int = v
		}
		if p.byteOrder == binary.BigEndian {
			binary.BigEndian.PutUint16(target, uint16(v_int))
		} else {
			binary.LittleEndian.PutUint16(target, uint16(v_int))
		}
		// Add other types...
	}
	return nil
}

// Decode converts bytes to response based on field definitions.
func (p *DynamicProtocol) Decode(data []byte) (*protocol.Response, error) {
	result := make(map[string]interface{})

	for _, f := range p.config.Fields {
		if f.Offset+f.Length > len(data) {
			continue // Partial packet or mismatch
		}

		segment := data[f.Offset : f.Offset+f.Length]

		switch f.Type {
		case "byte":
			result[f.Name] = segment[0]
		case "uint16":
			if p.byteOrder == binary.BigEndian {
				result[f.Name] = binary.BigEndian.Uint16(segment)
			} else {
				result[f.Name] = binary.LittleEndian.Uint16(segment)
			}
		case "string":
			result[f.Name] = string(bytes.Trim(segment, "\x00"))
		}
	}

	return &protocol.Response{
		Success:   true,
		Data:      result,
		RawData:   data,
		Timestamp: time.Now(),
	}, nil
}

func (p *DynamicProtocol) Validate(data []byte) error {
	return nil // Logic depends on fields
}
