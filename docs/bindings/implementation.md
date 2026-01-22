# ComX-Bridge Binding Implementation Details

This document explains the internal implementation code of how ComX-Bridge exposes C API and gRPC API. It is intended for Bridge developers or contributors.

## 1. C API Implementation (CGO)

Uses Go's **CGO** feature to expose (`export`) Go functions as C functions.

### Source Code Structure (`pkg/capi/capi.go`)

```go
package main

/*
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

// Callback function pointer definition
typedef void (*ComxDataCallback)(const uint8_t* data, int len, void* userdata);
typedef void (*ComxEventCallback)(int event_type, const char* message, void* userdata);
*/
import "C"
import (
    "context"
    "encoding/json"
    "sync"
    "unsafe"
    "github.com/commatea/ComX-Bridge/pkg/core"
    "github.com/commatea/ComX-Bridge/pkg/config"
)

// Handle Management Map (Keep Go objects from being GC'd)
var (
    engines   = make(map[uintptr]*core.Engine)
    gateways  = make(map[uintptr]*core.Gateway)
    mu        sync.RWMutex
    handleIdx uintptr
)

// Create Engine (Export)
//export comx_engine_create_with_config
func comx_engine_create_with_config(configJSON *C.char) uintptr {
    jsonStr := C.GoString(configJSON)
    // ... Parse Config and Create Engine ...
    handle := nextHandle()
    engines[handle] = engine
    return handle
}

// Start Engine
//export comx_engine_start
func comx_engine_start(handle uintptr) C.int {
    // ... Find Engine and Call Start ...
    if err := engine.Start(context.Background()); err != nil {
        return -1
    }
    return 0
}

// ... Other Gateway functions ...
```

### Build Process

```makefile
# Shared Library (.dll / .so)
build-shared:
	go build -buildmode=c-shared -o comx.dll ./pkg/capi/

# Static Archive (.a)
build-static:
	go build -buildmode=c-archive -o libcomx.a ./pkg/capi/
```

## 2. gRPC API Implementation

Implemented using `google.golang.org/grpc` package.

### Full Proto Definition (`api/proto/comx.proto`)

```protobuf
syntax = "proto3";
package comx.v1;
option go_package = "github.com/commatea/ComX-Bridge/api/proto/v1";

service ComxService {
    rpc GetStatus(GetStatusRequest) returns (StatusResponse);
    rpc ListGateways(ListGatewaysRequest) returns (ListGatewaysResponse);
    rpc AddGateway(AddGatewayRequest) returns (GatewayInfo);
    // ...
    rpc Send(SendRequest) returns (SendResponse);
    rpc Receive(ReceiveRequest) returns (ReceiveResponse);
    rpc Subscribe(SubscribeRequest) returns (stream Message);
}

// ... Message Definitions (Omitted) ...
```

### Server Implementation (`pkg/api/grpc/server.go`)

```go
type ComxGrpcServer struct {
    pb.UnimplementedComxServiceServer
    engine *core.Engine
}

func (s *ComxGrpcServer) Send(ctx context.Context, req *pb.SendRequest) (*pb.SendResponse, error) {
    gw, err := s.engine.GetGateway(req.Gateway)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "gateway not found")
    }
    
    // ... Data Send Logic ...
    return &pb.SendResponse{BytesSent: int32(n)}, nil
}
```
