package virt

import (
	"context"

	"github.com/containers/common/pkg/strongunits"
	"github.com/pkg/errors"
	"github.com/walteh/runm/core/runc/runtime"
	"github.com/walteh/runm/core/virt/vmm"
)

type RunmVMRuntimeCreator[VM vmm.VirtualMachine] struct {
	// publisher events.Publisher
	hpv       vmm.Hypervisor[VM]
	maxMemory strongunits.StorageUnits
	vcpus     int
}

func (me *RunmVMRuntimeCreator[VM]) Create(ctx context.Context, opts *runtime.RuntimeOptions) (runtime.Runtime, error) {
	vm, err := NewRunmVMRuntime(ctx, me.hpv, opts, me.maxMemory, me.vcpus)
	if err != nil {
		return nil, errors.Errorf("failed to create VM: %w", err)
	}
	return vm, nil
}

func NewRunmVMRuntimeCreator[VM vmm.VirtualMachine](hpv vmm.Hypervisor[VM], maxMemory strongunits.StorageUnits, vcpus int) *RunmVMRuntimeCreator[VM] {
	return &RunmVMRuntimeCreator[VM]{
		hpv:       hpv,
		maxMemory: maxMemory,
		vcpus:     vcpus,
	}
}
