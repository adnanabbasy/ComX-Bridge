# WebSocket API

ComX-Bridge's WebSocket API provides **real-time bidirectional communication**, allowing streaming of gateway data and sending remote commands.

## 1. Connection

### Endpoint
```
ws://localhost:8081/ws
```

### Configuration (config.yaml)
```yaml
api:
  websocket:
    enabled: true
    port: 8081
    path: "/ws"
    ping_interval: 30s
    allowed_origins:
      - "*"  # Allow all origins (Restriction required for production)
```

## 2. Message Format

All messages are in JSON format.

### Basic Structure
```json
{
  "type": "message_type",
  "id": "request_id",
  "gateway": "gateway_name",
  "data": { ... },
  "error": "error_message"
}
```

### Message Types

| Type | Direction | Description |
|------|------|------|
| `subscribe` | Client → Server | Subscribe to Gateway |
| `unsubscribe` | Client → Server | Unsubscribe |
| `send` | Client → Server | Send Data |
| `status` | Client → Server | Query Status |
| `data` | Server → Client | Received Data |
| `ack` | Server → Client | Acknowledge Request |
| `error` | Server → Client | Error Response |

## 3. Usage Examples

### JavaScript Client
```javascript
// Connect WebSocket
const ws = new WebSocket('ws://localhost:8081/ws');

ws.onopen = function() {
    console.log('Connected to ComX-Bridge');
    
    // Subscribe to Gateway
    ws.send(JSON.stringify({
        type: 'subscribe',
        id: 'sub-1',
        gateway: 'modbus-gw'
    }));
};

ws.onmessage = function(event) {
    const msg = JSON.parse(event.data);
    
    switch (msg.type) {
        case 'data':
            console.log('Received from', msg.gateway, ':', msg.data);
            break;
        case 'ack':
            console.log('Request acknowledged:', msg.id);
            break;
        case 'error':
            console.error('Error:', msg.error);
            break;
    }
};

// Send Data
function sendData(gateway, data) {
    ws.send(JSON.stringify({
        type: 'send',
        id: 'send-' + Date.now(),
        gateway: gateway,
        data: {
            data: Array.from(new TextEncoder().encode(data))
        }
    }));
}
```

### Python Client
```python
import asyncio
import websockets
import json

async def main():
    uri = "ws://localhost:8081/ws"
    
    async with websockets.connect(uri) as ws:
        # Subscribe to Gateway
        await ws.send(json.dumps({
            "type": "subscribe",
            "id": "sub-1",
            "gateway": "sensor-gw"
        }))
        
        # Message Receive Loop
        async for message in ws:
            msg = json.loads(message)
            if msg["type"] == "data":
                print(f"Data from {msg['gateway']}: {msg['data']}")
            elif msg["type"] == "error":
                print(f"Error: {msg['error']}")

asyncio.run(main())
```

## 4. Gateway Subscribe/Unsubscribe

### Subscribe Request
```json
{
  "type": "subscribe",
  "id": "unique-request-id",
  "gateway": "modbus-gw"
}
```

### Subscribe Response (Success)
```json
{
  "type": "ack",
  "id": "unique-request-id",
  "data": {"message": "subscribed"}
}
```

### Unsubscribe Request
```json
{
  "type": "unsubscribe",
  "id": "unique-request-id",
  "gateway": "modbus-gw"
}
```

## 5. Send Data

### Send Request
```json
{
  "type": "send",
  "id": "send-123",
  "gateway": "modbus-gw",
  "data": {
    "data": [1, 3, 0, 0, 0, 10]
  }
}
```

### Send Response
```json
{
  "type": "ack",
  "id": "send-123",
  "data": {"message": "sent"}
}
```

## 6. Query Status

### Status Request
```json
{
  "type": "status",
  "id": "status-1"
}
```

### Status Response
```json
{
  "type": "status",
  "id": "status-1",
  "data": {
    "status": {...},
    "gateways": ["modbus-gw", "mqtt-gw", "sensor-gw"]
  }
}
```

## 7. Received Data

When data is received from a subscribed gateway, it is automatically pushed.

```json
{
  "type": "data",
  "gateway": "modbus-gw",
  "data": {
    "bytes": [1, 3, 2, 0, 100, 185, 144]
  }
}
```

## 8. Error Handling

### Error Response
```json
{
  "type": "error",
  "id": "request-id",
  "error": "gateway not found"
}
```

### Common Errors
| Error Message | Cause |
|------------|------|
| `gateway required` | Gateway name missing |
| `gateway not found` | Gateway does not exist |
| `invalid data format` | Data format error |
| `unknown message type` | Unsupported message type |

## 9. Keep Alive

The server sends a Ping frame every 30 seconds to maintain the connection.
If the client does not respond, the connection is closed.
