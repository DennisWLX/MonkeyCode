package repo

import (
	"context"

	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/domain"
)

// SkillRepo 技能仓库实现
type SkillRepo struct {
	// 这里可以添加数据库连接等依赖
}

// NewSkillRepo 创建技能仓库
func NewSkillRepo(i *do.Injector) (domain.SkillRepo, error) {
	return &SkillRepo{}, nil
}

// ListSkills 获取技能列表
func (r *SkillRepo) ListSkills(ctx context.Context) ([]*domain.Skill, error) {
	// 返回示例技能数据
	return []*domain.Skill{
		{
			ID:          "1",
			SkillID:     "skill-1",
			Name:        "代码分析",
			Description: "分析代码结构和功能",
			Content:     "用于分析代码的技能",
			Categories:  []string{"开发", "分析"},
			Tags:        []string{"代码", "分析"},
			ArgsSchema:  map[string]any{"file_path": "string"},
		},
		{
			ID:          "2",
			SkillID:     "skill-2",
			Name:        "文档生成",
			Description: "生成代码文档",
			Content:     "用于生成代码文档的技能",
			Categories:  []string{"开发", "文档"},
			Tags:        []string{"文档", "生成"},
			ArgsSchema:  map[string]any{"dir_path": "string"},
		},
		{
			ID:          "3",
			SkillID:     "skill-3",
			Name:        "测试生成",
			Description: "生成单元测试",
			Content:     "用于生成单元测试的技能",
			Categories:  []string{"开发", "测试"},
			Tags:        []string{"测试", "生成"},
			ArgsSchema:  map[string]any{"file_path": "string"},
		},
	}, nil
}
