# GitHub App 功能完善实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完善 GitHub App 集成功能,实现 State 参数验证、更新已存在记录、改进错误提示和编写单元测试

**Architecture:** 在现有 GitHub App 回调处理基础上,添加 State 参数生成和验证机制,实现记录更新逻辑,优化错误处理,并为关键组件编写单元测试

**Tech Stack:** Go, Echo web framework, Ent ORM, JWT, Redis (可选)

---

## Task 1: 实现 State 参数验证和用户身份关联

**Files:**
- Create: `/Users/wanglx/dennis/project/MonkeyCode/backend/pkg/crypto/state.go`
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/domain/gitidentity.go`
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/usecase/identity.go`
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/handler/v1/github_app_setup.go`

- [ ] **Step 1: 创建 State 参数生成和验证工具**

创建文件 `backend/pkg/crypto/state.go`:

```go
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StateData State 参数数据
type StateData struct {
	UserID    uuid.UUID `json:"user_id"`
	Timestamp int64     `json:"timestamp"`
	Nonce     string    `json:"nonce"`
}

// GenerateState 生成 State 参数
func GenerateState(userID uuid.UUID, secret string) (string, error) {
	data := StateData{
		UserID:    userID,
		Timestamp: time.Now().Unix(),
		Nonce:     uuid.New().String()[:8],
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("marshal state data: %w", err)
	}

	// 添加签名
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(jsonData)
	signature := h.Sum(nil)

	// 组合: base64(json) + "." + base64(signature)
	encodedData := base64.URLEncoding.EncodeToString(jsonData)
	encodedSig := base64.URLEncoding.EncodeToString(signature)

	return encodedData + "." + encodedSig, nil
}

// VerifyState 验证 State 参数
func VerifyState(state, secret string, maxAge int64) (*StateData, error) {
	// 分离数据和签名
	parts := splitState(state)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid state format")
	}

	encodedData, encodedSig := parts[0], parts[1]

	// 解码数据
	jsonData, err := base64.URLEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("decode state data: %w", err)
	}

	// 验证签名
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(jsonData)
	expectedSig := h.Sum(nil)

	actualSig, err := base64.URLEncoding.DecodeString(encodedSig)
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	if !hmac.Equal(actualSig, expectedSig) {
		return nil, fmt.Errorf("invalid state signature")
	}

	// 解析数据
	var data StateData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("unmarshal state data: %w", err)
	}

	// 验证时间戳
	if maxAge > 0 {
		age := time.Now().Unix() - data.Timestamp
		if age > maxAge {
			return nil, fmt.Errorf("state expired (age: %d seconds)", age)
		}
	}

	return &data, nil
}

