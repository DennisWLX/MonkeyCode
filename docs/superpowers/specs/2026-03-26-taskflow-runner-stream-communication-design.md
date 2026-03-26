# Taskflow 到 Runner gRPC 双向流通信机制设计

## 概述

本文档描述 Taskflow 与 Runner 之间的 gRPC 双向流通信机制，用于实现 VM 创建、删除、任务执行等命令的实时传递和结果反馈。

## 背景

当前架构中，Backend 调用 Taskflow 的 HTTP API 创建 VM，Taskflow 仅将 VM 记录存入 Redis，但缺少将命令传递给 Runner 的机制。Runner 虽然通过 gRPC 注册和发送心跳，但没有接收命令的通道。

## 设计目标

1. **实时通信**：Taskflow 能实时向 Runner 推送命令
2. **结果反馈**：Runner 能将执行结果返回给 Taskflow
3. **状态同步**：VM 状态变更能及时反映到系统中
4. **离线检测**：Runner 离线时能立即拒绝新命令

## 架构设计

### 整体架构

```
┌─────────────┐                    ┌─────────────┐                    ┌─────────────┐
│   Backend   │ ──── HTTP ────▶    │  Taskflow   │ ◀─── gRPC 流 ───▶ │   Runner    │
│             │   /internal/vm     │             │                    │             │
└─────────────┘                    └─────────────┘                    └─────────────┘
                                          │
                                          ▼
                                   ┌─────────────┐
                                   │    Redis    │
                                   │  (状态存储)  │
                                   └─────────────┘
```

### 通信流程

1. Runner 启动时调用 `Register` 注册，然后建立 `CommandStream` 双向流
2. Backend 调用 Taskflow HTTP API 创建 VM
3. Taskflow 检查 Runner 在线状态
4. 若 Runner 在线，通过流推送 `CreateVMCommand`
5. Runner 执行命令，通过流返回 `CommandResult`
6. Taskflow 更新 Redis 中的 VM 状态

## 详细设计

### 1. Proto 定义

文件：`taskflow/pkg/proto/taskflow.proto`

新增双向流服务和消息类型：

```protobuf
service RunnerService {
    rpc Register(RegisterRequest) returns (RegisterResponse);
    rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
    rpc CreateVM(CreateVMRequest) returns (CreateVMResponse);
    rpc DeleteVM(DeleteVMRequest) returns (DeleteVMResponse);
    rpc ListVMs(ListVMsRequest) returns (ListVMsResponse);
    rpc GetVMInfo(GetVMInfoRequest) returns (GetVMInfoResponse);
    rpc CreateTask(CreateTaskRequest) returns (CreateTaskResponse);
    rpc StopTask(StopTaskRequest) returns (StopTaskResponse);
    rpc TerminalStream(stream TerminalData) returns (stream TerminalData);
    rpc ReportStream(ReportRequest) returns (stream ReportEntry);
    
    // 新增：双向流通信
    rpc CommandStream(stream RunnerMessage) returns (stream TaskflowCommand);
}

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

### 2. Taskflow 端实现

#### 2.1 StreamManager

文件：`taskflow/internal/runner/stream_manager.go`

职责：管理 Runner 的双向流连接

```go
package runner

import (
    "sync"
    pb "github.com/chaitin/MonkeyCode/taskflow/pkg/proto"
)

type StreamManager struct {
    mu      sync.RWMutex
    streams map[string]*RunnerStream
}

type RunnerStream struct {
    RunnerID string
    Stream   pb.RunnerService_CommandStreamServer
    sendChan chan *pb.TaskflowCommand
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
        sendChan: make(chan *pb.TaskflowCommand, 100),
    }
    m.streams[runnerID] = rs
    return rs
}

func (m *StreamManager) Unregister(runnerID string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if rs, ok := m.streams[runnerID]; ok {
        close(rs.sendChan)
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
    case rs.sendChan <- cmd:
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
```

#### 2.2 GRPC Server 扩展

文件：`taskflow/internal/server/grpc.go`

新增 `CommandStream` 方法：

```go
func (s *GRPCServer) CommandStream(stream pb.RunnerService_CommandStreamServer) error {
    ctx := stream.Context()
    
    // 等待 Runner 发送就绪消息
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
    
    // 注册流
    rs := s.streamManager.Register(runnerID, stream)
    defer s.streamManager.Unregister(runnerID)
    
    // 启动发送协程
    errChan := make(chan error, 2)
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case cmd, ok := <-rs.sendChan:
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
    
    // 接收协程
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
                s.handleHeartbeat(ctx, m.Heartbeat)
            }
        }
    }()
    
    select {
    case <-ctx.Done():
        return ctx.Err()
    case err := <-errChan:
        return err
    }
}

