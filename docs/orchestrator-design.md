# MonkeyCode Orchestrator 架构设计

## 一、项目概述

Orchestrator 是 MonkeyCode 分布式 AI 代码执行平台的核心服务，负责：
- 管理 Docker 容器生命周期
- 执行 AI 编程任务
- 提供交互式终端
- 支持多种 AI 编程助手
- 与 TaskFlow 协调服务通信

## 二、系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户浏览器                                 │
│              (WebSocket / HTTP API)                             │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                 Orchestrator 服务 (:8080)                        │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                      HTTP Server                             │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────┐ │ │
│  │  │ Terminal │  │  Task    │  │   VM     │  │Port Forward│ │ │
│  │  │ Handler  │  │ Handler  │  │ Handler  │  │  Handler   │ │ │
│  │  └──────────┘  └──────────┘  └──────────┘  └────────────┘ │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                             │                                   │
│                             ▼                                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                    AI Agent Manager                          │ │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────────┐   │ │
│  │  │  OpenCode  │  │  Cursor    │  │   Claude Code      │   │ │
│  │  │  Agent     │  │  Agent     │  │   Agent            │   │ │
│  │  └────────────┘  └────────────┘  └────────────────────┘   │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                             │                                   │
│                             ▼                                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                     Task Executor                            │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                             │                                   │
│                             ▼                                   │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                    Docker Manager                            │ │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────────────┐   │ │
│  │  │ Container  │  │   Image    │  │    Network         │   │ │
│  │  │ Manager    │  │  Manager   │  │    Manager         │   │ │
│  │  └────────────┘  └────────────┘  └────────────────────┘   │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                    gRPC Client                               │ │
│  │                 (TaskFlow 通信)                              │ │
│  └──────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                             │
                             │ gRPC
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      TaskFlow 服务 (:50051)                       │
└─────────────────────────────────────────────────────────────────┘
```

## 三、核心模块

### 3.1 HTTP Server

提供 REST API 和 WebSocket 接口。

**端口**: 8080

**端点**:
| 端点 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/terminal` | WS | 交互式终端 |
| `/task` | POST | 创建任务 |
| `/internal/vm` | POST/DELETE | VM 管理 |
| `/internal/port-forward` | GET/POST | 端口转发 |

### 3.2 AI Agent Manager

管理多种 AI 编程助手。

```go
type AgentManager struct {
    agents map[string]Agent
}

type Agent interface {
    Execute(ctx context.Context, opts ExecuteOptions, progress chan<- string) (string, error)
    Name() string
    Version() string
}
```

**支持的 Agent**:

| Agent | 状态 | 说明 |
|-------|------|------|
| `opencode` | ✅ 已实现 | OpenCode AI 编码助手 |
| `cursor` | ⬜ 规划中 | Cursor IDE CLI |
| `claude` | ⬜ 规划中 | Claude Code CLI |

### 3.3 Task Executor

执行 AI 编程任务。

**流程**:
```
创建任务 → 选择 Agent → 执行 → 报告进度 → 返回结果
```

### 3.4 Docker Manager

管理 Docker 容器。

**功能**:
- 容器生命周期管理
- 镜像拉取
- 网络配置
- 资源限制（CPU、内存）

### 3.5 gRPC Client

与 TaskFlow 通信。

**功能**:
- 注册 Orchestrator
- 心跳保活
- 上报状态
- 创建/删除 VM 记录

## 四、数据流

### 4.1 任务执行流程

```
1. 用户通过 API 创建任务
   POST /task
   {
     "agent": "opencode",
     "vm_id": "vm-123",
     "text": "帮我写一个 Hello World",
     "model": "gpt-4"
   }

2. Task Executor 创建 Task 对象
   Task {
     ID: "task-uuid",
     Agent: "opencode",
     Status: pending
   }

3. 获取或创建 VM (Docker 容器)
   VM {
     ID: "vm-123",
     ContainerID: "container-456",
     Status: running
   }

4. 在容器中执行 AI Agent
   opencode run --model gpt-4 "帮我写一个 Hello World"

5. 实时报告进度
   progress: "正在分析代码结构..."
   progress: "正在生成代码..."
   progress: "完成！"

6. 返回结果
   {
     "status": "completed",
     "result": "代码已生成...",
     "files": ["main.py"]
   }
```

### 4.2 终端会话流程

```
1. 用户连接 WebSocket
   ws://localhost:8080/terminal?vm_id=vm-123

2. 获取 VM 对应的容器
   ContainerID: "container-456"

3. Docker Exec 创建交互式 bash
   docker exec -it container-456 /bin/bash

4. 双向转发数据
   WebSocket ↔ PTY ↔ Container

5. 会话结束，清理资源
```

## 五、环境变量配置

```bash
# ===========================================
# Orchestrator 配置
# ===========================================

# 认证
TOKEN=your-auth-token

# TaskFlow 连接
GRPC_HOST=localhost
GRPC_PORT=50051
GRPC_URL=localhost:50051

# 宿主机标识
HOST_ID=host-uuid

# 日志
LOG_LEVEL=info

# 资源限制
MAX_CORES=8
MAX_MEMORY=16384

# AI API 配置
OPENAI_API_KEY=sk-xxx
OPENAI_BASE_URL=https://api.openai.com/v1
```