func splitState(state string) []string {
	for i := len(state) - 1; i >= 0; i-- {
		if state[i] == '.' {
			return []string{state[:i], state[i+1:]}
		}
	}
	return nil
}
```

- [ ] **Step 2: 在 Domain 层添加 State 相关方法**

修改 `backend/domain/gitidentity.go`,在 `GitIdentityUsecase` 接口中添加:

```go
// GitIdentityUsecase Git 身份认证业务逻辑接口
type GitIdentityUsecase interface {
	// ... 现有方法 ...
	GenerateGitHubAppState(ctx context.Context, uid uuid.UUID) (string, error)
}
```

- [ ] **Step 3: 在 Usecase 层实现 State 生成方法**

修改 `backend/biz/git/usecase/identity.go`,添加方法:

```go
// GenerateGitHubAppState 生成 GitHub App 安装的 State 参数
func (u *GitIdentityUsecase) GenerateGitHubAppState(ctx context.Context, uid uuid.UUID) (string, error) {
	// 使用配置中的 secret 或生成一个
	secret := u.cfg.AdminToken
	if secret == "" {
		secret = "github-app-state-secret"
	}

	state, err := crypto.GenerateState(uid, secret)
	if err != nil {
		u.logger.ErrorContext(ctx, "failed to generate state",
			"error", err,
			"user_id", uid)
		return "", err
	}

	return state, nil
}
```

- [ ] **Step 4: 在 Handler 层验证 State 参数**

修改 `backend/biz/git/handler/v1/github_app_setup.go` 的 `Setup` 方法:

```go
func (h *GitHubAppSetupHandler) Setup(c *web.Context, req domain.GitHubAppSetupReq) error {
	ctx := c.Request().Context()

	h.logger.InfoContext(ctx, "github app setup callback received",
		"installation_id", req.InstallationID,
		"setup_action", req.SetupAction,
		"state", req.State)

	// 验证 state 参数
	var uid uuid.UUID
	if req.State != "" {
		secret := h.cfg.AdminToken
		if secret == "" {
			secret = "github-app-state-secret"
		}

		stateData, err := crypto.VerifyState(req.State, secret, 3600) // 1小时有效期
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to verify state",
				"error", err,
				"state", req.State)
			redirectURL := fmt.Sprintf("%s/console/settings?github_setup=error&reason=invalid_state&message=%s",
				h.cfg.Server.BaseURL,
				"无效的状态参数")
			return c.Redirect(http.StatusFound, redirectURL)
		}
		uid = stateData.UserID
	} else {
		// 如果没有 state 参数,使用默认用户 ID (向后兼容)
		uid = uuid.MustParse("00000000-0000-0000-0000-000000000000")
	}

	// 调用业务逻辑处理
	resp, err := h.usecase.HandleGitHubAppSetup(ctx, &req, uid)
	// ... 其余代码 ...
}
```

- [ ] **Step 5: 更新 Usecase 方法签名**

修改 `backend/biz/git/usecase/identity.go` 的 `HandleGitHubAppSetup` 方法:

```go
// HandleGitHubAppSetup 处理 GitHub App 安装回调
func (u *GitIdentityUsecase) HandleGitHubAppSetup(ctx context.Context, req *domain.GitHubAppSetupReq, uid uuid.UUID) (*domain.GitHubAppSetupResp, error) {
	// ... 现有代码 ...
	// 移除: uid := uuid.MustParse("00000000-0000-0000-0000-000000000000")
	// 使用传入的 uid 参数
	// ... 其余代码 ...
}
```

- [ ] **Step 6: 更新 Domain 接口**

修改 `backend/domain/gitidentity.go`:

```go
HandleGitHubAppSetup(ctx context.Context, req *GitHubAppSetupReq, uid uuid.UUID) (*GitHubAppSetupResp, error)
```

- [ ] **Step 7: 验证代码编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go build ./...`
Expected: 编译成功

---

## Task 2: 实现更新已存在的记录功能

**Files:**
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/repo/identity.go`
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/domain/gitidentity.go`
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/usecase/identity.go`

- [ ] **Step 1: 在 Domain 层添加更新方法**

修改 `backend/domain/gitidentity.go`,在 `GitIdentityRepo` 接口中添加:

```go
UpdateFromGitHubApp(ctx context.Context, id uuid.UUID, username, email string) error
```

- [ ] **Step 2: 在 Repository 层实现更新方法**

修改 `backend/biz/git/repo/identity.go`,添加方法:

```go
// UpdateFromGitHubApp 更新 GitHub App 安装的用户信息
func (r *GitIdentityRepo) UpdateFromGitHubApp(ctx context.Context, id uuid.UUID, username, email string) error {
	return r.db.GitIdentity.UpdateOneID(id).
		SetUsername(username).
		SetEmail(email).
		Exec(ctx)
}
```

- [ ] **Step 3: 在 Usecase 层实现更新逻辑**

修改 `backend/biz/git/usecase/identity.go` 的 `HandleGitHubAppSetup` 方法:

```go
// 如果已存在,更新用户信息
if existing != nil {
	// 更新用户信息
	if err := u.repo.UpdateFromGitHubApp(ctx, existing.ID, username, email); err != nil {
		u.logger.ErrorContext(ctx, "failed to update git identity from github app",
			"error", err,
			"installation_id", req.InstallationID,
			"identity_id", existing.ID)
		return nil, err
	}

	u.logger.InfoContext(ctx, "github app installation updated successfully",
		"installation_id", req.InstallationID,
		"identity_id", existing.ID,
		"username", username)

	return &domain.GitHubAppSetupResp{
		Success:      true,
		AccountLogin: username,
		Message:      "GitHub App installation updated successfully",
	}, nil
}
```

- [ ] **Step 4: 验证代码编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go build ./...`
Expected: 编译成功

