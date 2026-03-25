# MonkeyCode 通讯时序图

## 一、Runner 注册流程

```mermaid
sequenceDiagram
    participant R as Runner
    participant TF as TaskFlow
    participant B as Backend
    participant Redis as Redis

    R->>R: 读取 TOKEN 环境变量
    R->>TF: Register(token, hostname, localhost, cores, memory)
    TF->>B: CheckToken(token)
    B->>B: 验证 Token
    B-->>TF: TokenInfo(user_id, team_id)
    TF->>Redis: RegisterRunner(runner_info, ttl=60s)
    TF->>B: ReportHostInfo(host_info)
    TF-->>R: RegisterResponse(runner_id)
    R->>R: 启动心跳协程

    loop 每 10 秒
        R->>TF: Heartbeat(vm_count, task_count)
        TF->>Redis: UpdateLastSeen(runner_id)
        TF-->>R: HeartbeatResponse
    end
```

---

## 二、VM 创建流程

```mermaid
sequenceDiagram
    participant Browser
    participant B as Backend
    participant TF as TaskFlow
    participant R as Runner
    participant Docker
    participant Git

    Browser->>B: CreateVM(user_id, git_url, image_url, ttl)
    B->>TF: CreateVM(host_id, image, cores, memory)
    TF->>R: 选择可用 Runner
    R->>Docker: Pull Image
    Docker-->>R: Image Ready
    R->>Docker: Create Container
    Docker-->>R: Container Created
    R->>Docker: Start Container
    Docker-->>R: Container Running
    alt Git 仓库配置
        R->>Git: Clone Repository
        Git-->>R: Clone Complete
    end
    R-->>TF: VM Ready(vm_id, status)
    TF-->>B: VM Created
    B-->>Browser: VM Info

    Browser->>Browser: 连接 Terminal
    Browser->>R: WebSocket /terminal?vm_id=xxx
    R->>Docker: Docker Exec Create(/bin/bash)
    Docker-->>R: Exec ID
    R->>Docker: Docker Exec Attach
    Docker-->>R: PTY Connected

    loop 交互
        Browser->>R: Terminal Input
        R->>Docker: Write PTY
        Docker-->>R: PTY Output
        R-->>Browser: Terminal Output
    end
```

---

## 三、Task 执行流程

```mermaid
sequenceDiagram
    participant Browser
    participant B as Backend
    participant TF as TaskFlow
    participant R as Runner
    participant Agent as OpenCode/Claude

    Browser->>B: CreateTask(vm_id, text, llm_config)
    B->>TF: CreateTask(vm_id, task_info)
    TF->>R: 创建 Task 记录
    R->>R: 更新 Task 状态为 running
    TF-->>B: Task Created
    B-->>Browser: Task Accepted

    alt 任务执行
        R->>Agent: 执行任务
        loop 实时进度
            Agent-->>R: Progress Update
            R-->>TF: Stream Progress
            TF-->>B: Stream Progress
            B-->>Browser: SSE Progress
        end

        alt 成功
            Agent-->>R: Task Complete(result)
            R->>R: 更新 Task 状态为 completed
            R-->>TF: Task Complete
            TF-->>B: Task Complete
            B-->>Browser: Final Result
        else 失败
            Agent-->>R: Task Failed(error)
            R->>R: 更新 Task 状态为 failed
            R-->>TF: Task Failed
            TF-->>B: Task Failed
            B-->>Browser: Error Response
        end
    end
```

---

## 四、Token 认证流程

```mermaid
sequenceDiagram
    participant U as 用户
    participant B as Backend
    participant TF as TaskFlow

    rect rgb(200, 200, 200)
        Note over U,TF: 用户 Token 认证流程
        U->>B: 请求 + Token
        B->>B: 检查 Token 类型
        alt TokenKind = "orchestrator"
            B->>B: 直接使用
            B-->>U: 认证成功
        else TokenKind = "agent"
            B-->>U: 返回 Token
        end
    end

    rect rgb(200, 220, 200)
        Note over U,TF: Runner Token 认证流程
        R->>TF: Register(token)
        TF->>B: CheckToken(token)
        B->>B: 验证 Token
        alt Token 无效
            B-->>TF: Error
            TF-->>R: Register Failed
        else Token 有效
            B-->>TF: TokenInfo
            TF->>TF: 存储 Runner 信息
            TF-->>R: Register Success
        end
    end
```

