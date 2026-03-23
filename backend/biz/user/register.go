package user

import (
	"github.com/samber/do"

	v1 "github.com/chaitin/MonkeyCode/backend/biz/user/handler/v1"
	walletHandler "github.com/chaitin/MonkeyCode/backend/biz/user/handler/v1/wallet"
	"github.com/chaitin/MonkeyCode/backend/biz/user/repo"
	walletRepo "github.com/chaitin/MonkeyCode/backend/biz/user/repo/wallet"
	"github.com/chaitin/MonkeyCode/backend/biz/user/usecase"
	walletUsecase "github.com/chaitin/MonkeyCode/backend/biz/user/usecase/wallet"
)

// RegisterUser 注册 user 模块
func RegisterUser(i *do.Injector) {
	// 注册用户相关组件
	do.Provide(i, repo.NewUserRepo)
	do.Provide(i, usecase.NewUserUsecase)
	do.Provide(i, v1.NewAuthHandler)
	do.MustInvoke[*v1.AuthHandler](i)

	// 注册钱包相关组件
	do.Provide(i, walletRepo.NewWalletRepo)
	do.Provide(i, walletUsecase.NewWalletUsecase)
	do.Provide(i, walletHandler.NewWalletHandler)
	do.MustInvoke[*walletHandler.WalletHandler](i)
}
