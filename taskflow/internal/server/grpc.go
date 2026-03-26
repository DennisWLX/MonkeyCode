package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/chaitin/MonkeyCode/taskflow/internal/backend"
	"github.com/chaitin/MonkeyCode/taskflow/internal/runner"
	"github.com/chaitin/MonkeyCode/taskflow/internal/store"
	pb "github.com/chaitin/MonkeyCode/taskflow/pkg/proto"
)

type GRPCServer struct {
	pb.UnimplementedRunnerServiceServer
	store         *store.RedisStore
	manager       *runner.Manager
	streamManager *runner.StreamManager
	logger        *slog.Logger
	backend       *backend.Client
}

func NewGRPCServer(s *store.RedisStore, m *runner.Manager, sm *runner.StreamManager, b *backend.Client, logger *slog.Logger) *GRPCServer {
	return &GRPCServer{
		store:         s,
		manager:       m,
		streamManager: sm,
		backend:       b,
		logger:        logger,
	}
}

func (s *GRPCServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	s.logger.Info("runner registering", "hostname", req.Hostname, "ip", req.Ip, "token", req.Token)

	token, err := s.backend.CheckToken(ctx, &backend.CheckTokenReq{
		Token: req.Token,
	})
	if err != nil {
		s.logger.Error("failed to check token", "error", err)
		return &pb.RegisterResponse{
			Success: false,
			Message: "invalid token",
		}, nil
	}

	s.logger.Info("token verified", "user_id", token.User.ID, "kind", token.Kind)

	runnerID := token.Token

	r := &store.Runner{
		ID:       runnerID,
		UserID:   token.User.ID,
		Hostname: req.Hostname,
		IP:       req.Ip,
		Status:   "online",
		LastSeen: time.Now().Unix(),
		Capacity: map[string]int64{
			"cores":  int64(req.Cores),
			"memory": req.Memory,
			"disk":   req.Disk,
		},
	}

	if err := s.store.RegisterRunner(ctx, r, 60*time.Second); err != nil {
		s.logger.Error("failed to register runner", "error", err)
		return nil, err
	}

	if err := s.store.AddUserRunner(ctx, token.User.ID, runnerID); err != nil {
		s.logger.Error("failed to add user runner", "error", err)
	}

	if err := s.manager.Register(ctx, runnerID, token.User.ID); err != nil {
		s.logger.Error("failed to register runner in manager", "error", err)
	}

	hostInfo := &backend.HostInfo{
		ID:         runnerID,
		UserID:     token.User.ID,
		Hostname:   req.Hostname,
		Name:       req.Hostname,
		Arch:       "amd64",
		OS:         "linux",
		Cores:      req.Cores,
		Memory:     uint64(req.Memory),
		Disk:       uint64(req.Disk),
		PublicIP:   req.Ip,
		InternalIP: req.Ip,
		CreatedAt:  time.Now().Unix(),
	}

	if err := s.backend.ReportHostInfo(ctx, hostInfo); err != nil {
		s.logger.Error("failed to report host info to backend", "error", err)
	} else {
		s.logger.Info("host info reported to backend", "runner_id", runnerID)
	}

	s.logger.Info("runner registered", "runner_id", runnerID, "user_id", token.User.ID)

	return &pb.RegisterResponse{
		RunnerId: runnerID,
		Success:  true,
		Message:  "registered successfully",
	}, nil
}

func (s *GRPCServer) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) (*pb.HeartbeatResponse, error) {
	s.manager.UpdateHeartbeat(req.RunnerId)

	r, err := s.store.GetRunner(ctx, req.RunnerId)
	if err != nil {
		s.logger.Warn("heartbeat from unknown runner", "runner_id", req.RunnerId)
		return &pb.HeartbeatResponse{Success: false}, nil
	}

	r.LastSeen = time.Now().Unix()
	if err := s.store.RegisterRunner(ctx, r, 60*time.Second); err != nil {
		s.logger.Error("failed to update runner heartbeat", "error", err)
		return nil, err
	}

	return &pb.HeartbeatResponse{Success: true}, nil
}

func (s *GRPCServer) CreateVM(ctx context.Context, req *pb.CreateVMRequest) (*pb.CreateVMResponse, error) {
	s.logger.Info("creating vm", "vm_id", req.VmId, "user_id", req.UserId)

	vm := &store.VM{
		ID:        req.VmId,
		UserID:    req.UserId,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
		ImageURL:  req.ImageUrl,
		GitURL:    req.GitUrl,
	}

	if err := s.store.SetVM(ctx, vm); err != nil {
		s.logger.Error("failed to create vm", "error", err)
		return nil, err
	}

	if err := s.store.AddUserVM(ctx, req.UserId, req.VmId); err != nil {
		s.logger.Error("failed to add user vm", "error", err)
	}

	return &pb.CreateVMResponse{
		VmId:    req.VmId,
		Success: true,
		Message: "vm created",
	}, nil
}

func (s *GRPCServer) DeleteVM(ctx context.Context, req *pb.DeleteVMRequest) (*pb.DeleteVMResponse, error) {
	s.logger.Info("deleting vm", "vm_id", req.VmId)

	if err := s.store.DeleteVM(ctx, req.VmId); err != nil {
		s.logger.Error("failed to delete vm", "error", err)
		return &pb.DeleteVMResponse{Success: false, Message: err.Error()}, nil
	}

	return &pb.DeleteVMResponse{Success: true, Message: "vm deleted"}, nil
}

