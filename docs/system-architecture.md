# MonkeyCode 系统架构文档

## 文档信息

- **创建时间**: 2024年
- **项目**: MonkeyCode 分布式 AI 代码执行平台
- **版本**: v1.0

---

## 一、系统概述

MonkeyCode 是一个分布式 AI 代码执行平台，采用微服务架构，包含以下核心组件：

| 组件 | 描述 | 端口 |
|------|------|------|
| **Backend** | 用户 API 网关、认证服务 | :8888 |
| **TaskFlow** | 任务编排、VM 管理、状态协调 | :50051 (gRPC), :8889 (HTTP) |
| **Runner** | 代码执行节点、Docker 容器管理 | :8080 |

---

## 二、系统架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                          用户浏览器 / 客户端                        │
│                       (Web UI / API 调用)                        │
└───────────────────────────────┬─────────────────────────────────┘
                                │
                                │ HTTPS
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Backend 服务                             │
│                         (用户认证、API 网关)                       │
│                             (:8888)                               │
│                                                               │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    核心功能                              │  │
│  │  • 用户认证 (Token 验证)                               │  │
│  │  • VM 生命周期管理                                      │  │
│  │  • Terminal 会话管理                                    │  │
│  │  • PortForward 管理                                    │  │
│  │  • 任务状态同步                                         │  │
│  └───────────────────────────────────────────────────────────┘  │
└───────────────────────────────┬─────────────────────────────────┘
                                │
              ┌─────────────────┴─────────────────┐
              │                                   │
              │ HTTP/gRPC                        │ HTTP
              ▼                                   ▼
┌─────────────────────────────┐  ┌───────────────────────────────────┐
│      TaskFlow 服务            │  │       外部服务                    │
│      (:50051 gRPC)          │  │       (AI 模型、外部 API)          │
│      (:8889 HTTP)          │  │                                   │
│                             │  │  • OpenAI API                    │
│  ┌───────────────────────┐  │  │  • Claude API                   │
│  │     gRPC Server       │  │  │  • GitHub API                   │
│  │  • Register (Runner)  │  │  └───────────────────────────────────┘
│  │  • Heartbeat         │  │
│  │  • CreateVM          │  │
│  │  • DeleteVM          │  │
│  │  • Terminal         │  │
│  └───────────────────────┘  │
│                             │
│  ┌───────────────────────┐  │
│  │    Redis Store       │  │
│  │  • Runner 信息        │  │
│  │  • VM 状态           │  │
│  │  • Task 记录         │  │
│  │  • PortForward       │  │
│  └───────────────────────┘  │
└─────────────┬───────────────┘
              │
              │ gRPC
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Runner 集群 (多个节点)                        │
│                                                               │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐              │
│  │  Runner 1  │  │  Runner 2  │  │  Runner 3  │              │
│  │   (:8080)  │  │   (:8080)  │  │   (:8080)  │              │
│  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘              │
│        │                │                │                       │
│        ▼                ▼                ▼                       │
│  ┌───────────┐  ┌───────────┐  ┌───────────┐                │
│  │  Docker   │  │  Docker   │  │  Docker   │                │
│  │  容器池    │  │  容器池    │  │  容器池    │                │
│  └───────────┘  └───────────┘  └───────────┘                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 三、组件详细说明

### 3.1 Backend 服务

**职责**：
- 用户认证和授权
- API 请求路由
- 业务逻辑编排
- 与 TaskFlow 通信

**关键文件**：
```
backend/
├── biz/host/usecase/host.go    # 主机/VM 业务逻辑
├── pkg/taskflow/               # TaskFlow 客户端
│   ├── client.go              # gRPC/HTTP 客户端
│   └── types.go               # 类型定义
└── pkg/llm/                  # LLM 客户端
    └── client.go              # OpenAI/Anthropic 客户端
```

**端口配置**：
| 变量 | 默认值 | 说明 |
|------|--------|------|
| HTTP_ADDR | :8888 | HTTP 监听地址 |

---

### 3.2 TaskFlow 服务

**职责**：
- Runner 节点注册和心跳管理
- VM 生命周期管理
- Terminal 会话协调
- PortForward 状态管理
- Redis 状态存储