---

## 五、PortForward 流程

```mermaid
sequenceDiagram
    participant Browser
    participant B as Backend
    participant TF as TaskFlow
    participant R as Runner
    participant External as 外部服务

    Browser->>B: CreatePortForward(vm_id, local_port)
    B->>TF: CreatePortForward(vm_id, port)
    TF->>R: 获取 VM 容器 ID
    R->>R: 启动端口转发
    R-->>TF: Forward Created(access_url)
    TF-->>B: Forward Created
    B-->>Browser: Access URL

    External->>Browser: 访问 http://host:port
    Browser-->>External: Service Response

    alt 关闭转发
        Browser->>B: ClosePortForward(forward_id)
        B->>TF: ClosePortForward
        TF->>R: Stop Forward
        R->>R: 停止转发
        R-->>TF: Forward Closed
        TF-->>B: Closed
        B-->>Browser: Success
    end
```

---

## 六、状态同步流程

```mermaid
sequenceDiagram
    participant R as Runner
    participant TF as TaskFlow
    participant Redis

    loop Runner 生命周期
        R->>TF: Register
        TF->>Redis: SET runner:{id} info EX 60
        TF->>B: ReportHostInfo
    end

    loop 心跳 (每 10s)
        R->>TF: Heartbeat
        TF->>Redis: EXPIRE runner:{id} 60
        TF-->>R: OK
    end

    loop VM 状态变化
        R->>R: VM 状态变化
        R-->>TF: VM Status Update
        TF->>Redis: HSET vm:{id} status
    end

    loop Task 状态变化
        R->>R: Task 状态变化
        R-->>TF: Task Status Update
        TF->>Redis: HSET task:{id} status
    end

    alt Runner 宕机
        Note over Redis: 60s 后 runner:{id} 自动过期
        TF->>TF: 检测 Runner 离线
        TF->>TF: 清理相关资源
    end
```

---

## 七、文件操作流程

```mermaid
sequenceDiagram
    participant Browser
    participant B as Backend
    participant TF as TaskFlow
    participant R as Runner
    participant Container

    rect rgb(200, 200, 255)
        Note over Browser,Container: 文件上传
        Browser->>B: UploadFile(vm_id, path, content)
        B->>TF: FileOperation(upload)
        TF->>R: 文件数据
        R->>Container: tar + docker cp
        Container-->>R: Upload Success
        R-->>TF: Success
        TF-->>B: Success
        B-->>Browser: Upload Complete
    end

    rect rgb(200, 255, 200)
        Note over Browser,Container: 文件下载
        Browser->>B: DownloadFile(vm_id, path)
        B->>TF: FileOperation(download)
        TF->>R: 请求文件
        R->>Container: docker cp + tar
        Container-->>R: 文件内容
        R-->>TF: 文件数据
        TF-->>B: 文件数据
        B-->>Browser: 文件下载
    end

    rect rgb(255, 200, 200)
        Note over Browser,Container: 文件列表
        Browser->>B: ListFiles(vm_id, path)
        B->>TF: FileOperation(list)
        TF->>R: 列出文件
        R->>Container: ls + stat
        Container-->>R: 文件列表
        R-->>TF: 文件列表
        TF-->>B: 文件列表
        B-->>Browser: 文件列表
    end
```

---

## 八、错误处理流程

```mermaid
sequenceDiagram
    participant R as Runner
    participant TF as TaskFlow
    participant B as Backend

    alt Runner 注册失败
        R->>TF: Register
        TF->>B: CheckToken
        B-->>TF: Invalid Token
        TF-->>R: Register Failed(error)
        R->>R: 打印错误日志
        R->>R: 退出或重试
    end

    alt VM 创建失败
        R->>Docker: Pull Image
        Docker-->>R: Image Pull Failed
        R-->>TF: VM Create Failed(error)
        TF->>TF: 记录错误
        TF-->>B: VM Create Failed
        B-->>Browser: Error Response
    end

    alt Task 执行失败
        R->>Agent: Execute Task
        Agent-->>R: Execution Failed(error)
        R->>R: 更新 Task 状态为 failed
        R-->>TF: Task Failed(error)
        TF->>TF: 记录错误
        TF-->>B: Task Failed
        B-->>Browser: Error Response
    end

    alt Runner 离线
        Note over TF: 心跳超时
        TF->>TF: Runner 标记为 offline
        TF->>TF: 清理资源
        TF-->>B: Runner Offline
    end
```

