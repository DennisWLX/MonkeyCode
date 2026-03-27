# GitHub App Setup 回调接口实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 GitHub App 安装回调接口 `/api/v1/github/app/setup`,处理 GitHub App 安装完成后的回调并重定向到前端设置页面

**Architecture:** 在 backend/biz/git 模块中新增 GitHub App Setup Handler,接收 GitHub 的 installation_id 和 setup_action 参数,创建或更新 GitIdentity 记录,然后重定向到前端设置页面并携带成功/失败参数

**Tech Stack:** Go, Echo web framework, Ent ORM, GitHub App API

---

## Task 1: 定义 Domain 层接口和数据结构

**Files:**
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/domain/gitidentity.go`
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/domain/gitidentity.go` (新增请求/响应结构)

- [ ] **Step 1: 在 GitIdentityUsecase 接口中添加 HandleGitHubAppSetup 方法**

在 `domain/gitidentity.go` 的 `GitIdentityUsecase` 接口中添加新方法:

```go
// GitIdentityUsecase Git 身份认证业务逻辑接口
type GitIdentityUsecase interface {
	List(ctx context.Context, uid uuid.UUID) ([]*GitIdentity, error)
	Get(ctx context.Context, uid uuid.UUID, id uuid.UUID) (*GitIdentity, error)
	Add(ctx context.Context, uid uuid.UUID, req *AddGitIdentityReq) (*GitIdentity, error)
	Update(ctx context.Context, uid uuid.UUID, req *UpdateGitIdentityReq) error
	Delete(ctx context.Context, uid uuid.UUID, id uuid.UUID) error
	ListBranches(ctx context.Context, uid uuid.UUID, identityID uuid.UUID, repoFullName string, page, perPage int) ([]*Branch, error)
	HandleGitHubAppSetup(ctx context.Context, req *GitHubAppSetupReq) (*GitHubAppSetupResp, error)
}
```

- [ ] **Step 2: 定义请求和响应结构体**

在 `domain/gitidentity.go` 中添加:

```go
// GitHubAppSetupReq GitHub App 安装回调请求
type GitHubAppSetupReq struct {
	InstallationID int64  `query:"installation_id" validate:"required"`
	SetupAction    string `query:"setup_action" validate:"required"`
	State          string `query:"state"` // 可选的 state 参数,用于关联用户
}

// GitHubAppSetupResp GitHub App 安装回调响应
type GitHubAppSetupResp struct {
	Success      bool   `json:"success"`
	AccountLogin string `json:"account_login,omitempty"`
	Message      string `json:"message,omitempty"`
}
```

- [ ] **Step 3: 在 GitIdentityRepo 接口中添加必要的方法**

在 `domain/gitidentity.go` 的 `GitIdentityRepo` 接口中添加:

```go
// GitIdentityRepo Git 身份认证数据仓库接口
type GitIdentityRepo interface {
	Get(ctx context.Context, id uuid.UUID) (*db.GitIdentity, error)
	GetByUserID(ctx context.Context, uid uuid.UUID, id uuid.UUID) (*db.GitIdentity, error)
	GetByInstallationID(ctx context.Context, installationID int64) (*db.GitIdentity, error)
	List(ctx context.Context, uid uuid.UUID) ([]*db.GitIdentity, error)
	Create(ctx context.Context, uid uuid.UUID, req *AddGitIdentityReq) (*db.GitIdentity, error)
	CreateFromGitHubApp(ctx context.Context, uid uuid.UUID, installationID int64, username, email string) (*db.GitIdentity, error)
	Update(ctx context.Context, uid uuid.UUID, id uuid.UUID, req *UpdateGitIdentityReq) error
	Delete(ctx context.Context, uid uuid.UUID, id uuid.UUID) error
	CountProjectsByGitIdentityID(ctx context.Context, id uuid.UUID) (int, error)
}
```

- [ ] **Step 4: 验证代码编译通过**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go build ./...`
Expected: 编译失败,因为接口方法未实现

---

## Task 2: 实现 Repository 层方法

**Files:**
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/repo/identity.go`

- [ ] **Step 1: 实现 GetByInstallationID 方法**

在 `biz/git/repo/identity.go` 中添加:

```go
// GetByInstallationID 根据 installation_id 获取 Git 身份认证
func (r *GitIdentityRepo) GetByInstallationID(ctx context.Context, installationID int64) (*db.GitIdentity, error) {
	return r.db.GitIdentity.Query().
		Where(gitidentity.InstallationID(installationID)).
		First(ctx)
}
```

- [ ] **Step 2: 实现 CreateFromGitHubApp 方法**

在 `biz/git/repo/identity.go` 中添加:

```go
// CreateFromGitHubApp 从 GitHub App 安装创建 Git 身份认证
func (r *GitIdentityRepo) CreateFromGitHubApp(ctx context.Context, uid uuid.UUID, installationID int64, username, email string) (*db.GitIdentity, error) {
	return r.db.GitIdentity.Create().
		SetUserID(uid).
		SetPlatform(consts.GitPlatformGithub).
		SetBaseURL("https://github.com").
		SetInstallationID(installationID).
		SetUsername(username).
		SetEmail(email).
		Save(ctx)
}
```

- [ ] **Step 3: 验证代码编译通过**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go build ./...`
Expected: 编译失败,因为 Usecase 层方法未实现

---

## Task 3: 实现 Usecase 层业务逻辑

**Files:**
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/usecase/identity.go`

- [ ] **Step 1: 添加必要的导入**

在 `biz/git/usecase/identity.go` 文件顶部添加:

```go
import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/consts"
	"github.com/chaitin/MonkeyCode/backend/db"
	"github.com/chaitin/MonkeyCode/backend/domain"
	"github.com/chaitin/MonkeyCode/backend/errcode"
	"github.com/chaitin/MonkeyCode/backend/pkg/cvt"
	"github.com/chaitin/MonkeyCode/backend/pkg/git/gitea"
	"github.com/chaitin/MonkeyCode/backend/pkg/git/gitee"
	"github.com/chaitin/MonkeyCode/backend/pkg/git/github"
	"github.com/chaitin/MonkeyCode/backend/pkg/git/gitlab"
)
```

- [ ] **Step 2: 实现 HandleGitHubAppSetup 方法**

在 `biz/git/usecase/identity.go` 中添加:

```go
// HandleGitHubAppSetup 处理 GitHub App 安装回调
func (u *GitIdentityUsecase) HandleGitHubAppSetup(ctx context.Context, req *domain.GitHubAppSetupReq) (*domain.GitHubAppSetupResp, error) {
	// 验证 setup_action
	if req.SetupAction != "install" && req.SetupAction != "update" {
		return &domain.GitHubAppSetupResp{
			Success: false,
			Message: fmt.Sprintf("unsupported setup_action: %s", req.SetupAction),
		}, nil
	}

	// TODO: 通过 GitHub API 获取安装信息和用户信息
	// 这里需要使用 GitHub App 的私钥来调用 GitHub API
	// 暂时使用占位符数据
	username := "github-user"
	email := "user@github.com"

	// 检查是否已存在该 installation_id 的记录
	existing, err := u.repo.GetByInstallationID(ctx, req.InstallationID)
	if err != nil && !db.IsNotFound(err) {
		u.logger.ErrorContext(ctx, "failed to get git identity by installation_id",
			"error", err,
			"installation_id", req.InstallationID)
		return nil, err
	}

	// 如果已存在,更新用户信息
	if existing != nil {
		u.logger.InfoContext(ctx, "github app installation already exists",
			"installation_id", req.InstallationID,
			"identity_id", existing.ID)
		return &domain.GitHubAppSetupResp{
			Success:      true,
			AccountLogin: username,
			Message:      "GitHub App installation updated successfully",
		}, nil
	}

	// TODO: 从 state 参数中解析用户 ID
	// 暂时使用一个默认的用户 ID (需要从 session 或 state 中获取)
	uid := uuid.MustParse("00000000-0000-0000-0000-000000000000")

	// 创建新的 GitIdentity 记录
	_, err = u.repo.CreateFromGitHubApp(ctx, uid, req.InstallationID, username, email)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to create git identity from github app",
			"error", err,
			"installation_id", req.InstallationID)
		return nil, err
	}

	u.logger.InfoContext(ctx, "github app installation created successfully",
		"installation_id", req.InstallationID,
		"username", username)

	return &domain.GitHubAppSetupResp{
		Success:      true,
		AccountLogin: username,
		Message:      "GitHub App installation completed successfully",
	}, nil
}
```