## 六、Docker 镜像

### 6.1 基础镜像

基于 `devbox:bookworm`，添加 OpenCode。

```dockerfile
FROM ghcr.io/chaitin/monkeycode/devbox:bookworm

# 安装 OpenCode
RUN curl -fsSL https://opencode.ai/install.sh | sh

# 验证
RUN opencode --version
```

### 6.2 镜像标签

| 标签 | 说明 |
|------|------|
| `latest` | 最新版本 |
| `v1.x.x` | 语义版本 |

## 七、部署架构

### 7.1 单机部署

```yaml
# docker-compose.yml
services:
  orchestrator:
    image: ghcr.io/chaitin/monkeycode/orchestrator:latest
    ports:
      - "8080:8080"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    env_file:
      - .env
```

### 7.2 多机部署

每台宿主机运行一个 Orchestrator 实例，通过 TaskFlow 协调。

## 八、API 文档

### 8.1 创建任务

```bash
POST /task
Content-Type: application/json

{
    "vm_id": "vm-123",        # VM ID（可选，不提供则创建新 VM）
    "user_id": "user-1",      # 用户 ID
    "agent": "opencode",      # AI 助手类型
    "text": "帮我写一个 Hello World 程序",
    "model": "gpt-4",         # AI 模型
    "api_key": "sk-xxx",      # API 密钥（可选）
    "api_base": "https://api.openai.com/v1"  # API 地址
}

响应: 201 Created
{
    "id": "task-uuid",
    "status": "pending",
    "vm_id": "vm-123",
    "agent": "opencode",
    "created_at": "2024-01-01T00:00:00Z"
}
```

### 8.2 获取任务状态

```bash
GET /task/:id

响应: 200 OK
{
    "id": "task-uuid",
    "status": "running",  # pending, running, completed, failed
    "progress": "正在生成代码...",
    "result": "",
    "error": ""
}
```

### 8.3 WebSocket 终端

```bash
ws://localhost:8080/terminal?vm_id=vm-123

# 发送命令
{
    "type": "input",
    "data": "ls -la\n"
}

# 接收输出
{
    "type": "output",
    "data": "total 64\n..."
}

# 调整终端大小
{
    "type": "resize",
    "cols": 80,
    "rows": 24
}
```

## 九、安全考虑

1. **容器隔离**: 每个 VM 运行在独立容器中
2. **资源限制**: CPU、内存限制防止资源滥用
3. **Token 认证**: 所有 API 调用需要认证 Token
4. **网络隔离**: 容器网络与宿主机隔离

## 十、扩展计划

### Phase 1: 基础功能 ✅
- [x] Docker 容器管理
- [x] OpenCode 集成
- [x] 任务执行
- [x] WebSocket 终端

### Phase 2: 多 Agent 支持
- [ ] Cursor CLI 集成
- [ ] Claude Code 集成
- [ ] Agent 抽象层

### Phase 3: 高级功能
- [ ] GPU 支持
- [ ] 多架构镜像
- [ ] 分布式存储

## 十一、目录结构

```
orchestrator/
├── cmd/
│   └── server/
│       └── main.go           # 入口
├── internal/
│   ├── agent/                # AI Agent
│   │   ├── manager.go        # Agent 管理器
│   │   ├── opencode.go       # OpenCode 实现
│   │   ├── cursor.go         # Cursor 实现（规划）
│   │   └── claude.go         # Claude Code 实现（规划）
│   ├── docker/              # Docker 管理
│   │   ├── manager.go
│   │   ├── container.go
│   │   ├── image.go
│   │   └── network.go
│   ├── executor/            # 任务执行器
│   │   └── executor.go
│   ├── handler/             # HTTP 处理器
│   │   ├── task.go
│   │   ├── terminal.go
│   │   └── vm.go
│   ├── task/                # 任务管理
│   │   └── manager.go
│   ├── terminal/             # 终端管理
│   │   ├── manager.go
│   │   └── pty.go
│   ├── vm/                   # VM 管理
│   │   └── manager.go
│   ├── config/               # 配置
│   │   └── config.go
│   └── grpc/                 # gRPC 客户端
│       └── client.go
├── pkg/                      # 公共包
│   └── proto/                # Protocol Buffers
├── Dockerfile
├── docker-compose.yml
└── README.md
```

## 十二、依赖

### Go 依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| `docker/docker` | v28.5.2 | Docker SDK |
| `grpc/grpc` | v1.79.3 | gRPC |
| `gorilla/websocket` | v1.5.3 | WebSocket |
| `google/uuid` | v1.6.0 | UUID 生成 |

### 系统依赖

| 依赖 | 版本 | 用途 |
|------|------|------|
| Docker | 20.10+ | 容器运行时 |
| Git | 2.0+ | 代码克隆 |
| OpenCode | latest | AI 编码助手 |
