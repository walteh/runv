package main

import (
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/containerd/console"
	"github.com/creack/pty"

	runc "github.com/containerd/go-runc"
)

func main() {
	// ── 1) Setup structured logging ─────────────────────────────────────────
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
	slog.SetDefault(slog.New(handler))

	// ── 2) Open a PTY (we’ll use this master FD in both sockets) ────────────
	master, slave, err := pty.Open()
	if err != nil {
		slog.Error("open pty", "error", err)
		return
	}
	defer master.Close()
	defer slave.Close()
	slog.Info("PTY opened", "master-fd", master.Fd(), "slave-fd", slave.Fd())

	// ── 3) Create two temp console sockets ─────────────────────────────────
	sockA, err := runc.NewTempConsoleSocket()
	if err != nil {
		slog.Error("NewTempConsoleSocket A", "error", err)
		return
	}
	defer sockA.Close()
	slog.Info("socket A", "path", sockA.Path())

	sockB, err := runc.NewTempConsoleSocket()
	if err != nil {
		slog.Error("NewTempConsoleSocket B", "error", err)
		return
	}
	defer sockB.Close()
	slog.Info("socket B", "path", sockB.Path())

	// ── 4) Simulate 'runc' sending the PTY master FD into each socket ──────
	dialAndSend := func(path string) {
		conn, err := net.Dial("unix", path)
		if err != nil {
			slog.Error("dial socket", "path", path, "error", err)
			return
		}
		defer conn.Close()

		// Build the control message carrying our master FD
		rights := unix.UnixRights(int(master.Fd()))
		n, oobn, err := conn.(*net.UnixConn).
			WriteMsgUnix(nil, rights, nil)
		slog.Info("sent FD", "socket", path, "n", n, "oobn", oobn, "error", err)
	}
	go dialAndSend(sockA.Path())
	go dialAndSend(sockB.Path())

	// ── 5) Call ReceiveMaster() on each socket to get console.Console ─────
	var cA, cB console.Console
	var wgReceive sync.WaitGroup
	wgReceive.Add(2)

	go func() {
		defer wgReceive.Done()
		c, err := sockA.ReceiveMaster()
		if err != nil {
			slog.Error("ReceiveMaster A", "error", err)
			return
		}
		slog.Info("ReceiveMaster A complete", "console-fd", c.Fd())
		cA = c
	}()
	go func() {
		defer wgReceive.Done()
		c, err := sockB.ReceiveMaster()
		if err != nil {
			slog.Error("ReceiveMaster B", "error", err)
			return
		}
		slog.Info("ReceiveMaster B complete", "console-fd", c.Fd())
		cB = c
	}()

	// Wait for both masters...
	wgReceive.Wait()

	// ── 6) Proxy data between the two consoles ─────────────────────────────
	slog.Info("starting proxy")

	var wgProxy sync.WaitGroup
	wgProxy.Add(2)

	// A → B
	go func() {
		defer wgProxy.Done()
		n, err := io.Copy(cB, cA)
		slog.Info("A→B copy done", "bytes", n, "error", err)
	}()
	// B → A
	go func() {
		defer wgProxy.Done()
		n, err := io.Copy(cA, cB)
		slog.Info("B→A copy done", "bytes", n, "error", err)
	}()

	// ── 7) Exercise the pipe ──────────────────────────────────────────────
	time.Sleep(100 * time.Millisecond)
	msg := []byte("ping from A\n")
	n, err := cA.Write(msg)
	slog.Info("A wrote", "bytes", n, "error", err)

	buf := make([]byte, 128)
	n, err = cB.Read(buf)
	slog.Info("B read", "msg", string(buf[:n]), "error", err)

	// Let proxy loops run a bit, then clean up
	time.Sleep(100 * time.Millisecond)
	cA.Close()
	cB.Close()
	wgProxy.Wait()

	slog.Info("done")
}
