package service

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

const maxContactsPerType = 5

var validContactTypes = map[string]bool{
	"email":    true,
	"phone":    true,
	"feishu":   true,
	"wecom":    true,
	"dingtalk": true,
	"webhook":  true,
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var phoneRegex = regexp.MustCompile(`^\+?[0-9\-]{5,20}$`)

// UserContactService provides CRUD operations for user contacts.
type UserContactService struct {
	repo   *repository.UserContactRepository
	logger *zap.Logger
}

// NewUserContactService creates a new UserContactService.
func NewUserContactService(repo *repository.UserContactRepository, logger *zap.Logger) *UserContactService {
	return &UserContactService{repo: repo, logger: logger}
}

// List returns all contacts for a user.
func (s *UserContactService) List(ctx context.Context, userID uint) ([]model.UserContact, error) {
	contacts, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to list user contacts", zap.Uint("user_id", userID), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return contacts, nil
}

// Create creates a new user contact with validation.
func (s *UserContactService) Create(ctx context.Context, contact *model.UserContact) error {
	if err := s.validate(contact); err != nil {
		return err
	}

	// Check uniqueness.
	existing, _ := s.repo.GetByUserAndValue(ctx, contact.UserID, contact.Type, contact.Value)
	if existing != nil {
		return apperr.WithMessage(apperr.ErrConflict, "contact already exists")
	}

	// Enforce max contacts per type.
	count, err := s.repo.CountByUserAndType(ctx, contact.UserID, contact.Type)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	if count >= maxContactsPerType {
		return apperr.WithMessage(apperr.ErrBusiness, "maximum contacts per type reached (5)")
	}

	// If this is the first contact of its type, make it default.
	if count == 0 {
		contact.IsDefault = true
	}

	if err := s.repo.Create(ctx, contact); err != nil {
		s.logger.Error("failed to create user contact", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Update updates an existing user contact.
func (s *UserContactService) Update(ctx context.Context, contact *model.UserContact) error {
	existing, err := s.repo.GetByID(ctx, contact.ID)
	if err != nil {
		return apperr.ErrNotFound
	}

	if err := s.validate(contact); err != nil {
		return err
	}

	// If type or value changed, check uniqueness.
	if existing.Type != contact.Type || existing.Value != contact.Value {
		dup, _ := s.repo.GetByUserAndValue(ctx, existing.UserID, contact.Type, contact.Value)
		if dup != nil && dup.ID != contact.ID {
			return apperr.WithMessage(apperr.ErrConflict, "contact already exists")
		}
	}

	existing.Type = contact.Type
	existing.Value = contact.Value
	existing.Name = contact.Name

	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to update user contact", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes a user contact. Only the owner can delete.
func (s *UserContactService) Delete(ctx context.Context, id, userID uint) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrNotFound
	}
	if existing.UserID != userID {
		return apperr.ErrForbidden
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete user contact", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// SetDefault sets a contact as the default for its type. Only the owner can set.
func (s *UserContactService) SetDefault(ctx context.Context, id, userID uint) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrNotFound
	}
	if existing.UserID != userID {
		return apperr.ErrForbidden
	}

	// Clear current default for this type.
	if err := s.repo.ClearDefault(ctx, userID, existing.Type); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	existing.IsDefault = true
	if err := s.repo.Update(ctx, existing); err != nil {
		s.logger.Error("failed to set default user contact", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns a contact by ID. Only the owner can access.
func (s *UserContactService) GetByID(ctx context.Context, id, userID uint) (*model.UserContact, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	if existing.UserID != userID {
		return nil, apperr.ErrForbidden
	}
	return existing, nil
}

// validate validates a user contact.
func (s *UserContactService) validate(contact *model.UserContact) error {
	contact.Type = strings.TrimSpace(contact.Type)
	contact.Value = strings.TrimSpace(contact.Value)
	contact.Name = strings.TrimSpace(contact.Name)

	if !validContactTypes[contact.Type] {
		return apperr.WithMessage(apperr.ErrInvalidParam, "invalid contact type, must be one of: email, phone, feishu, wecom, dingtalk, webhook")
	}
	if contact.Value == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "contact value is required")
	}

	switch contact.Type {
	case "email":
		if !emailRegex.MatchString(contact.Value) {
			return apperr.WithMessage(apperr.ErrInvalidParam, "invalid email format")
		}
	case "phone":
		if !phoneRegex.MatchString(contact.Value) {
			return apperr.WithMessage(apperr.ErrInvalidParam, "invalid phone format")
		}
	case "webhook":
		u, err := url.Parse(contact.Value)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return apperr.WithMessage(apperr.ErrInvalidParam, "invalid webhook URL")
		}
	}

	if contact.Name == "" {
		contact.Name = contact.Type
	}

	return nil
}