---

## Task 3: 改进错误提示信息

**Files:**
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/handler/v1/github_app_setup.go`
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/errcode/errcode.go`
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/errcode/locale.zh.toml`
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/errcode/locale.en.toml`

- [ ] **Step 1: 添加错误码定义**

修改 `backend/errcode/errcode.go`,添加错误码:

```go
var (
	// ... 现有错误码 ...
	ErrGitHubAppInvalidState    = New(20001, "github_app.invalid_state")
	ErrGitHubAppGetUserInfo     = New(20002, "github_app.get_user_info_failed")
	ErrGitHubAppCreateIdentity  = New(20003, "github_app.create_identity_failed")
	ErrGitHubAppUpdateIdentity  = New(20004, "github_app.update_identity_failed")
)
```

- [ ] **Step 2: 添加中文错误信息**

修改 `backend/errcode/locale.zh.toml`,添加:

```toml
"github_app.invalid_state" = "无效的状态参数"
"github_app.get_user_info_failed" = "获取 GitHub 用户信息失败"
"github_app.create_identity_failed" = "创建 Git 身份认证失败"
"github_app.update_identity_failed" = "更新 Git 身份认证失败"
```

- [ ] **Step 3: 添加英文错误信息**

修改 `backend/errcode/locale.en.toml`,添加:

```toml
"github_app.invalid_state" = "Invalid state parameter"
"github_app.get_user_info_failed" = "Failed to get GitHub user information"
"github_app.create_identity_failed" = "Failed to create Git identity"
"github_app.update_identity_failed" = "Failed to update Git identity"
```

- [ ] **Step 4: 在 Handler 层使用错误码**

修改 `backend/biz/git/handler/v1/github_app_setup.go`:

```go
func (h *GitHubAppSetupHandler) Setup(c *web.Context, req domain.GitHubAppSetupReq) error {
	ctx := c.Request().Context()

	// ... 现有代码 ...

	// 验证 state 参数
	if req.State != "" {
		secret := h.cfg.AdminToken
		if secret == "" {
			secret = "github-app-state-secret"
		}

		stateData, err := crypto.VerifyState(req.State, secret, 3600)
		if err != nil {
			h.logger.ErrorContext(ctx, "failed to verify state",
				"error", err,
				"state", req.State)
			redirectURL := fmt.Sprintf("%s/console/settings?github_setup=error&code=%d&message=%s",
				h.cfg.Server.BaseURL,
				errcode.ErrGitHubAppInvalidState.Code(),
				errcode.ErrGitHubAppInvalidState.Message())
			return c.Redirect(http.StatusFound, redirectURL)
		}
		uid = stateData.UserID
	}

	// ... 其余代码 ...
}
```

- [ ] **Step 5: 在 Usecase 层使用错误码**

修改 `backend/biz/git/usecase/identity.go`:

```go
// 获取用户信息
var username, email string
if u.ghApp != nil {
	var err error
	username, email, err = u.ghApp.GetUserInfo(ctx, req.InstallationID)
	if err != nil {
		u.logger.WarnContext(ctx, "failed to get user info from github app",
			"error", err,
			"installation_id", req.InstallationID)
		return nil, errcode.ErrGitHubAppGetUserInfo.Wrap(err)
	}
}

// ... 创建记录 ...
_, err = u.repo.CreateFromGitHubApp(ctx, uid, req.InstallationID, username, email)
if err != nil {
	u.logger.ErrorContext(ctx, "failed to create git identity from github app",
		"error", err,
		"installation_id", req.InstallationID)
	return nil, errcode.ErrGitHubAppCreateIdentity.Wrap(err)
}

// ... 更新记录 ...
if err := u.repo.UpdateFromGitHubApp(ctx, existing.ID, username, email); err != nil {
	u.logger.ErrorContext(ctx, "failed to update git identity from github app",
		"error", err,
		"installation_id", req.InstallationID,
		"identity_id", existing.ID)
	return nil, errcode.ErrGitHubAppUpdateIdentity.Wrap(err)
}
```

