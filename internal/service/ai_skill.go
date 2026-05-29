package service

import (
	"fmt"
	"time"

	"github.com/sreagent/sreagent/internal/model"
	"github.com/sreagent/sreagent/internal/repository"

	"go.uber.org/zap"
)

type AISkillService struct {
	repo   *repository.AISkillRepository
	logger *zap.Logger
}

func NewAISkillService(repo *repository.AISkillRepository, logger *zap.Logger) *AISkillService {
	return &AISkillService{repo: repo, logger: logger}
}

func (s *AISkillService) Create(skill *model.AISkill) error {
	return s.repo.Create(skill)
}

func (s *AISkillService) GetByID(id uint) (*model.AISkill, error) {
	skill, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	files, err := s.repo.GetFilesBySkillID(id)
	if err != nil {
		s.logger.Warn("failed to load skill files", zap.Error(err))
	} else {
		ptrs := make([]*model.AISkillFile, len(files))
		for i := range files {
			ptrs[i] = &files[i]
		}
		skill.Files = ptrs
	}
	skill.Builtin = skill.CreatedBy == "system"
	return skill, nil
}

func (s *AISkillService) Update(skill *model.AISkill) error {
	existing, err := s.repo.GetByID(skill.ID)
	if err != nil {
		return err
	}
	if existing.CreatedBy == "system" {
		return fmt.Errorf("cannot modify built-in skills")
	}
	skill.CreatedBy = existing.CreatedBy
	skill.CreatedAt = existing.CreatedAt
	return s.repo.Update(skill)
}

func (s *AISkillService) Delete(id uint) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing.CreatedBy == "system" {
		return fmt.Errorf("cannot delete builtin skill")
	}
	return s.repo.Delete(id)
}

func (s *AISkillService) List(search string) ([]model.AISkill, error) {
	skills, err := s.repo.List(search)
	if err != nil {
		return nil, err
	}
	for i := range skills {
		skills[i].Builtin = skills[i].CreatedBy == "system"
	}
	return skills, nil
}

func (s *AISkillService) GetFiles(skillID uint) ([]model.AISkillFile, error) {
	return s.repo.GetFilesBySkillID(skillID)
}

func (s *AISkillService) GetFile(fileID uint) (*model.AISkillFile, error) {
	return s.repo.GetFileByID(fileID)
}

func (s *AISkillService) AddFile(skillID uint, file *model.AISkillFile) error {
	file.SkillID = skillID
	file.Size = int64(len(file.Content))
	return s.repo.CreateFile(file)
}

func (s *AISkillService) DeleteFile(fileID uint) error {
	return s.repo.DeleteFile(fileID)
}

func (s *AISkillService) ImportSkill(skill *model.AISkill, files []model.AISkillFile) error {
	now := time.Now()
	skill.CreatedAt = now
	skill.UpdatedAt = now
	if err := s.repo.Create(skill); err != nil {
		return err
	}
	for i := range files {
		files[i].SkillID = skill.ID
		files[i].Size = int64(len(files[i].Content))
		files[i].CreatedAt = now
		files[i].UpdatedAt = now
		if err := s.repo.CreateFile(&files[i]); err != nil {
			return err
		}
	}
	return nil
}