**关键文件**：
```
taskflow/
├── internal/
│   ├── server/
│   │   ├── grpc.go            # gRPC 服务端
│   │   └── http.go            # HTTP 服务器
│   ├── handler/                # HTTP 处理器
│   │   ├── host.go           # Runner 管理
│   │   ├── vm.go            # VM 管理
│   │   ├── task.go          # 任务管理
│   │   └── portforward.go   # 端口转发
│   ├── runner/manager.go    # Runner 连接管理
│   └── store/redis.go      # Redis 存储
├── pkg/proto/                # Protocol Buffers
│   └── taskflow.proto       # gRPC 定义
└── internal/backend/client.go # Backend 客户端
```

**端口配置**：
| 变量 | 默认值 | 说明 |
|------|--------|------|
| HTTP_ADDR | :8889 | HTTP 监听地址 |
| GRPC_ADDR | :50051 | gRPC 监听地址 |

**gRPC 服务定义**：
```protobuf
service RunnerService {
  rpc Register(RegisterRequest) returns (RegisterResponse);
  rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
  rpc CreateVM(CreateVMRequest) returns (CreateVMResponse);
  rpc DeleteVM(DeleteVMRequest) returns (DeleteVMResponse);
  rpc ListVMs(ListVMsRequest) returns (ListVMsResponse);
  rpc CreateTask(CreateTaskRequest) returns (CreateTaskResponse);
  rpc StopTask(StopTaskRequest) returns (StopTaskResponse);
  rpc TerminalStream(stream TerminalData) returns (stream TerminalData);
  rpc ReportStream(stream ReportData) returns (stream ReportAck);
}
```

---

### 3.3 Runner 服务

**职责**：
- Docker 容器管理
- AI Agent 执行（OpenCode 等）
- 任务执行和进度报告
- Terminal 会话（WebSocket）
- PortForward 实现

**关键文件**：
```
runner/
├── cmd/runner/main.go         # 入口
├── internal/
│   ├── client/grpc.go        # TaskFlow gRPC 客户端
│   ├── docker/                # Docker SDK 封装
│   │   ├── container.go       # 容器操作
│   │   ├── image.go          # 镜像操作
│   │   └── network.go       # 网络操作
│   ├── agent/opencode.go     # OpenCode Agent
│   ├── task/manager.go       # 任务管理
│   ├── executor/executor.go  # 任务执行器
│   ├── vm/manager.go        # VM (Docker 容器) 管理
│   ├── terminal/             # 终端管理
│   │   ├── manager.go       # 会话管理
│   │   ├── pty.go          # PTY 处理
│   │   └── websocket.go     # WebSocket 处理
│   └── portforward/forward.go # 端口转发
└── config.yaml               # 配置文件
```

**端口配置**：
| 变量 | 默认值 | 说明 |
|------|--------|------|
| TOKEN | - | **必需** 认证令牌 |
| GRPC_URL | localhost:50051 | TaskFlow gRPC 地址 |
| GRPC_HOST | localhost | gRPC 主机 |
| GRPC_PORT | 50051 | gRPC 端口 |

---

## 四、通讯流程详解

### 4.1 Runner 注册流程

```
Runner                                    TaskFlow                               Backend
  │                                        │                                      │
  │  1. 启动时读取 TOKEN                     │                                      │
  │                                        │                                      │
  │─── Register ─────────────────────────►│                                      │
  │     { token, hostname, ip,           │                                      │
  │       cores, memory, disk }           │                                      │
  │                                        │                                      │
  │                                        │─── CheckToken ──────────────────►│
  │                                        │     { token }                       │
  │                                        │                                      │
  │                                        │◄── TokenInfo ──────────────────│
  │                                        │     { user_id, team_id }             │
  │                                        │                                      │
  │                                        │ 2. 存储 Runner 信息到 Redis          │
  │                                        │     • ID: token                      │
  │                                        │     • Status: online                 │
  │                                        │     • LastSeen: timestamp            │
  │                                        │                                      │
  │                                        │─── ReportHostInfo ────────────────►│
  │                                        │     { id, hostname, cores,           │
  │                                        │       memory, public_ip }            │
  │                                        │                                      │
  │◄── RegisterResponse ────────────────│                                      │
  │       { runner_id }                   │                                      │
  │                                        │                                      │
  │  3. 启动心跳协程 (每 10 秒)           │                                      │
  │                                        │                                      │
  │─── Heartbeat ───────────────────────►│                                      │
  │     { vm_count, task_count }         │                                      │
  │                                        │                                      │
  │     (重复直到 Runner 停止)             │                                      │
```

