package oom

import (
	"context"
	"log/slog"

	"github.com/containerd/containerd/v2/core/events"
	"github.com/walteh/runm/core/runc/runtime"
	"gitlab.com/tozd/go/errors"

	eventstypes "github.com/containerd/containerd/api/events"
	coreruntime "github.com/containerd/containerd/v2/core/runtime"
)

// Watcher watches OOM events
type Watcher interface {
	Close() error
	Run(ctx context.Context) error
}

type watcher struct {
	publisher     events.Publisher
	cgroupAdapter runtime.CgroupAdapter
}

type item struct {
	id  string
	ev  runtime.CgroupEvent
	err error
}

func NewWatcher(publisher events.Publisher, cgroupAdapter runtime.CgroupAdapter) Watcher {
	return &watcher{
		publisher:     publisher,
		cgroupAdapter: cgroupAdapter,
	}
}

func (w *watcher) Close() error {
	return nil
}

func (w *watcher) Run(ctx context.Context) error {

	eventCh, errCh, err := w.cgroupAdapter.OpenEventChan(ctx)
	if err != nil {
		return errors.Errorf("failed to open event channel: %w", err)
	}

	lastOOMMap := make(map[string]uint64) // key: id, value: ev.OOM
	itemCh := make(chan item)

	defer func() {
		close(itemCh)
	}()

	go func() {
		for {
			i := item{id: "root"}
			select {
			case ev := <-eventCh:
				i.ev = ev
				itemCh <- i
			case err := <-errCh:
				// channel is closed when cgroup gets deleted
				if err != nil {
					i.err = err
					itemCh <- i
					// we no longer get any event/err when we got an err
					slog.Error("error from *cgroupsv2.Manager.EventChan", "error", err)
				}
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			close(itemCh)
			return ctx.Err()
		case i := <-itemCh:
			if i.err != nil {
				delete(lastOOMMap, i.id)
				continue
			}
			lastOOM := lastOOMMap[i.id]
			if i.ev.OOMKill > lastOOM {
				if err := w.publisher.Publish(ctx, coreruntime.TaskOOMEventTopic, &eventstypes.TaskOOM{
					ContainerID: i.id,
				}); err != nil {
					return errors.Errorf("failed to publish OOM event: %w", err)
				}
			}
			if i.ev.OOMKill > 0 {
				lastOOMMap[i.id] = i.ev.OOMKill
			}
		}
	}

}
