// ComX-Bridge CLI
//
// A unified communication platform for industrial and IoT protocols.
// Abstracts RS232/RS485/TCP/UDP/HTTP/MQTT into a single communication layer
// with protocol plugin support.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/commatea/ComX-Bridge/pkg/api/grpc"
	"github.com/commatea/ComX-Bridge/pkg/api/rest"
	"github.com/commatea/ComX-Bridge/pkg/config"
	"github.com/commatea/ComX-Bridge/pkg/core"
	"github.com/commatea/ComX-Bridge/pkg/protocol/bacnet"
	"github.com/commatea/ComX-Bridge/pkg/protocol/modbus"
	"github.com/commatea/ComX-Bridge/pkg/protocol/opcua"
	"github.com/commatea/ComX-Bridge/pkg/protocol/raw"
	"github.com/commatea/ComX-Bridge/pkg/transport/ble"
	"github.com/commatea/ComX-Bridge/pkg/transport/http"
	"github.com/commatea/ComX-Bridge/pkg/transport/mqtt"
	"github.com/commatea/ComX-Bridge/pkg/transport/serial"
	"github.com/commatea/ComX-Bridge/pkg/transport/tcp"
	"github.com/commatea/ComX-Bridge/pkg/transport/udp"
	"github.com/commatea/ComX-Bridge/pkg/transport/websocket"
	"github.com/spf13/cobra"
)

var (
	version   = "1.0.0"
	buildTime = "dev"
	gitCommit = "unknown"
)

var (
	cfgFile    string
	verbose    bool
	jsonOutput bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "comx",
		Short: "ComX-Bridge - Unified Communication Engine",
		Long: `ComX-Bridge is a unified communication platform that abstracts
various communication protocols (RS232/RS485/TCP/UDP/HTTP/MQTT)
into a single layer with protocol plugin support.

"어떤 통신이든 연결하는 하나의 엔진"`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, gitCommit, buildTime),
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default: ./config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output in JSON format")

	// Add commands
	rootCmd.AddCommand(
		newStartCmd(),
		newStatusCmd(),
		newGatewayCmd(),
		newPluginCmd(),
		newSendCmd(),
		newAnalyzeCmd(),
		newVersionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// newStartCmd creates the start command.
func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the ComX-Bridge engine",
		Long:  "Start the ComX-Bridge engine with all configured gateways.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStart()
		},
	}

	return cmd
}

// runStart starts the engine.
func runStart() error {
	// Load configuration
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply Command Line Flags overrides
	if verbose {
		cfg.Logging.Level = "debug"
	}
	if jsonOutput {
		cfg.Logging.Format = "json"
	}

	// Create engine
	engine, err := core.NewEngine(cfg)
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}

	// Registries
	tr := core.NewTransportRegistry()
	tr.Register(serial.NewFactory())
	tr.Register(tcp.NewFactory())
	tr.Register(udp.NewFactory())
	tr.Register(mqtt.NewFactory())
	tr.Register(websocket.NewFactory())
	tr.Register(http.NewFactory())
	tr.Register(ble.NewFactory())
	engine.SetTransportRegistry(tr)

	pr := core.NewProtocolRegistry()
	pr.Register(&raw.Factory{})
	pr.Register(&bacnet.Factory{})
	pr.Register(&opcua.Factory{})
	pr.Register(&modbus.RTUFactory{})
	pr.Register(&modbus.TCPFactory{})
	engine.SetProtocolRegistry(pr)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start engine
	fmt.Println("Starting ComX-Bridge...")
	if err := engine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}

	// Start API Server if enabled
	var apiServer *rest.Server
	var grpcServer *grpc.Server
	if cfg.API.Enabled {
		// REST Server
		apiServer = rest.NewServer(engine, rest.ServerConfig{Port: cfg.API.Port})
		if err := apiServer.Start(); err != nil {
			return fmt.Errorf("failed to start API server: %w", err)
		}

		// gRPC Server (Default port 9090 for now, or add to config)
		grpcConfig := grpc.DefaultServerConfig()
		grpcConfig.Port = 9090 // TODO: Add to config
		grpcConfig.Engine = engine
		grpcServer = grpc.NewServer(engine, grpcConfig)
		if err := grpcServer.Start(); err != nil {
			return fmt.Errorf("failed to start gRPC server: %w", err)
		}
	}

	fmt.Println("ComX-Bridge is running. Press Ctrl+C to stop.")

	// Wait for signal
	<-sigCh
	fmt.Println("\nShutting down...")

	// Stop API Server
	if apiServer != nil {
		if err := apiServer.Stop(context.Background()); err != nil {
			fmt.Printf("Error stopping API server: %v\n", err)
		}
	}
	if grpcServer != nil {
		if err := grpcServer.Stop(context.Background()); err != nil {
			fmt.Printf("Error stopping gRPC server: %v\n", err)
		}
	}

	if err := engine.Stop(); err != nil {
		return fmt.Errorf("failed to stop engine: %w", err)
	}

	fmt.Println("ComX-Bridge stopped.")
	return nil
}

