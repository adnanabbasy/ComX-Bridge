package raw

import (
	"time"

	"github.com/commatea/ComX-Bridge/pkg/parser"
	"github.com/commatea/ComX-Bridge/pkg/protocol"
)

// RawProtocol implements a raw binary protocol.
type RawProtocol struct {
	config protocol.Config
	p      parser.Parser
}

// New creates a new RawProtocol.
func New(config protocol.Config) (protocol.Protocol, error) {
	// Create a parser based on options, default to delimiter (empty = read all?)
	// For now, let's assume we use a specialized "Any" parser or just Delimiter with timeout (handled by transport)
	// Actually, better to default to a custom parser that just returns what's there if possible,
	// but Parser interface is byte-stream based.
	// For simulation, let's use a Delimiter parser without delimiters (effectively relying on buffer size or chunks).

	// Create a factory (we assume one exists in parser package or we map it manually)
	// Since we don't have a parser factory available here readily (unless we import parser/delimiter etc),
	// we will assume the raw protocol mostly handles "Send" as-is.
	// For "Receive", it relies on the transport delivering data.

	// TODO: Implement proper parser creation. For now, nil parser (will panic if used).
	// But in Raw mode, maybe we don't use the standard Engine loop that relies on Parser?
	// The Engine uses `gw.Subscribe`.

	return &RawProtocol{
		config: config,
	}, nil
}

func (r *RawProtocol) Name() string {
	return "raw"
}

func (r *RawProtocol) Version() string {
	return "1.0"
}

func (r *RawProtocol) Encode(request *protocol.Request) ([]byte, error) {
	if data, ok := request.Data.([]byte); ok {
		return data, nil
	}
	if str, ok := request.Data.(string); ok {
		return []byte(str), nil
	}
	return nil, nil // Or error
}

func (r *RawProtocol) Decode(data []byte) (*protocol.Response, error) {
	return &protocol.Response{
		Success:   true,
		Data:      data,
		RawData:   data,
		Timestamp: time.Now(),
	}, nil
}

func (r *RawProtocol) Parser() parser.Parser {
	return r.p
}

func (r *RawProtocol) Validate(data []byte) error {
	return nil
}

func (r *RawProtocol) Configure(config protocol.Config) error {
	r.config = config
	return nil
}

// Factory implements protocol.Factory
type Factory struct{}

func (f *Factory) Type() string {
	return "raw"
}

func (f *Factory) Create(config protocol.Config) (protocol.Protocol, error) {
	return New(config)
}

func (f *Factory) Validate(config protocol.Config) error {
	return nil
}
