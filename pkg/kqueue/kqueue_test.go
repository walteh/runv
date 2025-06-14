//go:build darwin || freebsd || netbsd || openbsd
// +build darwin freebsd netbsd openbsd

package kqueue

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/containerd/console"
	"github.com/creack/pty" // preferred over github.com/kr/pty  [oai_citation:0‡github.com](https://github.com/creack/pty?utm_source=chatgpt.com)
	"github.com/stretchr/testify/require"
)

func TestKqueueConsole_ReadAndWrite(t *testing.T) {
	// 1. Allocate a real PTY pair.
	master, slave, err := pty.Open() // Opens (master, slave) pair  [oai_citation:1‡pkg.go.dev](https://pkg.go.dev/github.com/kr/pty?utm_source=chatgpt.com)
	require.NoError(t, err)
	defer master.Close()
	defer slave.Close()

	// 2. Wrap the master side so kqueue sees it as a tty.
	console, err := console.ConsoleFromFile(master) // registers master as an EpollConsole  [oai_citation:2‡pkg.go.dev](https://pkg.go.dev/github.com/containerd/console?utm_source=chatgpt.com)
	require.NoError(t, err)
	defer console.Close()

	// // 3. Re-open the slave file for container I/O.
	// slaveFile, err := os.OpenFile(slavePath, unix.O_RDWR, 0)
	// require.NoError(t, err)
	// defer slaveFile.Close()

	// 4. Spawn a command that writes “test” repeatedly to its stdout/stderr.
	iteration := 10
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("for i in $(seq 1 %d); do printf test; done", iteration),
	)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = slave, slave, slave

	// 5. Set up the kqueue-based shim.
	kq, err := NewKqueuer()
	require.NoError(t, err)
	defer kq.Close()
	kqConsole, err := kq.Add(console)
	require.NoError(t, err)

	// 6. Kick off the event loop.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		kq.Wait()
	}()

	// 7. Capture guest→shim output.
	var gotRead bytes.Buffer
	var readWG sync.WaitGroup
	readWG.Add(1)
	go func() {
		defer readWG.Done()
		io.Copy(&gotRead, kqConsole) // tests Read path
	}()

	// 8. Run the child command.
	require.NoError(t, cmd.Run())

	// 9. Validate the Read path.
	expected := strings.Repeat("test", iteration)
	require.Equal(t, expected, gotRead.String())

	// 10. Now exercise shim→guest Write.
	var gotWrite bytes.Buffer
	var writeWG sync.WaitGroup
	writeWG.Add(1)
	go func() {
		defer writeWG.Done()
		io.Copy(&gotWrite, slave) // tests Write path
	}()

	payload := []byte("ping\n")
	n, err := kqConsole.Write(payload)
	require.NoError(t, err)
	require.Equal(t, len(payload), n)

	// allow a moment for I/O
	time.Sleep(20 * time.Millisecond)

	// 11. Validate the Write path.
	require.Equal(t, "ping\n", gotWrite.String())

	err = kqConsole.Shutdown(kq.CloseConsole)
	require.NoError(t, err)

	err = kq.Close()
	require.NoError(t, err)

	err = kq.Close()
	require.ErrorIs(t, err, os.ErrClosed)

	// close the slave file
	err = slave.Close()
	require.NoError(t, err)

	// close the master file
	err = master.Close()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		wg.Wait()
		return kqConsole.closed
	}, 1*time.Second, 100*time.Millisecond, "kqueue event loop did not exit")

	require.Eventually(t, func() bool {
		readWG.Wait()
		return kqConsole.closed
	}, 1*time.Second, 100*time.Millisecond, "read path did not exit")

	require.Eventually(t, func() bool {
		writeWG.Wait()
		return kqConsole.closed
	}, 1*time.Second, 100*time.Millisecond, "write path did not exit")

	// 12. Cleanly shutdown.
	// readWG.Wait()
	// writeWG.Wait()

	// fmt.Println("gotWrite", gotWrite.String())
	// fmt.Println("payload", string(payload))

	// wg.Wait()
}

// func TestKqueueConsole_ReadAndWrite_Large(t *testing.T) {
// 	// 1) Allocate PTY pair
// 	master, slave, err := pty.Open()
// 	require.NoError(t, err)
// 	defer master.Close()
// 	defer slave.Close()

// 	// 2) Wrap master in kqueue console
// 	console, _, err := console.NewPtyFromFile(master)
// 	require.NoError(t, err)
// 	defer console.Close()

// 	kq, err := NewKqueuer()
// 	require.NoError(t, err)
// 	defer kq.Close()

// 	kqConsole, err := kq.Add(console)
// 	require.NoError(t, err)

// 	// 3) Start kqueue event loop
// 	go kq.Wait()

// 	//  4. Exercise shim → guest (Write path)
// 	//     Read from the ORIGINAL slave FD, not a reopened path
// 	var wg sync.WaitGroup
// 	var got []byte
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		buf := make([]byte, 4)
// 		// Read exactly 4 bytes ("ping")
// 		_, _ = io.ReadFull(slave, buf)
// 		got = buf
// 	}()

// 	// 5) Invoke the Write path
// 	n, err := kqConsole.Write([]byte("ping"))
// 	require.NoError(t, err)
// 	require.Equal(t, 4, n)

// 	// 6) Wait for goroutine to finish and verify
// 	wg.Wait()
// 	require.Equal(t, "ping", string(got))
// }