func (s *GRPCServer) ListVMs(ctx context.Context, req *pb.ListVMsRequest) (*pb.ListVMsResponse, error) {
	s.logger.Debug("listing vms", "runner_id", req.RunnerId)

	return &pb.ListVMsResponse{Vms: []*pb.VMInfo{}}, nil
}

func (s *GRPCServer) GetVMInfo(ctx context.Context, req *pb.GetVMInfoRequest) (*pb.GetVMInfoResponse, error) {
	s.logger.Debug("getting vm info", "vm_id", req.VmId)

	vm, err := s.store.GetVM(ctx, req.VmId)
	if err != nil {
		return nil, err
	}

	return &pb.GetVMInfoResponse{
		Vm: &pb.VMInfo{
			Id:          vm.ID,
			ContainerId: vm.ContainerID,
			Status:      vm.Status,
			CreatedAt:   vm.CreatedAt,
		},
	}, nil
}

func (s *GRPCServer) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	s.logger.Info("creating task", "task_id", req.TaskId, "vm_id", req.VmId)

	task := &store.Task{
		ID:        req.TaskId,
		VMID:      req.VmId,
		Status:    "pending",
		Agent:     req.Model,
		CreatedAt: time.Now().Unix(),
	}

	if err := s.store.SetTask(ctx, task); err != nil {
		s.logger.Error("failed to create task", "error", err)
		return nil, err
	}

	return &pb.CreateTaskResponse{Success: true, Message: "task created"}, nil
}

func (s *GRPCServer) StopTask(ctx context.Context, req *pb.StopTaskRequest) (*pb.StopTaskResponse, error) {
	s.logger.Info("stopping task", "task_id", req.TaskId)

	task, err := s.store.GetTask(ctx, req.TaskId)
	if err != nil {
		return nil, err
	}

	task.Status = "stopped"
	if err := s.store.SetTask(ctx, task); err != nil {
		s.logger.Error("failed to stop task", "error", err)
		return nil, err
	}

	return &pb.StopTaskResponse{Success: true, Message: "task stopped"}, nil
}

func (s *GRPCServer) TerminalStream(stream pb.RunnerService_TerminalStreamServer) error {
	for {
		data, err := stream.Recv()
		if err != nil {
			return err
		}
		s.logger.Debug("terminal data", "vm_id", data.VmId, "terminal_id", data.TerminalId)
		if err := stream.Send(data); err != nil {
			return err
		}
	}
}

func (s *GRPCServer) ReportStream(req *pb.ReportRequest, stream pb.RunnerService_ReportStreamServer) error {
	for {
		entry := &pb.ReportEntry{
			TaskId:    req.TaskId,
			Source:    "taskflow",
			Timestamp: time.Now().Unix(),
			Data:      []byte("report entry"),
		}
		if err := stream.Send(entry); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
}

func (s *GRPCServer) CommandStream(stream pb.RunnerService_CommandStreamServer) error {
	ctx := stream.Context()

	firstMsg, err := stream.Recv()
	if err != nil {
		return err
	}

	ready := firstMsg.GetReady()
	if ready == nil {
		return errors.New("first message must be RunnerReady")
	}

	runnerID := ready.RunnerId
	s.logger.Info("runner stream connected", "runner_id", runnerID)

	rs := s.streamManager.Register(runnerID, stream)
	defer s.streamManager.Unregister(runnerID)

	errChan := make(chan error, 2)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case cmd, ok := <-rs.SendChan:
				if !ok {
					return
				}
				if err := stream.Send(cmd); err != nil {
					errChan <- err
					return
				}
			}
		}
	}()

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				errChan <- err
				return
			}

			switch m := msg.Message.(type) {
			case *pb.RunnerMessage_Result:
				s.handleCommandResult(ctx, m.Result)
			case *pb.RunnerMessage_Heartbeat:
				s.handleStreamHeartbeat(ctx, m.Heartbeat)
			}
		}
	}()

	select {
	case <-ctx.Done():
		s.logger.Info("runner stream disconnected", "runner_id", runnerID)
		return ctx.Err()
	case err := <-errChan:
		s.logger.Error("runner stream error", "runner_id", runnerID, "error", err)
		return err
	}
}

func (s *GRPCServer) handleCommandResult(ctx context.Context, result *pb.CommandResult) {
	s.logger.Info("command result received",
		"command_id", result.CommandId,
		"success", result.Success,
		"message", result.Message)

	if vmID, ok := result.Data["vm_id"]; ok {
		vm, err := s.store.GetVM(ctx, vmID)
		if err != nil {
			s.logger.Error("failed to get vm", "vm_id", vmID, "error", err)
			return
		}

		if containerID, ok := result.Data["container_id"]; ok {
			vm.ContainerID = containerID
		}

		if result.Success {
			vm.Status = "running"
		} else {
			vm.Status = "error"
		}

		if err := s.store.SetVM(ctx, vm); err != nil {
			s.logger.Error("failed to update vm", "vm_id", vmID, "error", err)
		}
	}
}

func (s *GRPCServer) handleStreamHeartbeat(ctx context.Context, hb *pb.HeartbeatMessage) {
	s.manager.UpdateHeartbeat(hb.RunnerId)

	r, err := s.store.GetRunner(ctx, hb.RunnerId)
	if err != nil {
		s.logger.Warn("heartbeat from unknown runner", "runner_id", hb.RunnerId)
		return
	}

	r.LastSeen = time.Now().Unix()
	if err := s.store.RegisterRunner(ctx, r, 60*time.Second); err != nil {
		s.logger.Error("failed to update runner heartbeat", "error", err)
	}
}
