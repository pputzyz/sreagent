package handler

import (
	"io"

	"github.com/gin-gonic/gin"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// AlertmanagerImportHandler handles Alertmanager config import requests.
type AlertmanagerImportHandler struct {
	svc *service.AlertmanagerImportService
}

// NewAlertmanagerImportHandler creates a new AlertmanagerImportHandler.
func NewAlertmanagerImportHandler(svc *service.AlertmanagerImportService) *AlertmanagerImportHandler {
	return &AlertmanagerImportHandler{svc: svc}
}

// Import parses an Alertmanager YAML config and imports receivers as Channels
// and inhibit_rules as InhibitionRules.
//
// Accepts:
//   - JSON body: {"yaml": "..."}
//   - Multipart form: file field named "file"
func (h *AlertmanagerImportHandler) Import(c *gin.Context) {
	var yamlContent []byte

	// Try JSON body first
	var jsonReq struct {
		YAML string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&jsonReq); err == nil && jsonReq.YAML != "" {
		yamlContent = []byte(jsonReq.YAML)
	} else {
		// Fall back to multipart file upload
		file, _, fErr := c.Request.FormFile("file")
		if fErr != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "yaml content or file is required"))
			return
		}
		defer file.Close()
		data, readErr := io.ReadAll(file)
		if readErr != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "failed to read uploaded file"))
			return
		}
		yamlContent = data
	}

	userID := GetCurrentUserID(c)

	result, err := h.svc.ImportConfig(c.Request.Context(), yamlContent, userID)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}

// ImportPresets imports Alertmanager inhibit_rules as PresetRule templates
// with category "inhibition".
func (h *AlertmanagerImportHandler) ImportPresets(c *gin.Context) {
	var yamlContent []byte

	var jsonReq struct {
		YAML string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&jsonReq); err == nil && jsonReq.YAML != "" {
		yamlContent = []byte(jsonReq.YAML)
	} else {
		file, _, fErr := c.Request.FormFile("file")
		if fErr != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "yaml content or file is required"))
			return
		}
		defer file.Close()
		data, readErr := io.ReadAll(file)
		if readErr != nil {
			Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "failed to read uploaded file"))
			return
		}
		yamlContent = data
	}

	count, err := h.svc.ImportPresetInhibitionsFromConfig(c.Request.Context(), yamlContent)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, gin.H{"imported": count})
}
