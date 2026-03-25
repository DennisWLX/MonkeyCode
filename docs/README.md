# MonkeyCode 文档索引

欢迎使用 MonkeyCode 分布式 AI 代码执行平台！

## 📚 文档目录

### 系统架构

| 文档 | 描述 |
|------|------|
| [系统架构文档](system-architecture.md) | 完整的系统架构、组件说明、通讯流程 |
| [时序图文档](sequence-diagrams.md) | 详细的时序图（Mermaid 格式） |
| [Orchestrator 设计文档](orchestrator-design.md) | Orchestrator 服务的设计文档 |

### 项目文档

| 项目 | 文档目录 |
|------|---------|
| **Backend** | [backend/README.md](../backend/README.md) |
| **TaskFlow** | [taskflow/README.md](../taskflow/README.md) |
| **Runner** | [runner/README.md](../runner/README.md) |
| **Orchestrator** | [orchestrator/README.md](../orchestrator/README.md) |
| **DevRunner** | [DevRunner/README.md](../DevRunner/README.md) |
| **Installer** | [installer/README.md](../installer/README.md) |

---

## 🔍 快速链接

### 核心概念

- **系统架构**: [system-architecture.md](system-architecture.md#一系统概述)
- **通讯流程**: [system-architecture.md](system-architecture.md#四通讯流程详解)
- **Token 认证**: [system-architecture.md](system-architecture.md#五token-认证流程)

### 部署指南

- **Runner Docker 部署**: [runner/README.md#docker-部署](../runner/README.md#docker-部署)
- **Orchestrator 部署**: [orchestrator/README.md](../orchestrator/README.md)
- **Installer 使用**: [installer/README.md](../installer/README.md)

### API 参考

- **Runner API**: [runner/README.md#api-接口](../runner/README.md#api-接口)
- **TaskFlow gRPC**: [taskflow/pkg/proto/taskflow.proto](../taskflow/pkg/proto/taskflow.proto)

---

## 📖 阅读顺序建议

### 新手入门

1. [系统架构文档 - 系统概述](system-architecture.md#一系统概述)
2. [系统架构文档 - 组件说明](system-architecture.md#三组件详细说明)
3. [Runner Docker 部署](../runner/README.md#docker-部署)

### 开发指南

1. [系统架构文档 - 通讯流程](system-architecture.md#四通讯流程详解)
2. [时序图文档](sequence-diagrams.md)
3. [Runner 项目文档](../runner/README.md)
4. [TaskFlow 项目文档](../taskflow/README.md)

### 架构设计

1. [系统架构文档 - 完整版](system-architecture.md)
2. [时序图文档](sequence-diagrams.md)
3. [Orchestrator 设计文档](orchestrator-design.md)

---

## 🛠️ 故障排查

### 常见问题

| 问题 | 解决方案 |
|------|---------|
| Runner 无法注册 | 检查 TOKEN、GRPC_URL 配置 |
| VM 创建失败 | 检查 Docker 是否运行 |
| Terminal 无法连接 | 检查 WebSocket 端点 |
| 无法连接 TaskFlow | 检查网络和防火墙 |

详细排查指南：[system-architecture.md#十一故障排查](system-architecture.md#十一故障排查)

---

## 📝 文档贡献

欢迎提交 Issue 和 Pull Request 来改进文档！

### 文档规范

- 使用 Markdown 格式
- 时序图使用 Mermaid 语法
- 代码块标注语言
- 链接使用相对路径

---

## 📞 获取帮助

- 提交 [GitHub Issue](https://github.com/chaitin/MonkeyCode/issues)
- 查看 [Wiki](https://github.com/chaitin/MonkeyCode/wiki)

---

*最后更新: 2024*
