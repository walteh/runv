package virt

import (
	"context"

	"github.com/containerd/containerd/events"
	"github.com/walteh/runm/core/runc/runtime"
	"github.com/walteh/runm/core/virt/vmm"
)

type RunmVMRuntimeCreator[VM vmm.VirtualMachine] struct {
	publisher events.Publisher
	hpv       vmm.Hypervisor[VM]
	cfg       vmm.ContainerizedVMConfig
}

func (me *RunmVMRuntimeCreator[VM]) Create(ctx context.Context, opts *runtime.RuntimeOptions) runtime.Runtime {

}
