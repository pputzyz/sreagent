package service

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

type PetService struct {
	repo   *repository.PetRepository
	logger *zap.Logger
}

func NewPetService(repo *repository.PetRepository, logger *zap.Logger) *PetService {
	return &PetService{repo: repo, logger: logger}
}

// GetOrCreate returns the user's pet, creating a default fox if none exists.
func (s *PetService) GetOrCreate(ctx context.Context, userID uint) (*model.Pet, error) {
	pet, err := s.repo.GetByUserID(ctx, userID)
	if err == nil {
		return pet, nil
	}
	if err != gorm.ErrRecordNotFound {
		s.logger.Error("failed to get pet", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	// Create default pet
	pet = &model.Pet{
		UserID:  userID,
		Name:    "小狐",
		Species: "fox",
		Level:   1,
		Exp:     0,
		Hunger:  30,
		Mood:    70,
	}
	if err := s.repo.Create(ctx, pet); err != nil {
		s.logger.Error("failed to create pet", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return pet, nil
}

// Update saves changes to a pet.
func (s *PetService) Update(ctx context.Context, pet *model.Pet) error {
	if err := s.repo.Update(ctx, pet); err != nil {
		s.logger.Error("failed to update pet", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Feed reduces hunger by 20 (min 0), adds 5 exp, and logs the interaction.
func (s *PetService) Feed(ctx context.Context, userID uint) (*model.Pet, error) {
	// Ensure pet exists
	pet, err := s.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Atomic update
	if err := s.repo.FeedAtomic(ctx, userID, 20, 5); err != nil {
		s.logger.Error("failed to feed pet", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Log interaction
	if err := s.repo.CreateInteraction(ctx, &model.PetInteraction{
		PetID: pet.ID,
		Type:  "feed",
		Value: 5,
	}); err != nil {
		s.logger.Warn("failed to log feed interaction", zap.Error(err))
	}

	// Re-fetch to get current state and handle level-up
	pet, err = s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.checkLevelUp(pet)
	if err := s.repo.Update(ctx, pet); err != nil {
		s.logger.Warn("failed to save level-up", zap.Error(err))
	}

	return pet, nil
}

// Play increases mood by 15 (max 100), adds 5 exp, and logs the interaction.
func (s *PetService) Play(ctx context.Context, userID uint) (*model.Pet, error) {
	// Ensure pet exists
	pet, err := s.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Atomic update
	if err := s.repo.PlayAtomic(ctx, userID, 15, 5); err != nil {
		s.logger.Error("failed to play with pet", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Log interaction
	if err := s.repo.CreateInteraction(ctx, &model.PetInteraction{
		PetID: pet.ID,
		Type:  "play",
		Value: 5,
	}); err != nil {
		s.logger.Warn("failed to log play interaction", zap.Error(err))
	}

	// Re-fetch to get current state and handle level-up
	pet, err = s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.checkLevelUp(pet)
	if err := s.repo.Update(ctx, pet); err != nil {
		s.logger.Warn("failed to save level-up", zap.Error(err))
	}

	return pet, nil
}

// AddChatExp adds 2 exp for chatting with the pet.
func (s *PetService) AddChatExp(ctx context.Context, userID uint) (*model.Pet, error) {
	// Ensure pet exists
	pet, err := s.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Atomic update
	if err := s.repo.AddExpAtomic(ctx, userID, 2); err != nil {
		s.logger.Error("failed to add chat exp", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Log interaction
	if err := s.repo.CreateInteraction(ctx, &model.PetInteraction{
		PetID: pet.ID,
		Type:  "chat",
		Value: 2,
	}); err != nil {
		s.logger.Warn("failed to log chat interaction", zap.Error(err))
	}

	// Re-fetch to get current state and handle level-up
	pet, err = s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.checkLevelUp(pet)
	if err := s.repo.Update(ctx, pet); err != nil {
		s.logger.Warn("failed to save level-up", zap.Error(err))
	}

	return pet, nil
}

// GetInteractions returns interaction history for the user's pet.
func (s *PetService) GetInteractions(ctx context.Context, userID uint, limit int) ([]model.PetInteraction, error) {
	pet, err := s.GetOrCreate(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListInteractions(ctx, pet.ID, limit)
}

// checkLevelUp levels up the pet if exp >= level * 100.
func (s *PetService) checkLevelUp(pet *model.Pet) {
	required := pet.Level * 100
	for pet.Exp >= required {
		pet.Exp -= required
		pet.Level++
		required = pet.Level * 100
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