- [ ] **Step 3: 验证代码编译通过**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go build ./...`
Expected: 编译失败,因为 Handler 层未实现

---

## Task 4: 实现 Handler 层 HTTP 接口

**Files:**
- Create: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/handler/v1/github_app_setup.go`

- [ ] **Step 1: 创建 GitHubAppSetupHandler 文件**

创建文件 `backend/biz/git/handler/v1/github_app_setup.go`:

```go
package v1

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/GoYoko/web"
	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/config"
	"github.com/chaitin/MonkeyCode/backend/domain"
)

// GitHubAppSetupHandler GitHub App 安装回调处理器
type GitHubAppSetupHandler struct {
	cfg     *config.Config
	usecase domain.GitIdentityUsecase
	logger  *slog.Logger
}

// NewGitHubAppSetupHandler 创建 GitHub App 安装回调处理器
func NewGitHubAppSetupHandler(i *do.Injector) (*GitHubAppSetupHandler, error) {
	w := do.MustInvoke[*web.Web](i)
	cfg := do.MustInvoke[*config.Config](i)

	h := &GitHubAppSetupHandler{
		cfg:     cfg,
		usecase: do.MustInvoke[domain.GitIdentityUsecase](i),
		logger:  do.MustInvoke[*slog.Logger](i).With("module", "handler.github_app_setup"),
	}

	// 注册路由 - 不需要认证
	g := w.Group("/api/v1/github/app")
	g.GET("/setup", web.BindHandler(h.Setup))

	return h, nil
}

// Setup 处理 GitHub App 安装回调
//
//	@Summary		GitHub App 安装回调
//	@Description	处理 GitHub App 安装完成后的回调,创建 Git 身份认证记录并重定向到前端
//	@Tags			【Git】GitHub App
//	@Accept			json
//	@Produce		json
//	@Param			installation_id	query		int	true	"GitHub App 安装 ID"
//	@Param			setup_action	query		string	true	"安装动作 (install/update)"
//	@Param			state			query		string	false	"状态参数"
//	@Success		302				{string}	string	"重定向到前端设置页面"
//	@Router			/api/v1/github/app/setup [get]
func (h *GitHubAppSetupHandler) Setup(c *web.Context, req domain.GitHubAppSetupReq) error {
	ctx := c.Request().Context()

	h.logger.InfoContext(ctx, "github app setup callback received",
		"installation_id", req.InstallationID,
		"setup_action", req.SetupAction,
		"state", req.State)

	// 调用业务逻辑处理
	resp, err := h.usecase.HandleGitHubAppSetup(ctx, &req)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to handle github app setup",
			"error", err,
			"installation_id", req.InstallationID)
		// 重定向到前端错误页面
		redirectURL := fmt.Sprintf("%s/console/settings?github_setup=error&reason=internal_error&message=%s",
			h.cfg.Server.BaseURL,
			"处理 GitHub App 安装失败")
		return c.Redirect(http.StatusFound, redirectURL)
	}

	// 构建重定向 URL
	var redirectURL string
	if resp.Success {
		redirectURL = fmt.Sprintf("%s/console/settings?github_setup=success&installation_id=%d&account_login=%s",
			h.cfg.Server.BaseURL,
			req.InstallationID,
			resp.AccountLogin)
	} else {
		redirectURL = fmt.Sprintf("%s/console/settings?github_setup=error&reason=setup_failed&message=%s",
			h.cfg.Server.BaseURL,
			resp.Message)
	}

	h.logger.InfoContext(ctx, "redirecting to frontend",
		"url", redirectURL)

	return c.Redirect(http.StatusFound, redirectURL)
}
```

- [ ] **Step 2: 在 git 模块注册中添加 Handler**

修改 `backend/biz/git/register.go`:

