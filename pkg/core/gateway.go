package core

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/metrics"
	"github.com/commatea/ComX-Bridge/pkg/parser"
	"github.com/commatea/ComX-Bridge/pkg/persistence"
	"github.com/commatea/ComX-Bridge/pkg/protocol"
	"github.com/commatea/ComX-Bridge/pkg/rules"
	"github.com/commatea/ComX-Bridge/pkg/transport"
	"github.com/google/uuid"
)

// Gateway errors.
var (
	ErrGatewayNotStarted = errors.New("gateway not started")
	ErrGatewayStopped    = errors.New("gateway stopped")
	ErrNoTransport       = errors.New("no transport configured")
	ErrNoProtocol        = errors.New("no protocol configured")
)

// GatewayState represents the gateway state.
type GatewayState int

const (
	GatewayStateStopped GatewayState = iota
	GatewayStateStarting
	GatewayStateRunning
	GatewayStateStopping
	GatewayStateError
)

func (s GatewayState) String() string {
	switch s {
	case GatewayStateStopped:
		return "stopped"
	case GatewayStateStarting:
		return "starting"
	case GatewayStateRunning:
		return "running"
	case GatewayStateStopping:
		return "stopping"
	case GatewayStateError:
		return "error"
	default:
		return "unknown"
	}
}

// Gateway combines a transport and protocol to create a communication channel.
type Gateway struct {
	mu sync.RWMutex

	name       string
	transport  transport.Transport
	protocol   protocol.Protocol
	parser     parser.Parser
	config     GatewayConfig
	store      persistence.Store
	ruleEngine rules.Engine

	// Runtime state
	state     GatewayState
	ctx       context.Context
	cancel    context.CancelFunc
	lastError error

	// Message handling
	subscribers []chan *Message
	subMu       sync.RWMutex

	// Statistics
	stats GatewayStats

	// Parse buffer
	parseBuffer *parser.Buffer
}

// GatewayStats holds gateway statistics.
type GatewayStats struct {
	MessagesReceived uint64        `json:"messages_received"`
	MessagesSent     uint64        `json:"messages_sent"`
	BytesReceived    uint64        `json:"bytes_received"`
	BytesSent        uint64        `json:"bytes_sent"`
	Errors           uint64        `json:"errors"`
	Reconnects       uint64        `json:"reconnects"`
	Uptime           time.Duration `json:"uptime"`
	StartedAt        *time.Time    `json:"started_at"`
}

// Message represents a gateway message.
type Message struct {
	// ID is a unique message identifier.
	ID string `json:"id"`

	// Gateway is the source gateway name.
	Gateway string `json:"gateway"`

	// Direction is the message direction.
	Direction MessageDirection `json:"direction"`

	// Data is the decoded message data.
	Data interface{} `json:"data"`

	// RawData is the raw bytes.
	RawData []byte `json:"raw_data"`

	// Timestamp is when the message was created.
	Timestamp time.Time `json:"timestamp"`

	// Metadata contains additional message metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MessageDirection represents message direction.
type MessageDirection int

const (
	MessageInbound MessageDirection = iota
	MessageOutbound
)

// NewGateway creates a new gateway instance.
func NewGateway(name string, tr transport.Transport, proto protocol.Protocol) *Gateway {
	gw := &Gateway{
		name:      name,
		transport: tr,
		protocol:  proto,
		state:     GatewayStateStopped,
	}

	// Set up parser from protocol if available
	if proto != nil {
		gw.parser = proto.Parser()
		if gw.parser != nil {
			gw.parseBuffer = parser.NewBuffer(65536, gw.parser)
		}
	}

	return gw
}

// Name returns the gateway name.
func (g *Gateway) Name() string {
	return g.name
}

// Start starts the gateway.
func (g *Gateway) Start(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == GatewayStateRunning {
		return nil
	}

	if g.transport == nil {
		return ErrNoTransport
	}

	g.state = GatewayStateStarting
	g.ctx, g.cancel = context.WithCancel(ctx)

	// Connect transport
	if err := g.transport.Connect(g.ctx); err != nil {
		g.state = GatewayStateError
		g.lastError = err
		return err
	}

	// Start receive loop
	go g.receiveLoop()

	// Start retry loop if persistence is enabled
	if g.store != nil {
		go g.retryLoop()
	}

	now := time.Now()
	g.stats.StartedAt = &now
	g.state = GatewayStateRunning

	return nil
}

// Stop stops the gateway.
func (g *Gateway) Stop() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == GatewayStateStopped {
		return nil
	}

	g.state = GatewayStateStopping

	// Cancel context
	if g.cancel != nil {
		g.cancel()
	}

	// Close transport
	if g.transport != nil {
		if err := g.transport.Close(); err != nil {
			g.lastError = err
		}
	}

	// Close all subscriber channels
	g.subMu.Lock()
	for _, ch := range g.subscribers {
		close(ch)
	}
	g.subscribers = nil
	g.subMu.Unlock()

	g.state = GatewayStateStopped
	return nil
}