- [ ] **Step 6: 验证代码编译**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go build ./...`
Expected: 编译成功

---

## Task 4: 编写单元测试

**Files:**
- Create: `/Users/wanglx/dennis/project/MonkeyCode/backend/pkg/crypto/state_test.go`
- Create: `/Users/wanglx/dennis/project/MonkeyCode/backend/pkg/git/github/github_app_test.go`
- Create: `/Users/wanglx/dennis/project/MonkeyCode/backend/biz/git/usecase/identity_test.go`

- [ ] **Step 1: 编写 State 参数测试**

创建文件 `backend/pkg/crypto/state_test.go`:

```go
package crypto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndVerifyState(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	// 生成 state
	state, err := GenerateState(userID, secret)
	assert.NoError(t, err)
	assert.NotEmpty(t, state)

	// 验证 state
	data, err := VerifyState(state, secret, 3600)
	assert.NoError(t, err)
	assert.Equal(t, userID, data.UserID)
	assert.NotZero(t, data.Timestamp)
	assert.NotEmpty(t, data.Nonce)
}

func TestVerifyStateWithWrongSecret(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"
	wrongSecret := "wrong-secret"

	state, err := GenerateState(userID, secret)
	assert.NoError(t, err)

	// 使用错误的 secret 验证
	_, err = VerifyState(state, wrongSecret, 3600)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state signature")
}

func TestVerifyStateWithExpiredState(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret"

	// 生成一个过期的 state (模拟 2 小时前)
	data := StateData{
		UserID:    userID,
		Timestamp: time.Now().Unix() - 7200, // 2 小时前
		Nonce:     uuid.New().String()[:8],
	}

	// 手动构建 state (绕过时间检查)
	jsonData, _ := json.Marshal(data)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(jsonData)
	signature := h.Sum(nil)

	encodedData := base64.URLEncoding.EncodeToString(jsonData)
	encodedSig := base64.URLEncoding.EncodeToString(signature)
	state := encodedData + "." + encodedSig

	// 验证过期 state (1 小时有效期)
	_, err := VerifyState(state, secret, 3600)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state expired")
}

func TestVerifyStateWithInvalidFormat(t *testing.T) {
	secret := "test-secret"

	// 无效格式
	_, err := VerifyState("invalid-state", secret, 3600)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state format")
}
```

- [ ] **Step 2: 安装测试依赖**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go get github.com/stretchr/testify`
Expected: 安装成功

- [ ] **Step 3: 运行 State 参数测试**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go test ./pkg/crypto -v`
Expected: 所有测试通过

- [ ] **Step 4: 编写 GitHub App 客户端测试**

创建文件 `backend/pkg/git/github/github_app_test.go`:

```go
package github

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGitHubApp(t *testing.T) {
	appID := int64(123456)
	privateKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy0AHB7MbzYLdZ7ZvVy7F7V
...
-----END RSA PRIVATE KEY-----`

	app, err := NewGitHubApp(appID, privateKey, nil)
	assert.NoError(t, err)
	assert.NotNil(t, app)
	assert.Equal(t, appID, app.appID)
}

func TestNewGitHubAppWithInvalidKey(t *testing.T) {
	appID := int64(123456)
	invalidKey := "invalid-key"

	app, err := NewGitHubApp(appID, invalidKey, nil)
	// 注意: NewGitHubApp 现在不会立即验证密钥格式
	assert.NoError(t, err)
	assert.NotNil(t, app)
}
```

- [ ] **Step 5: 运行 GitHub App 测试**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go test ./pkg/git/github -v`
Expected: 测试通过

- [ ] **Step 6: 编写 Usecase 层测试**

创建文件 `backend/biz/git/usecase/identity_test.go`:

```go
package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/chaitin/MonkeyCode/backend/domain"
)

// MockGitIdentityRepo 模拟 GitIdentityRepo
type MockGitIdentityRepo struct {
	mock.Mock
}

func (m *MockGitIdentityRepo) GetByInstallationID(ctx context.Context, installationID int64) (*db.GitIdentity, error) {
	args := m.Called(ctx, installationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.GitIdentity), args.Error(1)
}

