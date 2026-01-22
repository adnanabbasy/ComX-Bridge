package parser

import (
	"encoding/binary"
	"errors"
)

// LengthParser extracts packets based on a length field.
type LengthParser struct {
	config LengthConfig
}

// NewLengthParser creates a new length-based parser.
func NewLengthParser(config LengthConfig) (*LengthParser, error) {
	if config.LengthSize != 1 && config.LengthSize != 2 && config.LengthSize != 4 {
		return nil, errors.New("length size must be 1, 2, or 4 bytes")
	}
	if config.MaxPacketSize <= 0 {
		config.MaxPacketSize = 65536
	}
	if config.LengthEndian == "" {
		config.LengthEndian = "big"
	}
	return &LengthParser{config: config}, nil
}

// Type returns the parser type.
func (p *LengthParser) Type() Type {
	return TypeLength
}

// Parse extracts a complete packet from the buffer.
func (p *LengthParser) Parse(buffer []byte) (packet []byte, remaining []byte, err error) {
	// Need at least length offset + length size bytes
	minRequired := p.config.LengthOffset + p.config.LengthSize
	if len(buffer) < minRequired {
		return nil, buffer, ErrIncompletePacket
	}

	// Read length field
	lengthBytes := buffer[p.config.LengthOffset : p.config.LengthOffset+p.config.LengthSize]
	var length int

	switch p.config.LengthSize {
	case 1:
		length = int(lengthBytes[0])
	case 2:
		if p.config.LengthEndian == "little" {
			length = int(binary.LittleEndian.Uint16(lengthBytes))
		} else {
			length = int(binary.BigEndian.Uint16(lengthBytes))
		}
	case 4:
		if p.config.LengthEndian == "little" {
			length = int(binary.LittleEndian.Uint32(lengthBytes))
		} else {
			length = int(binary.BigEndian.Uint32(lengthBytes))
		}
	}

	// Apply length adjustment
	length += p.config.LengthAdjust

	// Calculate total packet size
	totalSize := p.config.HeaderSize + length
	if p.config.HeaderSize == 0 {
		totalSize = p.config.LengthOffset + p.config.LengthSize + length
	}

	// Validate size
	if totalSize > p.config.MaxPacketSize {
		return nil, buffer, ErrBufferOverflow
	}
	if totalSize <= 0 {
		return nil, buffer, ErrInvalidPacket
	}

	// Check if we have the complete packet
	if len(buffer) < totalSize {
		return nil, buffer, ErrIncompletePacket
	}

	// Extract packet
	packet = make([]byte, totalSize)
	copy(packet, buffer[:totalSize])
	remaining = buffer[totalSize:]

	return packet, remaining, nil
}

// Validate validates a complete packet.
func (p *LengthParser) Validate(packet []byte) error {
	minRequired := p.config.LengthOffset + p.config.LengthSize
	if len(packet) < minRequired {
		return ErrInvalidPacket
	}

	// Read and verify length
	lengthBytes := packet[p.config.LengthOffset : p.config.LengthOffset+p.config.LengthSize]
	var length int

	switch p.config.LengthSize {
	case 1:
		length = int(lengthBytes[0])
	case 2:
		if p.config.LengthEndian == "little" {
			length = int(binary.LittleEndian.Uint16(lengthBytes))
		} else {
			length = int(binary.BigEndian.Uint16(lengthBytes))
		}
	case 4:
		if p.config.LengthEndian == "little" {
			length = int(binary.LittleEndian.Uint32(lengthBytes))
		} else {
			length = int(binary.BigEndian.Uint32(lengthBytes))
		}
	}

	length += p.config.LengthAdjust

	totalSize := p.config.HeaderSize + length
	if p.config.HeaderSize == 0 {
		totalSize = p.config.LengthOffset + p.config.LengthSize + length
	}

	if len(packet) != totalSize {
		return ErrInvalidPacket
	}

	return nil
}

// Reset resets the parser state.
func (p *LengthParser) Reset() {
	// Length parser is stateless
}

// Common length-based configurations
var (
	// Length1ByteConfig is for 1-byte length field at offset 0.
	Length1ByteConfig = LengthConfig{
		LengthOffset:  0,
		LengthSize:    1,
		LengthEndian:  "big",
		LengthAdjust:  0,
		HeaderSize:    1,
		MaxPacketSize: 256,
	}

	// Length2ByteBEConfig is for 2-byte big-endian length field.
	Length2ByteBEConfig = LengthConfig{
		LengthOffset:  0,
		LengthSize:    2,
		LengthEndian:  "big",
		LengthAdjust:  0,
		HeaderSize:    2,
		MaxPacketSize: 65536,
	}

	// Length2ByteLEConfig is for 2-byte little-endian length field.
	Length2ByteLEConfig = LengthConfig{
		LengthOffset:  0,
		LengthSize:    2,
		LengthEndian:  "little",
		LengthAdjust:  0,
		HeaderSize:    2,
		MaxPacketSize: 65536,
	}

	// ModbusRTULengthConfig is for Modbus RTU (function-dependent).
	ModbusRTULengthConfig = LengthConfig{
		LengthOffset:  2,     // After slave ID and function code
		LengthSize:    1,     // 1-byte length
		LengthEndian:  "big",
		LengthAdjust:  2,     // Add CRC length
		HeaderSize:    3,     // Slave ID + Function + Length
		MaxPacketSize: 256,
	}
)
