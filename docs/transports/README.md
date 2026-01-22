# ComX-Bridge Transport Layer

ComX-Bridge provides various physical/logical communication methods via an abstract interface called **Transport**.
You can use them by specifying the `type` in the `transport` section of the configuration file (`config.yaml`).

---

## 1. Serial (RS-232 / RS-485)

The most basic serial communication. Used to connect with legacy industrial equipment.

**Type**: `serial`

### Configuration Example

```yaml
transport:
  type: "serial"
  address: "COM3"          # Windows: COM1, COM3... / Linux: /dev/ttyUSB0, /dev/ttyS0
  options:
    baudrate: 9600         # Baud rate (1200, 2400, 4800, 9600, 19200, 38400, 57600, 115200)
    databits: 8            # Data bits (5, 6, 7, 8)
    parity: "none"         # Parity (none, odd, even, mark, space)
    stopbits: 1            # Stop bits (1, 2)
    timeout: "1s"          # Read timeout
```

---

## 2. TCP (Client / Server)

Ethernet-based TCP communication. Supports both Client and Server modes.

**Type**: `tcp`

### Client Mode (Connect to Device)
ComX-Bridge actively connects to the equipment (Server).

```yaml
transport:
  type: "tcp"
  address: "192.168.0.10:502"  # Target Device IP:Port
  options:
    mode: "client"             # Default
    timeout: "5s"              # Connect timeout
    keepalive: true            # Enable Keep-Alive
```

### Server Mode (Wait for Connection)
ComX-Bridge opens a port and waits for the equipment (Client) to connect.

```yaml
transport:
  type: "tcp"
  address: "0.0.0.0:5000"      # Bind Address
  options:
    mode: "server"
```

---

## 3. UDP (Unicast / Multicast)

Connectionless UDP communication. Used for high-speed data transmission or broadcasts.

**Type**: `udp`

### Configuration Example

```yaml
transport:
  type: "udp"
  address: "0.0.0.0:8080"      # Listen Port (Target address for sending is specified in Send() call or fixed)
  options:
    mode: "unicast"            # unicast, multicast, broadcast
    buffer_size: 4096          # Receive buffer size
```

---

## 4. MQTT (Message Queuing Telemetry Transport)

Includes a built-in MQTT client, an IoT standard protocol. Connects to a broker to send/receive messages.

**Type**: `mqtt`

### Configuration Example

```yaml
transport:
  type: "mqtt"
  # address is the broker address
  address: "tcp://broker.emqx.io:1883" 
  options:
    client_id: "comx-gateway-01"
    topic: "factory/machine1/data"  # Topic to Subscribe
    response_topic: "factory/machine1/cmd" # Topic to send responses (Optional)
    qos: 1                          # 0, 1, 2
    username: "user"                # (Optional)
    password: "pass"                # (Optional)
    clean_session: true
```

---

## 5. WebSocket

Supports web-based real-time bidirectional communication.

**Type**: `websocket`

### Client Mode

```yaml
transport:
  type: "websocket"
  address: "ws://echo.websocket.org"
  options:
    mode: "client"
    handshake_timeout: "5s"
```

### Server Mode

```yaml
transport:
  type: "websocket"
  address: "0.0.0.0:8080"
  options:
    mode: "server"
    path: "/ws"                 # Endpoint path
```

---

## 6. HTTP (REST)

Sends and receives data via HTTP requests.

**Type**: `http`

### Client Mode (Webhook)

```yaml
transport:
  type: "http"
  address: "http://api.myserver.com/v1/data"
  options:
    mode: "client"
    method: "POST"
    timeout: "10s"
    headers:
      Authorization: "Bearer my-token"
      Content-Type: "application/json"
```

---

## 7. BLE (Bluetooth Low Energy)

Communicates with real BLE devices using the Bluetooth adapter of the PC or embedded device. (Windows/Linux/macOS supported, based on `tinygo.org/x/bluetooth`)

**Type**: `ble`

### Configuration Example

```yaml
transport:
  type: "ble"
  options:
    # Scan Filter (One of two required)
    device_name: "MySensor"     # Search by device name
    device_id: "XX:XX:XX..."    # Search by MAC Address (Linux/Windows) or UUID (macOS)

    # Required Service/Characteristic UUIDs
    service_uuid: "6E400001-B5A3-F393-E0A9-E50E24DCCA9E"           # Nordic UART Service example
    characteristic_uuid: "6E400003-B5A3-F393-E0A9-E50E24DCCA9E"    # RX/TX Characteristic

    scan_timeout: "10s"         # Max scan wait time
```

**How it works:**
1. Scans for a device matching the configured `device_name` or `device_id`.
2. Connects to the device and finds the `characteristic_uuid` within the specified `service_uuid`.
3. Enables Notifications for that Characteristic to Receive data.
4. Sending data (Send) performs a Write (Response or WithoutResponse) to that Characteristic.

> **Note**: To use BLE, Bluetooth hardware and drivers must be correctly installed in the execution environment. Windows 10/11+ or Linux with BlueZ 5.x+ is required.
