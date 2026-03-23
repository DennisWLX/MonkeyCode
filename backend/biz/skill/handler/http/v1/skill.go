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

// SkillHandler 技能处理器
type SkillHandler struct {
	usecase         domain.SkillUsecase
	config          *config.Config
	authMiddleware  *middleware.AuthMiddleware
	auditMiddleware *middleware.AuditMiddleware
	logger          *slog.Logger
}

// NewSkillHandler 创建技能处理器 (samber/do 风格)
func NewSkillHandler(i *do.Injector) (*SkillHandler, error) {
	w := do.MustInvoke[*web.Web](i)
	auth := do.MustInvoke[*middleware.AuthMiddleware](i)
	audit := do.MustInvoke[*middleware.AuditMiddleware](i)
	logger := do.MustInvoke[*slog.Logger](i)

	h := &SkillHandler{
		usecase:         do.MustInvoke[domain.SkillUsecase](i),
		config:          do.MustInvoke[*config.Config](i),
		authMiddleware:  auth,
		auditMiddleware: audit,
		logger:          logger.With("module", "handler.skill"),
	}

	// 注册技能相关路由
	skills := w.Group("/api/v1/skills")
	skills.Use(auth.Auth())
	skills.GET("", web.BaseHandler(h.ListSkills))

	return h, nil
}

// ListSkills 获取技能列表
//
//	@Summary	获取已启用的 Skills 列表
//	@Description	获取系统中已启用的技能列表
//	@Tags		【用户】技能管理
//	@Accept		json
//	@Produce	json
//	@Security	MonkeyCodeAuth
//	@Success	200		{object}	web.Resp{data=[]domain.Skill}	"成功"
//	@Failure	401		{object}	web.Resp						"未授权"
//	@Failure	500		{object}	web.Resp						"服务器内部错误"
//	@Router		/api/v1/skills [get]
func (h *SkillHandler) ListSkills(c *web.Context) error {
	user := middleware.GetUser(c)
	if user == nil {
		return errcode.ErrUnauthorized
	}

	skills, err := h.usecase.ListSkills(c.Request().Context())
	if err != nil {
		h.logger.ErrorContext(c.Request().Context(), "list skills failed", "error", err)
		return err
	}

	return c.Success(skills)
}
