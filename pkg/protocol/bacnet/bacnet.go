// Package bacnet provides BACnet protocol implementation.
package bacnet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/commatea/ComX-Bridge/pkg/parser"
	"github.com/commatea/ComX-Bridge/pkg/protocol"
)

// Constants
const (
	BVLCEnabled = 0x81
	FuncUnicast = 0x0a
)

// Protocol implements the BACnet protocol.
type Protocol struct{}

// New creates a new BACnet protocol instance.
func New(_ protocol.Config) *Protocol {
	return &Protocol{}
}

func (p *Protocol) Name() string {
	return "bacnet"
}

func (p *Protocol) Version() string {
	return "0.1.0"
}

func (p *Protocol) Encode(request *protocol.Request) ([]byte, error) {
	// Simple encoder: Wrap data in BVLC header
	// BVLC Type (1) + Func (1) + Length (2) + Data
	var payload []byte
	if b, ok := request.Data.([]byte); ok {
		payload = b
	} else if s, ok := request.Data.(string); ok {
		payload = []byte(s)
	}

	length := 4 + len(payload)

	buf := new(bytes.Buffer)
	buf.WriteByte(BVLCEnabled)
	buf.WriteByte(FuncUnicast)
	binary.Write(buf, binary.BigEndian, uint16(length))
	buf.Write(payload)

	return buf.Bytes(), nil
}

func (p *Protocol) Decode(data []byte) (*protocol.Response, error) {
	if len(data) < 4 {
		return nil, errors.New("bacnet: data too short")
	}

	if data[0] != BVLCEnabled {
		return nil, fmt.Errorf("bacnet: invalid BVLC type: %x", data[0])
	}

	length := binary.BigEndian.Uint16(data[2:4])
	if int(length) != len(data) {
		return nil, fmt.Errorf("bacnet: length mismatch expected %d got %d", length, len(data))
	}

	// Strip BVLC header (4 bytes)
	return &protocol.Response{
		Data: data[4:],
	}, nil
}

func (p *Protocol) Parser() parser.Parser {
	return nil
}

func (p *Protocol) Validate(data []byte) error {
	if len(data) < 4 {
		return errors.New("data too short")
	}
	if data[0] != BVLCEnabled {
		return errors.New("invalid signature")
	}
	return nil
}

func (p *Protocol) Configure(config protocol.Config) error {
	return nil
}

// Factory creates BACnet protocol instances.
type Factory struct{}

func (f *Factory) Type() string {
	return "bacnet"
}

func (f *Factory) Create(config protocol.Config) (protocol.Protocol, error) {
	return New(config), nil
}

func (f *Factory) Validate(config protocol.Config) error {
	return nil
}
