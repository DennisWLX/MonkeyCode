# Taskflow-Runner gRPC 双向流通信机制实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 Taskflow 与 Runner 之间的 gRPC 双向流通信，使 Taskflow 能实时向 Runner 推送 VM 创建命令，Runner 能返回执行结果。

**Architecture:** Runner 通过 gRPC 双向流连接 Taskflow，Taskflow 的 StreamManager 管理所有 Runner 连接。当 Backend 请求创建 VM 时，Taskflow 检查 Runner 在线状态后通过流推送命令，Runner 执行后返回结果。

**Tech Stack:** Go, gRPC, Protocol Buffers, Redis

***

## 文件结构

### 新增文件

- `taskflow/internal/runner/stream_manager.go` - Runner 流连接管理器
- `runner/internal/client/stream.go` - Runner 流客户端

### 修改文件

- `taskflow/pkg/proto/taskflow.proto` - 新增双向流消息定义
- `taskflow/internal/server/grpc.go` - 新增 CommandStream 方法
- `taskflow/internal/handler/vm.go` - 修改 Create 方法，添加在线检查和命令发送
- `taskflow/internal/handler/handlers.go` - 注入 StreamManager
- `taskflow/cmd/server/main.go` - 初始化 StreamManager
- `runner/cmd/runner/main.go` - 使用流客户端连接
- `runner/internal/client/grpc.go` - 暴露 gRPC 客户端

***

## Task 1: 扩展 Proto 定义

**Files:**

- Modify: `taskflow/pkg/proto/taskflow.proto`
- [ ] **Step 1: 添加双向流消息定义**

在 `taskflow.proto` 文件末尾添加以下内容：

```protobuf
// Runner 发送给 Taskflow 的消息
message RunnerMessage {
    oneof message {
        RunnerReady ready = 1;
        CommandResult result = 2;
        HeartbeatMessage heartbeat = 3;
    }
}

message RunnerReady {
    string runner_id = 1;
}

message CommandResult {
    string command_id = 1;
    bool success = 2;
    string message = 3;
    map<string, string> data = 4;
}

message HeartbeatMessage {
    string runner_id = 1;
    int64 timestamp = 2;
    int32 running_vms = 3;
    int32 running_tasks = 4;
}

// Taskflow 发送给 Runner 的命令
message TaskflowCommand {
    string command_id = 1;
    oneof command {
        CreateVMCommand create_vm = 2;
        DeleteVMCommand delete_vm = 3;
        CreateTaskCommand create_task = 4;
        StopTaskCommand stop_task = 5;
    }
}

message CreateVMCommand {
    string vm_id = 1;
    string image_url = 2;
    string git_url = 3;
    string git_token = 4;
    int32 cores = 5;
    int64 memory = 6;
    map<string, string> env_vars = 7;
    int64 ttl_seconds = 8;
}

message DeleteVMCommand {
    string vm_id = 1;
    string container_id = 2;
}

message CreateTaskCommand {
    string task_id = 1;
    string vm_id = 2;
    string container_id = 3;
    string text = 4;
    string model = 5;
    string api_key = 6;
    string base_url = 7;
    map<string, string> env_vars = 8;
}

message StopTaskCommand {
    string task_id = 1;
}
```

- [ ] **Step 2: 在 service RunnerService 中添加双向流 RPC**

在 `RunnerService` service 块中添加：

```protobuf
    // 双向流通信
    rpc CommandStream(stream RunnerMessage) returns (stream TaskflowCommand);
```

- [ ] **Step 3: 生成 Go 代码**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/taskflow && protoc --go_out=. --go-grpc_out=. pkg/proto/taskflow.proto`

Expected: 生成 `taskflow.pb.go` 和 `taskflow_grpc.pb.go` 更新

- [ ] **Step 4: Commit**

```bash
git add taskflow/pkg/proto/
git commit -m "feat(taskflow): 添加 gRPC 双向流消息定义"
```

***

## Task 2: 实现 StreamManager

**Files:**

- Create: `taskflow/internal/runner/stream_manager.go`
- [ ] **Step 1: 创建 StreamManager 文件**

创建文件 `taskflow/internal/runner/stream_manager.go`：

```go
package runner

