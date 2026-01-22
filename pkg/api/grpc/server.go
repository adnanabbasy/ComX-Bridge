// Package grpc provides gRPC API server implementation.
package grpc

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/commatea/ComX-Bridge/pkg/api/middleware"
	"github.com/commatea/ComX-Bridge/pkg/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server is the gRPC API server.
type Server struct {
	mu       sync.RWMutex
	engine   EngineInterface
	server   *grpc.Server
	listener net.Listener
	config   ServerConfig
	running  bool
}

// ServerConfig holds gRPC server configuration.
type ServerConfig struct {
	// Port is the gRPC server port.
	Port int `yaml:"port" json:"port"`

	// EnableReflection enables gRPC reflection for debugging.
	EnableReflection bool `yaml:"enable_reflection" json:"enable_reflection"`

	// MaxRecvMsgSize is the max receive message size in bytes.
	MaxRecvMsgSize int `yaml:"max_recv_msg_size" json:"max_recv_msg_size"`

	// MaxSendMsgSize is the max send message size in bytes.
	MaxSendMsgSize int `yaml:"max_send_msg_size" json:"max_send_msg_size"`

	// Engine reference for config access
	Engine *core.Engine
}

// DefaultServerConfig returns default server configuration.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Port:             9090,
		EnableReflection: true,
		MaxRecvMsgSize:   4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:   4 * 1024 * 1024, // 4MB
	}
}

// EngineInterface defines the engine methods needed by the gRPC server.
type EngineInterface interface {
	Status() core.EngineStatus
	GetGateway(name string) (*core.Gateway, error)
	ListGateways() []string
}

// GatewayStatus represents gateway status.
type GatewayStatus struct {
	Connected        bool
	MessagesReceived uint64
	MessagesSent     uint64
	BytesReceived    uint64
	BytesSent        uint64
	Errors           uint64
}

// NewServer creates a new gRPC server.
func NewServer(engine EngineInterface, config ServerConfig) *Server {
	return &Server{
		engine: engine,
		config: config,
	}
}

// Start starts the gRPC server.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	// Create gRPC server with options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(s.config.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.config.MaxSendMsgSize),
	}

	// Apply Auth Middleware
	if s.config.Engine.Config().API.Auth.Enabled {
		config := s.config.Engine.Config().API.Auth
		authInterceptor := middleware.NewGRPCAuthInterceptor(config.Users, config.JWTSecret)
		opts = append(opts,
			grpc.UnaryInterceptor(authInterceptor.Unary()),
			grpc.StreamInterceptor(authInterceptor.Stream()),
		)
		fmt.Println("gRPC Authentication enabled")
	}

	s.server = grpc.NewServer(opts...)

	// Register ComX service
	RegisterComxServiceServer(s.server, &comxServiceImpl{engine: s.engine})

	// Enable reflection for debugging
	if s.config.EnableReflection {
		reflection.Register(s.server)
	}

	// Start listener
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}
	s.listener = listener

	// Start serving
	go func() {
		if err := s.server.Serve(listener); err != nil {
			// Log error
		}
	}()

	s.running = true
	return nil
}

// Stop stops the gRPC server.
func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	// Graceful stop with timeout
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		s.server.Stop()
	}

	s.running = false
	return nil
}

// comxServiceImpl implements the ComxService gRPC service.
type comxServiceImpl struct {
	UnimplementedComxServiceServer
	engine EngineInterface
}

// GetStatus returns engine status.
func (s *comxServiceImpl) GetStatus(ctx context.Context, req *GetStatusRequest) (*StatusResponse, error) {
	status := s.engine.Status()
	gateways := s.engine.ListGateways()

	gwStatus := make(map[string]*GatewayStatusProto)
	for _, name := range gateways {
		gw, err := s.engine.GetGateway(name)
		if err != nil {
			continue
		}
		st := gw.Status()
		gwStatus[name] = &GatewayStatusProto{
			Name: name,
			// State:            boolToConnectionState(st.Connected), // core.GatewayStatus has State (enum) not Connected logic directly?
			// Let's check core.GatewayStatus struct. It has State (GatewayState).
			State:            connectionStateFromCore(st.State),
			MessagesSent:     st.Stats.MessagesSent,
			MessagesReceived: st.Stats.MessagesReceived,
			BytesSent:        st.Stats.BytesSent,
			BytesReceived:    st.Stats.BytesReceived,
			Errors:           st.Stats.Errors,
		}
	}

	_ = status // Use status if needed

	return &StatusResponse{
		Running:      true,
		GatewayCount: int32(len(gateways)),
		Gateways:     gwStatus,
	}, nil
}

// ListGateways lists all gateways.
func (s *comxServiceImpl) ListGateways(ctx context.Context, req *ListGatewaysRequest) (*ListGatewaysResponse, error) {
	names := s.engine.ListGateways()
	gateways := make([]*GatewayInfo, 0, len(names))

	for _, name := range names {
		gw, err := s.engine.GetGateway(name)
		if err != nil {
			continue
		}
		st := gw.Status()
		gateways = append(gateways, &GatewayInfo{
			Name:  name,
			State: connectionStateFromCore(st.State),
		})
	}

	return &ListGatewaysResponse{
		Gateways: gateways,
	}, nil
}

// GetGateway gets gateway info.
func (s *comxServiceImpl) GetGateway(ctx context.Context, req *GetGatewayRequest) (*GatewayInfo, error) {
	gw, err := s.engine.GetGateway(req.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "gateway not found: %s", req.Name)
	}

	st := gw.Status()
	return &GatewayInfo{
		Name:  gw.Name(),
		State: connectionStateFromCore(st.State),
	}, nil
}

// Send sends data to a gateway.
func (s *comxServiceImpl) Send(ctx context.Context, req *SendRequest) (*SendResponse, error) {
	gw, err := s.engine.GetGateway(req.Gateway)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "gateway not found: %s", req.Gateway)
	}

	n, err := gw.SendRaw(ctx, req.Data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "send failed: %v", err)
	}

	return &SendResponse{
		BytesSent: int32(n),
	}, nil
}

// Receive receives data from a gateway.
func (s *comxServiceImpl) Receive(ctx context.Context, req *ReceiveRequest) (*ReceiveResponse, error) {
	// core.Gateway does not support polling Receive.
	// We can simulate it by subscribing for 1 message if needed, but for now return Unimplemented.
	return nil, status.Errorf(codes.Unimplemented, "receive not supported in this version")
}

// Subscribe streams messages from a gateway.
func (s *comxServiceImpl) Subscribe(req *SubscribeRequest, stream ComxService_SubscribeServer) error {
	gw, err := s.engine.GetGateway(req.Gateway)
	if err != nil {
		return status.Errorf(codes.NotFound, "gateway not found: %s", req.Gateway)
	}

	ch := gw.Subscribe()
	// Core Subscribe returns <-chan *Message. Message has Data interface{} and RawData []byte.

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return nil
			}
			if err := stream.Send(&Message{
				Data:      msg.RawData,
				Timestamp: msg.Timestamp.UnixNano(),
			}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
}

// Helper functions
func connectionStateFromCore(state core.GatewayState) ConnectionState {
	if state == core.GatewayStateRunning {
		return ConnectionState_CONNECTED
	}
	return ConnectionState_DISCONNECTED
}
