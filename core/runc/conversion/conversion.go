package conversion

import (
	"context"
	"encoding/json"
	"os"

	gorunc "github.com/containerd/go-runc"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/walteh/runv/core/runc/runtime"
	runvv1 "github.com/walteh/runv/proto/v1"
)

func ConvertStatsOut(stats *runvv1.RuncStats) *gorunc.Stats {
	if stats == nil {
		return nil
	}

	panic("unimplemented: ConvertStatsOut")
}

// convertStats converts runc.Stats to runvv1.RuncStats
func ConvertStatsIn(stats *gorunc.Stats) *runvv1.RuncStats {
	if stats == nil {
		return nil
	}

	runcStats := &runvv1.RuncStats{}

	// CPU stats
	cpu := &runvv1.RuncCpu{}

	// CPU Usage
	usage := &runvv1.RuncCpuUsage{}
	usage.SetTotal(stats.Cpu.Usage.Total)
	usage.SetPercpu(stats.Cpu.Usage.Percpu)
	usage.SetKernel(stats.Cpu.Usage.Kernel)
	usage.SetUser(stats.Cpu.Usage.User)
	cpu.SetUsage(usage)

	// CPU Throttling
	throttling := &runvv1.RuncThrottling{}
	throttling.SetPeriods(stats.Cpu.Throttling.Periods)
	throttling.SetThrottledPeriods(stats.Cpu.Throttling.ThrottledPeriods)
	throttling.SetThrottledTime(stats.Cpu.Throttling.ThrottledTime)
	cpu.SetThrottling(throttling)

	runcStats.SetCpu(cpu)

	// Memory
	memory := &runvv1.RuncMemory{}
	memory.SetCache(stats.Memory.Cache)

	// Memory Usage
	usageEntry := &runvv1.RuncMemoryEntry{}
	usageEntry.SetLimit(stats.Memory.Usage.Limit)
	usageEntry.SetUsage(stats.Memory.Usage.Usage)
	usageEntry.SetMax(stats.Memory.Usage.Max)
	usageEntry.SetFailcnt(stats.Memory.Usage.Failcnt)
	memory.SetUsage(usageEntry)

	// Memory Swap
	swapEntry := &runvv1.RuncMemoryEntry{}
	swapEntry.SetLimit(stats.Memory.Swap.Limit)
	swapEntry.SetUsage(stats.Memory.Swap.Usage)
	swapEntry.SetMax(stats.Memory.Swap.Max)
	swapEntry.SetFailcnt(stats.Memory.Swap.Failcnt)
	memory.SetSwap(swapEntry)

	// Memory Kernel
	kernelEntry := &runvv1.RuncMemoryEntry{}
	kernelEntry.SetLimit(stats.Memory.Kernel.Limit)
	kernelEntry.SetUsage(stats.Memory.Kernel.Usage)
	kernelEntry.SetMax(stats.Memory.Kernel.Max)
	kernelEntry.SetFailcnt(stats.Memory.Kernel.Failcnt)
	memory.SetKernel(kernelEntry)

	// Memory KernelTCP
	kernelTCPEntry := &runvv1.RuncMemoryEntry{}
	kernelTCPEntry.SetLimit(stats.Memory.KernelTCP.Limit)
	kernelTCPEntry.SetUsage(stats.Memory.KernelTCP.Usage)
	kernelTCPEntry.SetMax(stats.Memory.KernelTCP.Max)
	kernelTCPEntry.SetFailcnt(stats.Memory.KernelTCP.Failcnt)
	memory.SetKernelTcp(kernelTCPEntry)

	// Memory Raw
	memory.SetRaw(stats.Memory.Raw)

	runcStats.SetMemory(memory)

	// PIDs
	pids := &runvv1.RuncPids{}
	pids.SetCurrent(stats.Pids.Current)
	pids.SetLimit(stats.Pids.Limit)
	runcStats.SetPids(pids)

	// Blkio
	blkio := &runvv1.RuncBlkio{}

	// Convert all blkio entries
	blkio.SetIoServiceBytesRecursive(ConvertBlkioEntriesIn(stats.Blkio.IoServiceBytesRecursive))
	blkio.SetIoServicedRecursive(ConvertBlkioEntriesIn(stats.Blkio.IoServicedRecursive))
	blkio.SetIoQueuedRecursive(ConvertBlkioEntriesIn(stats.Blkio.IoQueuedRecursive))
	blkio.SetIoServiceTimeRecursive(ConvertBlkioEntriesIn(stats.Blkio.IoServiceTimeRecursive))
	blkio.SetIoWaitTimeRecursive(ConvertBlkioEntriesIn(stats.Blkio.IoWaitTimeRecursive))
	blkio.SetIoMergedRecursive(ConvertBlkioEntriesIn(stats.Blkio.IoMergedRecursive))
	blkio.SetIoTimeRecursive(ConvertBlkioEntriesIn(stats.Blkio.IoTimeRecursive))
	blkio.SetSectorsRecursive(ConvertBlkioEntriesIn(stats.Blkio.SectorsRecursive))

	runcStats.SetBlkio(blkio)

	// Hugetlb
	if stats.Hugetlb != nil {
		hugetlbMap := make(map[string]*runvv1.RuncHugetlb)
		for k, v := range stats.Hugetlb {
			hugetlb := &runvv1.RuncHugetlb{}
			hugetlb.SetUsage(v.Usage)
			hugetlb.SetMax(v.Max)
			hugetlb.SetFailcnt(v.Failcnt)
			hugetlbMap[k] = hugetlb
		}
		runcStats.SetHugetlb(hugetlbMap)
	}

	// NetworkInterfaces
	if len(stats.NetworkInterfaces) > 0 {
		networkInterfaces := make([]*runvv1.RuncNetworkInterface, len(stats.NetworkInterfaces))
		for i, ni := range stats.NetworkInterfaces {
			netIf := &runvv1.RuncNetworkInterface{}
			netIf.SetName(ni.Name)
			netIf.SetRxBytes(ni.RxBytes)
			netIf.SetRxPackets(ni.RxPackets)
			netIf.SetRxErrors(ni.RxErrors)
			netIf.SetRxDropped(ni.RxDropped)
			netIf.SetTxBytes(ni.TxBytes)
			netIf.SetTxPackets(ni.TxPackets)
			netIf.SetTxErrors(ni.TxErrors)
			netIf.SetTxDropped(ni.TxDropped)
			networkInterfaces[i] = netIf
		}
		runcStats.SetNetworkInterfaces(networkInterfaces)
	}

	return runcStats
}

