// Package opcua provides OPC-UA protocol implementation.
package opcua

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/commatea/ComX-Bridge/pkg/parser"
	"github.com/commatea/ComX-Bridge/pkg/protocol"
)

// Protocol implements the OPC-UA protocol.
type Protocol struct{}

// New creates a new OPC-UA protocol instance.
func New(_ protocol.Config) *Protocol {
	return &Protocol{}
}

func (p *Protocol) Name() string {
	return "opc-ua"
}

func (p *Protocol) Version() string {
	return "0.1.0"
}

func (p *Protocol) Encode(request *protocol.Request) ([]byte, error) {
	// Simple OCPF Header Encoder
	// MsgType(3) + ChunkType(1) + Size(4)
	// Example: HELF (Hello Final)

	msgType := "HEL"
	chunkType := "F"

	var payload []byte
	if b, ok := request.Data.([]byte); ok {
		payload = b
	} else if s, ok := request.Data.(string); ok {
		payload = []byte(s)
	}

	length := 8 + len(payload)

	buf := new(bytes.Buffer)
	buf.WriteString(msgType)
	buf.WriteString(chunkType)
	binary.Write(buf, binary.LittleEndian, uint32(length))
	buf.Write(payload)

	return buf.Bytes(), nil
}

func (p *Protocol) Decode(data []byte) (*protocol.Response, error) {
	if len(data) < 8 {
		return nil, errors.New("opc-ua: data too short")
	}

	// Read Size
	length := binary.LittleEndian.Uint32(data[4:8])
	if int(length) != len(data) {
		return nil, fmt.Errorf("opc-ua: length mismatch expected %d got %d", length, len(data))
	}

	return &protocol.Response{
		Data: data[8:], // Body
	}, nil
}

func (p *Protocol) Parser() parser.Parser {
	return &OCPFParser{}
}

func (p *Protocol) Validate(data []byte) error {
	if len(data) < 8 {
		return errors.New("data too short")
	}
	return nil
}

func (p *Protocol) Configure(config protocol.Config) error {
	return nil
}

// OCPFParser implements parser.Parser for OPC-UA (OCPF).
type OCPFParser struct{}

func (p *OCPFParser) Type() parser.Type {
	return parser.TypeHeaderCRC // Roughly fits or Custom
}

func (p *OCPFParser) Parse(buffer []byte) ([]byte, []byte, error) {
	// OCPF Header is 8 bytes minimum
	if len(buffer) < 8 {
		return nil, buffer, parser.ErrIncompletePacket
	}

	// 0-3: MsgType, 3: ChunkType
	// 4-8: Total Size (Little Endian)
	length := binary.LittleEndian.Uint32(buffer[4:8])

	if int(length) > len(buffer) {
		return nil, buffer, parser.ErrIncompletePacket
	}

	return buffer[:length], buffer[length:], nil
}

func (p *OCPFParser) Validate(packet []byte) error {
	if len(packet) < 8 {
		return parser.ErrInvalidPacket
	}
	return nil
}

func (p *OCPFParser) Reset() {}

// Factory creates OPC-UA protocol instances.
type Factory struct{}

func (f *Factory) Type() string {
	return "opc-ua"
}

func (f *Factory) Create(config protocol.Config) (protocol.Protocol, error) {
	return New(config), nil
}

func (f *Factory) Validate(config protocol.Config) error {
	return nil
}
