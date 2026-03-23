package repo

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/samber/do"

	"github.com/chaitin/MonkeyCode/backend/domain"
)

type SkillManifest struct {
	Skills []SkillEntry `json:"skills"`
}

type SkillEntry struct {
	SkillID      string            `json:"skill_id"`
	Name         map[string]string `json:"name"`
	Description  map[string]string `json:"description"`
	Categories   []string          `json:"categories"`
	Tags         []string          `json:"tags"`
	SkillMDPath  string            `json:"skill_md_path"`
	IsEnabled    bool              `json:"is_enabled"`
}

type SkillRepo struct {
	skillPath string
}

func NewSkillRepo(i *do.Injector) (domain.SkillRepo, error) {
	skillPath := os.Getenv("SKILL_PATH")
	if skillPath == "" {
		homeDir, _ := os.UserHomeDir()
		skillPath = filepath.Join(homeDir, "dennis", "project", "MonkeyCodeOfficialPlugins", "skills")
	}
	return &SkillRepo{skillPath: skillPath}, nil
}

func (r *SkillRepo) ListSkills(ctx context.Context) ([]*domain.Skill, error) {
	manifestPath := filepath.Join(r.skillPath, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest SkillManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}

	skills := make([]*domain.Skill, 0, len(manifest.Skills))
	for i, entry := range manifest.Skills {
		if !entry.IsEnabled {
			continue
		}

		name := entry.Name["zh"]
		if name == "" {
			name = entry.Name["en"]
		}

		desc := entry.Description["zh"]
		if desc == "" {
			desc = entry.Description["en"]
		}

		content, _ := r.loadSkillContent(entry.SkillMDPath)

		skill := &domain.Skill{
			ID:          entry.SkillID,
			SkillID:     entry.SkillID,
			Name:        name,
			Description: desc,
			Content:     content,
			Categories:  entry.Categories,
			Tags:        entry.Tags,
			ArgsSchema:  nil,
		}
		_ = i
		skills = append(skills, skill)
	}

	return skills, nil
}

func (r *SkillRepo) loadSkillContent(skillMDPath string) (string, error) {
	fullPath := filepath.Join(r.skillPath, skillMDPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