**详细代码**：

**Runner 端** (`runner/internal/client/grpc.go`)：
```go
func (c *Client) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    resp, err := c.client.Register(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("register failed: %w", err)
    }
    return resp, nil
}

// 心跳协程
go func() {
    ticker := time.NewTicker(10 * time.Second)
    for {
        <-ticker.C
        c.Heartbeat(ctx, &pb.HeartbeatRequest{
            RunnerId: runnerID,
            VmCount:  vmMgr.RunningCount(),
            TaskCount: taskMgr.RunningCount(),
        })
    }
}()
```

**TaskFlow 端** (`taskflow/internal/server/grpc.go`)：
```go
func (s *GRPCServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    // 验证 Token
    token, err := s.backend.CheckToken(ctx, &backend.CheckTokenReq{
        Token: req.Token,
    })
    if err != nil {
        return &pb.RegisterResponse{Success: false}, nil
    }

    // 存储 Runner 信息
    r := &store.Runner{
        ID:       token.Token,
        UserID:   token.User.ID,
        Hostname: req.Hostname,
        IP:       req.Ip,
        Status:   "online",
        Capacity: map[string]int64{
            "cores":  req.Cores,
            "memory": req.Memory,
            "disk":   req.Disk,
        },
    }
    s.store.RegisterRunner(ctx, r, 60*time.Second)

    // 上报 Host 信息到 Backend
    s.backend.ReportHostInfo(ctx, &backend.HostInfo{
        ID:       token.Token,
        UserID:   token.User.ID,
        Hostname: req.Hostname,
        Cores:    req.Cores,
        Memory:   req.Memory,
    })

    return &pb.RegisterResponse{
        Success:  true,
        RunnerId: token.Token,
    }, nil
}
```

---

### 4.2 VM 创建流程

```
Browser                     Backend                  TaskFlow                  Runner
  │                           │                        │                        │
  │  1. 点击"创建 VM"          │                        │                        │
  │─── CreateVM ────────────►│                        │                        │
  │     { user_id, git_url,  │                        │                        │
  │       image_url, ttl }    │                        │                        │
  │                           │                        │                        │
  │                           │─── CreateVM ─────────►│                        │
  │                           │     { host_id, image, │                        │
  │                           │       cores, memory } │                        │
  │                           │                        │                        │
  │                           │                        │─── 分配 Runner ─────────│
  │                           │                        │                        │
  │                           │                        │─── Pull Image ─────────►│
  │                           │                        │     docker pull ubuntu │
  │                           │                        │                        │
  │                           │                        │◄── Image Ready ─────────│
  │                           │                        │                        │
  │                           │                        │─── Create Container ──►│
  │                           │                        │     docker create ...   │
  │                           │                        │                        │
  │                           │                        │◄── Container Created ──│
  │                           │                        │                        │
  │                           │                        │─── Start Container ───►│
  │                           │                        │     docker start ...    │
  │                           │                        │                        │
  │                           │                        │◄── Container Running ──│
  │                           │                        │                        │
  │                           │                        │─── Clone Git ──────────►│
  │                           │                        │     git clone ...       │
  │                           │                        │                        │
  │                           │◄── VM Created ────────│◄── VM Ready ──────────│
  │                           │     { vm_id, status } │     { vm_id, status } │
  │                           │                        │                        │
  │◄── VM Info ──────────────│                        │                        │
  │     { vm_id, status:     │                        │                        │
  │       running }          │                        │                        │
```

---

### 4.3 Terminal 会话流程

