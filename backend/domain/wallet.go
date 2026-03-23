package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// WalletUsecase 钱包业务逻辑接口
type WalletUsecase interface {
	GetWallet(ctx context.Context, userID uuid.UUID) (*Wallet, error)
	ExchangeCode(ctx context.Context, userID uuid.UUID, code string) error
	ListTransactions(ctx context.Context, userID uuid.UUID, req *ListTransactionRequest) (*ListTransactionResponse, error)
}

// WalletRepo 钱包仓库接口
type WalletRepo interface {
	GetWallet(ctx context.Context, userID uuid.UUID) (*Wallet, error)
	UpdateWallet(ctx context.Context, wallet *Wallet) error
	CreateTransaction(ctx context.Context, transaction *TransactionLog) error
	ListTransactions(ctx context.Context, userID uuid.UUID, req *ListTransactionRequest) ([]*TransactionLog, *PageInfo, error)
}

// Wallet 钱包定义
type Wallet struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
	Balance float64  `json:"balance"`     // 充值的余额
	Bonus   float64  `json:"bonus"`       // 赠送余额
}

// TransactionLog 交易记录
type TransactionLog struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	Kind          string     `json:"kind"`           // 交易类型
	Amount        float64    `json:"amount"`         // 总金额
	AmountPrincipal float64  `json:"amount_principal"` // 余额变动
	AmountBonus    float64   `json:"amount_bonus"`    // 赠送金额变动
	Remark        string     `json:"remark"`         // 交易简介
	CreatedAt     time.Time  `json:"created_at"`     // 交易时间
}

// ListTransactionRequest 交易记录查询请求
type ListTransactionRequest struct {
	Start     *int64  `query:"start" json:"start"`         // 开始时间戳
	End       *int64  `query:"end" json:"end"`           // 结束时间戳
	Page      *int    `query:"page" json:"page"`         // 分页
	Size      *int    `query:"size" json:"size"`         // 每页多少条记录
	Sort      *string `query:"sort" json:"sort"`         // 根据 created_at 排序A；asc/desc；默认为 desc
	NextToken *string `query:"next_token" json:"next_token"` // 下一页标识
}

// ListTransactionResponse 交易记录查询响应
type ListTransactionResponse struct {
	Transactions []*TransactionLog `json:"transactions"`
	Page         *PageInfo         `json:"page"`
}

// PageInfo 分页信息
type PageInfo struct {
	HasNextPage bool   `json:"has_next_page"`
	NextToken   string `json:"next_token"`
	TotalCount  int64  `json:"total_count"`
}

// ExchangeReq 兑换码请求
type ExchangeReq struct {
	Code string `json:"code" validate:"required"` // 兑换码
}
