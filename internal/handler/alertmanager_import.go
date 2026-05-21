package handler

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"

	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

// readYAMLInput extracts YAML content from a request.
// It tries a JSON body with a "yaml" field first, then falls back to a multipart file upload.
func readYAMLInput(c *gin.Context) ([]byte, error) {
	var body struct {
		YAML string `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&body); err == nil && body.YAML != "" {
		return []byte(body.YAML), nil
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("provide JSON body with 'yaml' field or upload a file")
	}
	defer file.Close()
	const maxUploadSize = 10 << 20 // 10 MB
	if header.Size > maxUploadSize {
		return nil, fmt.Errorf("file too large (max 10MB)")
	}
	return io.ReadAll(io.LimitReader(file, maxUploadSize+1))
}

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
	yamlContent, err := readYAMLInput(c)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}

	userID := GetCurrentUserID(c)

	result, err := h.svc.ImportConfig(c.Request.Context(), yamlContent, userID)
	if err != nil {
		Error(c, err)
		return
	}

	Success(c, result)
}
