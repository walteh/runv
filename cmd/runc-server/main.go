package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/mdlayher/vsock"
	"github.com/walteh/runv/core/runc"
)

var (
	// vsock parameters
	contextID = flag.Uint("cid", 3, "Context ID to listen on for vsock")
	port      = flag.Uint("port", 10000, "Port to listen on for vsock")

	// unix socket parameters
	socketPath = flag.String("socket", "", "Path to unix socket (if empty, vsock will be used)")

	// runc parameters
	runcRoot      = flag.String("root", "/run/runc", "runc root directory")
	runcBinary    = flag.String("runc-binary", "runc", "Path to runc binary")
	debug         = flag.Bool("debug", false, "Enable debug logging for runc")
	systemdCgroup = flag.Bool("systemd-cgroup", false, "Use systemd cgroup manager")
	rootless      = flag.Bool("rootless", false, "Set to true to enable rootless mode")
	autoRootless  = flag.Bool("auto-rootless", true, "Auto-detect rootless mode")
)

func main() {
	flag.Parse()

	// Create server config
	config := &runc.ServerConfig{
		RuncRoot:      *runcRoot,
		RuncBinary:    *runcBinary,
		Debug:         *debug,
		PdeathSignal:  syscall.SIGKILL,
		SystemdCgroup: *systemdCgroup,
	}

	// Handle rootless mode
	if !*autoRootless {
		config.Rootless = rootless
	}

	var listener net.Listener
	var err error

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

	// Create listener based on configuration
	if *socketPath != "" {
		// Remove any existing socket file
		if err := os.RemoveAll(*socketPath); err != nil {
			log.Fatalf("Failed to remove existing socket file: %v", err)
		}

		listener, err = net.Listen("unix", *socketPath)
		if err != nil {
			log.Fatalf("Failed to listen on unix socket %s: %v", *socketPath, err)
		}
		log.Printf("Listening on unix socket %s", *socketPath)
	} else {
		listener, err = vsock.ListenContextID(uint32(*contextID), uint32(*port), nil)
		if err != nil {
			log.Fatalf("Failed to listen on vsock cid=%d port=%d: %v", *contextID, *port, err)
		}
		log.Printf("Listening on vsock cid=%d port=%d", *contextID, *port)
	}
	defer listener.Close()

	// Start server
	srv, err := runc.RunServer(listener, config)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for signal
	log.Println("Server started, press Ctrl+C to stop")
	done := runc.SetupSignalHandler(srv)
	<-done
	log.Println("Server stopped gracefully")
}
