# MonkeyCode Runner

MonkeyCode Runner 是分布式 AI 代码执行平台的核心执行节点，负责管理 Docker 容器、执行任务、提供交互式终端等。

## 功能特性

- 🐳 **Docker 容器管理** - 自动创建和管理隔离的容器环境
- 💻 **交互式终端** - 基于 WebSocket 的实时终端会话
- 📝 **任务执行** - 异步任务管理和执行
- 🔌 **端口转发** - 容器端口到宿主机的转发
- 📡 **TaskFlow 集成** - 与任务编排服务通信

## 快速开始

### 前置要求

- Docker 20.10+
- Go 1.21+

### 本地运行

```bash
# 设置环境变量
export TOKEN=your-auth-token
export GRPC_URL=localhost:50051

# 运行
make run
```

### Docker 部署

#### 方法 1: 使用 Docker Compose（推荐）

```bash
# 创建 .env 文件
cat > .env << EOF
TOKEN=your-auth-token
GRPC_URL=localhost:50051
HOST_STORAGE_DIR=/data/monkeycode_runner/data
EOF

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

#### 方法 2: 直接使用 Docker

```bash
# 构建镜像
make docker-build

# 运行容器
make docker-run

# 或使用完整配置
make docker-run-full
```

#### 方法 3: 直接运行镜像

```bash
# 拉取镜像（如果已推送）
docker pull ghcr.io/chaitin/monkeycode/runner:latest

# 运行容器
docker run -d \
  --name monkeycode_runner \
  --restart unless-stopped \
  -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /data/monkeycode_runner/data:/app/data \
  -e TOKEN=your-token \
  -e GRPC_URL=localhost:50051 \
  ghcr.io/chaitin/monkeycode/runner:latest
```

## 环境变量

| 变量 | 默认值 | 必填 | 说明 |
|------|--------|------|------|
| `TOKEN` | - | ✅ | 认证令牌 |
| `GRPC_URL` | localhost:50051 | ❌ | TaskFlow gRPC 地址 |
| `GRPC_HOST` | localhost | ❌ | gRPC 主机 |
| `GRPC_PORT` | 50051 | ❌ | gRPC 端口 |

## API 接口

### 健康检查

```bash
GET /health

# 响应: ok
```

### 创建任务

```bash
POST /task
Content-Type: application/json

{
    "vm_id": "vm-123",
    "user_id": "user-1",
    "text": "帮我写一个 Hello World 程序",
    "model": "gpt-4"
}

# 响应: 202 Accepted
{
    "id": "task-uuid",
    "status": "pending",
    ...
}
```

### WebSocket 终端

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
```

### 端口转发

```bash
# 创建转发
POST /internal/port-forward
{
    "container_id": "container-123",
    "host_port": 9000,
    "container_port": 8080,
    "protocol": "tcp"
}

# 列出转发
GET /internal/port-forward

# 关闭转发
POST /internal/port-forward/close
{
    "id": "forward-id"
}
```

## Docker 配置说明

### 卷挂载

```yaml
volumes:
  # Docker socket - 必须挂载，允许 runner 管理 Docker 容器
  - /var/run/docker.sock:/var/run/docker.sock

  # 数据目录 - 持久化存储
  - ${HOST_STORAGE_DIR:-./data}:/app/data
```

### 权限说明

Runner 需要以下权限：

1. **Docker Socket 访问** - 用于管理 Docker 容器
2. **网络访问** - 与 TaskFlow 和外部服务通信
3. **文件系统访问** - 读写工作目录

### 资源限制

```yaml
deploy:
  resources:
    limits:
      cpus: '8'        # 最多 8 CPU 核心
      memory: 16G      # 最多 16GB 内存
    reservations:
      cpus: '2'        # 预留 2 CPU 核心
      memory: 4G       # 预留 4GB 内存
```

## 构建和发布

### 构建镜像

```bash
# 单架构构建
make docker-build

# 多架构构建（需要 Docker buildx）
make docker-buildx
```

### 发布镜像

```bash
# 登录到 GitHub Container Registry
docker login ghcr.io -u GITHUB_USERNAME

# 推送镜像
make docker-push
```

## 运维管理

### 查看容器状态

```bash
docker-compose ps
# 或
docker ps | grep runner
```

### 查看日志

```bash
docker-compose logs -f
# 或
make docker-logs
```

### 进入容器

```bash
docker exec -it monkeycode_runner /bin/sh
```

### 重启服务

```bash
docker-compose restart
# 或
docker restart monkeycode_runner
```

### 停止服务

```bash
docker-compose down
# 或
make docker-stop
```

## 故障排查

### 容器无法启动

```bash
# 查看详细日志
docker-compose logs runner

# 检查 Docker 状态
docker info

# 检查端口占用
lsof -i :8080
```

### 无法连接到 TaskFlow

```bash
# 检查环境变量
docker exec monkeycode_runner env | grep GRPC

# 测试网络连接
docker exec monkeycode_runner curl -v telnet://localhost:50051
```

### Docker Socket 权限问题

```bash
# 如果遇到权限问题，确保 Docker socket 权限正确
ls -la /var/run/docker.sock
```

## 项目结构

```
runner/
├── cmd/runner/main.go       # 入口
├── internal/
│   ├── agent/               # AI Agent
│   ├── client/              # gRPC 客户端
│   ├── config/              # 配置
│   ├── docker/              # Docker 管理
│   ├── executor/            # 任务执行器
│   ├── file/                # 文件管理
│   ├── portforward/         # 端口转发
│   ├── task/                # 任务管理
│   ├── terminal/            # 终端管理
│   └── vm/                  # VM 管理
├── Dockerfile               # Docker 镜像
├── docker-compose.yml       # Docker Compose 配置
├── Makefile                 # 构建脚本
└── README.md               # 本文档
```

## 开发

### 本地开发

```bash
# 克隆代码
git clone https://github.com/chaitin/MonkeyCode.git
cd MonkeyCode/runner

# 安装依赖
go mod download

# 运行
make run

# 测试
make test

# 构建
make build
```

### 代码规范

```bash
# 格式化代码
go fmt ./...

# 运行 vet
go vet ./...
```

## 许可证

MIT License