// Send sends data through the gateway.
func (g *Gateway) Send(ctx context.Context, request *protocol.Request) (*protocol.Response, error) {
	g.mu.RLock()
	if g.state != GatewayStateRunning {
		g.mu.RUnlock()
		return nil, ErrGatewayNotStarted
	}
	tr := g.transport
	proto := g.protocol
	g.mu.RUnlock()

	// Encode request
	data, err := proto.Encode(request)
	if err != nil {
		return nil, err
	}

	// Send data
	n, err := tr.Send(ctx, data)
	if err != nil {
		g.mu.Lock()
		g.stats.Errors++
		g.mu.Unlock()
		metrics.IncPacket(g.name, metrics.DirectionOutbound, metrics.StatusFailed)
		metrics.IncError(g.name, "send_error")

		// Buffer message on failure
		if g.store != nil {
			g.bufferMessage(data)
		}

		return nil, err
	}

	g.mu.Lock()
	g.stats.MessagesSent++
	g.stats.BytesSent += uint64(n)
	g.mu.Unlock()

	// For request-response protocols, wait for response
	// This is a simplified implementation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// In a real implementation, we'd wait for and match the response
	}

	metrics.IncPacket(g.name, metrics.DirectionOutbound, metrics.StatusSuccess)
	return nil, nil
}

// SendRaw sends raw bytes through the gateway.
func (g *Gateway) SendRaw(ctx context.Context, data []byte) (int, error) {
	g.mu.RLock()
	if g.state != GatewayStateRunning {
		g.mu.RUnlock()
		return 0, ErrGatewayNotStarted
	}
	tr := g.transport
	g.mu.RUnlock()

	n, err := tr.Send(ctx, data)
	if err != nil {
		g.mu.Lock()
		g.stats.Errors++
		g.mu.Unlock()
		metrics.IncPacket(g.name, metrics.DirectionOutbound, metrics.StatusFailed)
		metrics.IncError(g.name, "send_raw_error")

		// Buffer message on failure
		if g.store != nil {
			g.bufferMessage(data)
		}

		return n, err
	}

	g.mu.Lock()
	g.stats.MessagesSent++
	g.stats.BytesSent += uint64(n)
	g.mu.Unlock()

	metrics.IncPacket(g.name, metrics.DirectionOutbound, metrics.StatusSuccess)
	return n, nil
}

// Subscribe returns a channel that receives messages.
func (g *Gateway) Subscribe() <-chan *Message {
	ch := make(chan *Message, 100)

	g.subMu.Lock()
	g.subscribers = append(g.subscribers, ch)
	g.subMu.Unlock()

	return ch
}

// Unsubscribe removes a subscription.
func (g *Gateway) Unsubscribe(ch <-chan *Message) {
	g.subMu.Lock()
	defer g.subMu.Unlock()

	for i, sub := range g.subscribers {
		if sub == ch {
			g.subscribers = append(g.subscribers[:i], g.subscribers[i+1:]...)
			close(sub)
			break
		}
	}
}

// Status returns the gateway status.
func (g *Gateway) Status() GatewayStatus {
	g.mu.RLock()
	defer g.mu.RUnlock()

	status := GatewayStatus{
		Name:  g.name,
		State: g.state,
		Stats: g.stats,
	}

	if g.stats.StartedAt != nil {
		status.Stats.Uptime = time.Since(*g.stats.StartedAt)
	}

	if g.transport != nil {
		status.TransportInfo = g.transport.Info()
	}

	if g.lastError != nil {
		errStr := g.lastError.Error()
		status.LastError = &errStr
	}

	return status
}

