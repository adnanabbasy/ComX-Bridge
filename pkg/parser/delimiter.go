package parser

import (
	"bytes"
)

// DelimiterParser extracts packets based on start/end delimiters.
type DelimiterParser struct {
	config DelimiterConfig
}

// NewDelimiterParser creates a new delimiter-based parser.
func NewDelimiterParser(config DelimiterConfig) *DelimiterParser {
	if config.MaxPacketSize <= 0 {
		config.MaxPacketSize = 65536
	}
	return &DelimiterParser{config: config}
}

// Type returns the parser type.
func (p *DelimiterParser) Type() Type {
	return TypeDelimiter
}

// Parse extracts a complete packet from the buffer.
func (p *DelimiterParser) Parse(buffer []byte) (packet []byte, remaining []byte, err error) {
	if len(buffer) == 0 {
		return nil, buffer, ErrIncompletePacket
	}

	startIdx := 0

	// If start delimiter is defined, find it
	if len(p.config.StartDelimiter) > 0 {
		idx := bytes.Index(buffer, p.config.StartDelimiter)
		if idx == -1 {
			// No start delimiter found, discard data before potential partial delimiter
			keepBytes := len(p.config.StartDelimiter) - 1
			if keepBytes > len(buffer) {
				keepBytes = len(buffer)
			}
			return nil, buffer[len(buffer)-keepBytes:], ErrIncompletePacket
		}
		startIdx = idx
	}

	// Find end delimiter
	if len(p.config.EndDelimiter) == 0 {
		return nil, buffer, ErrInvalidPacket
	}

	searchStart := startIdx
	if len(p.config.StartDelimiter) > 0 {
		searchStart = startIdx + len(p.config.StartDelimiter)
	}

	endIdx := bytes.Index(buffer[searchStart:], p.config.EndDelimiter)
	if endIdx == -1 {
		// Check for buffer overflow
		if len(buffer) > p.config.MaxPacketSize {
			return nil, buffer[len(buffer)-len(p.config.EndDelimiter):], ErrBufferOverflow
		}
		return nil, buffer[startIdx:], ErrIncompletePacket
	}

	endIdx += searchStart

	// Calculate packet boundaries
	packetStart := startIdx
	packetEnd := endIdx + len(p.config.EndDelimiter)

	// Check max size
	if packetEnd-packetStart > p.config.MaxPacketSize {
		return nil, buffer[packetEnd:], ErrBufferOverflow
	}

	// Extract packet
	if p.config.IncludeDelimiters {
		packet = make([]byte, packetEnd-packetStart)
		copy(packet, buffer[packetStart:packetEnd])
	} else {
		dataStart := packetStart
		dataEnd := endIdx
		if len(p.config.StartDelimiter) > 0 {
			dataStart = packetStart + len(p.config.StartDelimiter)
		}
		packet = make([]byte, dataEnd-dataStart)
		copy(packet, buffer[dataStart:dataEnd])
	}

	remaining = buffer[packetEnd:]
	return packet, remaining, nil
}

// Validate validates a complete packet.
func (p *DelimiterParser) Validate(packet []byte) error {
	if len(packet) == 0 {
		return ErrInvalidPacket
	}

	if p.config.IncludeDelimiters {
		// Check start delimiter
		if len(p.config.StartDelimiter) > 0 {
			if !bytes.HasPrefix(packet, p.config.StartDelimiter) {
				return ErrInvalidPacket
			}
		}

		// Check end delimiter
		if !bytes.HasSuffix(packet, p.config.EndDelimiter) {
			return ErrInvalidPacket
		}
	}

	return nil
}

// Reset resets the parser state.
func (p *DelimiterParser) Reset() {
	// Delimiter parser is stateless
}

// Common delimiter configurations
var (
	// CRLFDelimiter is a carriage return + line feed delimiter.
	CRLFDelimiter = DelimiterConfig{
		EndDelimiter:      []byte{'\r', '\n'},
		IncludeDelimiters: false,
		MaxPacketSize:     4096,
	}

	// LFDelimiter is a line feed delimiter.
	LFDelimiter = DelimiterConfig{
		EndDelimiter:      []byte{'\n'},
		IncludeDelimiters: false,
		MaxPacketSize:     4096,
	}

	// STXETXDelimiter is a STX/ETX delimiter (ASCII control characters).
	STXETXDelimiter = DelimiterConfig{
		StartDelimiter:    []byte{0x02}, // STX
		EndDelimiter:      []byte{0x03}, // ETX
		IncludeDelimiters: true,
		MaxPacketSize:     65536,
	}

	// NullDelimiter is a null character delimiter.
	NullDelimiter = DelimiterConfig{
		EndDelimiter:      []byte{0x00},
		IncludeDelimiters: false,
		MaxPacketSize:     4096,
	}
)
