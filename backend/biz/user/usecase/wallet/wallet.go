package wallet

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/domain"
)

// WalletUsecase 钱包业务逻辑实现
type WalletUsecase struct {
	repo domain.WalletRepo
}

// NewWalletUsecase 创建钱包业务逻辑
func NewWalletUsecase(i *do.Injector) (domain.WalletUsecase, error) {
	return &WalletUsecase{
		repo: do.MustInvoke[domain.WalletRepo](i),
	}, nil
}

// GetWallet 获取用户钱包
func (u *WalletUsecase) GetWallet(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error) {
	return u.repo.GetWallet(ctx, userID)
}

// ExchangeCode 兑换码兑换
func (u *WalletUsecase) ExchangeCode(ctx context.Context, userID uuid.UUID, code string) error {
	// 获取用户钱包
	wallet, err := u.repo.GetWallet(ctx, userID)
	if err != nil {
		return err
	}

	// 模拟兑换码兑换，这里可以根据实际情况添加兑换码验证逻辑
	// 假设兑换码有效，添加 100 元到余额
	wallet.Balance += 100.0

	// 更新钱包
	err = u.repo.UpdateWallet(ctx, wallet)
	if err != nil {
		return err
	}

	// 创建交易记录
	transaction := &domain.TransactionLog{
		ID:              uuid.New(),
		UserID:          userID,
		Kind:            "voucher_exchange",
		Amount:          100.0,
		AmountPrincipal: 100.0,
		AmountBonus:     0.0,
		Remark:          "兑换码兑换",
		CreatedAt:       time.Now(),
	}

	return u.repo.CreateTransaction(ctx, transaction)
}

// ListTransactions 获取交易记录列表
func (u *WalletUsecase) ListTransactions(ctx context.Context, userID uuid.UUID, req *domain.ListTransactionRequest) (*domain.ListTransactionResponse, error) {
	transactions, page, err := u.repo.ListTransactions(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	return &domain.ListTransactionResponse{
		Transactions: transactions,
		Page:         page,
	}, nil
}
