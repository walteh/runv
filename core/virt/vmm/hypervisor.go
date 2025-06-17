package vmm

import (
	"context"
	"io"
	"net"
	"strings"
	"time"

	"gitlab.com/tozd/go/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/walteh/runm/core/gvnet"
	grpcruntime "github.com/walteh/runm/core/runc/runtime/grpc"
	"github.com/walteh/runm/core/virt/virtio"
	"github.com/walteh/runm/linux/constants"
	runmv1 "github.com/walteh/runm/proto/v1"
)

const (
	ExecVSockPort = 2019
)

//go:mock
type Hypervisor[VM VirtualMachine] interface {
	NewVirtualMachine(ctx context.Context, id string, opts *NewVMOptions, bootLoader virtio.Bootloader) (VM, error)
	OnCreate() <-chan VM
}

type RunningVM[VM VirtualMachine] struct {
	// streamExecReady bool
	// manager                *VSockManager
	runtime    *grpcruntime.GRPCClientRuntime
	bootloader virtio.Bootloader

	// streamexec   *streamexec.Client
	portOnHostIP uint16
	wait         chan error
	vm           VM
	netdev       gvnet.Proxy
	workingDir   string
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	// connStatus      <-chan VSockManagerState
	start time.Time
}

// func (r *RunningVM[VM]) guestService(ctx context.Context) harpoonv1.TTRPCGuestServiceClient {
// 	r.guestServiceConnectionMu.Lock()
// 	defer r.guestServiceConnectionMu.Unlock()

// 	if r.guestServiceConnection == nil {
// 		conn, err := r.vm.VSockConnect(ctx, uint32(constants.VsockPort))
// 		if err != nil {
// 			slog.Error("failed to dial vsock", "error", err)
// 			return nil
// 		}
// 		r.guestServiceConnection = harpoonv1.NewTTRPCGuestServiceClient(ttrpc.NewClient(conn))
// 	}

// 	return r.guestServiceConnection
// }

