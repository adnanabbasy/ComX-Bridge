# ComX-Bridge Protocol Guide

ComX-Bridge provides a **Protocol** layer for data interpretation.
It converts raw byte streams received via Transport into meaningful messages (Response/Event).

---

## 1. Modbus (RTU / TCP)

A standard protocol in industrial automation.

### Modbus RTU (Serial)
Modbus operating over serial lines (RS-485, etc.).

**Type**: `modbus-rtu`

```yaml
protocol:
  type: "modbus-rtu"
  options:
    slave_id: 1        # Target Slave ID (1~247)
    timeout: "1s"      # Response wait timeout
```

**Operation:**
- **Request**: Create frame including Function Code, Start Address, Quantity + Calculate CRC16
- **Response**: Validate CRC16 of received data and parse data

### Modbus TCP (Ethernet)
Modbus operating over TCP/IP networks.

**Type**: `modbus-tcp`

```yaml
protocol:
  type: "modbus-tcp"
  options:
    slave_id: 1        # Unit Identifier
    timeout: "1s"
```

**Operation:**
- Create MBAP (Modbus Application Protocol) Header (Transaction ID managed automatically)
- No Checksum/CRC (Guaranteed by TCP)

---

## 2. BACnet (Building Automation)

Standard for building automation control networks.
Current version supports the basic structure based on **IP (BVLC)**.

**Type**: `bacnet`

```yaml
protocol:
  type: "bacnet"
  options:
    device_id: 1001    # (Optional) Device Identifier
```

**Features:**
- **BVLC (BACnet Virtual Link Control) Parsing**: Interpret Header (`0x81`), Function Code, and Length fields.
- **Data Encapsulation**: Encapsulate upper Application Layer (APDU) data into BVLC packets for transmission.

---

## 3. OPC-UA (Industrial Automation)

Next-generation platform-independent industrial communication standard.
Current version supports **Binary Protocol (OCPF)** header parsing.

**Type**: `opc-ua`

```yaml
protocol:
  type: "opc-ua"
  options:
    endpoint_url: "opc.tcp://localhost:4840"
```

**Features:**
- **Message Header Parsing**: Interpret Message Type (`HEL`, `ACK`, `MSG`, etc.), Chunk Type (`F`, `C`, `A`), Message Size.
- **Binary Structure Validation**: Packet integrity check according to OCPF spec.

---

## 4. Raw Binary (Pass-through)

Passes data through without any specific protocol processing.
Used when simple transmission is needed or when handling custom protocols directly in the upper application.

**Type**: `raw`

```yaml
protocol:
  type: "raw"
  options:
    delimiter: "\n"    # (Optional) Packet delimiter (Hex string e.g. "03" or Text "\n")
    debug: true        # Hex dump logging
```

---

## 5. Dynamic Protocol (AI-Inferred)

For unknown or proprietary protocols, the AI Engine can dynamically infer packet structure at runtime.

**Type**: `dynamic`

```yaml
protocol:
  type: "dynamic"
  options:
    learn_samples: 100    # Number of packets to analyze before inference
    auto_update: true     # Re-learn if pattern changes
```

**Features:**
- **AI Structure Inference**: Uses the Protocol Analyzer to detect fields, lengths, and checksums.
- **Runtime Adaptation**: Adjusts parsing rules as more data is collected.
- **Fallback**: If inference fails, falls back to `raw` pass-through.

> [!NOTE]
> Requires AI Engine to be enabled (`ai.enabled: true` in config).

---

## 6. Protocol Extension (Custom)

ComX-Bridge allows adding new protocols via the Plugin System.
Can be implemented using Go Language Plugins or Scripts (Lua/JS). (Details: [docs/plugins](../plugins))
