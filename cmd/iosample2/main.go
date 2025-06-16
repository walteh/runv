package main

import (
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"log/slog"

	"github.com/creack/pty"

	"github.com/containerd/console"
	runc "github.com/containerd/go-runc"
)

const bridgeSock = "/tmp/bridge.sock" // constant Unix-domain socket path  [oai_citation:3‡eli.thegreenplace.net](https://eli.thegreenplace.net/2019/unix-domain-sockets-in-go/?utm_source=chatgpt.com)

func init() {
	// Configure slog for structured, leveled logs  [oai_citation:4‡pkg.go.dev](https://pkg.go.dev/log/slog?utm_source=chatgpt.com)
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{AddSource: true})
	slog.SetDefault(slog.New(h))
}

func main() {

	role := ""
	if len(os.Args) > 1 {
		role = os.Args[1]
	}

	switch role {
	case "linux_a":
		slog.Info("starting linux_a")

		if len(os.Args) < 3 {
			slog.Error("linux_a missing socket path argument")
			return
		}
		linuxA(os.Args[2])
		return
	case "darwin_b":
		slog.Info("starting darwin_b")

		if len(os.Args) < 3 {
			slog.Error("darwin_b missing socket path argument")
			return
		}
		darwinB(os.Args[2])
		return
	default:
		slog.Info("starting coordinator")

		// Clean up any stale socket file  [oai_citation:5‡dev.to](https://dev.to/douglasmakey/understanding-unix-domain-sockets-in-golang-32n8?utm_source=chatgpt.com)
		os.Remove(bridgeSock)

		master, slave, err := pty.Open()
		if err != nil {
			slog.Error("failed to open pty", "error", err)
			return
		}
		defer slave.Close()

		sockA, err := runc.NewTempConsoleSocket()
		if err != nil {
			slog.Error("failed to create temp console socket A", "error", err)
			return
		}
		defer sockA.Close()

		sockB, err := runc.NewTempConsoleSocket()
		if err != nil {
			slog.Error("failed to create temp console socket B", "error", err)
			return
		}
		defer sockB.Close()

		// Spawn linux_a
		cmdA := exec.Command(os.Args[0], "linux_a", sockA.Path()) // use os/exec to re-exec yourself  [oai_citation:6‡pkg.go.dev](https://pkg.go.dev/os/exec?utm_source=chatgpt.com)
		cmdA.Stdout = os.Stdout
		cmdA.Stderr = os.Stderr
		if err := cmdA.Start(); err != nil {
			slog.Error("failed to start linux_a", "error", err)
			return
		}

		// Spawn darwin_b
		cmdB := exec.Command(os.Args[0], "darwin_b", sockB.Path()) // second child  [oai_citation:7‡pkg.go.dev](https://pkg.go.dev/os/exec?utm_source=chatgpt.com)
		cmdB.Stdout = os.Stdout
		cmdB.Stderr = os.Stderr
		if err := cmdB.Start(); err != nil {
			slog.Error("failed to start darwin_b", "error", err)
			return
		}

		sendFD := func(sockPath string) error {
			conn, err := net.Dial("unix", sockPath)
			if err != nil {
				return err
			}
			defer conn.Close()
			rights := syscall.UnixRights(int(master.Fd()))
			_, _, err = conn.(*net.UnixConn).WriteMsgUnix(nil, rights, nil)
			return err
		}
		if err := sendFD(sockA.Path()); err != nil {
			slog.Error("sendFD to A", "error", err)
		}
		if err := sendFD(sockB.Path()); err != nil {
			slog.Error("sendFD to B", "error", err)
		}

		// Wait for both to exit
		cmdA.Wait()
		cmdB.Wait()
		return
	}
}

