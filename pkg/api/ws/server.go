// Package ws provides WebSocket API server implementation.
package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Server is the WebSocket API server.
type Server struct {
	mu       sync.RWMutex
	engine   EngineInterface
	config   ServerConfig
	upgrader websocket.Upgrader
	clients  map[*Client]bool
	running  bool
	server   *http.Server
}

// ServerConfig holds WebSocket server configuration.
type ServerConfig struct {
	// Port is the WebSocket server port.
	Port int `yaml:"port" json:"port"`

	// Path is the WebSocket endpoint path.
	Path string `yaml:"path" json:"path"`

	// PingInterval is the ping interval for keepalive.
	PingInterval time.Duration `yaml:"ping_interval" json:"ping_interval"`

	// WriteTimeout is the write timeout.
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// ReadBufferSize is the read buffer size.
	ReadBufferSize int `yaml:"read_buffer_size" json:"read_buffer_size"`

	// WriteBufferSize is the write buffer size.
	WriteBufferSize int `yaml:"write_buffer_size" json:"write_buffer_size"`

	// AllowedOrigins is the list of allowed origins.
	AllowedOrigins []string `yaml:"allowed_origins" json:"allowed_origins"`
}

// DefaultServerConfig returns default configuration.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:            8081,
		Path:            "/ws",
		PingInterval:    30 * time.Second,
		WriteTimeout:    10 * time.Second,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		AllowedOrigins:  []string{"*"},
	}
}

// EngineInterface defines the engine methods needed by the WebSocket server.
type EngineInterface interface {
	Status() interface{}
	GetGateway(name string) (GatewayInterface, error)
	ListGateways() []string
}

// GatewayInterface defines gateway methods.
type GatewayInterface interface {
	Name() string
	SendRaw(ctx context.Context, data []byte) (int, error)
	Subscribe() (<-chan []byte, error)
}

// Client represents a WebSocket client.
type Client struct {
	conn       *websocket.Conn
	server     *Server
	send       chan []byte
	subscribed map[string]bool
	mu         sync.RWMutex
}

// Message types
const (
	MsgTypeSubscribe   = "subscribe"
	MsgTypeUnsubscribe = "unsubscribe"
	MsgTypeSend        = "send"
	MsgTypeStatus      = "status"
	MsgTypeData        = "data"
	MsgTypeError       = "error"
	MsgTypeAck         = "ack"
)

