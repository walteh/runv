package grpcruntime

import (
	"context"
	"log/slog"

	"github.com/walteh/runm/core/runc/runtime"
	runmv1 "github.com/walteh/runm/proto/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

var _ runtime.EventHandler = (*GRPCClientRuntime)(nil)

// Receive implements runtime.EventPublisher.
func (me *GRPCClientRuntime) Receive(ctx context.Context) (<-chan *runtime.PublishEvent, error) {
	ech := make(chan *runtime.PublishEvent)

	stream, err := me.eventPublisher.ReceiveEvents(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	go func() {
		defer close(ech)

		for {
			refId, err := stream.Recv()
			if err != nil {
				slog.Error("failed to receive event", "error", err)
				return
			}
			ech <- &runtime.PublishEvent{
				Topic: refId.GetTopic(),
				Data:  refId.GetRawJson(),
			}
		}
	}()

	return ech, nil
}

func (me *GRPCClientRuntime) Publish(ctx context.Context, event *runtime.PublishEvent) error {
	req := &runmv1.PublishEventRequest{}
	req.SetTopic(event.Topic)
	req.SetRawJson(event.Data)

	_, err := me.eventPublisher.PublishEvent(ctx, req)
	return err
}
