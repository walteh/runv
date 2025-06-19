package goruncruntime

import (
	"context"

	"github.com/walteh/runm/core/runc/runtime"
)

var _ runtime.EventHandler = (*GoRuncEventHandler)(nil)

type GoRuncEventHandler struct {
}

func NewGoRuncEventHandler() runtime.EventHandler {
	return &GoRuncEventHandler{}
}

// Publish implements runtime.EventHandler.
func (r *GoRuncEventHandler) Publish(ctx context.Context, event *runtime.PublishEvent) error {
	// No-op implementation for now
	// In a real implementation, this would publish the event to some event bus
	return nil
}

// Receive implements runtime.EventHandler.
func (r *GoRuncEventHandler) Receive(ctx context.Context) (<-chan *runtime.PublishEvent, error) {
	ch := make(chan *runtime.PublishEvent)
	go func() {
		<-ctx.Done()
		close(ch)
	}()

	return ch, nil
}
