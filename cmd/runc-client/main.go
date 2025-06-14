package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/walteh/runv/core/runc/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	// vsock parameters
	contextID = flag.Uint("cid", 2, "Context ID to connect to for vsock")
	port      = flag.Uint("port", 10000, "Port to connect to for vsock")

	// unix socket parameters
	socketPath = flag.String("socket", "", "Path to unix socket (if empty, vsock will be used)")

	// command parameters
	command = flag.String("cmd", "ping", "Command to execute (ping, list, state, etc.)")
	id      = flag.String("id", "", "Container ID for commands that require it")
	root    = flag.String("root", "", "Root directory for list command")
	timeout = flag.Duration("timeout", 5*time.Second, "Timeout for connection and operations")
)

func main() {
	flag.Parse()

	// Check environment variables first
	envSocketPath := os.Getenv("RUNV_SOCKET")
	if envSocketPath != "" {
		*socketPath = envSocketPath
	}

	// Check for vsock environment variable
	envVsock := os.Getenv("RUNV_VSOCK")
	if envVsock != "" && *socketPath == "" {
		parts := strings.Split(envVsock, ",")
		if len(parts) == 2 {
			if cid, err := strconv.ParseUint(parts[0], 10, 32); err == nil {
				*contextID = uint(cid)
			}
			if p, err := strconv.ParseUint(parts[1], 10, 32); err == nil {
				*port = uint(p)
			}
		}
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Create client connection
	var conn *grpc.ClientConn
	var err error
	var target string

	if *socketPath != "" {
		// Connect to Unix socket
		target = "unix://" + *socketPath
		conn, err = grpc.DialContext(ctx, target,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock())
	} else {
		// Connect to vsock
		target = fmt.Sprintf("vsock://%d:%d", *contextID, *port)

		// Use a custom dialer for vsock
		dialer := func(ctx context.Context, addr string) (net.Conn, error) {
			return vsock.Dial(uint32(*contextID), uint32(*port), nil)
		}

		conn, err = grpc.DialContext(ctx, target,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(dialer),
			grpc.WithBlock())
	}

	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", target, err)
	}
	defer conn.Close()

	// Create client
	runcClient, err := client.NewRuncClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer runcClient.Close()

	// Execute command
	switch *command {
	case "ping":
		if err := runcClient.Ping(ctx); err != nil {
			log.Fatalf("Ping failed: %v", err)
		}
		fmt.Println("Ping successful")

	case "list":
		containers, err := runcClient.List(ctx, *root)
		if err != nil {
			log.Fatalf("List failed: %v", err)
		}
		fmt.Printf("Found %d containers\n", len(containers))
		for _, c := range containers {
			fmt.Printf("ID: %s, PID: %d, Status: %s\n", c.GetId(), c.GetPid(), c.GetStatus())
		}

	case "state":
		if *id == "" {
			log.Fatal("Container ID is required for state command")
		}
		container, err := runcClient.State(ctx, *id)
		if err != nil {
			log.Fatalf("State failed: %v", err)
		}
		fmt.Printf("Container %s: PID=%d, Status=%s\n", container.GetId(), container.GetPid(), container.GetStatus())

	case "version":
		runc, commit, spec, err := runcClient.Version(ctx)
		if err != nil {
			log.Fatalf("Version failed: %v", err)
		}
		fmt.Printf("runc: %s\ncommit: %s\nspec: %s\n", runc, commit, spec)

	default:
		log.Fatalf("Unknown command: %s", *command)
	}
}
