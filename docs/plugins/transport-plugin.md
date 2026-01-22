# ComX-Bridge Transport Plugin Development Guide

This document describes how to develop a Transport plugin for ComX-Bridge.
Transport plugins implement physical/logical data transmission channels such as TCP, Serial, BLE, etc.

## 📁 Example Code

`plugins/examples/` (To be provided)

---

## 1. Interface Definition

A Transport plugin must implement the interface in the `pkg/transport` package.

```go
type Transport interface {
    // 1. Connection Management
    Connect(ctx context.Context) error
    Close() error
    IsConnected() bool

    // 2. Data Send/Receive
    Send(ctx context.Context, data []byte) (int, error)
    Receive(ctx context.Context) ([]byte, error)

    // 3. Config and Info
    Configure(config Config) error
    Info() Info
}
```

---

## 2. Step-by-Step Development Guide

### Step 1: Define Struct

```go
type MyTransport struct {
    config  transport.Config
    conn    net.Conn
    mu      sync.Mutex
    running bool
}
```

### Step 2: Implement Connect

Write the actual connection logic. Use `ctx` for timeouts and cancellation.

```go
func (t *MyTransport) Connect(ctx context.Context) error {
    t.mu.Lock()
    defer t.mu.Unlock()

    if t.conn != nil {
        return nil // Already connected
    }

    dialer := net.Dialer{Timeout: 5 * time.Second}
    conn, err := dialer.DialContext(ctx, "tcp", t.config.Address)
    if err != nil {
        return err
    }

    t.conn = conn
    t.running = true
    return nil
}
```

### Step 3: Implement Send

```go
func (t *MyTransport) Send(ctx context.Context, data []byte) (int, error) {
    if !t.IsConnected() {
        return 0, errors.New("not connected")
    }

    // Set deadline
    if deadline, ok := ctx.Deadline(); ok {
        t.conn.SetWriteDeadline(deadline)
    }

    return t.conn.Write(data)
}
```

### Step 4: Implement Receive

Read data. Handle buffering if necessary.

```go
func (t *MyTransport) Receive(ctx context.Context) ([]byte, error) {
    buffer := make([]byte, 1024)
    
    // Read can block, so setting a timeout or using a separate goroutine is recommended
    t.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
    
    n, err := t.conn.Read(buffer)
    if err != nil {
        return nil, err
    }
    
    return buffer[:n], nil
}
```

---

## 3. Best Practices

1.  **Thread Safety**: `Send` and `Connect/Close` can be called concurrently from multiple goroutines. Must use `Mutex` to protect internal state.
2.  **Reconnect**: Rather than performing automatic reconnection within the Transport itself, it is better to quickly return a disconnection error so that the Engine can execute its reconnection policy.
3.  **Resource Cleanup**: `Close()` must immediately clean up all sockets and resources.

---

## Next Steps

*   [Protocol Plugin Development Guide](../protocol-plugin.md)
*   [AI Plugin Development Guide](../ai-plugin.md)
