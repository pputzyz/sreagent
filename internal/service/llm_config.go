package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/pkg/safehttp"
	"github.com/sreagent/sreagent/internal/repository"

	"github.com/sreagent/sreagent/internal/pkg/crypto"
)

// LLMConfigService provides business logic for LLM configs.
type LLMConfigService struct {
	repo   *repository.LLMConfigRepository
	db     *gorm.DB
	client *http.Client
	logger *zap.Logger
}

// NewLLMConfigService creates a new LLMConfigService.
func NewLLMConfigService(
	repo *repository.LLMConfigRepository,
	db *gorm.DB,
	logger *zap.Logger,
) *LLMConfigService {
	return &LLMConfigService{
		repo:   repo,
		db:     db,
		client: safehttp.NewSafeClient(30 * time.Second),
		logger: logger,
	}
}

// Create creates a new LLM config. Encrypts API key and clears other defaults
// if this config is marked as default.
func (s *LLMConfigService) Create(ctx context.Context, v *model.LLMConfig) error {
	if err := v.Verify(); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
	}

	// Encrypt API key if present and not already masked
	if v.APIKey != "" && !model.IsMaskedAPIKey(v.APIKey) && !crypto.IsEncrypted(v.APIKey) {
		enc, err := crypto.EncryptString(v.APIKey)
		if err != nil {
			s.logger.Error("failed to encrypt API key", zap.Error(err))
			return apperr.Wrap(apperr.ErrInternal, err)
		}
		v.APIKey = enc
	}

	if v.IsDefault {
		if err := s.clearDefaultInTx(ctx); err != nil {
			return err
		}
	}

	if err := s.repo.Create(ctx, v); err != nil {
		s.logger.Error("failed to create LLM config", zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// GetByID returns an LLM config by its ID. The API key is decrypted for
// internal use; callers that return it to the frontend should mask it.
func (s *LLMConfigService) GetByID(ctx context.Context, id uint) (*model.LLMConfig, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to get LLM config", zap.Uint("id", id), zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrNotFound, err)
	}
	s.decryptKey(v)
	return v, nil
}

// Update updates an existing LLM config. If the incoming API key is masked,
// the existing encrypted key is preserved.
func (s *LLMConfigService) Update(ctx context.Context, existing *model.LLMConfig, input *model.LLMConfig) error {
	if err := input.Verify(); err != nil {
		return apperr.WithMessage(apperr.ErrInvalidParam, err.Error())
	}

	// Preserve immutable fields
	input.ID = existing.ID
	input.CreatedAt = existing.CreatedAt
	input.CreatedBy = existing.CreatedBy

	// If the incoming API key is masked, keep the existing encrypted key
	if model.IsMaskedAPIKey(input.APIKey) {
		input.APIKey = existing.APIKey
	} else if input.APIKey != "" && !crypto.IsEncrypted(input.APIKey) {
		// Encrypt a new plaintext key
		enc, err := crypto.EncryptString(input.APIKey)
		if err != nil {
			s.logger.Error("failed to encrypt API key", zap.Error(err))
			return apperr.Wrap(apperr.ErrInternal, err)
		}
		input.APIKey = enc
	}

	if input.IsDefault {
		if err := s.clearDefaultInTx(ctx); err != nil {
			return err
		}
	}

	if err := s.repo.Update(ctx, input); err != nil {
		s.logger.Error("failed to update LLM config", zap.Uint("id", existing.ID), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// Delete deletes an LLM config by ID.
func (s *LLMConfigService) Delete(ctx context.Context, id uint) error {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		s.logger.Error("failed to delete LLM config", zap.Uint("id", id), zap.Error(err))
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

// List returns a paginated list of LLM configs. API keys are masked.
func (s *LLMConfigService) List(ctx context.Context, page, pageSize int) ([]model.LLMConfig, int64, error) {
	list, total, err := s.repo.List(ctx, page, pageSize)
	if err != nil {
		s.logger.Error("failed to list LLM configs", zap.Error(err))
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	// Mask API keys before returning
	for i := range list {
		s.decryptKey(&list[i])
		list[i].APIKey = list[i].MaskAPIKey()
	}
	return list, total, nil
}

// PickDefault returns the default LLM config (decrypted).
func (s *LLMConfigService) PickDefault(ctx context.Context) (*model.LLMConfig, error) {
	v, err := s.repo.PickDefault(ctx)
	if err != nil {
		s.logger.Error("failed to pick default LLM config", zap.Error(err))
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	if v == nil {
		return nil, nil
	}
	s.decryptKey(v)
	return v, nil
}

// TestConnection makes a lightweight API call to verify the config is reachable.
func (s *LLMConfigService) TestConnection(ctx context.Context, v *model.LLMConfig) error {
	s.decryptKey(v)

	apiURL := v.APIURL
	if apiURL == "" {
		return apperr.WithMessage(apperr.ErrInvalidParam, "api_url is required for connection test")
	}

	// Build a minimal chat completions request
	reqBody := map[string]interface{}{
		"model":      v.ModelName,
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		"max_tokens": 1,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	var endpoint string
	switch v.Provider {
	case "ollama":
		endpoint = apiURL + "/v1/chat/completions"
	default:
		endpoint = apiURL + "/v1/chat/completions"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return apperr.Wrap(apperr.ErrExternalAPI, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if v.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+v.APIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return apperr.WithMessage(apperr.ErrExternalAPI, fmt.Sprintf("connection failed: %v", err))
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return apperr.WithMessage(apperr.ErrExternalAPI,
			fmt.Sprintf("LLM API returned HTTP %d", resp.StatusCode))
	}
	return nil
}

// decryptKey decrypts the API key in-place if it is encrypted.
func (s *LLMConfigService) decryptKey(v *model.LLMConfig) {
	if v.APIKey == "" || !crypto.IsEncrypted(v.APIKey) {
		return
	}
	dec, err := crypto.DecryptString(v.APIKey)
	if err != nil {
		s.logger.Warn("failed to decrypt LLM config API key, returning masked",
			zap.Uint("id", v.ID), zap.Error(err))
		v.APIKey = "****"
		return
	}
	v.APIKey = dec
}

// clearDefaultInTx clears all is_default flags inside a transaction.
func (s *LLMConfigService) clearDefaultInTx(ctx context.Context) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.ClearDefault(tx)
	})
}
