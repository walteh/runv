package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/walteh/runv/core/runc/plug"
	"github.com/walteh/runv/core/runc/runtime"

	runtimemock "github.com/walteh/runv/gen/mocks/core/runc/runtime"
)

var mockRuntime = &runtimemock.MockRuntime{
	LogFilePathFunc: func() string {
		fmt.Println("LogFilePathFunc called")
		return "/tmp/runc.log"
	},
}

func server(ctx context.Context) error {

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plug.Handshake,
		Plugins:         plug.NewRuntimePluginSet(mockRuntime),

		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})

	return nil
}

func client(ctx context.Context, command string) error {
	execuable, err := os.Executable()
	if err != nil {
		return err
	}

	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  plug.Handshake,
		Plugins:          plug.PluginMap,
		Cmd:              exec.Command(execuable, "server"),
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})
	defer client.Kill()

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		return err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(plug.PluginName)
	if err != nil {
		return err
	}

	// We should have a KV store now! This feels like a normal interface
	// implementation but is in fact over an RPC connection.
	kv := raw.(runtime.Runtime)

	switch command {
	case "ping":
		fmt.Println("log file path: ", kv.LogFilePath())
	default:
		return fmt.Errorf("please only use 'ping', given: %q", os.Args[0])
	}

	return nil
}

func main() {
	// We don't want to see the plugin logs.
	log.SetOutput(io.Discard)

	ctx := context.Background()

	arg := os.Args[1]
	if arg == "server" {
		if err := server(ctx); err != nil {
			fmt.Printf("error: %+v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := client(ctx, arg); err != nil {
		fmt.Printf("error: %+v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