// linuxA connects to the bridge and proxies its console into it
func linuxA(path string) {
	// Dial the console socket from parent
	conn, err := net.Dial("unix", path)
	if err != nil {
		slog.Error("linuxA: dial console socket", "error", err)
		return
	}
	defer conn.Close()
	unixConn := conn.(*net.UnixConn)

	slog.Info("linuxA: dialed console socket", "path", path)

	// Receive one FD via SCM_RIGHTS
	oob := make([]byte, syscall.CmsgSpace(4))
	_, oobn, _, _, err := unixConn.ReadMsgUnix(nil, oob)
	if err != nil {
		slog.Error("linuxA: ReadMsgUnix", "error", err)
		return
	}
	slog.Info("linuxA: received FD", "oobn", oobn)
	cms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		slog.Error("linuxA: ParseSocketControlMessage", "error", err)
		return
	}
	slog.Info("linuxA: parsed socket control message", "cms", cms)
	fds, err := syscall.ParseUnixRights(&cms[0])
	if err != nil {
		slog.Error("linuxA: ParseUnixRights", "error", err)
		return
	}
	slog.Info("linuxA: parsed unix rights", "fds", fds)
	ptm := os.NewFile(uintptr(fds[0]), "pty-master")
	cons, err := console.ConsoleFromFile(ptm)
	if err != nil {
		slog.Error("linuxA: ConsoleFromFile", "error", err)
		return
	}
	defer cons.Close()
	slog.Info("linuxA: got console FD", "fd", cons.Fd())

	// Dial the bridge
	slog.Info("linuxA: dialing bridge socket", "path", bridgeSock)
	bridge, err := net.Dial("unix", bridgeSock) // net.Dial for Unix sockets  [oai_citation:10‡pkg.go.dev](https://pkg.go.dev/net?utm_source=chatgpt.com)
	if err != nil {
		slog.Error("linuxA: net.Dial", "error", err)
		return
	}
	defer bridge.Close()

	slog.Info("linuxA: proxying console → bridge")
	// proxy bidirectionally using io.Copy  [oai_citation:11‡geeksforgeeks.org](https://www.geeksforgeeks.org/io-copy-function-in-golang-with-examples/?utm_source=chatgpt.com)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(bridge, cons)
	}()
	go func() {
		defer wg.Done()
		io.Copy(cons, bridge)
	}()
	wg.Wait()
	slog.Info("linuxA: done proxying")
}

// darwinB listens on the bridge and proxies its console into it
func darwinB(path string) {
	// Dial the console socket from parent
	conn, err := net.Dial("unix", path)
	if err != nil {
		slog.Error("darwinB: dial console socket", "error", err)
		return
	}
	defer conn.Close()
	slog.Info("darwinB: dialed console socket", "path", path)
	unixConn := conn.(*net.UnixConn)

	// Receive one FD via SCM_RIGHTS
	oob := make([]byte, syscall.CmsgSpace(4))
	_, oobn, _, _, err := unixConn.ReadMsgUnix(nil, oob)
	if err != nil {
		slog.Error("darwinB: ReadMsgUnix", "error", err)
		return
	}
	slog.Info("darwinB: received FD", "oobn", oobn)
	cms, err := syscall.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		slog.Error("darwinB: ParseSocketControlMessage", "error", err)
		return
	}
	slog.Info("darwinB: parsed socket control message", "cms", cms)
	fds, err := syscall.ParseUnixRights(&cms[0])
	if err != nil {
		slog.Error("darwinB: ParseUnixRights", "error", err)
		return
	}
	slog.Info("darwinB: parsed unix rights", "fds", fds)
	ptm := os.NewFile(uintptr(fds[0]), "pty-master")
	cons, err := console.ConsoleFromFile(ptm)
	if err != nil {
		slog.Error("darwinB: ConsoleFromFile", "error", err)
		return
	}
	defer cons.Close()
	slog.Info("darwinB: got console FD", "fd", cons.Fd())

	// Listen on the same bridge path
	slog.Info("darwinB: listening on bridge socket", "path", bridgeSock)
	ln, err := net.Listen("unix", bridgeSock) // net.Listen for Unix sockets  [oai_citation:14‡dev.to](https://dev.to/douglasmakey/understanding-unix-domain-sockets-in-golang-32n8?utm_source=chatgpt.com)
	if err != nil {
		slog.Error("darwinB: net.Listen", "error", err)
		return
	}
	defer ln.Close()

	// Accept one connection
	conn, err = ln.Accept() // Listener.Accept unblocks on connect  [oai_citation:15‡pkg.go.dev](https://pkg.go.dev/net?utm_source=chatgpt.com)
	if err != nil {
		slog.Error("darwinB: Accept", "error", err)
		return
	}
	defer conn.Close()
	slog.Info("darwinB: accepted bridge connection")

	slog.Info("darwinB: proxying console ↔ bridge")
	// proxy bidirectionally using io.Copy  [oai_citation:16‡geeksforgeeks.org](https://www.geeksforgeeks.org/io-copy-function-in-golang-with-examples/?utm_source=chatgpt.com)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(conn, cons)
	}()
	go func() {
		defer wg.Done()
		io.Copy(cons, conn)
	}()
	wg.Wait()
	slog.Info("darwinB: done proxying")
}
