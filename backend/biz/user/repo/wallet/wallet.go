package wallet

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/domain"
)

// WalletRepo 钱包仓库实现
type WalletRepo struct {
	// 这里可以添加数据库连接等依赖
	wallets       map[uuid.UUID]*domain.Wallet
	transactions  map[uuid.UUID][]*domain.TransactionLog
}

// NewWalletRepo 创建钱包仓库
func NewWalletRepo(i *do.Injector) (domain.WalletRepo, error) {
	return &WalletRepo{
		wallets:      make(map[uuid.UUID]*domain.Wallet),
		transactions: make(map[uuid.UUID][]*domain.TransactionLog),
	}, nil
}

// GetWallet 获取用户钱包
func (r *WalletRepo) GetWallet(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error) {
	// 检查是否已存在钱包
	if wallet, ok := r.wallets[userID]; ok {
		return wallet, nil
	}

	// 创建新钱包
	wallet := &domain.Wallet{
		ID:      uuid.New(),
		UserID:  userID,
		Balance: 100.0,  // 初始余额
		Bonus:   50.0,   // 初始赠送余额
	}

	r.wallets[userID] = wallet
	return wallet, nil
}

// UpdateWallet 更新钱包
func (r *WalletRepo) UpdateWallet(ctx context.Context, wallet *domain.Wallet) error {
	r.wallets[wallet.UserID] = wallet
	return nil
}

// CreateTransaction 创建交易记录
func (r *WalletRepo) CreateTransaction(ctx context.Context, transaction *domain.TransactionLog) error {
	userID := transaction.UserID
	if _, ok := r.transactions[userID]; !ok {
		r.transactions[userID] = []*domain.TransactionLog{}
	}
	r.transactions[userID] = append(r.transactions[userID], transaction)
	return nil
}

// ListTransactions 获取交易记录列表
func (r *WalletRepo) ListTransactions(ctx context.Context, userID uuid.UUID, req *domain.ListTransactionRequest) ([]*domain.TransactionLog, *domain.PageInfo, error) {
	// 检查是否有交易记录
	if _, ok := r.transactions[userID]; !ok {
		// 创建示例交易记录
		r.createSampleTransactions(userID)
	}

	transactions := r.transactions[userID]
	totalCount := int64(len(transactions))

	// 分页处理
	page := 1
	size := 10
	if req.Page != nil && *req.Page > 0 {
		page = *req.Page
	}
	if req.Size != nil && *req.Size > 0 {
		size = *req.Size
	}

	start := (page - 1) * size
	end := start + size
	if start >= len(transactions) {
		return []*domain.TransactionLog{}, &domain.PageInfo{
			HasNextPage: false,
			NextToken:   "",
			TotalCount:  totalCount,
		}, nil
	}

	if end > len(transactions) {
		end = len(transactions)
	}

	hasNextPage := end < len(transactions)
	nextToken := ""
	if hasNextPage {
		nextToken = transactions[end-1].ID.String()
	}

	return transactions[start:end], &domain.PageInfo{
		HasNextPage: hasNextPage,
		NextToken:   nextToken,
		TotalCount:  totalCount,
	}, nil
}

// createSampleTransactions 创建示例交易记录
func (r *WalletRepo) createSampleTransactions(userID uuid.UUID) {
	now := time.Now()
	transactions := []*domain.TransactionLog{
		{
			ID:              uuid.New(),
			UserID:          userID,
			Kind:            "signup_bonus",
			Amount:          50.0,
			AmountPrincipal: 0.0,
			AmountBonus:     50.0,
			Remark:          "注册奖励",
			CreatedAt:       now.Add(-7 * 24 * time.Hour),
		},
		{
			ID:              uuid.New(),
			UserID:          userID,
			Kind:            "voucher_exchange",
			Amount:          100.0,
			AmountPrincipal: 100.0,
			AmountBonus:     0.0,
			Remark:          "兑换码兑换",
			CreatedAt:       now.Add(-3 * 24 * time.Hour),
		},
		{
			ID:              uuid.New(),
			UserID:          userID,
			Kind:            "model_consumption",
			Amount:          10.0,
			AmountPrincipal: 10.0,
			AmountBonus:     0.0,
			Remark:          "模型使用消耗",
			CreatedAt:       now.Add(-1 * 24 * time.Hour),
		},
	}
	r.transactions[userID] = transactions
}
