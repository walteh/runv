package library

import (
	"context"
	"time"

	runc "github.com/containerd/go-runc"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// RuncLibrary defines an interface for interacting with runc containers.
// This interface mirrors the functionality provided by the go-runc package
// to allow for easy mocking and testing.
type RuncLibrary interface {
	// State returns the state of the container for the given id.
	State(context.Context, string) (*runc.Container, error)

	// Create creates a new container and returns its pid.
	Create(context.Context, string, string, *runc.CreateOpts) error

	// Start starts a created container.
	Start(context.Context, string) error

	// Run creates and starts a container and returns its pid.
	Run(context.Context, string, string, *runc.CreateOpts) (int, error)

	// Delete deletes the container.
	Delete(context.Context, string, *runc.DeleteOpts) error

	// Kill sends the specified signal to the container.
	Kill(context.Context, string, int, *runc.KillOpts) error

	// Stats returns runtime specific metrics for a container.
	Stats(context.Context, string) (*runc.Stats, error)

	// Events returns events for the container.
	Events(context.Context, string, time.Duration) (chan *runc.Event, error)

	// Pause pauses the container.
	Pause(context.Context, string) error

	// Resume resumes the container.
	Resume(context.Context, string) error

	// Ps lists processes in the container.
	Ps(context.Context, string) ([]int, error)

	// Checkpoint checkpoints the container.
	Checkpoint(context.Context, string, *runc.CheckpointOpts, ...runc.CheckpointAction) error

	// Restore restores the container from a checkpoint.
	Restore(context.Context, string, string, *runc.RestoreOpts) (int, error)

	// Exec executes an additional process in the container.
	Exec(context.Context, string, specs.Process, *runc.ExecOpts) error

	// List lists all containers.
	List(context.Context) ([]*runc.Container, error)

	// Version returns the version of runc.
	Version(context.Context) (runc.Version, error)
}
