package vf

import (
	"fmt"

	"github.com/walteh/runm/core/virt/virtio"
)

func (vmConfig *vzVirtioDeviceApplier) applyRosettaShare(dev *virtio.RosettaShare) error {
	return fmt.Errorf("rosetta is unsupported on non-arm64 platforms")
}
