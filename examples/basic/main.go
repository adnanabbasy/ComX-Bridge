// Basic example demonstrating ComX-Bridge usage.
//
// This example shows how to:
// 1. Create and configure transports
// 2. Set up a gateway with a protocol
// 3. Send and receive data
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/commatea/ComX-Bridge/pkg/core"
	"github.com/commatea/ComX-Bridge/pkg/transport"
	"github.com/commatea/ComX-Bridge/pkg/transport/tcp"
)

func main() {
	fmt.Println("ComX-Bridge Basic Example")
	fmt.Println("==========================")

	// Example 1: Direct TCP Transport Usage
	fmt.Println("\n1. Direct Transport Usage")
	directTransportExample()

	// Example 2: Gateway Usage
	fmt.Println("\n2. Gateway Usage")
	gatewayExample()

	// Example 3: Engine Usage
	fmt.Println("\n3. Engine Usage")
	engineExample()
}

// directTransportExample shows direct transport usage.
func directTransportExample() {
	// Create TCP transport configuration
	config := transport.Config{
		Type:       "tcp",
		Address:    "example.com:80",
		BufferSize: 4096,
		Timeout:    5 * time.Second,
		Options: map[string]interface{}{
			"keepalive": true,
			"no_delay":  true,
		},
	}

	// Create TCP client
	client, err := tcp.NewClient(config)
	if err != nil {
		log.Printf("Failed to create TCP client: %v", err)
		return
	}

	// Get transport info
	info := client.Info()
	fmt.Printf("Transport ID: %s\n", info.ID)
	fmt.Printf("Transport Type: %s\n", info.Type)
	fmt.Printf("Address: %s\n", info.Address)
	fmt.Printf("State: %s\n", info.State)

	// In a real scenario, you would connect and send data:
	// ctx := context.Background()
	// if err := client.Connect(ctx); err != nil {
	//     log.Fatal(err)
	// }
	// defer client.Close()
	//
	// request := []byte("GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
	// n, err := client.Send(ctx, request)
	// if err != nil {
	//     log.Fatal(err)
	// }
	// fmt.Printf("Sent %d bytes\n", n)
	//
	// response, err := client.Receive(ctx)
	// if err != nil {
	//     log.Fatal(err)
	// }
	// fmt.Printf("Received: %s\n", string(response))
}

// gatewayExample shows gateway usage.
func gatewayExample() {
	// Create transport
	tcpConfig := transport.Config{
		Type:    "tcp",
		Address: "localhost:502",
		Timeout: 5 * time.Second,
	}

	client, err := tcp.NewClient(tcpConfig)
	if err != nil {
		log.Printf("Failed to create transport: %v", err)
		return
	}

	// Create gateway (without protocol for this example)
	gateway := core.NewGateway("example-gateway", client, nil)

	fmt.Printf("Gateway Name: %s\n", gateway.Name())

	// Subscribe to messages
	messages := gateway.Subscribe()

	// Start gateway in a goroutine
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		for msg := range messages {
			fmt.Printf("Received message: %v\n", msg)
		}
	}()

	// Get status
	status := gateway.Status()
	fmt.Printf("Gateway State: %s\n", status.State)
	fmt.Printf("Messages Received: %d\n", status.Stats.MessagesReceived)
	fmt.Printf("Messages Sent: %d\n", status.Stats.MessagesSent)

	// In a real scenario:
	// if err := gateway.Start(ctx); err != nil {
	//     log.Fatal(err)
	// }
	// defer gateway.Stop()
	//
	// Send raw data:
	// n, err := gateway.SendRaw(ctx, []byte("Hello"))
	// if err != nil {
	//     log.Fatal(err)
	// }
	// fmt.Printf("Sent %d bytes\n", n)

	_ = ctx // Use ctx
}

// engineExample shows engine usage.
func engineExample() {
	// Create engine configuration
	config := &core.Config{
		Gateways: []core.GatewayConfig{
			{
				Name:    "tcp-gateway",
				Enabled: true,
				Transport: transport.Config{
					Type:    "tcp",
					Address: "localhost:502",
					Timeout: 5 * time.Second,
				},
				AutoReconnect: true,
			},
		},
		Logging: core.LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}

	// Create engine
	engine, err := core.NewEngine(config)
	if err != nil {
		log.Printf("Failed to create engine: %v", err)
		return
	}

	// Register event handler
	engine.OnEvent(core.EventHandlerFunc(func(event core.Event) {
		fmt.Printf("Engine Event: %v\n", event.Type)
	}))

	// List gateways
	gateways := engine.ListGateways()
	fmt.Printf("Configured Gateways: %v\n", gateways)

	// Get status
	status := engine.Status()
	fmt.Printf("Engine Started: %v\n", status.Started)

	// In a real scenario:
	// ctx := context.Background()
	// if err := engine.Start(ctx); err != nil {
	//     log.Fatal(err)
	// }
	// defer engine.Stop()
	//
	// Get a gateway:
	// gw, err := engine.GetGateway("tcp-gateway")
	// if err != nil {
	//     log.Fatal(err)
	// }
	// gwStatus := gw.Status()
	// fmt.Printf("Gateway %s state: %s\n", gw.Name(), gwStatus.State)
}
