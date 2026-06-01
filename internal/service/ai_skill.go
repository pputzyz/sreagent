package service

import (
	"context"
	"time"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
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

func (s *AISkillService) Create(ctx context.Context, skill *model.AISkill) error {
	return s.repo.Create(ctx, skill)
}

func (s *AISkillService) GetByID(ctx context.Context, id uint) (*model.AISkill, error) {
	skill, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	files, err := s.repo.GetFilesBySkillID(ctx, id)
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

func (s *AISkillService) Update(ctx context.Context, skill *model.AISkill) error {
	existing, err := s.repo.GetByID(ctx, skill.ID)
	if err != nil {
		return err
	}
	if existing.CreatedBy == "system" {
		return apperr.WithMessage(apperr.ErrBusiness, "cannot modify built-in skills")
	}
	skill.CreatedBy = existing.CreatedBy
	skill.CreatedAt = existing.CreatedAt
	return s.repo.Update(ctx, skill)
}

func (s *AISkillService) Delete(ctx context.Context, id uint) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.CreatedBy == "system" {
		return apperr.WithMessage(apperr.ErrBusiness, "cannot delete builtin skill")
	}
	return s.repo.Delete(ctx, id)
}

func (s *AISkillService) List(ctx context.Context, search string) ([]model.AISkill, error) {
	skills, err := s.repo.List(ctx, search)
	if err != nil {
		return nil, err
	}
	for i := range skills {
		skills[i].Builtin = skills[i].CreatedBy == "system"
	}
	return skills, nil
}

func (s *AISkillService) GetFiles(ctx context.Context, skillID uint) ([]model.AISkillFile, error) {
	return s.repo.GetFilesBySkillID(ctx, skillID)
}

func (s *AISkillService) GetFile(ctx context.Context, fileID uint) (*model.AISkillFile, error) {
	return s.repo.GetFileByID(ctx, fileID)
}

func (s *AISkillService) AddFile(ctx context.Context, skillID uint, file *model.AISkillFile) error {
	file.SkillID = skillID
	file.Size = int64(len(file.Content))
	return s.repo.CreateFile(ctx, file)
}

func (s *AISkillService) DeleteFile(ctx context.Context, fileID uint) error {
	return s.repo.DeleteFile(ctx, fileID)
}

func (s *AISkillService) ImportSkill(ctx context.Context, skill *model.AISkill, files []model.AISkillFile) error {
	now := time.Now()
	skill.CreatedAt = now
	skill.UpdatedAt = now
	if err := s.repo.Create(ctx, skill); err != nil {
		return err
	}
	for i := range files {
		files[i].SkillID = skill.ID
		files[i].Size = int64(len(files[i].Content))
		files[i].CreatedAt = now
		files[i].UpdatedAt = now
		if err := s.repo.CreateFile(ctx, &files[i]); err != nil {
			return err
		}
	}
	return nil
}
