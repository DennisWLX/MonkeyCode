package v1

import (
	"log/slog"

	"github.com/GoYoko/web"
	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/config"
	"github.com/chaitin/MonkeyCode/backend/domain"
	"github.com/chaitin/MonkeyCode/backend/errcode"
	"github.com/chaitin/MonkeyCode/backend/middleware"
)

// TeamAuditHandler 团队审计日志处理器
type TeamAuditHandler struct {
	usecase         domain.AuditUsecase
	config          *config.Config
	authMiddleware  *middleware.AuthMiddleware
	auditMiddleware *middleware.AuditMiddleware
	logger          *slog.Logger
}

// NewTeamAuditHandler 创建团队审计日志处理器 (samber/do 风格)
func NewTeamAuditHandler(i *do.Injector) (*TeamAuditHandler, error) {
	w := do.MustInvoke[*web.Web](i)
	auth := do.MustInvoke[*middleware.AuthMiddleware](i)
	audit := do.MustInvoke[*middleware.AuditMiddleware](i)
	logger := do.MustInvoke[*slog.Logger](i)

	h := &TeamAuditHandler{
		usecase:         do.MustInvoke[domain.AuditUsecase](i),
		config:          do.MustInvoke[*config.Config](i),
		authMiddleware:  auth,
		auditMiddleware: audit,
		logger:          logger.With("module", "handler.team_audit"),
	}

	// 注册审计日志相关路由
	audits := w.Group("/api/v1/teams/audits")
	audits.Use(auth.TeamAuth())
	audits.GET("", web.BindHandler(h.ListAudits))

	return h, nil
}

// ListAudits 获取审计日志列表
//
//	@Summary	获取审计日志列表
//	@Description	获取团队审计日志列表，支持分页
//	@Tags		【Team 管理员】审计日志
//	@Accept		json
//	@Produce	json
//	@Security	MonkeyCodeAITeamAuth
//	@Param		limit	query		int		false	"每页数量"
//	@Param		cursor	query		string	false	"分页游标"
//	@Success	200		{object}	web.Resp{data=domain.ListAuditsResponse}	"成功"
//	@Failure	401		{object}	web.Resp									"未授权"
//	@Failure	500		{object}	web.Resp									"服务器内部错误"
//	@Router		/api/v1/teams/audits [get]
func (h *TeamAuditHandler) ListAudits(c *web.Context, req domain.ListAuditsRequest) error {
	teamUser := middleware.GetTeamUser(c)
	if teamUser == nil {
		return errcode.ErrUnauthorized
	}

	resp, err := h.usecase.ListAudits(c.Request().Context(), teamUser, &req)
	if err != nil {
		h.logger.ErrorContext(c.Request().Context(), "list audits failed", "error", err)
		return err
	}

	return c.Success(resp)
}
