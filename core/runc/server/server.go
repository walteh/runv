package server

import (
	"context"

	runc "github.com/containerd/go-runc"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	runvv1 "github.com/walteh/runv/proto/v1"
	"google.golang.org/grpc"
)

// Server implements the RuncServiceServer interface.
//
//go:opts
type Server struct {
	runc *runc.Runc
}

var _ runvv1.RuncServiceServer = (*Server)(nil)

// NewRuncServer creates a new RuncServiceServer with default options.
func NewRuncServer() *Server {
	return NewRuncServerWithOptions(&runc.Runc{
		Command: runc.DefaultCommand,
	})
}

// NewRuncServerWithOptions creates a new RuncServiceServer with the given runc client.
func NewRuncServerWithOptions(runcCmd *runc.Runc) *Server {
	s := NewServer(
		WithRunc(runcCmd),
	)
	return &s
}

// Register registers the server with a gRPC server.
func (s *Server) Register(grpcServer *grpc.Server) {
	runvv1.RegisterRuncServiceServer(grpcServer, s)
}

// Ping implements the RuncServiceServer Ping method.
func (s *Server) Ping(ctx context.Context, req *runvv1.PingRequest) (*runvv1.PingResponse, error) {
	return &runvv1.PingResponse{}, nil
}

// List implements the RuncServiceServer List method.
func (s *Server) List(ctx context.Context, req *runvv1.RuncListRequest) (*runvv1.RuncListResponse, error) {
	resp := &runvv1.RuncListResponse{}

	// Create a copy of the runc client with the requested root if specified
	r := &runc.Runc{
		Command:       s.runc.Command,
		Root:          req.GetRoot(),
		Debug:         s.runc.Debug,
		Log:           s.runc.Log,
		LogFormat:     s.runc.LogFormat,
		PdeathSignal:  s.runc.PdeathSignal,
		Setpgid:       s.runc.Setpgid,
		Criu:          s.runc.Criu,
		SystemdCgroup: s.runc.SystemdCgroup,
		Rootless:      s.runc.Rootless,
	}

	containers, err := r.List(ctx)
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	runcContainers := make([]*runvv1.RuncContainer, len(containers))
	for i, container := range containers {
		c := &runvv1.RuncContainer_builder{
			Id:               container.ID,
			Pid:              int32(container.Pid),
			Status:           container.Status,
			Bundle:           container.Bundle,
			Rootfs:           container.Rootfs,
			CreatedTimestamp: container.Created.UnixNano(),
			Annotations:      container.Annotations,
		}
		runcContainers[i] = c.Build()
	}

	resp.SetContainers(runcContainers)
	return resp, nil
}

