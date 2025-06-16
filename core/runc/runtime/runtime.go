package runtime

import (
	"context"
	"io"
	"net"
	"os/exec"
	"time"

	"github.com/containerd/console"
	gorunc "github.com/containerd/go-runc"
	"github.com/opencontainers/runtime-spec/specs-go"
)

type RuntimeOptions struct {
	Root          string
	Path          string
	Namespace     string
	Runtime       string
	SystemdCgroup bool
}

type RuntimeCreator interface {
	Create(ctx context.Context, opts *RuntimeOptions) Runtime
}

type SocketAllocator interface {
	AllocateSocket(ctx context.Context) (AllocatedSocket, error)
	BindConsoleToSocket(ctx context.Context, consoleReferenceId ConsoleSocket, socketReferenceId AllocatedSocket) error
	BindIOToSockets(ctx context.Context, ioReferenceId IO, socketReferenceIds [3]AllocatedSocket) error
}

//go:mock
type Runtime interface {
	// io: yes
	// ✅
	NewPipeIO(ioUID, ioGID int, opts ...gorunc.IOOpt) (IO, error)
	// io: yes
	NewTempConsoleSocket(ctx context.Context) (ConsoleSocket, error)
	// io: yes
	// ✅
	NewNullIO() (IO, error)
	// io: yes
	// console: yes
	// channel: yes
	// fd: yes
	Create(ctx context.Context, id, bundle string, opts *gorunc.CreateOpts) error
	// io: yes
	// console: yes
	// channel: yes
	Exec(ctx context.Context, id string, spec specs.Process, opts *gorunc.ExecOpts) error
	// fd: yes
	Checkpoint(ctx context.Context, id string, opts *gorunc.CheckpointOpts, actions ...gorunc.CheckpointAction) error
	// io: yes
	Restore(ctx context.Context, id, bundle string, opts *gorunc.RestoreOpts) (int, error)
	// ✅
	Kill(ctx context.Context, id string, signal int, opts *gorunc.KillOpts) error
	Start(ctx context.Context, id string) error
	// ✅
	Delete(ctx context.Context, id string, opts *gorunc.DeleteOpts) error
	// ✅
	Update(ctx context.Context, id string, resources *specs.LinuxResources) error
	LogFilePath() string
	Pause(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) error
	Ps(ctx context.Context, id string) ([]int, error)
	ReadPidFile(path string) (int, error)
}

type ConsoleSocket interface {
	ReceiveMaster() (console.Console, error)
	Path() string
	Close() error
}

// Platform handles platform-specific behavior that may differs across
// // platform implementations
// type Platform interface {
// 	CopyConsole(ctx context.Context, console Console, id, stdin, stdout, stderr string, wg *sync.WaitGroup) (Console, error)
// 	ShutdownConsole(ctx context.Context, console Console) error
// 	Close() error
// }

type RuntimeConsole interface {
	console.File

	// Resize resizes the console to the provided window size
	Resize(console.WinSize) error
	// ResizeFrom resizes the calling console to the size of the
	// provided console
	ResizeFrom(RuntimeConsole) error
	// SetRaw sets the console in raw mode
	SetRaw() error
	// DisableEcho disables echo on the console
	DisableEcho() error
	// Reset restores the console to its original state
	Reset() error
	// Size returns the window size of the console
	Size() (console.WinSize, error)
}

type IO interface {
	Stdin() io.WriteCloser
	Stdout() io.ReadCloser
	Stderr() io.ReadCloser
	Close() error
	// unused
	Set(cmd *exec.Cmd)
}

// RuncLibrary defines an interface for interacting with runc containers.
// This interface mirrors the functionality provided by the go-runc package
// to allow for easy mocking and testing.
type RuntimeExtras interface {
	// State returns the state of the container for the given id.
	State(context.Context, string) (*gorunc.Container, error)

	// Run creates and starts a container and returns its pid.
	Run(context.Context, string, string, *gorunc.CreateOpts) (int, error)

	// Stats returns runtime specific metrics for a container.
	Stats(context.Context, string) (*gorunc.Stats, error)

	// Events returns events for the container.
	Events(context.Context, string, time.Duration) (chan *gorunc.Event, error)

	// List lists all containers.
	List(context.Context) ([]*gorunc.Container, error)

	// Version returns the version of runc.
	Version(context.Context) (gorunc.Version, error)

	Top(context.Context, string, string) (*gorunc.TopResults, error)
}

type AllocatedSocketReference interface {
	ReferableByReferenceId
}

type AllocatedSocket interface {
	isAllocatedSocket()
	io.Closer
	Conn() *net.UnixConn
}

type VsockAllocatedSocket interface {
	AllocatedSocket
	Port() uint32
}

type UnixAllocatedSocket interface {
	AllocatedSocket
	Path() string
}

type ServerStateGetter interface {
	GetOpenIO(referenceId string) (IO, bool)
	GetOpenSocket(referenceId string) (AllocatedSocket, bool)
	GetOpenConsole(referenceId string) (ConsoleSocket, bool)
}

type ServerStateSetter interface {
	StoreOpenIO(referenceId string, io IO)
	StoreOpenSocket(referenceId string, socket AllocatedSocket)
	StoreOpenConsole(referenceId string, console ConsoleSocket)
}

type ReferableByReferenceId interface {
	GetReferenceId() string
}

type VsockProxier interface {
	Proxy(ctx context.Context, port uint32) (*net.UnixConn, string, error)
}

type VsockFdProxier interface {
	ProxyFd(ctx context.Context, port uint32) (*net.Conn, uintptr, error)
}