func (s *GRPCServer) handleCommandResult(ctx context.Context, result *pb.CommandResult) {
    s.logger.Info("command result received", 
        "command_id", result.CommandId, 
        "success", result.Success,
        "message", result.Message)
    
    // 更新 VM 状态
    if containerID, ok := result.Data["container_id"]; ok {
        vmID := result.Data["vm_id"]
        vm, err := s.store.GetVM(ctx, vmID)
        if err != nil {
            return
        }
        vm.ContainerID = containerID
        if result.Success {
            vm.Status = "running"
        } else {
            vm.Status = "error"
        }
        s.store.SetVM(ctx, vm)
    }
}
```

#### 2.3 VM Handler 修改

文件：`taskflow/internal/handler/vm.go`

修改 `Create` 方法：

```go
func (h *VMHandler) Create(c echo.Context) error {
    var req CreateVMRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, err.Error())
    }

    if req.UserID == "" {
        return echo.NewHTTPError(http.StatusBadRequest, "user_id required")
    }

    // 检查 Runner 是否在线
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

    // 发送创建命令
    cmd := &pb.TaskflowCommand{
        CommandId: uuid.New().String(),
        Command: &pb.TaskflowCommand_CreateVm{
            CreateVm: &pb.CreateVMCommand{
                VmId:      vmID,
                ImageUrl:  req.ImageURL,
                GitUrl:    req.GitURL,
                GitToken:  req.GitToken,
                Cores:     req.Cores,
                Memory:    req.Memory,
                EnvVars:   req.EnvVars,
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

### 3. Runner 端实现

#### 3.1 Stream Client

文件：`runner/internal/client/stream.go`

```go
package client

import (
    "context"
    "log/slog"
    
    pb "github.com/chaitin/MonkeyCode/taskflow/pkg/proto"
    "github.com/chaitin/MonkeyCode/runner/internal/vm"
    "github.com/chaitin/MonkeyCode/runner/internal/task"
)

type StreamClient struct {
    client   pb.RunnerServiceClient
    stream   pb.RunnerService_CommandStreamClient
    vmMgr    *vm.Manager
    taskMgr  *task.Manager
    logger   *slog.Logger
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
    
    // 发送就绪消息
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

#### 3.2 Main 程序修改

文件：`runner/cmd/runner/main.go`

```go
func main() {
    // ... 现有初始化代码 ...
    
    grpcClient, err := grpcclient.New(cfg.GRPCAddr, logger)
    if err != nil {
        logger.Error("failed to connect to taskflow", "error", err)
        os.Exit(1)
    }
    defer grpcClient.Close()
    
    // 注册 Runner
    runnerID, err := grpcClient.Register(ctx, cfg.Token, hostname, ip, 8, 16384, 100000)
    if err != nil {
        logger.Error("failed to register", "error", err)
        os.Exit(1)
    }
    
    // 创建流客户端并连接
    streamClient := client.NewStreamClient(grpcClient.Client(), vmMgr, taskMgr, logger)
    if err := streamClient.Connect(ctx, runnerID); err != nil {
        logger.Error("failed to connect stream", "error", err)
        os.Exit(1)
    }
    
    // 使用流客户端发送心跳
    go startStreamHeartbeat(ctx, streamClient, runnerID, vmMgr, taskExecutor, logger)
    
    // ... 其余代码 ...
}

func startStreamHeartbeat(ctx context.Context, streamClient *client.StreamClient, runnerID string, vmMgr *vm.Manager, taskExecutor *task.Executor, logger *slog.Logger) {
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

## 错误处理

### Runner 离线

当 Backend 请求创建 VM 时，Taskflow 检查 Runner 是否在线：
- 若离线：返回 HTTP 503 Service Unavailable
- 若在线：发送命令并等待结果

### 流断开

当 Runner 的流断开时：
1. StreamManager 从 map 中移除该 Runner
2. 后续请求会收到 "runner offline" 错误
3. Runner 重连后重新建立流

### 命令执行失败

Runner 执行命令失败时：
1. 通过流返回 `CommandResult`，`success=false`
2. Taskflow 更新 VM 状态为 `error`
3. 用户可通过 API 查询状态

## 测试计划

### 单元测试

1. `StreamManager` 的注册、注销、发送命令测试
2. `StreamClient` 的消息序列化/反序列化测试
3. VM Handler 的在线检查逻辑测试

### 集成测试

1. Runner 连接 -> 发送命令 -> 接收结果 完整流程
2. Runner 离线时的错误处理
3. 流断开后的重连测试

## 实现优先级

1. **P0 - 核心功能**
   - Proto 定义扩展
   - StreamManager 实现
   - CommandStream gRPC 方法
   - Runner StreamClient 实现
   - VM Handler 修改

2. **P1 - 完善功能**
   - DeleteVM 命令
   - CreateTask/StopTask 命令
   - 心跳迁移到流

3. **P2 - 优化**
   - 流重连机制
   - 命令超时处理
   - 监控指标

## 文件变更清单

### 新增文件

- `taskflow/internal/runner/stream_manager.go`
- `runner/internal/client/stream.go`

### 修改文件

- `taskflow/pkg/proto/taskflow.proto`
- `taskflow/internal/server/grpc.go`
- `taskflow/internal/handler/vm.go`
- `taskflow/internal/handler/handlers.go`
- `taskflow/cmd/server/main.go`
- `runner/cmd/runner/main.go`
- `runner/internal/client/grpc.go`
