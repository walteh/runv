package runv

import (
	"fmt"

	"github.com/containerd/console"
	"github.com/containerd/containerd/v2/pkg/stdio"
)

func NewPlatform() (stdio.Platform, error) {
	epoller, err := console.NewEpoller()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize epoller: %w", err)
	}
	go epoller.Wait()
	return newPlatform(epoller)
}
