package runtime

import (
	"context"
	"time"

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

//go:mock
type Runtime interface {
	Create(ctx context.Context, id, bundle string, opts *gorunc.CreateOpts) error
	Exec(ctx context.Context, id string, spec specs.Process, opts *gorunc.ExecOpts) error
	Kill(ctx context.Context, id string, signal int, opts *gorunc.KillOpts) error
	Checkpoint(ctx context.Context, id string, opts *gorunc.CheckpointOpts, actions ...gorunc.CheckpointAction) error
	Restore(ctx context.Context, id, bundle string, opts *gorunc.RestoreOpts) (int, error)
	Start(ctx context.Context, id string) error
	Delete(ctx context.Context, id string, opts *gorunc.DeleteOpts) error
	Update(ctx context.Context, id string, resources *specs.LinuxResources) error
	LogFilePath() string
	Pause(ctx context.Context, id string) error
	Resume(ctx context.Context, id string) error
	Ps(ctx context.Context, id string) ([]int, error)
	NewTempConsoleSocket() (*gorunc.Socket, error)
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