```
Browser                     Backend                  TaskFlow                  Runner                    Docker
  │                           │                        │                        │                       │
  │  1. 点击"连接终端"         │                        │                        │                       │
  │─── Connect ─────────────►│                        │                        │                       │
  │                           │                        │                        │                       │
  │                           │─── Terminal ──────────►│                        │                       │
  │                           │     { vm_id }           │                        │                       │
  │                           │                        │                        │                       │
  │                           │                        │◄─── Terminal ID ──────│                        │
  │                           │                        │                        │                       │
  │◄── WebSocket URL ────────│◄─── URL ─────────────│                        │                       │
  │                           │                        │                        │                       │
  │  2. 建立 WebSocket       │                        │                        │                       │
  │─── WS Connect ──────────────────────────────────────────────────────────►│                       │
  │                           │                        │                        │                       │
  │                           │                        │                        │─── Docker Exec ────►│
  │                           │                        │                        │     /bin/bash         │
  │                           │                        │                        │◄── PTY Created ──────│
  │                           │                        │                        │                       │
  │  3. 用户输入命令          │                        │                        │                       │
  │─── { type: "input",      │                        │                        │                       │
  │       data: "ls -la\n" } ────────────────────────────────────────────►│─── Write PTY ───────►│
  │                           │                        │                        │                       │
  │                           │                        │                        │◄── PTY Output ───────│
  │◄── { type: "output",     │                        │                        │                       │
  │       data: "total 64\n" } ────────────────────────────────────────────────│◄── Read PTY ────────│
  │                           │                        │                        │                       │
  │  (重复 3-4，直到用户关闭)   │                        │                        │                       │
```

**Terminal 数据格式**：
```json
// 输入
{
    "type": "input",
    "data": "ls -la\n"
}

// 输出
{
    "type": "output",
    "data": "total 64\ndrwxr-xr-x 5 root root 4096 ...\n"
}

// 调整大小
{
    "type": "resize",
    "cols": 80,
    "rows": 24
}
```

---

### 4.4 Task 执行流程

```
Browser                     Backend                  TaskFlow                  Runner
  │                           │                        │                        │
  │  1. 提交任务              │                        │                        │
  │─── CreateTask ──────────►│                        │                        │
  │     { vm_id, text,       │                        │                        │
  │       coding_agent, llm } │                        │                        │
  │                           │                        │                        │
  │                           │─── CreateTask ───────►│                        │
  │                           │                        │                        │
  │                           │                        │─── 获取 VM 信息 ────────│
  │                           │                        │                        │
  │                           │                        │─── Agent 执行 ────────►│
  │                           │                        │     opencode run ...    │
  │                           │                        │                        │
  │                           │                        │◄── Progress ──────────│
  │                           │◄── Stream ───────────│◄── Report ───────────│
  │◄── SSE Stream ──────────│                        │                        │
  │     { event: "progress",│                        │                        │
  │       data: "分析代码..." }│                        │                        │
  │                           │                        │                        │
  │     { event: "progress", │                        │                        │
  │       data: "生成代码..." }│                        │                        │
  │                           │                        │                        │
  │     { event: "complete", │                        │                        │
  │       data: { result,    │                        │                        │
  │              files } }    │                        │                        │
  │                           │                        │◄── Task Complete ─────│
  │                           │◄── Task Complete ────│                        │
  │                           │                        │                        │
  │◄── Final Result ─────────│                        │                        │
```

---

## 五、Token 认证流程

### 5.1 Token 类型

MonkeyCode 使用两种 Token：

| Token 类型 | 值 | 用途 | 验证方 |
|-----------|-----|------|--------|
| **Orchestrator Token** | 用户 Token | 用户认证 | Backend |
| **Agent Token** | Runner Token | Runner 认证 | TaskFlow → Backend |

### 5.2 Token 验证流程

```
用户请求
    │
    ▼
Backend
    │
    ├── 检查 Token 类型
    │
    ├── TokenKind = "orchestrator"
    │   └── 直接使用，用户认证
    │
    └── TokenKind = "agent"
        └── 转发到 TaskFlow 验证
            │
            ▼
        TaskFlow
            │
            └── backend.CheckToken(token)
                │
                ▼
            Backend
                │
                └── 验证 token，返回用户信息
                    │
                    ▼
                TokenInfo { user_id, team_id, ... }
```

### 5.3 Runner Token 特殊处理

```go
// taskflow/internal/server/grpc.go
func (s *GRPCServer) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    // 1. 验证 Token
    token, err := s.backend.CheckToken(ctx, &backend.CheckTokenReq{
        Token: req.Token,
    })
    if err != nil {
        return nil, err
    }

    // 2. TokenKind 必须是 "orchestrator"
    if token.Kind != "orchestrator" {
        return &pb.RegisterResponse{
            Success: false,
            Message: "invalid token type",
        }, nil
    }

    // 3. 使用 token.Token 作为 RunnerID
    runnerID := token.Token

    // 4. Runner 后续请求中携带此 runnerID
}
```