// State implements the RuncServiceServer State method.
func (s *Server) State(ctx context.Context, req *runvv1.RuncStateRequest) (*runvv1.RuncStateResponse, error) {
	resp := &runvv1.RuncStateResponse{}

	container, err := s.runc.State(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	c := &runvv1.RuncContainer{}
	c.SetId(container.ID)
	c.SetPid(int32(container.Pid))
	c.SetStatus(container.Status)
	c.SetBundle(container.Bundle)
	c.SetRootfs(container.Rootfs)
	c.SetCreatedTimestamp(container.Created.UnixNano())
	c.SetAnnotations(container.Annotations)

	resp.SetContainer(c)
	return resp, nil
}

// Create implements the RuncServiceServer Create method.
func (s *Server) Create(ctx context.Context, req *runvv1.RuncCreateRequest) (*runvv1.RuncCreateResponse, error) {
	resp := &runvv1.RuncCreateResponse{}

	opts := &runc.CreateOpts{
		PidFile:      req.GetPidFile(),
		NoPivot:      req.GetNoPivot(),
		NoNewKeyring: req.GetNoNewKeyring(),
		ExtraArgs:    req.GetExtraArgs(),
	}

	if req.GetDetach() {
		opts.Detach = true
	}

	err := s.runc.Create(ctx, req.GetId(), req.GetBundle(), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Start implements the RuncServiceServer Start method.
func (s *Server) Start(ctx context.Context, req *runvv1.RuncStartRequest) (*runvv1.RuncStartResponse, error) {
	resp := &runvv1.RuncStartResponse{}

	err := s.runc.Start(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Run implements the RuncServiceServer Run method.
func (s *Server) Run(ctx context.Context, req *runvv1.RuncRunRequest) (*runvv1.RuncRunResponse, error) {
	resp := &runvv1.RuncRunResponse{}

	opts := &runc.CreateOpts{
		PidFile:      req.GetPidFile(),
		NoPivot:      req.GetNoPivot(),
		NoNewKeyring: req.GetNoNewKeyring(),
		ExtraArgs:    req.GetExtraArgs(),
	}

	if req.GetDetach() {
		opts.Detach = true
	}

	status, err := s.runc.Run(ctx, req.GetId(), req.GetBundle(), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	resp.SetStatus(int32(status))
	return resp, nil
}

// Delete implements the RuncServiceServer Delete method.
func (s *Server) Delete(ctx context.Context, req *runvv1.RuncDeleteRequest) (*runvv1.RuncDeleteResponse, error) {
	resp := &runvv1.RuncDeleteResponse{}

	opts := &runc.DeleteOpts{
		Force:     req.GetForce(),
		ExtraArgs: req.GetExtraArgs(),
	}

	err := s.runc.Delete(ctx, req.GetId(), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Kill implements the RuncServiceServer Kill method.
func (s *Server) Kill(ctx context.Context, req *runvv1.RuncKillRequest) (*runvv1.RuncKillResponse, error) {
	resp := &runvv1.RuncKillResponse{}

	opts := &runc.KillOpts{
		All:       req.GetAll(),
		ExtraArgs: req.GetExtraArgs(),
	}

	err := s.runc.Kill(ctx, req.GetId(), int(req.GetSignal()), opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Stats implements the RuncServiceServer Stats method.
func (s *Server) Stats(ctx context.Context, req *runvv1.RuncStatsRequest) (*runvv1.RuncStatsResponse, error) {
	resp := &runvv1.RuncStatsResponse{}

	stats, err := s.runc.Stats(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	runcStats := convertStats(stats)
	resp.SetStats(runcStats)
	return resp, nil
}

// Pause implements the RuncServiceServer Pause method.
func (s *Server) Pause(ctx context.Context, req *runvv1.RuncPauseRequest) (*runvv1.RuncPauseResponse, error) {
	resp := &runvv1.RuncPauseResponse{}

	err := s.runc.Pause(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Resume implements the RuncServiceServer Resume method.
func (s *Server) Resume(ctx context.Context, req *runvv1.RuncResumeRequest) (*runvv1.RuncResumeResponse, error) {
	resp := &runvv1.RuncResumeResponse{}

	err := s.runc.Resume(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
	}
	return resp, nil
}

// Ps implements the RuncServiceServer Ps method.
func (s *Server) Ps(ctx context.Context, req *runvv1.RuncPsRequest) (*runvv1.RuncPsResponse, error) {
	resp := &runvv1.RuncPsResponse{}

	pids, err := s.runc.Ps(ctx, req.GetId())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	pidsList := make([]int32, len(pids))
	for i, pid := range pids {
		pidsList[i] = int32(pid)
	}

	resp.SetPids(pidsList)
	return resp, nil
}

// Top implements the RuncServiceServer Top method.
func (s *Server) Top(ctx context.Context, req *runvv1.RuncTopRequest) (*runvv1.RuncTopResponse, error) {
	resp := &runvv1.RuncTopResponse{}

	topResults, err := s.runc.Top(ctx, req.GetId(), req.GetPsOptions())
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	resp.SetHeaders(topResults.Headers)

	processesList := make([]*runvv1.RuncProcessData, len(topResults.Processes))
	for i, process := range topResults.Processes {
		p := &runvv1.RuncProcessData{}
		p.SetData(process)
		processesList[i] = p
	}

	resp.SetProcesses(processesList)
	return resp, nil
}

// Version implements the RuncServiceServer Version method.
func (s *Server) Version(ctx context.Context, req *runvv1.RuncVersionRequest) (*runvv1.RuncVersionResponse, error) {
	resp := &runvv1.RuncVersionResponse{}

	version, err := s.runc.Version(ctx)
	if err != nil {
		resp.SetGoError(err.Error())
		return resp, nil
	}

	resp.SetRunc(version.Runc)
	resp.SetCommit(version.Commit)
	resp.SetSpec(version.Spec)

	return resp, nil
}

// Exec implements the RuncServiceServer Exec method.
func (s *Server) Exec(ctx context.Context, req *runvv1.RuncExecRequest) (*runvv1.RuncExecResponse, error) {
	resp := &runvv1.RuncExecResponse{}

	if req.GetSpec() == nil {
		resp.SetGoError("spec is required")
		return resp, nil
	}

	processSpec := specs.Process{
		Terminal: req.GetSpec().GetTerminal() != "",
		Cwd:      req.GetSpec().GetCwd(),
		Args:     req.GetSpec().GetArgs(),
		Env:      req.GetSpec().GetEnv(),
		User: specs.User{
			UID: uint32(req.GetSpec().GetUserUid()),
			GID: uint32(req.GetSpec().GetUserGid()),
		},
	}

	// Convert additional groups
	for _, gid := range req.GetSpec().GetAdditionalGids() {
		processSpec.User.AdditionalGids = append(processSpec.User.AdditionalGids, uint32(gid))
	}

	opts := &runc.ExecOpts{
		PidFile:   req.GetPidFile(),
		ExtraArgs: req.GetExtraArgs(),
	}

	if req.GetDetach() {
		opts.Detach = true
	}

	err := s.runc.Exec(ctx, req.GetId(), processSpec, opts)
	if err != nil {
		resp.SetGoError(err.Error())
	}

	return resp, nil
}

// convertStats converts runc.Stats to runvv1.RuncStats
func convertStats(stats *runc.Stats) *runvv1.RuncStats {
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
	blkio.SetIoServiceBytesRecursive(convertBlkioEntries(stats.Blkio.IoServiceBytesRecursive))
	blkio.SetIoServicedRecursive(convertBlkioEntries(stats.Blkio.IoServicedRecursive))
	blkio.SetIoQueuedRecursive(convertBlkioEntries(stats.Blkio.IoQueuedRecursive))
	blkio.SetIoServiceTimeRecursive(convertBlkioEntries(stats.Blkio.IoServiceTimeRecursive))
	blkio.SetIoWaitTimeRecursive(convertBlkioEntries(stats.Blkio.IoWaitTimeRecursive))
	blkio.SetIoMergedRecursive(convertBlkioEntries(stats.Blkio.IoMergedRecursive))
	blkio.SetIoTimeRecursive(convertBlkioEntries(stats.Blkio.IoTimeRecursive))
	blkio.SetSectorsRecursive(convertBlkioEntries(stats.Blkio.SectorsRecursive))

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
func convertBlkioEntries(entries []runc.BlkioEntry) []*runvv1.RuncBlkioEntry {
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