// bufferMessage saves a failed message to the store.
func (g *Gateway) bufferMessage(data []byte) {
	msg := &persistence.Message{
		ID:        uuid.New().String(),
		Gateway:   g.name,
		Data:      data,
		CreatedAt: time.Now(),
	}
	if err := g.store.Save(msg); err != nil {
		// Log error (we don't have logger here conveniently, but metrics can track it)
		metrics.IncError(g.name, "persistence_save_error")
	}
}

// retryLoop continuously attempts to resend buffered messages.
func (g *Gateway) retryLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			if g.state != GatewayStateRunning {
				continue
			}

			// Get pending messages
			msgs, err := g.store.GetPending(g.name, 10)
			if err != nil {
				continue
			}

			for _, msg := range msgs {
				g.mu.RLock()
				tr := g.transport
				g.mu.RUnlock()

				if tr == nil {
					break
				}

				// Try to send
				_, err := tr.Send(context.Background(), msg.Data)
				if err == nil {
					// Success, delete from store
					g.store.Delete(msg.ID)
					metrics.IncPacket(g.name, metrics.DirectionOutbound, metrics.StatusSuccess)
				} else {
					// Still failing, stop for now
					metrics.IncError(g.name, "retry_error")
					break
				}
			}
		}
	}
}

// receiveLoop continuously receives and processes data.
func (g *Gateway) receiveLoop() {
	for {
		select {
		case <-g.ctx.Done():
			return
		default:
		}

		g.mu.RLock()
		tr := g.transport
		proto := g.protocol
		g.mu.RUnlock()

		if tr == nil {
			return
		}

		// Receive data
		data, err := tr.Receive(g.ctx)
		if err != nil {
			if g.ctx.Err() != nil {
				return
			}
			g.mu.Lock()
			g.stats.Errors++
			g.lastError = err
			g.mu.Unlock()
			metrics.IncError(g.name, "receive_error")
			continue
		}

		g.mu.Lock()
		g.stats.BytesReceived += uint64(len(data))
		g.mu.Unlock()

		// Parse packets if parser is configured
		var packets [][]byte
		if g.parseBuffer != nil {
			if err := g.parseBuffer.Write(data); err != nil {
				continue
			}
			packets, _ = g.parseBuffer.ParseAll()
		} else {
			packets = [][]byte{data}
		}

		// Process each packet
		for _, packet := range packets {
			// Apply Rules
			if g.ruleEngine != nil {
				var err error
				packet, err = g.ruleEngine.Execute(g.name, packet)
				if err != nil {
					metrics.IncError(g.name, "rule_error")
					continue
				}
				if packet == nil {
					// Rule decided to drop packet
					continue
				}
			}

			var decoded interface{}

			// Decode if protocol is configured
			if proto != nil {
				resp, err := proto.Decode(packet)
				if err == nil {
					decoded = resp
				}
			}

			// Create message
			msg := &Message{
				Gateway:   g.name,
				Direction: MessageInbound,
				Data:      decoded,
				RawData:   packet,
				Timestamp: time.Now(),
			}

			g.mu.Lock()
			g.stats.MessagesReceived++
			g.mu.Unlock()

			// Notify subscribers
			g.notifySubscribers(msg)

			// Metrics
			metrics.IncPacket(g.name, metrics.DirectionInbound, metrics.StatusSuccess)
		}
	}
}

// notifySubscribers sends a message to all subscribers.
func (g *Gateway) notifySubscribers(msg *Message) {
	g.subMu.RLock()
	defer g.subMu.RUnlock()

	for _, ch := range g.subscribers {
		select {
		case ch <- msg:
		default:
			// Channel full, skip
		}
	}
}

// GatewayStatus represents the gateway status.
type GatewayStatus struct {
	Name          string         `json:"name"`
	State         GatewayState   `json:"state"`
	TransportInfo transport.Info `json:"transport_info"`
	Stats         GatewayStats   `json:"stats"`
	LastError     *string        `json:"last_error,omitempty"`
}
