// Package parser provides packet parsing functionality for extracting
// complete packets from byte streams. It supports various parsing strategies
// including delimiter-based, length-based, and header+CRC-based parsing.
package parser

import (
	"errors"
)

// Common parser errors.
var (
	ErrIncompletePacket = errors.New("incomplete packet")
	ErrInvalidPacket    = errors.New("invalid packet")
	ErrBufferOverflow   = errors.New("buffer overflow")
	ErrChecksumMismatch = errors.New("checksum mismatch")
	ErrInvalidHeader    = errors.New("invalid header")
)

// Type represents the parser type.
type Type int

const (
	// TypeDelimiter parses packets based on start/end delimiters.
	// Example: STX...ETX, \r\n terminated
	TypeDelimiter Type = iota

	// TypeLength parses packets based on a length field.
	// Example: [LEN:2][DATA:LEN]
	TypeLength

	// TypeHeaderCRC parses packets with header and CRC validation.
	// Example: [HEADER][LEN][DATA][CRC]
	TypeHeaderCRC

	// TypeFixed parses fixed-length packets.
	TypeFixed

	// TypeCustom is a user-defined parser.
	TypeCustom
)

func (t Type) String() string {
	switch t {
	case TypeDelimiter:
		return "delimiter"
	case TypeLength:
		return "length"
	case TypeHeaderCRC:
		return "header_crc"
	case TypeFixed:
		return "fixed"
	case TypeCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// Parser extracts complete packets from a byte stream.
type Parser interface {
	// Type returns the parser type.
	Type() Type

	// Parse attempts to extract a complete packet from the buffer.
	// Returns:
	//   - packet: the extracted packet (nil if incomplete)
	//   - remaining: bytes remaining in buffer after extraction
	//   - err: any parsing error
	Parse(buffer []byte) (packet []byte, remaining []byte, err error)

	// Validate validates a complete packet.
	Validate(packet []byte) error

	// Reset resets the parser state.
	Reset()
}

// Config holds parser configuration.
type Config struct {
	// Type is the parser type.
	Type Type `yaml:"type" json:"type"`

	// MaxPacketSize is the maximum allowed packet size.
	MaxPacketSize int `yaml:"max_packet_size" json:"max_packet_size"`

	// Options contains type-specific options.
	Options map[string]interface{} `yaml:"options" json:"options"`
}

// DelimiterConfig holds delimiter parser configuration.
type DelimiterConfig struct {
	// StartDelimiter is the packet start delimiter (optional).
	StartDelimiter []byte `yaml:"start" json:"start"`

	// EndDelimiter is the packet end delimiter.
	EndDelimiter []byte `yaml:"end" json:"end"`

	// IncludeDelimiters includes delimiters in the returned packet.
	IncludeDelimiters bool `yaml:"include_delimiters" json:"include_delimiters"`

	// MaxPacketSize is the maximum packet size.
	MaxPacketSize int `yaml:"max_size" json:"max_size"`
}

// LengthConfig holds length-based parser configuration.
type LengthConfig struct {
	// LengthOffset is the byte offset of the length field.
	LengthOffset int `yaml:"length_offset" json:"length_offset"`

	// LengthSize is the size of the length field in bytes (1, 2, or 4).
	LengthSize int `yaml:"length_size" json:"length_size"`

	// LengthEndian is the byte order ("big" or "little").
	LengthEndian string `yaml:"length_endian" json:"length_endian"`

	// LengthAdjust is added to the length value to get total packet size.
	LengthAdjust int `yaml:"length_adjust" json:"length_adjust"`

	// HeaderSize is the fixed header size before data.
	HeaderSize int `yaml:"header_size" json:"header_size"`

	// MaxPacketSize is the maximum packet size.
	MaxPacketSize int `yaml:"max_size" json:"max_size"`
}

// HeaderCRCConfig holds header+CRC parser configuration.
type HeaderCRCConfig struct {
	// Header is the expected header bytes.
	Header []byte `yaml:"header" json:"header"`

	// LengthOffset is the byte offset of the length field from start.
	LengthOffset int `yaml:"length_offset" json:"length_offset"`

	// LengthSize is the size of the length field in bytes.
	LengthSize int `yaml:"length_size" json:"length_size"`

	// LengthEndian is the byte order ("big" or "little").
	LengthEndian string `yaml:"length_endian" json:"length_endian"`

	// LengthAdjust is added to length to get total packet size.
	LengthAdjust int `yaml:"length_adjust" json:"length_adjust"`

	// CRCType is the CRC algorithm ("crc16", "crc32", "checksum").
	CRCType string `yaml:"crc_type" json:"crc_type"`

	// CRCOffset is the byte offset of CRC from end (negative) or start.
	CRCOffset int `yaml:"crc_offset" json:"crc_offset"`

	// CRCSize is the size of CRC in bytes.
	CRCSize int `yaml:"crc_size" json:"crc_size"`

	// MaxPacketSize is the maximum packet size.
	MaxPacketSize int `yaml:"max_size" json:"max_size"`
}

// FixedConfig holds fixed-length parser configuration.
type FixedConfig struct {
	// PacketSize is the fixed packet size.
	PacketSize int `yaml:"size" json:"size"`
}

// Buffer manages incoming data for parsing.
type Buffer struct {
	data     []byte
	maxSize  int
	parser   Parser
}

// NewBuffer creates a new parse buffer.
func NewBuffer(maxSize int, parser Parser) *Buffer {
	return &Buffer{
		data:    make([]byte, 0, maxSize),
		maxSize: maxSize,
		parser:  parser,
	}
}

// Write adds data to the buffer.
func (b *Buffer) Write(data []byte) error {
	if len(b.data)+len(data) > b.maxSize {
		return ErrBufferOverflow
	}
	b.data = append(b.data, data...)
	return nil
}

// Parse attempts to extract a complete packet.
func (b *Buffer) Parse() ([]byte, error) {
	if len(b.data) == 0 {
		return nil, ErrIncompletePacket
	}

	packet, remaining, err := b.parser.Parse(b.data)
	if err != nil {
		return nil, err
	}

	b.data = remaining
	return packet, nil
}

// ParseAll extracts all complete packets from the buffer.
func (b *Buffer) ParseAll() ([][]byte, error) {
	var packets [][]byte

	for {
		packet, err := b.Parse()
		if err == ErrIncompletePacket {
			break
		}
		if err != nil {
			return packets, err
		}
		if packet == nil {
			break
		}
		packets = append(packets, packet)
	}

	return packets, nil
}

// Len returns the current buffer length.
func (b *Buffer) Len() int {
	return len(b.data)
}

// Reset clears the buffer.
func (b *Buffer) Reset() {
	b.data = b.data[:0]
	b.parser.Reset()
}

// Factory creates parser instances.
type Factory interface {
	// Create creates a new parser with the given config.
	Create(config Config) (Parser, error)
}
