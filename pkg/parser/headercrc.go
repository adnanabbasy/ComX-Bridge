package parser

import (
	"encoding/binary"
	"hash/crc32"

	"github.com/commatea/ComX-Bridge/pkg/utils/crc"
)

// HeaderCRCParser implements a parser that detects a header, reads a length, and validates CRC.
type HeaderCRCParser struct {
	config HeaderCRCConfig
	buffer []byte
}

// NewHeaderCRCParser creates a new HeaderCRCParser.
func NewHeaderCRCParser(config HeaderCRCConfig) *HeaderCRCParser {
	return &HeaderCRCParser{
		config: config,
	}
}

func (p *HeaderCRCParser) Type() Type {
	return TypeHeaderCRC
}

func (p *HeaderCRCParser) Parse(buffer []byte) (packet []byte, remaining []byte, err error) {
	// 1. Scan for header
	if len(p.config.Header) > 0 {
		// Simple scan
		headerFound := false
		for i := 0; i <= len(buffer)-len(p.config.Header); i++ {
			match := true
			for j := 0; j < len(p.config.Header); j++ {
				if buffer[i+j] != p.config.Header[j] {
					match = false
					break
				}
			}
			if match {
				// Found header at i
				buffer = buffer[i:]
				headerFound = true
				break
			}
		}
		if !headerFound {
			// Discard all but last len(header)-1 bytes just in case
			keep := len(p.config.Header) - 1
			if keep > 0 && len(buffer) > keep {
				return nil, buffer[len(buffer)-keep:], nil
			}
			return nil, buffer, nil
		}
	}

	// 2. Read length
	minLen := p.config.LengthOffset + p.config.LengthSize
	if len(buffer) < minLen {
		return nil, buffer, nil
	}

	var length int
	lenBytes := buffer[p.config.LengthOffset : p.config.LengthOffset+p.config.LengthSize]

	switch p.config.LengthSize {
	case 1:
		length = int(lenBytes[0])
	case 2:
		if p.config.LengthEndian == "little" {
			length = int(binary.LittleEndian.Uint16(lenBytes))
		} else {
			length = int(binary.BigEndian.Uint16(lenBytes))
		}
	case 4:
		if p.config.LengthEndian == "little" {
			length = int(binary.LittleEndian.Uint32(lenBytes))
		} else {
			length = int(binary.BigEndian.Uint32(lenBytes))
		}
	}

	length += p.config.LengthAdjust

	if length > p.config.MaxPacketSize {
		return nil, buffer[1:], ErrInvalidPacket // Skip header byte and retry?
	}

	if len(buffer) < length {
		return nil, buffer, nil
	}

	// 3. Extract candidate packet
	packet = buffer[:length]
	remaining = buffer[length:]

	// 4. Validate CRC
	if err := p.Validate(packet); err != nil {
		// Invalid CRC, skip header and retry scanning
		return nil, buffer[1:], nil
	}

	return packet, remaining, nil
}

func (p *HeaderCRCParser) Validate(packet []byte) error {
	if p.config.CRCSize == 0 {
		return nil
	}

	if p.config.CRCType == "crc16" || p.config.CRCType == "modbus" {
		if len(packet) < 2 {
			return ErrInvalidPacket
		}
		data := packet[:len(packet)-2]
		// Modbus CRC is Little Endian
		expected := binary.LittleEndian.Uint16(packet[len(packet)-2:])
		calc := crc.CalculateCRC16(data)
		if calc != expected {
			return ErrChecksumMismatch
		}
	} else if p.config.CRCType == "crc32" {
		if len(packet) < 4 {
			return ErrInvalidPacket
		}
		data := packet[:len(packet)-4]
		expected := binary.BigEndian.Uint32(packet[len(packet)-4:])
		calc := crc32.ChecksumIEEE(data)
		if calc != expected {
			return ErrChecksumMismatch
		}
	}

	return nil
}

func (p *HeaderCRCParser) Reset() {
	// Stateless
}
