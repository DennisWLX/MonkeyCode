package wallet

import (
	"log/slog"

	"github.com/GoYoko/web"
	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/config"
	"github.com/chaitin/MonkeyCode/backend/domain"
	"github.com/chaitin/MonkeyCode/backend/errcode"
	"github.com/chaitin/MonkeyCode/backend/middleware"
)

// WalletHandler 钱包处理器
type WalletHandler struct {
	config         *config.Config
	logger         *slog.Logger
	usecase        domain.WalletUsecase
	authMiddleware *middleware.AuthMiddleware
}

// NewWalletHandler 创建钱包处理器 (samber/do 风格)
func NewWalletHandler(i *do.Injector) (*WalletHandler, error) {
	w := do.MustInvoke[*web.Web](i)
	cfg := do.MustInvoke[*config.Config](i)
	logger := do.MustInvoke[*slog.Logger](i)
	usecase := do.MustInvoke[domain.WalletUsecase](i)
	auth := do.MustInvoke[*middleware.AuthMiddleware](i)

	h := &WalletHandler{
		config:         cfg,
		logger:         logger.With("module", "wallet.handler"),
		usecase:        usecase,
		authMiddleware: auth,
	}

	// 注册钱包相关路由
	v1 := w.Group("/api/v1/users/wallet")
	v1.Use(auth.Auth())

	// 获取钱包信息
	v1.GET("", web.BaseHandler(h.GetWallet))

	// 兑换码兑换
	v1.POST("/exchange", web.BindHandler(h.ExchangeCode))

	// 获取交易记录
	v1.GET("/transaction", web.BindHandler(h.ListTransactions))

	return h, nil
}

// GetWallet 获取用户钱包信息
//
//	@Summary	用户钱包
//	@Description	获取用户钱包信息，包括余额和赠送余额
//	@Tags		【用户】钱包
//	@Accept		json
//	@Produce	json
//	@Security	MonkeyCodeAIAuth
//	@Success	200		{object}	web.Resp{data=domain.Wallet}	"成功"
//	@Failure	401		{object}	web.Resp						"未授权"
//	@Failure	500		{object}	web.Resp						"服务器内部错误"
//	@Router		/api/v1/users/wallet [get]
func (h *WalletHandler) GetWallet(c *web.Context) error {
	user := middleware.GetUser(c)
	if user == nil {
		return errcode.ErrUnauthorized
	}

	wallet, err := h.usecase.GetWallet(c.Request().Context(), user.ID)
	if err != nil {
		h.logger.ErrorContext(c.Request().Context(), "get wallet failed", "error", err)
		return err
	}

	return c.Success(wallet)
}

// ExchangeCode 兑换码兑换
//
//	@Summary	兑现兑换码
//	@Description	使用兑换码兑换余额
//	@Tags		【用户】钱包
//	@Accept		json
//	@Produce	json
//	@Security	MonkeyCodeAIAuth
//	@Param		req	body		domain.ExchangeReq	true	"兑换码请求"
//	@Success	200		{object}	web.Resp{}					"成功"
//	@Failure	401		{object}	web.Resp					"未授权"
//	@Failure	500		{object}	web.Resp					"服务器内部错误"
//	@Router		/api/v1/users/wallet/exchange [post]
func (h *WalletHandler) ExchangeCode(c *web.Context, req domain.ExchangeReq) error {
	user := middleware.GetUser(c)
	if user == nil {
		return errcode.ErrUnauthorized
	}

	err := h.usecase.ExchangeCode(c.Request().Context(), user.ID, req.Code)
	if err != nil {
		h.logger.ErrorContext(c.Request().Context(), "exchange code failed", "error", err)
		return err
	}

	return c.Success(nil)
}

// ListTransactions 获取交易记录列表
//
//	@Summary	交易记录
//	@Description	获取用户的交易记录列表，支持分页和时间范围筛选
//	@Tags		【用户】钱包
//	@Accept		json
//	@Produce	json
//	@Security	MonkeyCodeAIAuth
//	@Param		end		query		int64	false	"结束时间戳"
//	@Param		next_token	query		string	false	"下一页标识"
//	@Param		page		query		int		false	"分页"
//	@Param		size		query		int		false	"每页多少条记录"
//	@Param		sort		query		string	false	"根据 created_at 排序A；asc/desc；默认为 desc"
//	@Param		start		query		int64	false	"开始时间戳"
//	@Success	200		{object}	web.Resp{data=domain.ListTransactionResponse}	"成功"
//	@Failure	401		{object}	web.Resp										"未授权"
//	@Failure	500		{object}	web.Resp										"服务器内部错误"
//	@Router		/api/v1/users/wallet/transaction [get]
func (h *WalletHandler) ListTransactions(c *web.Context, req domain.ListTransactionRequest) error {
	user := middleware.GetUser(c)
	if user == nil {
		return errcode.ErrUnauthorized
	}

	resp, err := h.usecase.ListTransactions(c.Request().Context(), user.ID, &req)
	if err != nil {
		h.logger.ErrorContext(c.Request().Context(), "list transactions failed", "error", err)
		return err
	}

	return c.Success(resp)
}
