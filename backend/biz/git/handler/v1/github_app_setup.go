package v1

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/GoYoko/web"
	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/config"
	"github.com/chaitin/MonkeyCode/backend/domain"
	"github.com/chaitin/MonkeyCode/backend/pkg/crypto"
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
//	@Param			installation_id	query		int		true	"GitHub App 安装 ID"
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

	// 验证 state 参数
	if req.State == "" {
		h.logger.ErrorContext(ctx, "missing state parameter",
			"installation_id", req.InstallationID)
		redirectURL := fmt.Sprintf("%s/console/settings?github_setup=error&reason=missing_state&message=%s",
			h.cfg.Server.BaseURL,
			"缺少必要的 state 参数，请从设置页面重新发起 GitHub App 安装")
		return c.Redirect(http.StatusFound, redirectURL)
	}

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
			"无效的状态参数或已过期，请从设置页面重新发起 GitHub App 安装")
		return c.Redirect(http.StatusFound, redirectURL)
	}

	uid := stateData.UserID

	// 调用业务逻辑处理
	resp, err := h.usecase.HandleGitHubAppSetup(ctx, &req, uid)
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
