package domain

import (
	"context"
)

// SkillUsecase 技能业务逻辑接口
type SkillUsecase interface {
	ListSkills(ctx context.Context) ([]*Skill, error)
}

// SkillRepo 技能仓库接口
type SkillRepo interface {
	ListSkills(ctx context.Context) ([]*Skill, error)
}

// Skill 技能定义
type Skill struct {
	ID          string         `json:"id"`
	SkillID     string         `json:"skill_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Content     string         `json:"content"`
	Categories  []string       `json:"categories"`
	Tags        []string       `json:"tags"`
	ArgsSchema  map[string]any `json:"args_schema"`
}