---

## 六、网络配置详解

### 6.1 GRPC 配置关系

| 变量 | 示例值 | 用途 |
|------|---------|------|
| **GRPC_HOST** | localhost | gRPC 主机地址 |
| **GRPC_PORT** | 50051 | gRPC 端口 |
| **GRPC_URL** | localhost:50051 | gRPC 完整地址（HOST:PORT） |

### 6.2 内部架构（开发环境）

```
┌──────────────────────────────────────────────────────────────┐
│                        本地环境                              │
│                                                        │
│   Runner ──── HTTP ────► Backend (:8888)                  │
│   Browser ──── HTTP ────► Backend (:8888)                  │
│                                                        │
└────────────────────────┬─────────────────────────────────┘
                         │
                         │ 本地网络
                         ▼
┌──────────────────────────────────────────────────────────────┐
│                       单机 / 本地网络                          │
│                                                        │
│   ┌────────────────────────────────────────────────────┐  │
│   │              Backend 服务                          │  │
│   │   监听 localhost:8888                             │  │
│   └────────────────────────────────────────────────────┘  │
│                                                        │
│   ┌────────────────────────────────────────────────────┐  │
│   │              TaskFlow 服务                          │  │
│   │   监听 localhost:50051 (gRPC)                    │  │
│   │   监听 localhost:8889 (HTTP)                      │  │
│   └────────────────────────────────────────────────────┘  │
│                                                        │
│   ┌────────────────────────────────────────────────────┐  │
│   │              Redis 服务                            │  │
│   │   监听 localhost:6379                             │  │
│   └────────────────────────────────────────────────────┘  │
│                                                        │
│   ┌────────────────────────────────────────────────────┐  │
│   │              Runner 节点                            │  │
│   │   监听 localhost:8080                            │  │
│   └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

### 6.3 配置示例（开发环境）

```yaml
# Runner 配置
GRPC_URL = localhost:50051

# 或分开配置
GRPC_HOST = localhost
GRPC_PORT = 50051

# Backend 配置
TASKFLOW_ADDR = localhost:50051
```

### 6.4 生产环境配置

```yaml
# Runner 配置
# GRPC_URL: TaskFlow gRPC 地址
GRPC_URL = localhost:50051

# 或分开配置
GRPC_HOST = localhost
GRPC_PORT = 50051
```

---

## 七、数据存储结构

### 7.1 Redis Key 设计

| Key Pattern | 类型 | 说明 | TTL |
|-------------|------|------|-----|
| `runner:{id}` | Hash | Runner 信息 | 60s |
| `vm:{id}` | Hash | VM 信息 | - |
| `task:{id}` | Hash | Task 信息 | - |
| `portforward:{id}` | Hash | PortForward 信息 | - |
| `user:runners:{user_id}` | Set | 用户-Runner 关联 | - |
| `user:vms:{user_id}` | Set | 用户-VM 关联 | - |
| `vm:{vm_id}:forwards` | Set | VM-端口转发关联 | - |

### 7.2 数据结构

```go
// Runner
type Runner struct {
    ID       string            // Runner ID (= Token)
    UserID   string            // 用户 ID
    Hostname string            // 主机名
    IP       string            // IP 地址
    Status   string            // online/offline
    LastSeen int64             // 最后活跃时间
    Capacity map[string]int64  // {cores, memory, disk}
}