---

## 九、并发控制流程

```mermaid
sequenceDiagram
    participant B as Backend
    participant TF as TaskFlow
    participant R1 as Runner 1
    participant R2 as Runner 2
    participant R3 as Runner N

    par 负载均衡
        B->>TF: CreateVM
        TF->>TF: 选择 Runner
        TF->>TF: 检查 Runner 容量
        TF->>R1: CreateVM
        R1-->>TF: VM Created
    and 并发创建
        B->>TF: CreateVM 1
        B->>TF: CreateVM 2
        B->>TF: CreateVM 3
        TF->>R1: CreateVM 1
        TF->>R2: CreateVM 2
        TF->>R3: CreateVM 3
    end

    par 心跳处理
        R1->>TF: Heartbeat 1
        R2->>TF: Heartbeat 2
        R3->>TF: Heartbeat N
        TF->>TF: 更新 Redis
    end

    par 容量检查
        TF->>TF: 检查 Runner 1 容量
        TF->>TF: 检查 Runner 2 容量
        TF->>TF: 检查 Runner N 容量
    end
```

---

## 十、重连机制流程

```mermaid
sequenceDiagram
    participant R as Runner
    participant TF as TaskFlow

    rect rgb(255, 200, 200)
        Note over R,TF: Runner 异常断开
        R->>TF: Heartbeat
        TF-->>R: Connection Lost
    end

    loop 重连尝试 (最多 5 次)
        R->>R: 等待 2^n 秒 (指数退避)
        R->>TF: Register
        alt 注册成功
            TF-->>R: Register Success
            R->>R: 重启心跳
        else 注册失败
            R->>R: 重试
        end
    end

    alt 重连成功
        R->>TF: 注册成功
        R->>TF: 恢复 VM 状态
        TF->>TF: 同步状态
    else 重连失败 (5 次)
        R->>R: 打印错误日志
        R->>R: 退出程序
    end
```

---

## 十一、健康检查流程

```mermaid
sequenceDiagram
    participant Monitor
    participant R as Runner
    participant TF as TaskFlow
    participant B as Backend

    loop 定期检查
        Monitor->>R: GET /health
        R-->>Monitor: OK

        Monitor->>TF: GET /health
        TF-->>Monitor: OK

        Monitor->>B: GET /health
        B-->>Monitor: OK
    end

    alt Runner 健康检查失败
        Monitor->>Monitor: 触发告警
        Monitor->>Monitor: 自动重启 Runner
    end

    loop TaskFlow 内部检查
        TF->>TF: 检查 Redis 连接
        TF->>TF: 检查 Runner 连接
    end
```

---

## 十二、数据流总览

```mermaid
flowchart LR
    subgraph External["外部"]
        Browser[浏览器]
        ExternalAPI[外部 API]
    end

    subgraph Backend["Backend 服务 (:8888)"]
        Auth[认证模块]
        Router[请求路由]
    end

    subgraph TaskFlow["TaskFlow 服务 (:50051)"]
        GRPCServer[gRPC 服务]
        Store[Redis 存储]
        Manager[Runner 管理]
    end

    subgraph Runner["Runner 集群 (:8080)"]
        HTTPAPI[HTTP API]
        Agent[AI Agent]
        Docker[Docker 容器]
    end

    Browser -->|HTTPS| Router
    Router -->|HTTP| GRPCServer
    ExternalAPI -->|HTTPS| Router
    GRPCServer -->|gRPC| HTTPAPI
    HTTPAPI -->|Docker API| Docker
    Agent -->|Exec| Docker
    GRPCServer -->|Redis| Store
    Manager -->|Register| GRPCServer
    Manager -->|Heartbeat| GRPCServer
```

---

*本文档为 MonkeyCode 通讯流程的时序图补充说明*