func connectToVsockWithRetry(ctx context.Context, vm VirtualMachine, port uint32) (net.Conn, error) {

	ticker := time.NewTicker(100 * time.Millisecond)
	timeout := time.NewTimer(3 * time.Second)
	defer ticker.Stop()
	defer timeout.Stop()

	lastError := error(errors.Errorf("initial error"))

	for {
		select {
		case <-ticker.C:
			conn, err := vm.VSockConnect(ctx, port)
			if err != nil {
				lastError = err
				continue
			}
			return conn, nil
		case <-timeout.C:
			return nil, errors.Errorf("timeout waiting for guest service connection: %w", lastError)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

func (r *RunningVM[VM]) GuestService(ctx context.Context) (*grpcruntime.GRPCClientRuntime, error) {
	if r.runtime != nil {
		return r.runtime, nil
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	timeout := time.NewTimer(3 * time.Second)
	defer ticker.Stop()
	defer timeout.Stop()

	lastError := error(errors.Errorf("initial error"))

	for {
		select {
		case <-ticker.C:
			conn, err := r.vm.VSockConnect(ctx, uint32(constants.RunmVsockPort))
			if err != nil {
				lastError = err
				continue
			}
			grpcConn, err := grpc.NewClient("runm-runtime",
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
					return conn, nil
				}),
			)
			r.runtime, err = grpcruntime.NewGRPCClientRuntimeFromConn(grpcConn)
			if err != nil {
				lastError = err
				continue
			}
			return r.runtime, nil
		case <-timeout.C:
			return nil, errors.Errorf("timeout waiting for guest service connection: %w", lastError)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// func NewRunningContainerdVM[VM VirtualMachine](ctx context.Context, vm VM, portOnHostIP uint16, start time.Time, workingDir string, ec1DataDir string, cfg *ContainerizedVMConfig) *RunningVM[VM] {

// 	return &RunningVM[VM]{
// 		start:                  start,
// 		vm:                     vm,
// 		stdin:                  cfg.StdinReader,
// 		stdout:                 cfg.StdoutWriter,
// 		stderr:                 cfg.StderrWriter,
// 		portOnHostIP:           portOnHostIP,
// 		wait:                   make(chan error, 1),
// 		manager:                nil,
// 		guestServiceConnection: nil,
// 		streamexec:             nil,
// 		workingDir:             workingDir,
// 		netdev:                 nil,
// 	}
// }

func (r *RunningVM[VM]) ForwardStdio(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	return ForwardStdio(ctx, r.vm, stdin, stdout, stderr)
}

// func NewRunningVM[VM VirtualMachine](ctx context.Context, vm VM, portOnHostIP uint16, start time.Time) *RunningVM[VM] {

// 	transporz := NewVSockManager(func(ctx context.Context) (io.ReadWriteCloser, error) {
// 		return vm.VSockConnect(ctx, uint32(ExecVSockPort))
// 	})

// 	tfunc := transport.NewFunctionTransport(func() (io.ReadWriteCloser, error) {
// 		slog.Info("dialing vm transport")
// 		conn, err := transporz.Dial(ctx)
// 		if err != nil {
// 			slog.Error("failed to dial vm transport", "error", err)
// 			return nil, errors.Errorf("dialing vm transport: %w", err)
// 		}
// 		slog.Info("dialed vm transport")
// 		return conn, nil
// 	}, nil)

// 	client := streamexec.NewClient(tfunc, func(conn io.ReadWriter) protocol.Protocol {
// 		return protocol.NewFramedProtocol(conn)
// 	})

// 	go func() {
// 		slog.Info("dialing vm")
// 		err := client.Connect(ctx)
// 		if err != nil {
// 			slog.Error("failed to connect to vm", "error", err)
// 		} else {
// 			slog.Info("connected to vm")
// 		}

// 	}()

// 	// connStatus := transporz.AddStateNotifier()

// 	return &RunningVM[VM]{
// 		start:   start,
// 		vm:      vm,
// 		manager: transporz,
// 		// connStatus:      connStatus,
// 		portOnHostIP: portOnHostIP,
// 		wait:         make(chan error, 1),
// 		// streamExecReady: false,
// 		streamexec: client,
// 	}
// }

func (r *RunningVM[VM]) WaitOnVmStopped() error {
	return <-r.wait
}

// func (r *RunningVM[VM]) WaitOnVMReadyToExec() <-chan struct{} {
// 	ch := make(chan struct{})

// 	if r.manager.State() == StateConnected {
// 		close(ch)
// 		return ch
// 	}
// 	check := r.manager.AddStateNotifier()
// 	go func() {
// 		defer close(check)
// 		for {
// 			select {
// 			case <-check:
// 				if r.manager.State() == StateConnected {
// 					close(ch)
// 				}
// 			}
// 		}
// 	}()
// 	return ch
// }

// func (r *RunningVM[VM]) WaitOnVMReady(ctx context.Context) <-chan struct{} {
// 	ch := make(chan struct{})

// 	cacheDir, err := host.EmphiricalVMCacheDir(ctx, r.vm.ID())

// 	// keep checking the ready file
// 	readyFile := filepath.Join(cacheDir, constants.ContainerReadyFile)
// 	dat, err := os.ReadFile(readyFile)
// 	if err != nil {
// 		slog.Error("problem reading ready file", "error", err)
// 		return ch
// 	}
// 	return ch
// }

func (r *RunningVM[VM]) VM() VM {
	return r.vm
}

func (r *RunningVM[VM]) PortOnHostIP() uint16 {
	return r.portOnHostIP
}

func (r *RunningVM[VM]) RunCommandSimple(ctx context.Context, command string) ([]byte, []byte, int64, error) {
	guestService, err := r.GuestService(ctx)
	if err != nil {
		return nil, nil, 0, errors.Errorf("getting guest service: %w", err)
	}

	fields := strings.Fields(command)

	argc := fields[0]
	argv := []string{}
	for _, field := range fields[1:] {
		argv = append(argv, field)
	}

	// req, err := harpoonv1.NewValidatedRunRequest(func(b *harpoonv1.RunRequest_builder) {
	// 	// b.Stdin = stdinData
	// })
	req, err := runmv1.NewGuestRunCommandRequestE(&runmv1.GuestRunCommandRequest_builder{
		Argc:    argc,
		Argv:    argv,
		EnvVars: map[string]string{},
		Stdin:   []byte{},
	})
	if err != nil {
		return nil, nil, 0, err
	}

	exec, err := guestService.Management().GuestRunCommand(ctx, req)
	if err != nil {
		return nil, nil, 0, err
	}

	return exec.GetStdout(), exec.GetStderr(), int64(exec.GetExitCode()), nil
}