// VM
type VM struct {
    ID            string               // VM ID
    HostID        string               // Runner ID
    UserID        string               // 用户 ID
    ContainerID   string               // Docker 容器 ID
    Status        VirtualMachineStatus // pending/running/stopped
    GitURL        string               // Git 仓库 URL
    ImageURL      string               // Docker 镜像 URL
    Cores         int32                // CPU 核心数
    Memory        uint64               // 内存 (MB)
    TTL          TTL                 // 生命周期
    CreatedAt    int64                // 创建时间
}
```

---

## 八、安全考虑

### 8.1 网络隔离

- Backend、TaskFlow、Redis 部署在内网
- 只有 Nginx/API Gateway 对外暴露
- Runner 节点通过 NAT 或内网访问 TaskFlow

### 8.2 Token 安全

- Runner Token 有效期由 Backend 管理
- TaskFlow 通过 Backend 验证 Token
- 心跳机制检测异常 Runner

### 8.3 容器隔离

- 每个 VM 运行在独立 Docker 容器
- 可配置资源限制（CPU、内存）
- Runner 部署在专用网络

---

## 九、部署架构

### 9.1 开发环境

```
┌────────────────────────────────────────────────────────────┐
│                      本地机器                               │
│                                                        │
│   Backend ──── TaskFlow ──── Runner                       │
│   (:8888)      (:50051)     (:8080)                     │
│                                                        │
│   Redis ──── PostgreSQL                                  │
│   (:6379)     (:5432)                                   │
└────────────────────────────────────────────────────────────┘
```

### 9.2 生产环境（推测）

```
┌────────────────────────────────────────────────────────────┐
│                         本地网络                             │
│                                                        │
│   ┌────────────────────────────────────────────────┐   │
│   │              Backend + TaskFlow                  │   │
│   │              localhost:8888, :50051              │   │
│   └───────────────┬────────────────────────────────┘   │
│                   │                                       │
└───────────────────┼───────────────────────────────────────┘
                    │
                    │ 本地网络
        ┌───────────┴───────────┐
        ▼                       ▼
┌───────────────────┐   ┌───────────────────┐
│   Backend         │   │   TaskFlow        │
│   localhost:8888   │   │   localhost:50051  │
└───────────────────┘   └───────────────────┘
```

---

## 十、配置参考

### 10.1 Runner 环境变量

```bash
# 必需
TOKEN=your-runner-token

# TaskFlow 连接
# 方式一：直接使用完整地址
GRPC_URL=localhost:50051

# 方式二：分开配置（可选）
GRPC_HOST=localhost
GRPC_PORT=50051

# 日志
LOG_LEVEL=info
```

**配置说明**：
| 变量 | 示例值 | 说明 |
|------|--------|------|
| `GRPC_URL` | localhost:50051 | TaskFlow gRPC 地址（推荐使用此配置） |
| `GRPC_HOST` | localhost | gRPC 主机地址 |
| `GRPC_PORT` | 50051 | gRPC 端口 |

### 10.2 TaskFlow 配置

```yaml
server:
  http_addr: ":8889"
  grpc_addr: ":50051"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

backend:
  addr: "http://localhost:8888"

runner:
  token: "test-runner-token"
  heartbeat_timeout: 60
```

### 10.3 Backend 配置

```yaml
server:
  http_addr: ":8888"

taskflow:
  addr: "localhost:50051"
```

---

## 十一、故障排查

### 11.1 Runner 无法注册

```bash
# 1. 检查 TOKEN 是否正确
echo $TOKEN

# 2. 检查网络连接
curl -v telnet://$GRPC_HOST:$GRPC_PORT

# 3. 查看 Runner 日志
docker logs monkeycode_runner

# 4. 检查 TaskFlow 是否正常
curl http://localhost:8889/health
```

### 11.2 VM 创建失败

```bash
# 1. 检查 Docker 是否运行
docker info

# 2. 检查镜像是否存在
docker images | grep ubuntu

# 3. 查看 Runner 容器日志
docker logs -f monkeycode_runner
```

### 11.3 Terminal 连接失败

```bash
# 1. 检查 WebSocket 端点
curl -v ws://localhost:8080/terminal?vm_id=test

# 2. 检查 VM 是否运行
docker ps | grep vm-

# 3. 检查容器内部 bash
docker exec -it <container_id> bash
```

---

## 十二、相关文档

- [Runner 项目文档](../runner/README.md)
- [TaskFlow 项目文档](../taskflow/README.md)
- [Backend 项目文档](../backend/README.md)
- [Orchestrator 设计文档](./orchestrator-design.md)
- [安装器设计文档](./installer-design.md)

---

## 十三、更新日志

| 日期 | 版本 | 更新内容 |
|------|------|----------|
| 2024 | v1.0 | 初始版本 |

---

*本文档由 AI 生成，如有疑问请联系维护者*
