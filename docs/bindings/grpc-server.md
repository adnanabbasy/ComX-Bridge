# ComX-Bridge gRPC Server Implementation Guide

This document explains how the **gRPC Server** operates within the ComX-Bridge engine and covers how to add or extend custom gRPC services as needed.

For gRPC API client usage, refer to the [gRPC API Guide](grpc-api.md).

## 1. Server Architecture

ComX-Bridge embeds a gRPC server using Go's `google.golang.org/grpc` package.

*   **Location**: `pkg/api/grpc/server.go`
*   **Proto Types**: `pkg/api/grpc/types.go`
*   **Role**: Accepts external RPC requests and relays them to Core Engine methods (`GetGateway`, `Send`, etc.).

## 2. Server Configuration (`config.yaml`)

Configure the gRPC server port and enabled status in the `api` section of `config.yaml`.

```yaml
api:
  grpc:
    enabled: true
    port: 9090                # gRPC Port
    enable_reflection: true   # Enable Reflection for debugging
    max_recv_msg_size: 4194304  # 4MB
    max_send_msg_size: 4194304  # 4MB
```

## 3. Server Internal Implementation

### Server Struct
```go
type Server struct {
    engine   EngineInterface
    server   *grpc.Server
    config   ServerConfig
    running  bool
}
```

### Start Server
```go
func NewServer(engine EngineInterface, config ServerConfig) *Server {
    return &Server{
        engine: engine,
        config: config,
    }
}

func (s *Server) Start() error {
    opts := []grpc.ServerOption{
        grpc.MaxRecvMsgSize(s.config.MaxRecvMsgSize),
        grpc.MaxSendMsgSize(s.config.MaxSendMsgSize),
    }
    
    s.server = grpc.NewServer(opts...)
    RegisterComxServiceServer(s.server, &comxServiceImpl{engine: s.engine})
    
    if s.config.EnableReflection {
        reflection.Register(s.server)
    }
    
    listener, _ := net.Listen("tcp", fmt.Sprintf(":%d", s.config.Port))
    go s.server.Serve(listener)
    
    return nil
}
```

## 4. Supported RPC Methods

| Method | Request | Response | Description |
|--------|------|------|------|
| `GetStatus` | `GetStatusRequest` | `StatusResponse` | Query Engine Status |
| `ListGateways` | `ListGatewaysRequest` | `ListGatewaysResponse` | List Gateways |
| `GetGateway` | `GetGatewayRequest` | `GatewayInfo` | Get Gateway Info |
| `Send` | `SendRequest` | `SendResponse` | Send Data |
| `Receive` | `ReceiveRequest` | `ReceiveResponse` | Receive Data |
| `Subscribe` | `SubscribeRequest` | `stream Message` | Real-time Subscription |

## 5. Adding Custom Services (Extension)

If you fork ComX-Bridge and want to expose your own business logic via gRPC:

1.  **Add Proto Definition**: Write `api/proto/my_service.proto`.
2.  **Generate Go Code**: Run `protoc` to generate `pb.go`.
3.  **Write Service Implementation**: Write `pkg/api/grpc/my_service.go`.
4.  **Register Server**: Call `RegisterMyServiceServer` in the `Start` method of `server.go`.

## 6. Authentication (JWT/API Key)

ComX-Bridge gRPC server supports **Per-RPC Credentials** using Metadata.

*   **API Key**: Pass `x-api-key` in metadata.
*   **JWT Token**: Pass `authorization` in metadata with value `Bearer <token>`.

### Interceptor Logic
The server includes a global Unary/Stream Interceptor that:
1.  Checks `authorization` metadata for Bearer Token.
2.  Validates JWT signature.
3.  If no JWT, checks `x-api-key` metadata.
4.  Rejects with `Unauthenticated` status if validation fails.

## 7. Security (TLS/SSL)

TLS encryption is mandatory when exposing gRPC externally in production environments.

### Generate Certificate (For Testing)
```bash
openssl req -newkey rsa:2048 -nodes -keyout server.key -x509 -days 365 -out server.crt
```

### Configuration
```yaml
api:
  grpc:
    tls:
      enabled: true
      cert_file: "./certs/server.crt"
      key_file: "./certs/server.key"
```

## 7. Performance Tuning

Consider the following options when streaming large amounts of data (`Subscribe` RPC).

*   **MaxSendMsgSize / MaxRecvMsgSize**: Increase limit if default 4MB is insufficient.
*   **Window Size**: Adjust flow control window size.
*   **Keepalive**: Keepalive settings for maintaining long-lived connections.

```go
opts := []grpc.ServerOption{
    grpc.KeepaliveParams(keepalive.ServerParameters{
        MaxConnectionIdle: 15 * time.Minute,
        Time:              5 * time.Minute,
    }),
}
```