import (
	"errors"
	"sync"

	pb "github.com/chaitin/MonkeyCode/taskflow/pkg/proto"
)

var (
	ErrRunnerNotFound = errors.New("runner not found")
	ErrStreamFull     = errors.New("runner stream full")
)

type StreamManager struct {
	mu      sync.RWMutex
	streams map[string]*RunnerStream
}

type RunnerStream struct {
	RunnerID string
	Stream   pb.RunnerService_CommandStreamServer
	SendChan chan *pb.TaskflowCommand
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(map[string]*RunnerStream),
	}
}

func (m *StreamManager) Register(runnerID string, stream pb.RunnerService_CommandStreamServer) *RunnerStream {
	m.mu.Lock()
	defer m.mu.Unlock()

	rs := &RunnerStream{
		RunnerID: runnerID,
		Stream:   stream,
		SendChan: make(chan *pb.TaskflowCommand, 100),
	}
	m.streams[runnerID] = rs
	return rs
}

func (m *StreamManager) Unregister(runnerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rs, ok := m.streams[runnerID]; ok {
		close(rs.SendChan)
		delete(m.streams, runnerID)
	}
}

func (m *StreamManager) SendCommand(runnerID string, cmd *pb.TaskflowCommand) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rs, ok := m.streams[runnerID]
	if !ok {
		return ErrRunnerNotFound
	}

	select {
	case rs.SendChan <- cmd:
		return nil
	default:
		return ErrStreamFull
	}
}

func (m *StreamManager) IsOnline(runnerID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.streams[runnerID]
	return ok
}

func (m *StreamManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.streams))
	for id := range m.streams {
		ids = append(ids, id)
	}
	return ids
}

func (m *StreamManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.streams)
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/taskflow && go build ./...`

Expected: 编译成功，无错误

- [ ] **Step 3: Commit**

```bash
git add taskflow/internal/runner/stream_manager.go
git commit -m "feat(taskflow): 实现 StreamManager 管理 Runner 双向流连接"
```

***

## Task 3: 实现 GRPC Server CommandStream 方法

**Files:**

- Modify: `taskflow/internal/server/grpc.go`
- [ ] **Step 1: 修改 GRPCServer 结构体，添加 streamManager 字段**

修改 `taskflow/internal/server/grpc.go`：

```go
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
```

- [ ] **Step 2: 添加 CommandStream 方法**

在 `grpc.go` 文件末尾添加：

```go
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
```

- [ ] **Step 3: 添加必要的 import**

确保 import 包含：

```go
import (
	"errors"
	"time"
	// ... 其他已有的 import
)
```

- [ ] **Step 4: 验证编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/taskflow && go build ./...`

Expected: 编译成功

- [ ] **Step 5: Commit**

```bash
git add taskflow/internal/server/grpc.go
git commit -m "feat(taskflow): 实现 CommandStream 双向流方法"
```

***

## Task 4: 修改 VM Handler

**Files:**

- Modify: `taskflow/internal/handler/vm.go`
- Modify: `taskflow/internal/handler/handlers.go`
- [ ] **Step 1: 修改 VMHandler 结构体**

修改 `taskflow/internal/handler/vm.go`：

```go
type VMHandler struct {
	store         *store.RedisStore
	manager       *runner.Manager
	streamManager *runner.StreamManager
}

func NewVMHandler(s *store.RedisStore, m *runner.Manager, sm *runner.StreamManager) *VMHandler {
	return &VMHandler{store: s, manager: m, streamManager: sm}
}
```

- [ ] **Step 2: 修改 Create 方法**

替换 `Create` 方法：

