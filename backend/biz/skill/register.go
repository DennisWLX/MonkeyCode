package skill

import (
	"github.com/samber/do"

	v1 "github.com/chaitin/MonkeyCode/backend/biz/skill/handler/http/v1"
	"github.com/chaitin/MonkeyCode/backend/biz/skill/repo"
	"github.com/chaitin/MonkeyCode/backend/biz/skill/usecase"
)

// RegisterSkill 注册 skill 模块
func RegisterSkill(i *do.Injector) error {
	// 注册仓库
	do.Provide(i, repo.NewSkillRepo)

	// 注册业务逻辑
	do.Provide(i, usecase.NewSkillUsecase)

	// 注册处理器
	do.Provide(i, v1.NewSkillHandler)

	// 初始化处理器
	do.MustInvoke[*v1.SkillHandler](i)

	return nil
}