// convertBlkioEntries converts runc.BlkioEntry to runvv1.RuncBlkioEntry
func ConvertBlkioEntriesIn(entries []gorunc.BlkioEntry) []*runvv1.RuncBlkioEntry {
	if entries == nil {
		return nil
	}

	result := make([]*runvv1.RuncBlkioEntry, len(entries))
	for i, entry := range entries {
		blkioEntry := &runvv1.RuncBlkioEntry{}
		blkioEntry.SetMajor(entry.Major)
		blkioEntry.SetMinor(entry.Minor)
		blkioEntry.SetOp(entry.Op)
		blkioEntry.SetValue(entry.Value)
		result[i] = blkioEntry
	}
	return result
}

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

func ConvertIoIn(io *runvv1.RuncIO) gorunc.IO {
	return &PassThroughIO{}
}

func ConvertCreateOptsIn(opts *runvv1.RuncCreateOptions) (*gorunc.CreateOpts, error) {
	var err error
	files := make([]*os.File, len(opts.GetExtraFiles()))
	for i, file := range opts.GetExtraFiles() {
		files[i], err = os.Open(file)
		if err != nil {
			return nil, err
		}
	}

	return &gorunc.CreateOpts{
		PidFile:       opts.GetPidFile(),
		IO:            ConvertIoIn(opts.GetIo()),
		NoPivot:       opts.GetNoPivot(),
		NoNewKeyring:  opts.GetNoNewKeyring(),
		ConsoleSocket: NewPassThroughConsoleSocket(opts.GetConsoleSocket().GetPath()),
		Detach:        opts.GetDetach(),
		ExtraFiles:    files,
		Started:       make(chan int),
	}, nil
}

func ConvertCreateOptsOut(opts *gorunc.CreateOpts) *runvv1.RuncCreateOptions {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertExecOptsIn(opts *runvv1.RuncExecOptions) *gorunc.ExecOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertExecOptsOut(opts *gorunc.ExecOpts) *runvv1.RuncExecOptions {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertKillOptsIn(opts *runvv1.RuncKillOptions) *gorunc.KillOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertKillOptsOut(opts *gorunc.KillOpts) *runvv1.RuncKillOptions {
	panic(runtime.ReflectNotImplementedError())
}

// checkpoint in out

func ConvertCheckpointOptsIn(opts *runvv1.RuncCheckpointOptions) *gorunc.CheckpointOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertCheckpointOptsOut(opts *gorunc.CheckpointOpts) *runvv1.RuncCheckpointOptions {
	panic(runtime.ReflectNotImplementedError())
}

// restore in out
func ConvertRestoreOptsIn(opts *runvv1.RuncRestoreOptions) *gorunc.RestoreOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertRestoreOptsOut(opts *gorunc.RestoreOpts) *runvv1.RuncRestoreOptions {
	panic(runtime.ReflectNotImplementedError())
}

// checkpoint actions in out

func ConvertCheckpointActionsIn(actions []runvv1.RuncCheckpointAction) []gorunc.CheckpointAction {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertCheckpointActionsOut(actions []gorunc.CheckpointAction) []runvv1.RuncCheckpointAction {
	panic(runtime.ReflectNotImplementedError())
}

// delete in out
func ConvertDeleteOptsIn(opts *runvv1.RuncDeleteOptions) *gorunc.DeleteOpts {
	panic(runtime.ReflectNotImplementedError())
}

func ConvertDeleteOptsOut(opts *gorunc.DeleteOpts) *runvv1.RuncDeleteOptions {
	panic(runtime.ReflectNotImplementedError())
}

// linux resources in out

func ConvertLinuxResourcesIn(resources *runvv1.RuncLinuxResources) (*specs.LinuxResources, error) {
	var linuxResources specs.LinuxResources
	if err := json.Unmarshal(resources.GetRawJson(), &linuxResources); err != nil {
		return nil, err
	}
	return &linuxResources, nil
}

func ConvertLinuxResourcesOut(resources *specs.LinuxResources) (*runvv1.RuncLinuxResources, error) {
	rawJson, err := json.Marshal(resources)
	if err != nil {
		return nil, err
	}
	resp := &runvv1.RuncLinuxResources{}
	resp.SetRawJson(rawJson)
	return resp, nil
}

func ConvertProcessSpecIn(resources *runvv1.RuncProcessSpec) (*specs.Process, error) {
	var ProcessSpec specs.Process
	if err := json.Unmarshal(resources.GetRawJson(), &ProcessSpec); err != nil {
		return nil, err
	}
	return &ProcessSpec, nil
}

func ConvertProcessSpecOut(resources *specs.Process) (*runvv1.RuncProcessSpec, error) {
	rawJson, err := json.Marshal(resources)
	if err != nil {
		return nil, err
	}
	resp := &runvv1.RuncProcessSpec{}
	resp.SetRawJson(rawJson)
	return resp, nil
}