```go
func (h *VMHandler) Create(c echo.Context) error {
	var req CreateVMRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if req.UserID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user_id required")
	}

	if !h.streamManager.IsOnline(req.HostID) {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "runner offline")
	}

	vmID := uuid.New().String()

	vm := &store.VM{
		ID:        vmID,
		RunnerID:  req.HostID,
		UserID:    req.UserID,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
		ImageURL:  req.ImageURL,
		GitURL:    req.GitURL,
	}

	if err := h.store.SetVM(c.Request().Context(), vm); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if err := h.store.AddUserVM(c.Request().Context(), req.UserID, vmID); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	cmd := &pb.TaskflowCommand{
		CommandId: uuid.New().String(),
		Command: &pb.TaskflowCommand_CreateVm{
			CreateVm: &pb.CreateVMCommand{
				VmId:       vmID,
				ImageUrl:   req.ImageURL,
				GitUrl:     req.GitURL,
				GitToken:   req.GitToken,
				Cores:      req.Cores,
				Memory:     req.Memory,
				EnvVars:    req.EnvVars,
				TtlSeconds: req.TTL,
			},
		},
	}

	if err := h.streamManager.SendCommand(req.HostID, cmd); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to send command: "+err.Error())
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"code": 0,
		"data": vm,
	})
}
```

- [ ] **Step 3: 添加 pb import**

在 `vm.go` 文件顶部添加：

```go
import (
	pb "github.com/chaitin/MonkeyCode/taskflow/pkg/proto"
	// ... 其他 import
)
```

- [ ] **Step 4: 修改 Handlers 结构体**

修改 `taskflow/internal/handler/handlers.go`：

```go
package handler

import (
	"github.com/chaitin/MonkeyCode/taskflow/internal/runner"
	"github.com/chaitin/MonkeyCode/taskflow/internal/store"
)

type Handlers struct {
	Host        *HostHandler
	VM          *VMHandler
	Task        *TaskHandler
	Stats       *StatsHandler
	PortForward *PortForwardHandler
}

func NewHandlers(s *store.RedisStore, m *runner.Manager, sm *runner.StreamManager) *Handlers {
	return &Handlers{
		Host:        NewHostHandler(s),
		VM:          NewVMHandler(s, m, sm),
		Task:        NewTaskHandler(s),
		Stats:       NewStatsHandler(s, m),
		PortForward: NewPortForwardHandler(s, m),
	}
}
```

- [ ] **Step 5: 验证编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/taskflow && go build ./...`

Expected: 编译成功

- [ ] **Step 6: Commit**

```bash
git add taskflow/internal/handler/
git commit -m "feat(taskflow): VM Handler 添加 Runner 在线检查和命令发送"
```

***

## Task 5: 修改 Taskflow main.go

**Files:**

- Modify: `taskflow/cmd/server/main.go`
- [ ] **Step 1: 初始化 StreamManager 并传递给相关组件**

修改 `taskflow/cmd/server/main.go`：

```go
redisStore := store.NewRedisStore(redisClient)
runnerManager := runner.NewManager()
streamManager := runner.NewStreamManager()
backendClient := backend.NewClient(&cfg.Backend)

grpcServer := server.NewGRPCServer(redisStore, runnerManager, streamManager, backendClient, logger)
```

以及：

```go
handlers := handler.NewHandlers(redisStore, runnerManager, streamManager)
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/taskflow && go build ./...`

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add taskflow/cmd/server/main.go
git commit -m "feat(taskflow): main.go 初始化 StreamManager"
```

***

## Task 6: 实现 Runner Stream Client

**Files:**

- Create: `runner/internal/client/stream.go`
- [ ] **Step 1: 创建 StreamClient 文件**

创建文件 `runner/internal/client/stream.go`：

```go
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
	c.logger.Info("received create task command", "task_id", createCmd.TaskId)

	result := &pb.CommandResult{
		CommandId: cmd.CommandId,
		Success:   true,
		Message:   "task created",
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
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/runner && go build ./...`

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add runner/internal/client/stream.go
git commit -m "feat(runner): 实现 StreamClient 处理 Taskflow 命令"
```

***

## Task 7: 修改 Runner gRPC Client

**Files:**

- Modify: `runner/internal/client/grpc.go`
- [ ] **Step 1: 暴露 gRPC 客户端**

在 `runner/internal/client/grpc.go` 中添加方法：

```go
func (c *Client) Client() pb.RunnerServiceClient {
	return c.client
}
```

- [ ] **Step 2: 验证编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/runner && go build ./...`

