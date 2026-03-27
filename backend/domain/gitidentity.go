package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/chaitin/MonkeyCode/backend/consts"
	"github.com/chaitin/MonkeyCode/backend/db"
)

// GitIdentityUsecase Git 身份认证业务逻辑接口
type GitIdentityUsecase interface {
	List(ctx context.Context, uid uuid.UUID) ([]*GitIdentity, error)
	Get(ctx context.Context, uid uuid.UUID, id uuid.UUID) (*GitIdentity, error)
	Add(ctx context.Context, uid uuid.UUID, req *AddGitIdentityReq) (*GitIdentity, error)
	Update(ctx context.Context, uid uuid.UUID, req *UpdateGitIdentityReq) error
	Delete(ctx context.Context, uid uuid.UUID, id uuid.UUID) error
	ListBranches(ctx context.Context, uid uuid.UUID, identityID uuid.UUID, repoFullName string, page, perPage int) ([]*Branch, error)
	HandleGitHubAppSetup(ctx context.Context, req *GitHubAppSetupReq, uid uuid.UUID) (*GitHubAppSetupResp, error)
	GenerateGitHubAppState(ctx context.Context, uid uuid.UUID) (string, error)
}

// GitIdentityRepo Git 身份认证数据仓库接口
type GitIdentityRepo interface {
	Get(ctx context.Context, id uuid.UUID) (*db.GitIdentity, error)
	GetByUserID(ctx context.Context, uid uuid.UUID, id uuid.UUID) (*db.GitIdentity, error)
	GetByInstallationID(ctx context.Context, installationID int64) (*db.GitIdentity, error)
	List(ctx context.Context, uid uuid.UUID) ([]*db.GitIdentity, error)
	Create(ctx context.Context, uid uuid.UUID, req *AddGitIdentityReq) (*db.GitIdentity, error)
	CreateFromGitHubApp(ctx context.Context, uid uuid.UUID, installationID int64, username, email string) (*db.GitIdentity, error)
	Update(ctx context.Context, uid uuid.UUID, id uuid.UUID, req *UpdateGitIdentityReq) error
	UpdateFromGitHubApp(ctx context.Context, id uuid.UUID, username, email string) error
	Delete(ctx context.Context, uid uuid.UUID, id uuid.UUID) error
	CountProjectsByGitIdentityID(ctx context.Context, id uuid.UUID) (int, error)
}

// AuthRepository 授权仓库信息
type AuthRepository struct {
	FullName    string `json:"full_name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

// GitPlatformClient 各 Git 平台的统一客户端接口
type GitPlatformClient[T any] interface {
	// GetAuthorizedRepositories 获取 PAT 可访问的仓库列表
	GetAuthorizedRepositories(ctx context.Context, token string) ([]T, error)
}

// AuthRepositoryInterface 用于约束平台客户端返回的仓库类型
type AuthRepositoryInterface interface {
	~struct {
		FullName    string
		URL         string
		Description string
	}
}

// GitIdentity Git 身份认证
type GitIdentity struct {
	ID                     uuid.UUID          `json:"id"`
	Platform               consts.GitPlatform `json:"platform"`
	BaseURL                string             `json:"base_url"`
	AccessToken            string             `json:"access_token"`
	Username               string             `json:"username"`
	Email                  string             `json:"email"`
	Remark                 string             `json:"remark"`
	IsInstallationApp      bool               `json:"is_installation_app"`
	AuthorizedRepositories []AuthRepository   `json:"authorized_repositories"`
	CreatedAt              time.Time          `json:"created_at"`
}

// From 从数据库模型转换为领域模型
func (g *GitIdentity) From(src *db.GitIdentity) *GitIdentity {
	if src == nil {
		return g
	}
	g.ID = src.ID
	g.Platform = src.Platform
	g.BaseURL = src.BaseURL
	g.AccessToken = src.AccessToken
	g.Username = src.Username
	g.Email = src.Email
	g.Remark = src.Remark
	g.CreatedAt = src.CreatedAt
	g.IsInstallationApp = src.InstallationID != 0
	return g
}

// AddGitIdentityReq 添加 Git 身份认证请求
type AddGitIdentityReq struct {
	Platform    consts.GitPlatform `json:"platform" validate:"required"`
	BaseURL     string             `json:"base_url" validate:"required"`
	AccessToken string             `json:"access_token" validate:"required"`
	Username    string             `json:"username" validate:"required"`
	Email       string             `json:"email" validate:"required"`
	Remark      string             `json:"remark,omitempty"`
}

// UpdateGitIdentityReq 更新 Git 身份认证请求
type UpdateGitIdentityReq struct {
	ID          uuid.UUID           `param:"id" validate:"required" json:"-" swaggerignore:"true"`
	Platform    *consts.GitPlatform `json:"platform,omitempty"`
	BaseURL     *string             `json:"base_url,omitempty"`
	AccessToken *string             `json:"access_token,omitempty"`
	Username    *string             `json:"username,omitempty"`
	Email       *string             `json:"email,omitempty"`
	Remark      *string             `json:"remark,omitempty"`
}

// GetGitIdentityReq 获取 Git 身份认证详情请求
type GetGitIdentityReq struct {
	ID uuid.UUID `param:"id" validate:"required"`
}

// DeleteGitIdentityReq 删除 Git 身份认证请求
type DeleteGitIdentityReq struct {
	ID uuid.UUID `param:"id" validate:"required"`
}

// ListBranchesReq 获取仓库分支列表请求
type ListBranchesReq struct {
	IdentityID          uuid.UUID `param:"identity_id" validate:"required" json:"-" swaggerignore:"true"`
	EscapedRepoFullName string    `param:"escaped_repo_full_name" validate:"required" json:"-" swaggerignore:"true"`
	Page                int       `query:"page" json:"-"`
	PerPage             int       `query:"per_page" json:"-"`
}

// Branch 分支信息
type Branch struct {
	Name string `json:"name"`
}

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

// GitHubAppInstallUrlResp GitHub App 安装 URL 响应
type GitHubAppInstallUrlResp struct {
	Url string `json:"url"`
}
