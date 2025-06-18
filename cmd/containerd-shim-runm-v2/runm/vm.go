package runm

import (
	"context"
	"log/slog"

	"github.com/containerd/containerd/v2/pkg/shim"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/walteh/run"
	"github.com/walteh/runm/core/runc/oom"
	"github.com/walteh/runm/core/runc/runtime"
	"github.com/walteh/runm/core/virt/vmm"
)

var _ runtime.Runtime = (*RunmVMRuntime[vmm.VirtualMachine])(nil)
var _ runtime.RuntimeExtras = (*RunmVMRuntime[vmm.VirtualMachine])(nil)
var _ runtime.CgroupAdapter = (*RunmVMRuntime[vmm.VirtualMachine])(nil)
var _ runtime.EventHandler = (*RunmVMRuntime[vmm.VirtualMachine])(nil)
var _ runtime.GuestManagement = (*RunmVMRuntime[vmm.VirtualMachine])(nil)

var _ run.Runnable = (*RunmVMRuntime[vmm.VirtualMachine])(nil)

type RunmVMRuntime[VM vmm.VirtualMachine] struct {
	runtime.Runtime
	runtime.RuntimeExtras
	runtime.CgroupAdapter
	runtime.EventHandler
	runtime.GuestManagement

	spec       *specs.Spec
	vm         *vmm.RunningVM[VM]
	oomWatcher *oom.Watcher

	runGroup *run.Group
}

func NewRunmVMRuntime[VM vmm.VirtualMachine](ctx context.Context, hpv vmm.Hypervisor[VM], publisher shim.Publisher, cfg vmm.OCIVMConfig) (*RunmVMRuntime[VM], error) {

	runGroup := run.New()

	vm, err := vmm.NewOCIVirtualMachine(ctx, hpv, cfg)
	if err != nil {
		return nil, err
	}

	srv, err := vm.GuestService(ctx)
	if err != nil {
		return nil, err
	}

	ep := oom.NewWatcher(publisher, srv)

	runGroup.Always(ep)

	return &RunmVMRuntime[VM]{
		vm:              vm,
		oomWatcher:      ep,
		spec:            cfg.Spec,
		Runtime:         srv,
		RuntimeExtras:   srv,
		CgroupAdapter:   srv,
		EventHandler:    srv,
		GuestManagement: srv,
		runGroup:        runGroup,
	}, nil
}

// Alive implements run.Runnable.
func (r *RunmVMRuntime[VM]) Alive() bool {
	panic("unimplemented")
}

// Close implements run.Runnable.
func (r *RunmVMRuntime[VM]) Close(context.Context) error {
	panic("unimplemented")
}

// Fields implements run.Runnable.
func (r *RunmVMRuntime[VM]) Fields() []slog.Attr {
	panic("unimplemented")
}

// Name implements run.Runnable.
func (r *RunmVMRuntime[VM]) Name() string {
	panic("unimplemented")
}

// Run implements run.Runnable.
// Subtle: this method shadows the method (RuntimeExtras).Run of RunmVMRuntime.RuntimeExtras.
func (r *RunmVMRuntime[VM]) Run(context.Context) error {
	panic("unimplemented")
}