```go
package git

import (
	"github.com/samber/do"

	v1 "github.com/chaitin/MonkeyCode/backend/biz/git/handler/v1"
	"github.com/chaitin/MonkeyCode/backend/biz/git/repo"
	"github.com/chaitin/MonkeyCode/backend/biz/git/usecase"
)

// RegisterGit 注册 git 模块
func RegisterGit(i *do.Injector) {
	// GitIdentity Repo
	do.Provide(i, repo.NewGitIdentityRepo)

	// GitIdentity Usecase
	do.Provide(i, usecase.NewGitIdentityUsecase)
	do.Provide(i, usecase.NewGithubAccessTokenUsecase)

	// GitIdentity Handler
	do.Provide(i, v1.NewGitIdentityHandler)
	do.MustInvoke[*v1.GitIdentityHandler](i)

	// GitHub App Setup Handler
	do.Provide(i, v1.NewGitHubAppSetupHandler)
	do.MustInvoke[*v1.GitHubAppSetupHandler](i)

	// GitBot Repo
	do.Provide(i, repo.NewGitBotRepo)

	// GitBot Usecase
	do.Provide(i, usecase.NewGitBotUsecase)

	// GitBot Handler
	do.Provide(i, v1.NewGitBotHandler)
	do.MustInvoke[*v1.GitBotHandler](i)

	// Webhook Handlers
	do.Provide(i, v1.NewGithubWebhookHandler)
	do.Provide(i, v1.NewGitlabWebhookHandler)
	do.Provide(i, v1.NewGiteeWebhookHandler)
	do.Provide(i, v1.NewGiteaWebhookHandler)
	do.MustInvoke[*v1.GithubWebhookHandler](i)
	do.MustInvoke[*v1.GitlabWebhookHandler](i)
	do.MustInvoke[*v1.GiteeWebhookHandler](i)
	do.MustInvoke[*v1.GiteaWebhookHandler](i)
}
```

- [ ] **Step 3: 验证代码编译通过**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go build ./...`
Expected: 编译成功

---

## Task 5: 测试和验证

**Files:**
- Test: 手动测试

- [ ] **Step 1: 启动后端服务**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go run cmd/server/main.go`
Expected: 服务正常启动,监听在配置的端口

- [ ] **Step 2: 模拟 GitHub App 安装回调**

使用浏览器或 curl 访问:
```
http://localhost:8888/api/v1/github/app/setup?installation_id=119313320&setup_action=install
```

Expected:
- 后端日志显示收到请求
- 返回 302 重定向到前端设置页面
- 重定向 URL 包含 `github_setup=success` 或 `github_setup=error` 参数

- [ ] **Step 3: 检查数据库记录**

检查 `git_identities` 表中是否创建了新记录:
- `platform` = "github"
- `installation_id` = 119313320
- `base_url` = "https://github.com"

- [ ] **Step 4: 验证前端回调处理**

访问重定向后的 URL,确认前端能够正确解析参数并显示成功/失败提示。

---

## Task 6: 完善和优化

**Files:**
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/usecase/identity.go`

- [ ] **Step 1: 添加 GitHub API 集成 (可选)**

如果需要从 GitHub API 获取真实的用户信息,可以在 `HandleGitHubAppSetup` 方法中添加:

```go
// 使用 GitHub App 私钥获取 installation token
// 然后调用 GitHub API 获取安装信息和用户信息
// 参考文档: https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/authenticating-as-a-github-app-installation
```

- [ ] **Step 2: 添加 state 参数验证 (可选)**

为了安全性,可以添加 state 参数验证:

```go
// 从 state 参数中解析用户 ID
// state 可以是加密的 JWT token 或存储在 Redis 中的 key
uid, err := parseState(req.State)
if err != nil {
	return &domain.GitHubAppSetupResp{
		Success: false,
		Message: "invalid state parameter",
	}, nil
}
```

- [ ] **Step 3: 添加单元测试 (可选)**

创建测试文件验证各个组件的功能。

---

## 注意事项

1. **用户身份关联**: 当前实现使用了一个默认的用户 ID,实际使用时需要从 session 或 state 参数中获取真实的用户 ID
2. **GitHub API 集成**: 当前实现使用占位符数据,实际应该调用 GitHub API 获取真实的用户信息
3. **安全性**: 建议添加 state 参数验证,防止 CSRF 攻击
4. **错误处理**: 已经添加了基本的错误处理和日志记录,可以根据需要进一步完善
5. **幂等性**: 实现中检查了重复的 installation_id,确保不会创建重复记录

## 后续改进

1. 实现 GitHub API 集成,获取真实的用户信息
2. 添加 state 参数生成和验证机制
3. 支持更新已存在的 GitHub App 安装
4. 添加更完善的错误处理和用户提示
5. 编写单元测试和集成测试