// WSMessage is a WebSocket message.
type WSMessage struct {
	Type    string          `json:"type"`
	ID      string          `json:"id,omitempty"`
	Gateway string          `json:"gateway,omitempty"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// NewServer creates a new WebSocket server.
func NewServer(engine EngineInterface, config ServerConfig) *Server {
	s := &Server{
		engine:  engine,
		config:  config,
		clients: make(map[*Client]bool),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin: func(r *http.Request) bool {
				if len(config.AllowedOrigins) == 0 {
					return true
				}
				origin := r.Header.Get("Origin")
				for _, allowed := range config.AllowedOrigins {
					if allowed == "*" || allowed == origin {
						return true
					}
				}
				return false
			},
		},
	}
	return s
}

// Start starts the WebSocket server.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc(s.config.Path, s.handleWebSocket)

	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.Port),
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			// Log error
		}
	}()

	s.running = true
	return nil
}

// Stop stops the WebSocket server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// Close all client connections
	for client := range s.clients {
		client.conn.Close()
	}

	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	s.running = false
	return nil
}

// handleWebSocket handles WebSocket upgrade and client connection.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &Client{
		conn:       conn,
		server:     s,
		send:       make(chan []byte, 256),
		subscribed: make(map[string]bool),
	}

	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	go client.writePump()
	go client.readPump()
}

// Broadcast sends a message to all connected clients.
func (s *Server) Broadcast(message []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		select {
		case client.send <- message:
		default:
			// Client buffer full, close connection
			s.removeClient(client)
		}
	}
}

// BroadcastToGateway sends a message to clients subscribed to a gateway.
func (s *Server) BroadcastToGateway(gateway string, data []byte) {
	msg := WSMessage{
		Type:    MsgTypeData,
		Gateway: gateway,
	}
	msg.Data, _ = json.Marshal(map[string]interface{}{
		"bytes": data,
	})

	msgBytes, _ := json.Marshal(msg)

	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		client.mu.RLock()
		subscribed := client.subscribed[gateway]
		client.mu.RUnlock()

		if subscribed {
			select {
			case client.send <- msgBytes:
			default:
			}
		}
	}
}

// removeClient removes a client.
func (s *Server) removeClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[client]; ok {
		delete(s.clients, client)
		close(client.send)
	}
}

// readPump reads messages from the client.
func (c *Client) readPump() {
	defer func() {
		c.server.removeClient(c)
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			c.sendError("", "invalid message format")
			continue
		}

		c.handleMessage(&msg)
	}
}

// writePump writes messages to the client.
func (c *Client) writePump() {
	ticker := time.NewTicker(c.server.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(c.server.config.WriteTimeout))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(c.server.config.WriteTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles an incoming message.
func (c *Client) handleMessage(msg *WSMessage) {
	switch msg.Type {
	case MsgTypeSubscribe:
		c.handleSubscribe(msg)
	case MsgTypeUnsubscribe:
		c.handleUnsubscribe(msg)
	case MsgTypeSend:
		c.handleSend(msg)
	case MsgTypeStatus:
		c.handleStatus(msg)
	default:
		c.sendError(msg.ID, "unknown message type")
	}
}

// handleSubscribe handles subscribe requests.
func (c *Client) handleSubscribe(msg *WSMessage) {
	if msg.Gateway == "" {
		c.sendError(msg.ID, "gateway required")
		return
	}

	// Verify gateway exists
	_, err := c.server.engine.GetGateway(msg.Gateway)
	if err != nil {
		c.sendError(msg.ID, "gateway not found")
		return
	}

	c.mu.Lock()
	c.subscribed[msg.Gateway] = true
	c.mu.Unlock()

	c.sendAck(msg.ID, "subscribed")
}

// handleUnsubscribe handles unsubscribe requests.
func (c *Client) handleUnsubscribe(msg *WSMessage) {
	c.mu.Lock()
	delete(c.subscribed, msg.Gateway)
	c.mu.Unlock()

	c.sendAck(msg.ID, "unsubscribed")
}

// handleSend handles send requests.
func (c *Client) handleSend(msg *WSMessage) {
	if msg.Gateway == "" {
		c.sendError(msg.ID, "gateway required")
		return
	}

	gw, err := c.server.engine.GetGateway(msg.Gateway)
	if err != nil {
		c.sendError(msg.ID, "gateway not found")
		return
	}

	var payload struct {
		Data []byte `json:"data"`
	}
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		c.sendError(msg.ID, "invalid data format")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = gw.SendRaw(ctx, payload.Data)
	if err != nil {
		c.sendError(msg.ID, err.Error())
		return
	}

	c.sendAck(msg.ID, "sent")
}

// handleStatus handles status requests.
func (c *Client) handleStatus(msg *WSMessage) {
	status := c.server.engine.Status()
	gateways := c.server.engine.ListGateways()

	data, _ := json.Marshal(map[string]interface{}{
		"status":   status,
		"gateways": gateways,
	})

	response := WSMessage{
		Type: MsgTypeStatus,
		ID:   msg.ID,
		Data: data,
	}

	respBytes, _ := json.Marshal(response)
	c.send <- respBytes
}

// sendError sends an error message.
func (c *Client) sendError(id, errMsg string) {
	msg := WSMessage{
		Type:  MsgTypeError,
		ID:    id,
		Error: errMsg,
	}
	msgBytes, _ := json.Marshal(msg)
	c.send <- msgBytes
}

// sendAck sends an acknowledgment.
func (c *Client) sendAck(id, message string) {
	data, _ := json.Marshal(map[string]string{"message": message})
	msg := WSMessage{
		Type: MsgTypeAck,
		ID:   id,
		Data: data,
	}
	msgBytes, _ := json.Marshal(msg)
	c.send <- msgBytes
}
