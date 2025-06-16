package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/walteh/runv/core/runc/runtime"
	"github.com/walteh/runv/core/runc/runtime/plug"
	"github.com/walteh/runv/pkg/logging"

	gorunc "github.com/containerd/go-runc"

	runtimemock "github.com/walteh/runv/gen/mocks/core/runc/runtime"
)

var mockRuntime = &runtimemock.MockRuntime{
	ReadPidFileFunc: func(ctx context.Context, path string) (int, error) {
		return 1234, nil
	},
	SharedDirFunc: func() string {
		return "/runv/shared"
	},
}

var mockSocketAllocator = &runtimemock.MockSocketAllocator{
	AllocateSocketFunc: func(ctx context.Context) (runtime.AllocatedSocket, error) {
		return nil, nil
	},
}

var mockRuntimeExtras = &runtimemock.MockRuntimeExtras{
	RunFunc: func(context1 context.Context, s string, s1 string, createOpts *gorunc.CreateOpts) (int, error) {
		return 1234, nil
	},
}

func server(ctx context.Context, logPath string) error {
	proxySock, err := setupServerLogProxy(ctx, logPath)
	if err != nil {
		return err
	}

	logging.NewDefaultDevLogger("server", proxySock,
		logging.WithValue(slog.Int("pid", os.Getpid())),
		logging.WithValue(slog.Int("ppid", os.Getppid())),
	)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: plug.Handshake,
		Logger:          hclog.Default(),
		Plugins:         plug.NewRuntimePluginSet(mockRuntime, mockRuntimeExtras, mockSocketAllocator),
		// A non-nil value here enables gRPC serving for this plugin...
		GRPCServer: plugin.DefaultGRPCServer,
	})

	return nil
}

func client(ctx context.Context, command string) error {
	logging.NewDefaultDevLogger(
		"client",
		os.Stdout,
		logging.WithValue(slog.Int("pid", os.Getpid())),
		logging.WithValue(slog.Int("ppid", os.Getppid())),
	)

	proxySock, err := setupClientLogProxy(ctx, os.Stdout)
	if err != nil {
		return err
	}

	execuable, err := os.Executable()
	if err != nil {
		return err
	}

	// We're a host. Start by launching the plugin process.
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  plug.Handshake,
		Plugins:          plug.PluginMap,
		Logger:           hclog.Default(),
		Cmd:              exec.Command(execuable, "server", proxySock),
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
		pid, err := kv.ReadPidFile(ctx, "/proc/1234/status")
		if err != nil {
			return err
		}
		fmt.Println("pid: ", pid)
	default:
		return fmt.Errorf("please only use 'ping', given: %q", os.Args[0])
	}

	return nil
}

func main() {
	// We don't want to see the plugin logs.

	ctx := context.Background()

	arg := os.Args[1]
	if arg == "server" {
		if len(os.Args) < 3 {
			fmt.Printf("usage: %s server <log-path>\n", os.Args[0])
			os.Exit(1)
		}
		if err := server(ctx, os.Args[2]); err != nil {
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

func setupServerLogProxy(ctx context.Context, path string) (io.Writer, error) {
	proxySock, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}

	return proxySock, nil
}
func setupClientLogProxy(ctx context.Context, w io.Writer) (string, error) {
	tmpFile, err := os.CreateTemp("", "log-proxy-socket")
	if err != nil {
		return "", err
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name())

	proxySock, err := net.Listen("unix", tmpFile.Name())
	if err != nil {
		return "", err
	}

	// fwd logs from the proxy socket to stdout
	go func() {
		defer proxySock.Close()
		for {
			if ctx.Err() != nil {
				return
			}
			conn, err := proxySock.Accept()
			if err != nil {
				slog.Error("Failed to accept log proxy connection", "error", err)
				return
			}
			defer conn.Close()
			go func() { _, _ = io.Copy(w, conn) }()
		}
	}()

	return tmpFile.Name(), nil
}
