package v1

import (
	"fmt"
	"log/slog"
	"net/url"

	"github.com/GoYoko/web"
	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/config"
	"github.com/chaitin/MonkeyCode/backend/domain"
	"github.com/chaitin/MonkeyCode/backend/errcode"
	"github.com/chaitin/MonkeyCode/backend/middleware"
)

// GitIdentityHandler Git 身份认证处理器
type GitIdentityHandler struct {
	cfg     *config.Config
	usecase domain.GitIdentityUsecase
	logger  *slog.Logger
}

// NewGitIdentityHandler 创建 Git 身份认证处理器
func NewGitIdentityHandler(i *do.Injector) (*GitIdentityHandler, error) {
	w := do.MustInvoke[*web.Web](i)
	auth := do.MustInvoke[*middleware.AuthMiddleware](i)
	cfg := do.MustInvoke[*config.Config](i)

	h := &GitIdentityHandler{
		cfg:     cfg,
		usecase: do.MustInvoke[domain.GitIdentityUsecase](i),
		logger:  do.MustInvoke[*slog.Logger](i).With("module", "handler.git_identity"),
	}

	g := w.Group("/api/v1/users/git-identities")
	g.Use(auth.Auth())
	g.GET("", web.BaseHandler(h.List))
	g.GET("/:id", web.BindHandler(h.Get))
	g.POST("", web.BindHandler(h.Add))
	g.PUT("/:id", web.BindHandler(h.Update))
	g.DELETE("/:id", web.BindHandler(h.Delete))
	g.GET("/:identity_id/:escaped_repo_full_name/branches", web.BindHandler(h.ListBranches))
	g.GET("/github-app/install-url", web.BaseHandler(h.GetGitHubAppInstallUrl))

	return h, nil
}

// List 获取当前用户的 Git 身份认证列表
func (h *GitIdentityHandler) List(c *web.Context) error {
	user := middleware.GetUser(c)
	list, err := h.usecase.List(c.Request().Context(), user.ID)
	if err != nil {
		return errcode.ErrDatabaseQuery.Wrap(err)
	}
	return c.Success(list)
}

// Get 获取单个 Git 身份认证详情
func (h *GitIdentityHandler) Get(c *web.Context, req domain.GetGitIdentityReq) error {
	user := middleware.GetUser(c)
	identity, err := h.usecase.Get(c.Request().Context(), user.ID, req.ID)
	if err != nil {
		return err
	}
	return c.Success(identity)
}

// Add 添加 Git 身份认证
func (h *GitIdentityHandler) Add(c *web.Context, req domain.AddGitIdentityReq) error {
	user := middleware.GetUser(c)
	identity, err := h.usecase.Add(c.Request().Context(), user.ID, &req)
	if err != nil {
		return errcode.ErrDatabaseQuery.Wrap(err)
	}
	return c.Success(identity)
}

// Update 更新 Git 身份认证
func (h *GitIdentityHandler) Update(c *web.Context, req domain.UpdateGitIdentityReq) error {
	user := middleware.GetUser(c)
	if err := h.usecase.Update(c.Request().Context(), user.ID, &req); err != nil {
		return errcode.ErrDatabaseQuery.Wrap(err)
	}
	return c.Success(nil)
}

// Delete 删除 Git 身份认证
func (h *GitIdentityHandler) Delete(c *web.Context, req domain.DeleteGitIdentityReq) error {
	user := middleware.GetUser(c)
	if err := h.usecase.Delete(c.Request().Context(), user.ID, req.ID); err != nil {
		return err
	}
	return c.Success(nil)
}

// ListBranches 获取仓库分支列表
func (h *GitIdentityHandler) ListBranches(c *web.Context, req domain.ListBranchesReq) error {
	user := middleware.GetUser(c)

	repoFullName, err := url.PathUnescape(req.EscapedRepoFullName)
	if err != nil {
		return errcode.ErrInvalidParameter.Wrap(err)
	}

	branches, err := h.usecase.ListBranches(c.Request().Context(), user.ID, req.IdentityID, repoFullName, req.Page, req.PerPage)
	if err != nil {
		return err
	}
	return c.Success(branches)
}

// GetGitHubAppInstallUrl 获取 GitHub App 安装 URL
//
//	@Summary		获取 GitHub App 安装 URL
//	@Description	生成带有 state 参数的 GitHub App 安装 URL
//	@Tags			【Git】身份认证
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	domain.GitHubAppInstallUrlResp
//	@Router			/api/v1/users/git-identities/github-app/install-url [get]
func (h *GitIdentityHandler) GetGitHubAppInstallUrl(c *web.Context) error {
	user := middleware.GetUser(c)

	state, err := h.usecase.GenerateGitHubAppState(c.Request().Context(), user.ID)
	if err != nil {
		h.logger.Error("failed to generate github app state", "error", err, "user_id", user.ID)
		return errcode.ErrInternalServer.Wrap(err)
	}

	baseURL := h.cfg.Github.AppInstallUrl
	if baseURL == "" {
		baseURL = "https://github.com/apps/dennis-code-dev/installations/new"
	}

	installURL := fmt.Sprintf("%s?state=%s", baseURL, url.QueryEscape(state))

	return c.Success(&domain.GitHubAppInstallUrlResp{
		Url: installURL,
	})
}