func (m *MockGitIdentityRepo) CreateFromGitHubApp(ctx context.Context, uid uuid.UUID, installationID int64, username, email string) (*db.GitIdentity, error) {
	args := m.Called(ctx, uid, installationID, username, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*db.GitIdentity), args.Error(1)
}

// ... 其他方法 ...

func TestHandleGitHubAppSetup(t *testing.T) {
	mockRepo := new(MockGitIdentityRepo)
	usecase := &GitIdentityUsecase{
		repo:   mockRepo,
		logger: slog.Default(),
	}

	ctx := context.Background()
	req := &domain.GitHubAppSetupReq{
		InstallationID: 123456,
		SetupAction:    "install",
	}
	uid := uuid.New()

	// 模拟不存在记录
	mockRepo.On("GetByInstallationID", ctx, int64(123456)).Return(nil, db.NotFoundError())

	// 模拟创建成功
	mockRepo.On("CreateFromGitHubApp", ctx, uid, int64(123456), "github-user", "user@github.com").
		Return(&db.GitIdentity{ID: uuid.New()}, nil)

	// 调用方法
	resp, err := usecase.HandleGitHubAppSetup(ctx, req, uid)

	// 验证结果
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "github-user", resp.AccountLogin)

	// 验证 mock 调用
	mockRepo.AssertExpectations(t)
}
```

- [ ] **Step 7: 运行 Usecase 测试**

Run: `cd /Users/wanglx/dennis/project/MonkeyCode/backend && go test ./biz/git/usecase -v`
Expected: 测试通过

---

## Task 5: 更新文档

**Files:**
- Modify: `/Users/wanglx/dennis/project/MonkeyCode/backend/docs/github-app-integration.md`

- [ ] **Step 1: 更新集成指南**

在 `backend/docs/github-app-integration.md` 中添加:

```markdown
### State 参数验证

为了安全性,系统支持 State 参数验证来关联用户身份:

#### 生成 State 参数

在前端调用绑定接口时,先生成 State 参数:

```typescript
// 前端代码示例
const response = await fetch('/api/v1/git/identity/github/state', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
});
const { state } = await response.json();

// 构建安装 URL
const installUrl = `${getGithubAppInstallUrl()}?state=${state}`;
window.open(installUrl, '_blank');
```

#### State 参数格式

State 参数格式为: `base64(json_data).base64(signature)`

其中 `json_data` 包含:
- `user_id`: 用户 ID
- `timestamp`: 生成时间戳
- `nonce`: 随机字符串

#### 验证流程

1. 用户点击"绑定 GitHub"
2. 前端请求生成 State 参数
3. 前端携带 State 参数跳转到 GitHub App 安装页面
4. GitHub 回调时携带 State 参数
5. 后端验证 State 参数的有效性和签名
6. 从 State 中提取用户 ID,创建 GitIdentity 记录

### 错误处理

系统提供详细的错误码和错误信息:

| 错误码 | 说明 |
|--------|------|
| 20001 | 无效的状态参数 |
| 20002 | 获取 GitHub 用户信息失败 |
| 20003 | 创建 Git 身份认证失败 |
| 20004 | 更新 Git 身份认证失败 |

前端可以根据错误码显示对应的错误提示。
```

- [ ] **Step 2: 验证文档完整性**

检查文档是否包含所有必要信息,包括:
- State 参数验证流程
- 错误处理说明
- 更新已存在记录的说明
- 测试覆盖说明

---

## 注意事项

1. **安全性**: State 参数使用 HMAC-SHA256 签名,防止篡改
2. **有效期**: State 参数默认有效期为 1 小时
3. **向后兼容**: 如果没有 State 参数,使用默认用户 ID
4. **测试覆盖**: 为关键组件编写单元测试,确保功能正确性
5. **错误处理**: 使用统一的错误码和错误信息

## 后续改进

1. 使用 Redis 存储 State 参数,支持更长的有效期和撤销功能
2. 添加集成测试,测试完整的 GitHub App 安装流程
3. 添加监控和告警,跟踪 GitHub App 安装成功率和失败原因
4. 支持多语言错误信息,根据用户语言设置返回对应的错误提示
