package usecase

import (
	"context"

	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/domain"
)

// SkillUsecase 技能业务逻辑实现
type SkillUsecase struct {
	repo domain.SkillRepo
}

// NewSkillUsecase 创建技能业务逻辑
func NewSkillUsecase(i *do.Injector) (domain.SkillUsecase, error) {
	return &SkillUsecase{
		repo: do.MustInvoke[domain.SkillRepo](i),
	}, nil
}

// ListSkills 获取技能列表
func (u *SkillUsecase) ListSkills(ctx context.Context) ([]*domain.Skill, error) {
	return u.repo.ListSkills(ctx)
}
