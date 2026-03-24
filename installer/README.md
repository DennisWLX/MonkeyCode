# MonkeyCode Runner Installer

MonkeyCode Runner 安装程序，用于在宿主机上安装和配置 MonkeyCode Runner 服务。

## 功能特性

- 🚀 **一键安装** - 自动检测系统环境，一键完成安装
- 🐳 **Docker 集成** - 自动安装 Docker（如未安装）
- 🔄 **自动更新** - 集成 Watchtower，自动更新容器镜像
- 🔒 **安全配置** - 自动生成认证令牌和安全配置
- 📊 **健康检查** - 内置容器健康检查，确保服务可用
- 🧹 **轻松卸载** - 使用 docker-compose 一键卸载

## 系统要求

- **操作系统**: Linux (Ubuntu, Debian, CentOS, RHEL) / macOS
- **架构**: x86_64 (amd64) / ARM64 (aarch64)
- **内存**: 最少 2GB RAM
- **磁盘**: 最少 10GB 可用空间
- **权限**: root 或 sudo 权限

## 快速开始

### 方法 1: 使用官方安装脚本（推荐）

```bash
# 获取安装命令
curl -fsSL https://monkeycode.ai/api/v1/users/hosts/install-command

# 或直接运行（需要从后台获取 token）
sudo bash -c "$(curl -fsSL 'https://monkeycode.ai/api/v1/users/hosts/install')"
```

### 方法 2: 手动安装

```bash
# 1. 下载 installer
curl -fsSL https://github.com/chaitin/MonkeyCode/releases/latest/download/installer_linux_amd64 -o installer

# 2. 添加执行权限
chmod +x installer

# 3. 运行安装程序
sudo ./installer --token YOUR_TOKEN --grpc-url localhost:50051

# 4. 或使用交互模式
sudo ./installer
```

## 安装选项

### 命令行参数

```bash
sudo ./installer [选项]

选项:
  --dir string       安装目录 (默认: /data/monkeycode_runner)
  --token string     认证令牌
  --grpc-host string    gRPC 主机地址
  --grpc-port string    gRPC 端口
  --grpc-url string     gRPC 完整地址
```

### 环境变量

在 `.env` 文件中配置：

```bash
# 认证配置
TOKEN=your_auth_token

# TaskFlow 连接配置
GRPC_HOST=localhost
GRPC_PORT=50051
GRPC_URL=localhost:50051

# 日志级别
LOG_LEVEL=info

# 资源限制
MAX_CORES=8
MAX_MEMORY=16384
```

## 架构说明

```
┌─────────────────────────────────────────────────────────────┐
│                      安装后架构                               │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                  Docker 容器                         │  │
│  │                                                       │  │
│  │  ┌────────────────┐    ┌─────────────────────────┐  │  │
│  │  │   Orchestrator │◄───┤   Runner HTTP API      │  │  │
│  │  │  (AI 编排器)    │    │       :8080            │  │  │
│  │  └────────────────┘    └─────────────────────────┘  │  │
│  │                                                       │  │
│  │  ┌────────────────┐                                  │  │
│  │  │   Watchtower    │    自动更新镜像                 │  │
│  │  └────────────────┘                                  │  │
│  │                                                       │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                 TaskFlow 服务器                       │  │
│  │                   (外部服务)                           │  │
│  │                    :50051 (gRPC)                     │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 目录结构

安装后，`/data/monkeycode_runner` 目录结构如下：

```
/data/monkeycode_runner/
├── docker-compose.yml    # Docker Compose 配置
├── .env                  # 环境变量配置
├── data/                # 数据目录
│   └── ...
└── logs/               # 日志目录
    └── ...
```

## 管理命令

### 查看容器状态

```bash
cd /data/monkeycode_runner
docker-compose ps
```

### 查看日志

```bash
# 查看所有容器日志
docker-compose logs

# 查看特定容器日志
docker-compose logs -f orchestrator
docker-compose logs -f watchtower
```

### 重启服务

```bash
cd /data/monkeycode_runner
docker-compose restart
```

### 更新服务

```bash
cd /data/monkeycode_runner
docker-compose pull
docker-compose up -d
```

### 卸载服务

```bash
cd /data/monkeycode_runner
docker-compose down -v  # -v 同时删除数据卷
rm -rf /data/monkeycode_runner
```

## API 接口

安装后，Runner 在 `:8080` 端口提供 HTTP API：

| 端点 | 方法 | 说明 |
|------|------|------|
| `/health` | GET | 健康检查 |
| `/terminal` | WebSocket | 交互式终端 |
| `/task` | POST | 创建任务 |
| `/internal/port-forward` | GET/POST | 端口转发管理 |

## 故障排查

### 容器无法启动

```bash
# 查看详细日志
docker-compose logs --tail=100 orchestrator

# 检查 Docker 状态
docker info

# 重启 Docker
sudo systemctl restart docker
```

### 无法连接到 TaskFlow

```bash
# 检查 .env 配置
cat /data/monkeycode_runner/.env | grep GRPC

# 测试网络连接
telnet localhost 50051
```

### 端口被占用

```bash
# 查看 8080 端口占用
sudo lsof -i :8080

# 修改 .env 配置使用其他端口
```

## 开发

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/chaitin/MonkeyCode.git
cd MonkeyCode/installer

# 下载依赖
go mod download

# 构建
make build

# 或交叉编译
make cross-compile
```

### 运行测试

```bash
make test
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License - 查看 [LICENSE](LICENSE) 文件了解更多详情。
