package client

import (
	runvv1 "github.com/walteh/runv/proto/v1"
)

// CreateOption is a functional option for configuring Create requests.
type CreateOption func(req *runvv1.RuncCreateRequest)

// WithDetach sets the detach option for Create requests.
func WithDetach(detach bool) CreateOption {
	return func(req *runvv1.RuncCreateRequest) {
		req.SetDetach(detach)
	}
}

// WithNoPivot sets the no pivot option for Create requests.
func WithNoPivot(noPivot bool) CreateOption {
	return func(req *runvv1.RuncCreateRequest) {
		req.SetNoPivot(noPivot)
	}
}

// WithNoNewKeyring sets the no new keyring option for Create requests.
func WithNoNewKeyring(noNewKeyring bool) CreateOption {
	return func(req *runvv1.RuncCreateRequest) {
		req.SetNoNewKeyring(noNewKeyring)
	}
}

// WithPidFile sets the pid file option for Create requests.
func WithPidFile(pidFile string) CreateOption {
	return func(req *runvv1.RuncCreateRequest) {
		req.SetPidFile(pidFile)
	}
}

// WithExtraArgs sets extra arguments for Create requests.
func WithExtraArgs(extraArgs ...string) CreateOption {
	return func(req *runvv1.RuncCreateRequest) {
		req.SetExtraArgs(extraArgs)
	}
}

// RunOption is a functional option for configuring Run requests.
type RunOption func(req *runvv1.RuncRunRequest)

// WithRunDetach sets the detach option for Run requests.
func WithRunDetach(detach bool) RunOption {
	return func(req *runvv1.RuncRunRequest) {
		req.SetDetach(detach)
	}
}

// WithRunNoPivot sets the no pivot option for Run requests.
func WithRunNoPivot(noPivot bool) RunOption {
	return func(req *runvv1.RuncRunRequest) {
		req.SetNoPivot(noPivot)
	}
}

// WithRunNoNewKeyring sets the no new keyring option for Run requests.
func WithRunNoNewKeyring(noNewKeyring bool) RunOption {
	return func(req *runvv1.RuncRunRequest) {
		req.SetNoNewKeyring(noNewKeyring)
	}
}

// WithRunPidFile sets the pid file option for Run requests.
func WithRunPidFile(pidFile string) RunOption {
	return func(req *runvv1.RuncRunRequest) {
		req.SetPidFile(pidFile)
	}
}

// WithRunExtraArgs sets extra arguments for Run requests.
func WithRunExtraArgs(extraArgs ...string) RunOption {
	return func(req *runvv1.RuncRunRequest) {
		req.SetExtraArgs(extraArgs)
	}
}

// ExecOption is a functional option for configuring Exec requests.
type ExecOption func(req *runvv1.RuncExecRequest)

// WithExecDetach sets the detach option for Exec requests.
func WithExecDetach(detach bool) ExecOption {
	return func(req *runvv1.RuncExecRequest) {
		req.SetDetach(detach)
	}
}

// WithExecPidFile sets the pid file option for Exec requests.
func WithExecPidFile(pidFile string) ExecOption {
	return func(req *runvv1.RuncExecRequest) {
		req.SetPidFile(pidFile)
	}
}

// WithExecExtraArgs sets extra arguments for Exec requests.
func WithExecExtraArgs(extraArgs ...string) ExecOption {
	return func(req *runvv1.RuncExecRequest) {
		req.SetExtraArgs(extraArgs)
	}
}

// NewProcessSpec creates a new process specification.
func NewProcessSpec(terminal, cwd string, args []string) *runvv1.RuncProcessSpec {
	spec := &runvv1.RuncProcessSpec{}
	spec.SetTerminal(terminal)
	spec.SetCwd(cwd)
	spec.SetArgs(args)
	return spec
}

// WithEnv adds environment variables to the process spec.
func WithEnv(env []string) func(*runvv1.RuncProcessSpec) {
	return func(spec *runvv1.RuncProcessSpec) {
		spec.SetEnv(env)
	}
}

// WithUser sets the user ID and group ID for the process spec.
func WithUser(uid, gid int32) func(*runvv1.RuncProcessSpec) {
	return func(spec *runvv1.RuncProcessSpec) {
		spec.SetUserUid(uid)
		spec.SetUserGid(gid)
	}
}

// WithAdditionalGids adds additional group IDs for the process spec.
func WithAdditionalGids(gids []int32) func(*runvv1.RuncProcessSpec) {
	return func(spec *runvv1.RuncProcessSpec) {
		spec.SetAdditionalGids(gids)
	}
}
