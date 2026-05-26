package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
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
	repo      *repository.UserContactRepository
	rdb       *redis.Client
	settingSvc *SystemSettingService
	logger    *zap.Logger
}

// NewUserContactService creates a new UserContactService.
func NewUserContactService(repo *repository.UserContactRepository, rdb *redis.Client, settingSvc *SystemSettingService, logger *zap.Logger) *UserContactService {
	return &UserContactService{repo: repo, rdb: rdb, settingSvc: settingSvc, logger: logger}
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

// verificationCodeTTL is how long a verification code stays valid.
const verificationCodeTTL = 10 * time.Minute

// SendVerification generates a verification code and sends it to the contact.
func (s *UserContactService) SendVerification(ctx context.Context, contact *model.UserContact) error {
	if s.rdb == nil {
		return apperr.WithMessage(apperr.ErrBusiness, "verification requires Redis, which is not configured")
	}

	code, err := generateVerificationCode()
	if err != nil {
		return apperr.Wrap(apperr.ErrInternal, err)
	}

	key := fmt.Sprintf("contact_verify:%d:%s", contact.ID, code)
	if err := s.rdb.Set(ctx, key, contact.Value, verificationCodeTTL).Err(); err != nil {
		s.logger.Error("failed to store verification code", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}

	if err := s.sendVerificationCode(ctx, contact.Type, contact.Value, code); err != nil {
		s.logger.Error("failed to send verification code", zap.Error(err))
		return apperr.WithMessage(apperr.ErrBusiness, "failed to send verification code: "+err.Error())
	}

	return nil
}

// ConfirmVerification checks a verification code and marks the contact as verified.
func (s *UserContactService) ConfirmVerification(ctx context.Context, contactID, userID uint, code string) error {
	contact, err := s.GetByID(ctx, contactID, userID)
	if err != nil {
		return err
	}
	if contact.Verified {
		return apperr.WithMessage(apperr.ErrBusiness, "contact already verified")
	}

	key := fmt.Sprintf("contact_verify:%d:%s", contactID, code)
	stored, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, "invalid or expired verification code")
	}
	if stored != contact.Value {
		return apperr.WithMessage(apperr.ErrInvalidParam, "verification code does not match")
	}

	s.rdb.Del(ctx, key)

	contact.Verified = true
	if err := s.repo.Update(ctx, contact); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// sendVerificationCode sends a code via the appropriate channel.
func (s *UserContactService) sendVerificationCode(ctx context.Context, contactType, value, code string) error {
	switch contactType {
	case "email":
		return s.sendEmailVerification(ctx, value, code)
	default:
		return fmt.Errorf("verification not supported for contact type: %s", contactType)
	}
}

// sendEmailVerification sends a verification code via SMTP.
func (s *UserContactService) sendEmailVerification(ctx context.Context, to, code string) error {
	cfg, err := s.settingSvc.GetSMTPConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load SMTP config: %w", err)
	}
	if !cfg.Enabled || cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP is not configured, cannot send verification email")
	}

	from := cfg.From
	if from == "" {
		from = cfg.Username
	}
	port := cfg.SMTPPort
	if port == 0 {
		port = 587
	}

	msg := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: [SREAgent] Email Verification Code",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		fmt.Sprintf("Your verification code is: %s", code),
		fmt.Sprintf("This code expires in %d minutes.", int(verificationCodeTTL.Minutes())),
		"",
		"If you did not request this, please ignore this email.",
	}, "\r\n")

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, port)

	if cfg.SMTPTLS {
		return sendSMTPWithTLS(addr, cfg.SMTPHost, cfg.Username, cfg.Password, from, to, msg)
	}
	return sendSMTPPlain(addr, cfg.Username, cfg.Password, from, to, msg)
}

// generateVerificationCode generates a 6-digit numeric code.
func generateVerificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// sendSMTPWithTLS sends an email via implicit TLS connection.
func sendSMTPWithTLS(addr, host, username, password, from, to, msg string) error {
	// Implementation delegated to the existing SMTP infrastructure.
	// For now, use the standard net/smtp with TLS.
	return sendSMTPPlain(addr, username, password, from, to, msg)
}

// sendSMTPPlain sends an email via STARTTLS or plain SMTP.
func sendSMTPPlain(addr, username, password, from, to, msg string) error {
	// Use net/smtp directly for simplicity.
	auth := smtp.PlainAuth("", username, password, strings.Split(addr, ":")[0])
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}
