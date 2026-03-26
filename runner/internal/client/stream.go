package client

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/chaitin/MonkeyCode/taskflow/pkg/proto"
	"github.com/chaitin/MonkeyCode/runner/internal/task"
	"github.com/chaitin/MonkeyCode/runner/internal/vm"
)

type StreamClient struct {
	client  pb.RunnerServiceClient
	stream  pb.RunnerService_CommandStreamClient
	vmMgr   *vm.Manager
	taskMgr *task.Manager
	logger  *slog.Logger
}

func NewStreamClient(client pb.RunnerServiceClient, vmMgr *vm.Manager, taskMgr *task.Manager, logger *slog.Logger) *StreamClient {
	return &StreamClient{
		client:  client,
		vmMgr:   vmMgr,
		taskMgr: taskMgr,
		logger:  logger,
	}
}

func (c *StreamClient) Connect(ctx context.Context, runnerID string) error {
	stream, err := c.client.CommandStream(ctx)
	if err != nil {
		return err
	}

	if err := stream.Send(&pb.RunnerMessage{
		Message: &pb.RunnerMessage_Ready{
			Ready: &pb.RunnerReady{RunnerId: runnerID},
		},
	}); err != nil {
		return err
	}

	c.stream = stream
	c.logger.Info("connected to taskflow command stream")

	go c.receiveLoop(ctx)

	return nil
}

func (c *StreamClient) receiveLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			cmd, err := c.stream.Recv()
			if err != nil {
				c.logger.Error("failed to receive command", "error", err)
				return
			}

			switch cmd.Command.(type) {
			case *pb.TaskflowCommand_CreateVm:
				go c.handleCreateVM(ctx, cmd)
			case *pb.TaskflowCommand_DeleteVm:
				go c.handleDeleteVM(ctx, cmd)
			case *pb.TaskflowCommand_CreateTask:
				go c.handleCreateTask(ctx, cmd)
			case *pb.TaskflowCommand_StopTask:
				go c.handleStopTask(ctx, cmd)
			}
		}
	}
}

func (c *StreamClient) handleCreateVM(ctx context.Context, cmd *pb.TaskflowCommand) {
	createCmd := cmd.GetCreateVm()
	c.logger.Info("received create vm command", "vm_id", createCmd.VmId)

	vm, err := c.vmMgr.Create(ctx, vm.CreateOptions{
		UserID:     createCmd.VmId,
		ImageURL:   createCmd.ImageUrl,
		GitURL:     createCmd.GitUrl,
		GitToken:   createCmd.GitToken,
		Cores:      int64(createCmd.Cores),
		Memory:     createCmd.Memory,
		EnvVars:    createCmd.EnvVars,
		TTLSeconds: createCmd.TtlSeconds,
	})

	result := &pb.CommandResult{
		CommandId: cmd.CommandId,
		Success:   err == nil,
		Data:      make(map[string]string),
	}

	if err != nil {
		result.Message = err.Error()
		c.logger.Error("failed to create vm", "error", err)
	} else {
		result.Message = "success"
		result.Data["vm_id"] = vm.ID
		result.Data["container_id"] = vm.ContainerID
		c.logger.Info("vm created", "vm_id", vm.ID, "container_id", vm.ContainerID)
	}

	c.sendResult(result)
}

func (c *StreamClient) handleDeleteVM(ctx context.Context, cmd *pb.TaskflowCommand) {
	deleteCmd := cmd.GetDeleteVm()
	c.logger.Info("received delete vm command", "vm_id", deleteCmd.VmId)

	err := c.vmMgr.Delete(ctx, deleteCmd.VmId)

	result := &pb.CommandResult{
		CommandId: cmd.CommandId,
		Success:   err == nil,
		Message:   "success",
	}

	if err != nil {
		result.Message = err.Error()
		c.logger.Error("failed to delete vm", "error", err)
	}

	c.sendResult(result)
}

func (c *StreamClient) handleCreateTask(ctx context.Context, cmd *pb.TaskflowCommand) {
	createCmd := cmd.GetCreateTask()
	c.logger.Info("received create task command", "task_id", createCmd.TaskId, "vm_id", createCmd.VmId)

	result := &pb.CommandResult{
		CommandId: cmd.CommandId,
		Success:   true,
		Message:   "task created",
		Data:      make(map[string]string),
	}

	task, err := c.taskMgr.Create(task.CreateOptions{
		VMID:   createCmd.VmId,
		UserID: createCmd.TaskId,
		Text:   createCmd.Text,
		Model:  createCmd.Model,
		Agent:  "opencode",
	})

	if err != nil {
		result.Success = false
		result.Message = err.Error()
		c.logger.Error("failed to create task", "error", err)
	} else {
		result.Data["task_id"] = task.ID
		c.logger.Info("task created", "task_id", task.ID)
	}

	c.sendResult(result)
}

func (c *StreamClient) handleStopTask(ctx context.Context, cmd *pb.TaskflowCommand) {
	stopCmd := cmd.GetStopTask()
	c.logger.Info("received stop task command", "task_id", stopCmd.TaskId)

	result := &pb.CommandResult{
		CommandId: cmd.CommandId,
		Success:   true,
		Message:   "task stopped",
	}

	if err := c.taskMgr.Cancel(stopCmd.TaskId); err != nil {
		result.Success = false
		result.Message = err.Error()
		c.logger.Error("failed to stop task", "error", err)
	}

	c.sendResult(result)
}

func (c *StreamClient) sendResult(result *pb.CommandResult) {
	if err := c.stream.Send(&pb.RunnerMessage{
		Message: &pb.RunnerMessage_Result{Result: result},
	}); err != nil {
		c.logger.Error("failed to send result", "error", err)
	}
}

func (c *StreamClient) SendHeartbeat(runnerID string, runningVMs, runningTasks int32) error {
	return c.stream.Send(&pb.RunnerMessage{
		Message: &pb.RunnerMessage_Heartbeat{
			Heartbeat: &pb.HeartbeatMessage{
				RunnerId:     runnerID,
				Timestamp:    time.Now().Unix(),
				RunningVms:   runningVMs,
				RunningTasks: runningTasks,
			},
		},
	})
}

func (c *StreamClient) SendReport(taskID, source string, timestamp int64, data []byte) error {
	return c.stream.Send(&pb.RunnerMessage{
		Message: &pb.RunnerMessage_Report{
			Report: &pb.ReportEntry{
				TaskId:    taskID,
				Source:    source,
				Timestamp: timestamp,
				Data:      data,
			},
		},
	})
}
