# Agent 报告回调链路设计

## 概述

本文档描述 Agent 执行过程中的实时报告回调链路实现，使 Runner 能将 Agent 输出实时推送到 Loki，供前端通过 WebSocket 实时显示。

## 背景

当前实现中：
- Runner 的 `Executor.reportCh` 已经收集 Agent 执行输出
- Backend 的 Loki Client 已经实现 Tail/History 方法读取日志
- **缺失**：Runner 到 Taskflow 的报告发送，以及 Taskflow 到 Loki 的存储

## 设计目标

1. Runner 通过 gRPC ReportStream 发送报告到 Taskflow
2. Taskflow 将报告存储到 Loki
3. 整个链路实时、可靠

## 架构设计

### 整体架构

```
┌─────────────┐                    ┌─────────────┐                    ┌─────────────┐
│   Runner    │                    │  Taskflow   │                    │    Loki     │
├─────────────┤                    ├─────────────┤                    ├─────────────┤
│ Executor    │                    │             │                    │             │
│   reportCh ─┼──▶ gRPC Report ───▶│ gRPC Server │──▶ HTTP POST ─────▶│   Storage   │
│             │    Stream          │             │    /loki/api/v1/   │             │
└─────────────┘                    └─────────────┘    push            └─────────────┘
```

### 数据流向

```
1. Agent 执行输出
   OpenCode stdout/stderr
         │
         ▼
2. Executor 收集
   progressCb(source, data) ──▶ reportCh
         │
         ▼
3. Runner 发送
   executor.Reports() ──▶ gRPC ReportStream.Send()
         │
         ▼
4. Taskflow 接收
   ReportStream.Recv() ──▶ Loki Client.Push()
         │
         ▼
5. Loki 存储
   {task_id="xxx"} labels
         │
         ▼
6. Backend 读取
   Loki.Tail() ──▶ WebSocket ──▶ Frontend
```

## 详细设计

### 1. Runner 端实现

#### 1.1 启动报告发送 goroutine

文件：`runner/cmd/runner/main.go`

在 main 函数中添加：

```go
// 启动报告发送
go startReportStream(ctx, grpcClient, taskExecutor, logger)

func startReportStream(ctx context.Context, grpcClient *grpcclient.Client, taskExecutor *task.Executor, logger *slog.Logger) {
    for {
        select {
        case <-ctx.Done():
            return
        case report := <-taskExecutor.Reports():
            err := grpcClient.SendReport(ctx, &pb.ReportRequest{
                TaskId:    report.TaskID,
                Source:    report.Source,
                Timestamp: report.Timestamp,
                Data:      report.Data,
            })
            if err != nil {
                logger.Error("failed to send report", "error", err)
            }
        }
    }
}
```

#### 1.2 gRPC Client 添加 SendReport 方法

文件：`runner/internal/client/grpc.go`

```go
func (c *Client) SendReport(ctx context.Context, req *pb.ReportRequest) error {
    stream, err := c.client.ReportStream(ctx)
    if err != nil {
        return err
    }
    return stream.Send(req)
}
```

### 2. Taskflow 端实现

#### 2.1 添加 Loki 客户端

文件：`taskflow/pkg/loki/client.go`

```go
package loki

type Client struct {
    baseURL string
    client  *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: strings.TrimRight(baseURL, "/"),
        client:  &http.Client{Timeout: 10 * time.Second},
    }
}

// Push 推送日志到 Loki
func (c *Client) Push(ctx context.Context, labels map[string]string, entries []Entry) error {
    // 构建 Loki push 请求
    // POST /loki/api/v1/push
}
```

#### 2.2 修改 GRPCServer 添加 Loki 客户端

文件：`taskflow/internal/server/grpc.go`

```go
type GRPCServer struct {
    pb.UnimplementedRunnerServiceServer
    store         *store.RedisStore
    manager       *runner.Manager
    streamManager *runner.StreamManager
    loki          *loki.Client
    logger        *slog.Logger
    backend       *backend.Client
}
```

#### 2.3 实现 ReportStream 方法

文件：`taskflow/internal/server/grpc.go`

```go
func (s *GRPCServer) ReportStream(stream pb.RunnerService_ReportStreamServer) error {
    ctx := stream.Context()
    
    for {
        req, err := stream.Recv()
        if err != nil {
            return err
        }
        
        // 推送到 Loki
        labels := map[string]string{
            "task_id": req.TaskId,
            "source":  req.Source,
        }
        
        entry := loki.Entry{
            Timestamp: time.Unix(req.Timestamp, 0),
            Line:      string(req.Data),
        }
        
        if err := s.loki.Push(ctx, labels, []loki.Entry{entry}); err != nil {
            s.logger.Error("failed to push to loki", "error", err)
        }
    }
}
```

### 3. 配置

#### 3.1 Taskflow 配置

文件：`taskflow/config/config.go`

```go
type Config struct {
    // ...
    Loki LokiConfig `yaml:"loki"`
}

type LokiConfig struct {
    URL string `yaml:"url"`
}
```

#### 3.2 Taskflow main.go

文件：`taskflow/cmd/server/main.go`

```go
lokiClient := loki.NewClient(cfg.Loki.URL)
grpcServer := server.NewGRPCServer(redisStore, runnerManager, streamManager, lokiClient, backendClient, logger)
```

## 文件变更清单

### 新增文件

- `taskflow/pkg/loki/client.go` - Loki 客户端

### 修改文件

- `runner/cmd/runner/main.go` - 启动报告发送 goroutine
- `runner/internal/client/grpc.go` - 添加 SendReport 方法
- `taskflow/internal/server/grpc.go` - 添加 Loki 客户端，实现 ReportStream
- `taskflow/config/config.go` - 添加 Loki 配置
- `taskflow/cmd/server/main.go` - 初始化 Loki 客户端

## 测试计划

1. **单元测试**
   - Loki Client Push 方法测试
   - Runner SendReport 方法测试

2. **集成测试**
   - Runner 发送报告 → Taskflow 接收 → Loki 存储
   - Backend 通过 Loki Tail 读取报告

## 验收标准

1. Agent 执行输出能实时发送到 Taskflow
2. Taskflow 能将报告存储到 Loki
3. Backend 能通过 Loki Tail 实时获取报告
4. 前端能通过 WebSocket 实时显示 Agent 输出
