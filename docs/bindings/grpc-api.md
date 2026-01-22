# ComX-Bridge gRPC API

The gRPC API provides a language-independent high-performance RPC interface, allowing client generation and usage in any language supported by gRPC, such as Java, Node.js, and Python.

## 1. Proto Definition (`comx.proto`)

Service specification defined in `api/proto/comx.proto`.

```protobuf
syntax = "proto3";
package comx.v1;

option go_package = "github.com/commatea/ComX-Bridge/api/proto/v1";

service ComxService {
    // Query Engine Status
    rpc GetStatus(GetStatusRequest) returns (StatusResponse);

    // Gateway Management
    rpc ListGateways(ListGatewaysRequest) returns (ListGatewaysResponse);
    rpc AddGateway(AddGatewayRequest) returns (GatewayInfo);
    
    // Data Send/Receive
    rpc Send(SendRequest) returns (SendResponse);
    rpc Receive(ReceiveRequest) returns (ReceiveResponse);
    
    // Real-time Subscription (Streaming)
    rpc Subscribe(SubscribeRequest) returns (stream Message);
}

message SendRequest {
    string gateway = 1;
    bytes data = 2;
}

message SendResponse {
    int32 bytes_sent = 1;
}

message SubscribeRequest {
    string gateway = 1;
}

message Message {
    bytes data = 1;
    int64 timestamp = 2;
}

// ... (Others Message omitted)
```

## 2. Code Generation

Generate client code for each language using the `protoc` compiler.

### Go
```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    api/proto/comx.proto
```

### Python
```bash
python -m grpc_tools.protoc -Iapi/proto --python_out=. --grpc_python_out=. api/proto/comx.proto
```

### C#
Visual Studio or .NET CLI is used. Add `<Protobuf Include="api/proto/comx.proto" />` to `.csproj` for auto-generation.

## 3. Usage Example (Python Client)

```python
import grpc
import comx_pb2
import comx_pb2_grpc

def run():
    # Connect gRPC Channel
    # metadata = (('x-api-key', 'your-secret-key'),) # Or use JWT: ('authorization', 'Bearer <token>')
    credentials = grpc.access_token_call_credentials('your-jwt-token') # If using standard JWT
    
    with grpc.insecure_channel('localhost:50051') as channel:
        stub = comx_pb2_grpc.ComxServiceStub(channel)
        
        # Send Data (with Metadata if not using call credentials)
        response = stub.Send(comx_pb2.SendRequest(
            gateway="modbus-tcp",
            data=b'\x01\x03\x00\x00\x00\x01'
        ), metadata=(('x-api-key', 'admin-key'),)) # Example passing key directly
        print(f"Bytes Sent: {response.bytes_sent}")
        
        # Subscribe Real-time Data
        for message in stub.Subscribe(comx_pb2.SubscribeRequest(gateway="modbus-tcp")):
            print(f"Received: {message.data}")

if __name__ == '__main__':
    run()
```