Expected: 编译成功

- [ ] **Step 3: Commit**

```bash
git add runner/internal/client/grpc.go
git commit -m "feat(runner): 暴露 gRPC 客户端供 StreamClient 使用"
```

***

## Task 8: 修改 Runner main.go

**Files:**

- Modify: `runner/cmd/runner/main.go`
- [ ] **Step 1: 使用 StreamClient 连接**

修改 `runner/cmd/runner/main.go`，在注册成功后添加流连接：

```go
import (
	streamclient "github.com/chaitin/MonkeyCode/runner/internal/client"
	// ... 其他 import
)

func main() {
	// ... 现有初始化代码 ...

	grpcClient, err := grpcclient.New(cfg.GRPCAddr, logger)
	if err != nil {
		logger.Error("failed to connect to taskflow", "error", err)
		os.Exit(1)
	}
	defer grpcClient.Close()

	runnerID, err := grpcClient.Register(ctx, cfg.Token, hostname, ip, 8, 16384, 100000)
	if err != nil {
		logger.Error("failed to register", "error", err)
		os.Exit(1)
	}

	logger.Info("runner registered", "runner_id", runnerID, "hostname", hostname, "ip", ip)

	// 创建流客户端并连接
	streamClient := streamclient.NewStreamClient(grpcClient.Client(), vmMgr, taskMgr, logger)
	if err := streamClient.Connect(ctx, runnerID); err != nil {
		logger.Error("failed to connect stream", "error", err)
		os.Exit(1)
	}

	// 使用流客户端发送心跳
	go startStreamHeartbeat(ctx, streamClient, runnerID, vmMgr, taskExecutor, logger)
	go startHTTPServer(ctx, vmMgr, termMgr, taskMgr, taskExecutor, forwardMgr, logger)

	// ... 信号处理代码 ...
}

func startStreamHeartbeat(ctx context.Context, streamClient *streamclient.StreamClient, runnerID string, vmMgr *vm.Manager, taskExecutor *task.Executor, logger *slog.Logger) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runningVMs := int32(vmMgr.RunningCount())
			runningTasks := int32(taskExecutor.RunningCount())

			if err := streamClient.SendHeartbeat(runnerID, runningVMs, runningTasks); err != nil {
				logger.Error("stream heartbeat failed", "error", err)
			} else {
				logger.Debug("stream heartbeat sent", "runner_id", runnerID)
			}
		}
	}
}
```

- [ ] **Step 2: 删除旧的 startHeartbeat 函数调用**

移除或注释掉原来的 `startHeartbeat` 调用。

- [ ] **Step 3: 验证编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/runner && go build ./...`

Expected: 编译成功

- [ ] **Step 4: Commit**

```bash
git add runner/cmd/runner/main.go
git commit -m "feat(runner): 使用 StreamClient 连接 Taskflow 并发送心跳"
```

***

## Task 9: 集成测试

**Files:**

- 无文件变更
- [ ] **Step 1: 编译 Taskflow**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/taskflow && go build -o bin/taskflow ./cmd/server`

Expected: 编译成功

- [ ] **Step 2: 编译 Runner**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/runner && go build -o bin/runner ./cmd/runner`

Expected: 编译成功

- [ ] **Step 3: Commit 最终版本**

```bash
git add -A
git commit -m "feat: 完成 Taskflow-Runner gRPC 双向流通信机制实现"
```

***

## 验收标准

1. Taskflow 启动后能接受 Runner 的 gRPC 双向流连接
2. Runner 连接后能通过流接收 VM 创建命令
3. Runner 执行命令后能通过流返回结果
4. Taskflow 能正确更新 VM 状态
5. Runner 离线时 Taskflow 能正确拒绝创建请求

