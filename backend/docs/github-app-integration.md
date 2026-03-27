# GitHub App 集成指南

本文档说明如何配置和使用 MonkeyCode 的 GitHub App 集成功能。

## 功能概述

GitHub App 集成允许用户通过安装 GitHub App 来授权 MonkeyCode 访问其 GitHub 仓库,无需手动创建 Personal Access Token (PAT)。

## 配置步骤

### 1. 创建 GitHub App

1. 访问 GitHub Settings → Developer settings → GitHub Apps → New GitHub App
2. 填写以下信息:
   - **GitHub App name**: MonkeyCode AI (或你的应用名称)
   - **Homepage URL**: `https://monkeycode-ai.com`
   - **Callback URL**: `https://monkeycode-ai.com/api/v1/github/app/setup`
   - **Setup URL**: `https://monkeycode-ai.com/api/v1/github/app/setup`
   - **Webhook URL**: `https://monkeycode-ai.com/api/v1/github/webhook/{bot_id}`
   
3. 设置权限 (Permissions):
   - **Repository permissions**:
     - Contents: Read and write
     - Issues: Read and write
     - Pull requests: Read and write
     - Webhooks: Read and write
   - **Account permissions**:
     - Email addresses: Read-only

4. 选择 "Where can this GitHub App be installed?":
   - Any account (推荐) 或 Only on this account

5. 点击 "Create GitHub App"

### 2. 获取配置信息

创建完成后,你需要获取以下信息:

- **App ID**: 在 GitHub App 页面顶部可以看到,例如: `123456`
- **Private Key**: 
  1. 在 GitHub App 页面底部找到 "Private keys"
  2. 点击 "Generate a private key"
  3. 下载并保存 `.pem` 文件
- **Webhook secret**: (可选) 用于验证 webhook 请求

### 3. 配置 MonkeyCode 后端

编辑 `config.yaml` 文件:

```yaml
github:
  enabled: true
  token: ""  # 可选,用于 PAT 模式
  oauth:
    client_id: ""  # OAuth Client ID (如果使用 OAuth)
    client_secret: ""  # OAuth Client Secret (如果使用 OAuth)
    redirect_url: ""  # OAuth 回调 URL
  app:
    app_id: 123456  # 你的 GitHub App ID
    private_key: |
      -----BEGIN RSA PRIVATE KEY-----
      MIIEpAIBAAKCAQEA...
      (你的私钥内容)
      -----END RSA PRIVATE KEY-----
    webhook_secret: "your-webhook-secret"  # 可选
```

**注意**: `private_key` 字段需要包含完整的 PEM 格式私钥,包括开始和结束标记。

### 4. 配置前端

在前端代码中,确保 `getGithubAppInstallUrl()` 函数返回正确的 GitHub App 安装 URL:

```typescript
export function getGithubAppInstallUrl(): string {
  if (typeof window !== "undefined" && window.location.origin === "https://monkeycode-ai.com") {
    return "https://github.com/apps/monkeycode-ai/installations/new"
  }
  return "https://github.com/apps/monkeycode-ai-dev/installations/new"
}
```

## 使用流程

### 用户绑定 GitHub 账号

1. 用户在设置页面点击"绑定 GitHub"按钮
2. 跳转到 GitHub App 安装页面
3. 用户选择要授权的仓库或组织
4. 确认安装后,GitHub 回调到 MonkeyCode
5. MonkeyCode 创建 GitIdentity 记录并重定向到前端
6. 前端显示绑定成功

### API 端点

- **安装回调**: `GET /api/v1/github/app/setup`
  - 参数:
    - `installation_id` (必需): GitHub App 安装 ID
    - `setup_action` (必需): 安装动作 (install/update)
    - `state` (可选): 状态参数,用于关联用户
  - 响应: 重定向到前端设置页面

## 高级功能

### State 参数验证

为了安全性,可以在用户点击"绑定 GitHub"时生成一个 state 参数,包含用户 ID 和时间戳,然后在回调时验证:

```go
// 生成 state
state := generateState(userID, timestamp)

// 验证 state
userID, err := verifyState(state)
```

### 动态获取 Access Token

当需要访问用户仓库时,可以使用 GitHub App 的 installation token:

```go
// 获取 installation token
token, err := ghApp.GetInstallationAccessToken(ctx, installationID)

// 使用 token 访问仓库
client := newClientWithToken(ctx, token)
repos, _, err := client.Repositories.List(ctx, "", nil)
```

## 故障排查

### 常见问题

1. **私钥格式错误**
   - 确保私钥包含完整的 PEM 格式
   - 检查 YAML 缩进是否正确
   - 使用 `|` 或 `|-` 保持换行

2. **权限不足**
   - 检查 GitHub App 的权限设置
   - 确认用户已授权所需仓库

3. **回调失败**
   - 检查回调 URL 是否正确
   - 确认后端服务可访问
   - 查看后端日志

### 日志调试

启用 debug 模式查看详细日志:

```yaml
debug: true
logger:
  level: "debug"
```

## 安全建议

1. **私钥保护**
   - 不要将私钥提交到代码仓库
   - 使用环境变量或密钥管理服务
   - 定期轮换私钥

2. **Webhook 验证**
   - 配置 webhook secret
   - 验证 webhook 签名

3. **最小权限原则**
   - 只请求必要的权限
   - 让用户选择要授权的仓库

## 相关文档

- [GitHub Apps 官方文档](https://docs.github.com/en/apps)
- [GitHub App 认证](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app)
- [GitHub App 权限](https://docs.github.com/en/rest/permissions/permissions-for-github-apps)