// newStatusCmd creates the status command.
func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show engine and gateway status",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Engine Status:")
			fmt.Println("  State: not running")
			fmt.Println("\nUse 'comx start' to start the engine.")
			return nil
		},
	}
}

// newGatewayCmd creates the gateway command.
func newGatewayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gateway",
		Short: "Manage gateways",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List configured gateways",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Configured Gateways:")
				fmt.Println("  (none configured)")
				return nil
			},
		},
		&cobra.Command{
			Use:   "add <name>",
			Short: "Add a new gateway",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Use config file to add gateways.")
				return nil
			},
		},
	)

	return cmd
}

// newPluginCmd creates the plugin command.
func newPluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plugin",
		Short: "Manage plugins",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List available plugins",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Available Plugins:")
				fmt.Println("\nTransports:")
				fmt.Println("  - serial    RS232/RS485 serial port")
				fmt.Println("  - tcp       TCP client/server")
				fmt.Println("  - udp       UDP client/server")
				fmt.Println("\nProtocols:")
				fmt.Println("  - modbus-rtu  Modbus RTU")
				fmt.Println("  - modbus-tcp  Modbus TCP")
				fmt.Println("  - raw         Raw binary")
				return nil
			},
		},
		&cobra.Command{
			Use:   "install <name>",
			Short: "Install a plugin",
			RunE: func(cmd *cobra.Command, args []string) error {
				fmt.Println("Plugin installation not yet implemented.")
				return nil
			},
		},
	)

	return cmd
}

// newSendCmd creates the send command.
func newSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send <gateway> <data>",
		Short: "Send data through a gateway",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			gateway := args[0]
			data := args[1]
			fmt.Printf("Sending to gateway '%s': %s\n", gateway, data)
			fmt.Println("Engine not running. Use 'comx start' first.")
			return nil
		},
	}

	return cmd
}

// newAnalyzeCmd creates the analyze command (AI feature).
func newAnalyzeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "AI-powered protocol analysis",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "packets <file>",
			Short: "Analyze packet samples from file",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				file := args[0]
				fmt.Printf("Analyzing packets from: %s\n", file)
				fmt.Println("AI analysis feature coming soon...")
				return nil
			},
		},
		&cobra.Command{
			Use:   "capture <gateway>",
			Short: "Capture and analyze live traffic",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				gateway := args[0]
				fmt.Printf("Capturing from gateway: %s\n", gateway)
				fmt.Println("Live capture feature coming soon...")
				return nil
			},
		},
	)

	return cmd
}

// newVersionCmd creates the version command.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("ComX-Bridge %s\n", version)
			fmt.Printf("  Commit:  %s\n", gitCommit)
			fmt.Printf("  Built:   %s\n", buildTime)
			fmt.Println()
			fmt.Println("A unified communication platform for industrial and IoT protocols.")
			fmt.Println("https://github.com/commatea/ComX-Bridge")
		},
	}
}
